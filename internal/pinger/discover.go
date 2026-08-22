package pinger

import (
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/resnostyle/ha-mqtt/internal/lib/ha"
)

var suffixRE = regexp.MustCompile(`_(\d+)$`)

func extractCastUUID(identifiers [][]string) string {
	for _, ident := range identifiers {
		if len(ident) >= 2 && ident[0] == "cast" && ident[1] != "" {
			return ident[1]
		}
	}
	return ""
}

func entityScore(entityID string) (int, int) {
	match := suffixRE.FindStringSubmatch(entityID)
	if len(match) == 2 {
		n, _ := strconv.Atoi(match[1])
		return 1, n
	}
	return 0, 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func DiscoverCastTargets(entities []ha.EntityRegistryEntry, devices []ha.DeviceRegistryEntry, settings Settings) []PingTarget {
	devicesByID := map[string]ha.DeviceRegistryEntry{}
	for _, d := range devices {
		devicesByID[d.ID] = d
	}

	candidates := map[string][]PingTarget{}
	for _, entry := range entities {
		entityID := entry.EntityID
		if !strings.HasPrefix(entityID, "media_player.") {
			continue
		}
		if entry.Platform != "cast" {
			continue
		}
		if entry.DisabledBy != nil {
			continue
		}
		if entry.HiddenBy != nil {
			continue
		}
		if entry.DeviceID == "" {
			continue
		}
		device, ok := devicesByID[entry.DeviceID]
		if !ok {
			continue
		}
		manufacturer := device.Manufacturer
		model := device.Model
		if len(settings.PingManufacturerFilter) > 0 {
			if _, allowed := settings.PingManufacturerFilter[manufacturer]; !allowed {
				continue
			}
		}
		if _, excluded := settings.PingExcludeModels[model]; excluded {
			continue
		}
		castUUID := extractCastUUID(device.Identifiers)
		if castUUID == "" {
			slog.Debug("skipping entity: no cast UUID", "entity", entityID)
			continue
		}
		friendly := firstNonEmpty(deref(entry.Name), deref(entry.OriginalName), device.NameByUser, device.Name, entityID)
		slugBase := strings.TrimPrefix(entityID, "media_player.")
		target := PingTarget{
			EntityID:     entityID,
			DeviceID:     entry.DeviceID,
			Slug:         Slugify(slugBase),
			FriendlyName: friendly,
			CastUUID:     castUUID,
			Manufacturer: manufacturer,
			Model:        model,
			AreaID:       firstNonEmpty(deref(entry.AreaID), deref(device.AreaID)),
			Host:         settings.PingHostOverrides[entityID],
		}
		candidates[entry.DeviceID] = append(candidates[entry.DeviceID], target)
	}

	selected := make([]PingTarget, 0, len(candidates))
	for deviceID, group := range candidates {
		sort.Slice(group, func(i, j int) bool {
			ai, an := entityScore(group[i].EntityID)
			bi, bn := entityScore(group[j].EntityID)
			if ai != bi {
				return ai < bi
			}
			if an != bn {
				return an < bn
			}
			return group[i].EntityID < group[j].EntityID
		})
		chosen := group[0]
		if len(group) > 1 {
			others := make([]string, 0, len(group)-1)
			for _, t := range group[1:] {
				others = append(others, t.EntityID)
			}
			slog.Debug("deduped cast device", "device", deviceID, "chosen", chosen.EntityID, "others", others)
		}
		selected = append(selected, chosen)
	}
	sort.Slice(selected, func(i, j int) bool {
		return strings.ToLower(selected[i].FriendlyName) < strings.ToLower(selected[j].FriendlyName)
	})
	slog.Info("discovered cast targets from home assistant", "count", len(selected))
	return selected
}

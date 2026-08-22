package pinger

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/resnostyle/ha-mqtt/internal/lib/env"
)

type PingTarget struct {
	EntityID     string
	DeviceID     string
	Slug         string
	FriendlyName string
	CastUUID     string
	Manufacturer string
	Model        string
	AreaID       string
	Host         string
}

func (t PingTarget) WithHost(host string) PingTarget {
	t.Host = host
	return t
}

type Settings struct {
	env.Common
	PingIntervalSeconds          int
	PingDiscoveryRefreshSeconds  int
	PingMethod                   string
	PingTimeoutMS                int
	PingTCPPort                  int
	PingStatsWindow              int
	PingManufacturerFilter       map[string]struct{}
	PingExcludeModels            map[string]struct{}
	PingHostOverrides            map[string]string
}

func FromEnv() (Settings, error) {
	common, err := env.LoadCommon("home/ping", "pinger-mqtt")
	if err != nil {
		return Settings{}, err
	}
	method := strings.ToLower(env.Get("PING_METHOD", "tcp8008"))
	if method != "tcp8008" && method != "icmp" {
		return Settings{}, fmt.Errorf("invalid PING_METHOD %q; expected tcp8008 or icmp", method)
	}
	interval, err := env.Int("PING_INTERVAL_SECONDS", 60)
	if err != nil {
		return Settings{}, err
	}
	refresh, err := env.Int("PING_DISCOVERY_REFRESH_SECONDS", 300)
	if err != nil {
		return Settings{}, err
	}
	timeout, err := env.Int("PING_TIMEOUT_MS", 2000)
	if err != nil {
		return Settings{}, err
	}
	port, err := env.Int("PING_TCP_PORT", 8008)
	if err != nil {
		return Settings{}, err
	}
	window, err := env.Int("PING_STATS_WINDOW", 60)
	if err != nil {
		return Settings{}, err
	}
	overrides, err := parseHostOverrides(os.Getenv("PING_HOST_OVERRIDES"))
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		Common:                      common,
		PingIntervalSeconds:         interval,
		PingDiscoveryRefreshSeconds: refresh,
		PingMethod:                  method,
		PingTimeoutMS:               timeout,
		PingTCPPort:                 port,
		PingStatsWindow:             window,
		PingManufacturerFilter:      env.CSV("PING_MANUFACTURER_FILTER", "Google Inc."),
		PingExcludeModels:           env.CSV("PING_EXCLUDE_MODELS", "Google Cast Group"),
		PingHostOverrides:           overrides,
	}, nil
}

func parseHostOverrides(raw string) (map[string]string, error) {
	overrides := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		entityID, host, ok := strings.Cut(part, ":")
		entityID = strings.TrimSpace(entityID)
		host = strings.TrimSpace(host)
		if !ok || entityID == "" || host == "" {
			return nil, fmt.Errorf("invalid PING_HOST_OVERRIDES entry %q; expected entity_id:host", part)
		}
		overrides[entityID] = host
	}
	return overrides, nil
}

var slugRE = regexp.MustCompile(`[^a-z0-9_]+`)

func Slugify(value string) string {
	slug := slugRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		return "device"
	}
	return slug
}

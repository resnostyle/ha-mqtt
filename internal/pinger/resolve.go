package pinger

import (
	"context"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const castService = "_googlecast._tcp"

var uuidRE = regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})`)

func NormalizeCastUUID(value string) string {
	cleaned := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "-", ""))
	if len(cleaned) != 32 {
		return strings.ToLower(strings.TrimSpace(value))
	}
	for _, c := range cleaned {
		if c < '0' || (c > '9' && c < 'a') || c > 'f' {
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return cleaned[0:8] + "-" + cleaned[8:12] + "-" + cleaned[12:16] + "-" + cleaned[16:20] + "-" + cleaned[20:32]
}

func UUIDFromServiceName(name string) string {
	match := uuidRE.FindStringSubmatch(name)
	if len(match) < 2 {
		return ""
	}
	return NormalizeCastUUID(match[1])
}

func ParseCastService(name, host string, properties map[string]string, addresses []string) (uuid string, hostIP string, ok bool) {
	rawID := ""
	if properties != nil {
		rawID = properties["id"]
	}
	if rawID != "" {
		uuid = NormalizeCastUUID(rawID)
	} else {
		uuid = UUIDFromServiceName(name)
		if uuid == "" {
			return "", "", false
		}
	}
	if len(addresses) > 0 {
		return uuid, addresses[0], true
	}
	resolved, err := net.LookupHost(strings.TrimRight(host, "."))
	if err != nil || len(resolved) == 0 {
		slog.Debug("could not resolve mDNS host", "host", host, "name", name)
		return "", "", false
	}
	return uuid, resolved[0], true
}

type Resolver struct {
	BrowseSeconds time.Duration
	Cache         map[string]string
	Browse        func(ctx context.Context) (map[string]string, error)
}

func NewResolver() *Resolver {
	r := &Resolver{
		BrowseSeconds: 3 * time.Second,
		Cache:         map[string]string{},
	}
	r.Browse = r.browseMDNS
	return r
}

func (r *Resolver) browseMDNS(ctx context.Context) (map[string]string, error) {
	timeout := r.BrowseSeconds
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	entries := make(chan *zeroconf.ServiceEntry, 16)
	if err := resolver.Browse(browseCtx, castService, "local.", entries); err != nil {
		return nil, err
	}
	found := map[string]string{}
	for {
		select {
		case <-browseCtx.Done():
			return found, nil
		case entry, ok := <-entries:
			if !ok {
				return found, nil
			}
			if entry == nil {
				continue
			}
			props := map[string]string{}
			for _, txt := range entry.Text {
				k, v, cut := strings.Cut(txt, "=")
				if cut {
					props[k] = v
				}
			}
			addrs := make([]string, 0, len(entry.AddrIPv4)+len(entry.AddrIPv6))
			for _, ip := range entry.AddrIPv4 {
				addrs = append(addrs, ip.String())
			}
			uuid, hostIP, ok := ParseCastService(entry.Instance+"."+entry.Service, entry.HostName, props, addrs)
			if ok {
				found[uuid] = hostIP
			}
		}
	}
}

func (r *Resolver) RefreshCache(ctx context.Context) (map[string]string, error) {
	slog.Info("refreshing mDNS host cache")
	found, err := r.Browse(ctx)
	if err != nil {
		return r.Cache, err
	}
	if r.Cache == nil {
		r.Cache = map[string]string{}
	}
	for k, v := range found {
		r.Cache[k] = v
	}
	slog.Info("mDNS browse found cast hosts", "count", len(found))
	return r.Cache, nil
}

func (r *Resolver) LookupHost(castUUID string) string {
	if r.Cache == nil {
		return ""
	}
	return r.Cache[NormalizeCastUUID(castUUID)]
}

func (r *Resolver) ResolveTargets(ctx context.Context, targets []PingTarget) ([]PingTarget, error) {
	if len(r.Cache) == 0 {
		if _, err := r.RefreshCache(ctx); err != nil {
			slog.Error("mDNS browse failed", "err", err)
		}
	}
	resolved := make([]PingTarget, 0, len(targets))
	for _, target := range targets {
		if target.Host != "" {
			resolved = append(resolved, target)
			continue
		}
		host := r.LookupHost(target.CastUUID)
		if host != "" {
			resolved = append(resolved, target.WithHost(host))
			continue
		}
		slog.Warn("no mDNS host for device",
			"name", target.FriendlyName,
			"entity", target.EntityID,
			"uuid", target.CastUUID,
		)
		resolved = append(resolved, target)
	}
	return resolved, nil
}

func (r *Resolver) ReapplyFailedHosts(targets []PingTarget, failedEntityIDs map[string]struct{}, pinnedHosts map[string]string) []PingTarget {
	if len(failedEntityIDs) == 0 {
		return targets
	}
	updated := make([]PingTarget, 0, len(targets))
	for _, target := range targets {
		if _, failed := failedEntityIDs[target.EntityID]; !failed {
			updated = append(updated, target)
			continue
		}
		if _, pinned := pinnedHosts[target.EntityID]; pinned {
			updated = append(updated, target)
			continue
		}
		host := r.LookupHost(target.CastUUID)
		if host != "" && host != target.Host {
			slog.Info("updated host after failure",
				"name", target.FriendlyName,
				"from", target.Host,
				"to", host,
			)
			updated = append(updated, target.WithHost(host))
			continue
		}
		updated = append(updated, target)
	}
	return updated
}

package monitor

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/resnostyle/ha-mqtt/internal/lib/env"
)

type HostTarget struct {
	Name string
	Slug string
	Host string
}

type Settings struct {
	env.MQTT
	Hosts           []HostTarget
	IntervalSeconds int
	Method          string
	TimeoutMS       int
	TCPPort         int
	StatsWindow     int
}

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func FromEnv() (Settings, error) {
	mqtt, err := env.LoadMQTT("home/monitor", "monitor-mqtt")
	if err != nil {
		return Settings{}, err
	}
	hosts, err := ParseHosts(os.Getenv("MONITOR_HOSTS"))
	if err != nil {
		return Settings{}, err
	}
	method := strings.ToLower(env.Get("MONITOR_METHOD", "icmp"))
	if method != "icmp" && method != "tcp" {
		return Settings{}, fmt.Errorf("invalid MONITOR_METHOD %q; expected icmp or tcp", method)
	}
	interval, err := env.Int("MONITOR_INTERVAL_SECONDS", 60)
	if err != nil {
		return Settings{}, err
	}
	timeout, err := env.Int("MONITOR_TIMEOUT_MS", 2000)
	if err != nil {
		return Settings{}, err
	}
	port, err := env.Int("MONITOR_TCP_PORT", 443)
	if err != nil {
		return Settings{}, err
	}
	window, err := env.Int("MONITOR_STATS_WINDOW", 60)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		MQTT:            mqtt,
		Hosts:           hosts,
		IntervalSeconds: interval,
		Method:          method,
		TimeoutMS:       timeout,
		TCPPort:         port,
		StatsWindow:     window,
	}, nil
}

func ParseHosts(raw string) ([]HostTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("missing required environment variable: MONITOR_HOSTS")
	}
	var hosts []HostTarget
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, host, ok := strings.Cut(part, ":")
		name = strings.ToLower(strings.TrimSpace(name))
		host = strings.TrimSpace(host)
		if !ok || name == "" || host == "" {
			return nil, fmt.Errorf("invalid MONITOR_HOSTS entry %q; expected name:host", part)
		}
		if !nameRE.MatchString(name) {
			return nil, fmt.Errorf("invalid monitor host name %q; use lowercase letters, numbers, underscores", name)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate monitor host name: %s", name)
		}
		seen[name] = struct{}{}
		hosts = append(hosts, HostTarget{
			Name: name,
			Slug: Slugify(name),
			Host: host,
		})
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("MONITOR_HOSTS is empty")
	}
	return hosts, nil
}

var slugRE = regexp.MustCompile(`[^a-z0-9_]+`)

func Slugify(value string) string {
	slug := slugRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		return "host"
	}
	return slug
}

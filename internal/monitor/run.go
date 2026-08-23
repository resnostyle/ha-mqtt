package monitor

import (
	"log/slog"

	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
	"github.com/resnostyle/ha-mqtt/internal/lib/probe"
)

type ProbeOutcome struct {
	Target HostTarget
	Result probe.Result
	Stats  probe.Stats
}

type Monitor struct {
	settings Settings
	prober   *probe.Prober
}

func New(settings Settings) *Monitor {
	return &Monitor{
		settings: settings,
		prober: probe.NewProber(probe.Config{
			Method:      settings.Method,
			TimeoutMS:   settings.TimeoutMS,
			TCPPort:     settings.TCPPort,
			StatsWindow: settings.StatsWindow,
		}),
	}
}

func (m *Monitor) ProbeAll(targets []HostTarget) []ProbeOutcome {
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.Slug)
	}
	m.prober.SyncIDs(ids)

	out := make([]ProbeOutcome, 0, len(targets))
	for _, target := range targets {
		result, stats := m.prober.Probe(target.Slug, target.Host, target.Name)
		out = append(out, ProbeOutcome{Target: target, Result: result, Stats: stats})
	}
	return out
}

func PublishDiscovery(settings Settings, mqtt mqttpub.Sink, targets []HostTarget) error {
	if !settings.MQTTDiscoveryEnabled {
		slog.Info("MQTT discovery disabled")
		return nil
	}
	total := 0
	for _, target := range targets {
		configs := BuildDiscoveryConfigs(settings.MQTTTopicPrefix, target)
		if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
			return err
		}
		total += len(configs)
		slog.Info("published discovery configs", "count", len(configs), "name", target.Name, "host", target.Host)
	}
	slog.Info("published mqtt discovery configs total", "count", total)
	return nil
}

func ProbeAndPublish(settings Settings, mqtt mqttpub.Sink, m *Monitor, targets []HostTarget) []ProbeOutcome {
	results := m.ProbeAll(targets)
	for _, r := range results {
		payload := BuildHostPayload(r.Target, r.Result, r.Stats, settings.Method)
		if err := mqtt.Publish(r.Target.Slug+"/current", payload, true); err != nil {
			slog.Error("publish failed", "name", r.Target.Name, "err", err)
		}
	}
	summary := BuildSummaryPayload(results, settings.Method)
	if err := mqtt.Publish("summary", summary, true); err != nil {
		slog.Error("summary publish failed", "err", err)
	} else {
		slog.Info("published monitor results",
			"hosts", summary["host_count"],
			"reachable", summary["reachable_count"],
		)
	}
	return results
}

package pinger

import (
	"context"
	"log/slog"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/lib/ha"
	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

type RegistryClient interface {
	ListEntityRegistry(ctx context.Context) ([]ha.EntityRegistryEntry, error)
	ListDeviceRegistry(ctx context.Context) ([]ha.DeviceRegistryEntry, error)
}

func PublishDiscovery(settings Settings, mqtt mqttpub.Sink, targets []PingTarget) error {
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
		slog.Info("published discovery configs", "count", len(configs), "name", target.FriendlyName, "entity", target.EntityID)
	}
	slog.Info("published mqtt discovery configs total", "count", total)
	return nil
}

func RefreshTargets(ctx context.Context, settings Settings, haClient RegistryClient, resolver *Resolver) ([]PingTarget, error) {
	entities, err := haClient.ListEntityRegistry(ctx)
	if err != nil {
		return nil, err
	}
	devices, err := haClient.ListDeviceRegistry(ctx)
	if err != nil {
		return nil, err
	}
	targets := DiscoverCastTargets(entities, devices, settings)
	return resolver.ResolveTargets(ctx, targets)
}

func RefreshFailedHosts(ctx context.Context, settings Settings, resolver *Resolver, targets []PingTarget, results []ProbeOutcome) []PingTarget {
	failed := map[string]struct{}{}
	for _, r := range results {
		if !r.Result.Reachable {
			failed[r.Target.EntityID] = struct{}{}
		}
	}
	if len(failed) == 0 {
		return targets
	}
	if _, err := resolver.RefreshCache(ctx); err != nil {
		slog.Error("mDNS refresh failed", "err", err)
	}
	return resolver.ReapplyFailedHosts(targets, failed, settings.PingHostOverrides)
}

func ProbeAndPublish(settings Settings, mqtt mqttpub.Sink, p *Pinger, targets []PingTarget) []ProbeOutcome {
	results := p.ProbeAll(targets)
	for _, r := range results {
		payload := BuildDevicePayload(r.Target, r.Result, r.Stats, settings.PingMethod)
		if err := mqtt.Publish(r.Target.Slug+"/current", payload, true); err != nil {
			slog.Error("publish failed", "entity", r.Target.EntityID, "err", err)
		}
	}
	summary := BuildSummaryPayload(results, settings.PingMethod)
	if err := mqtt.Publish("summary", summary, true); err != nil {
		slog.Error("summary publish failed", "err", err)
	} else {
		slog.Info("published ping results",
			"devices", summary["device_count"],
			"reachable", summary["reachable_count"],
		)
	}
	return results
}

func Due(last time.Time, interval time.Duration, now time.Time) bool {
	if last.IsZero() {
		return true
	}
	return now.Sub(last) >= interval
}

package main

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/lib/ha"
	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
	"github.com/resnostyle/ha-mqtt/internal/lib/poll"
	"github.com/resnostyle/ha-mqtt/internal/pinger"
)

func main() {
	settings, err := pinger.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	configureLogging(settings.LogLevel)

	slog.Info("starting ha-mqtt pinger",
		"interval", settings.PingIntervalSeconds,
		"discovery_refresh", settings.PingDiscoveryRefreshSeconds,
		"method", settings.PingMethod,
		"mqtt", settings.MQTTHost,
		"port", settings.MQTTPort,
		"discovery", settings.MQTTDiscoveryEnabled,
	)

	ctx, cancel := poll.NotifyContext()
	defer cancel()

	client := ha.New(settings.HAURL, settings.HAToken, 30*time.Second)
	resolver := pinger.NewResolver()
	probe := pinger.NewPinger(settings)

	mqtt, err := mqttpub.New(
		settings.MQTTHost,
		settings.MQTTPort,
		settings.MQTTClientID,
		settings.MQTTUsername,
		settings.MQTTPassword,
		settings.MQTTTopicPrefix,
	)
	if err != nil {
		slog.Error("mqtt connect failed", "err", err)
		os.Exit(1)
	}
	defer mqtt.Close()

	var targets []pinger.PingTarget
	var lastDiscovery time.Time
	refreshEvery := time.Duration(settings.PingDiscoveryRefreshSeconds) * time.Second

	for ctx.Err() == nil {
		if pinger.Due(lastDiscovery, refreshEvery, time.Now()) {
			next, err := pinger.RefreshTargets(ctx, settings, client, resolver)
			if err != nil {
				slog.Error("target discovery failed", "err", err)
			} else {
				targets = next
				lastDiscovery = time.Now()
				if err := pinger.PublishDiscovery(settings, mqtt, targets); err != nil {
					slog.Error("mqtt discovery publish failed", "err", err)
				}
			}
		}

		if len(targets) > 0 {
			results := pinger.ProbeAndPublish(settings, mqtt, probe, targets)
			targets = pinger.RefreshFailedHosts(ctx, settings, resolver, targets, results)
		} else {
			slog.Warn("no ping targets available; waiting for discovery")
		}
		poll.Wait(ctx, time.Duration(settings.PingIntervalSeconds)*time.Second)
	}
	slog.Info("exited")
}

func configureLogging(level string) {
	var lvl slog.Level
	switch strings.ToUpper(level) {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARN", "WARNING":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}

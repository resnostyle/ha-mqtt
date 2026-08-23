package main

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
	"github.com/resnostyle/ha-mqtt/internal/lib/poll"
	"github.com/resnostyle/ha-mqtt/internal/monitor"
)

func main() {
	settings, err := monitor.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	configureLogging(settings.LogLevel)

	slog.Info("starting ha-mqtt monitor",
		"hosts", len(settings.Hosts),
		"interval", settings.IntervalSeconds,
		"method", settings.Method,
		"mqtt", settings.MQTTHost,
		"port", settings.MQTTPort,
		"discovery", settings.MQTTDiscoveryEnabled,
	)

	ctx, cancel := poll.NotifyContext()
	defer cancel()

	m := monitor.New(settings)
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

	if err := monitor.PublishDiscovery(settings, mqtt, settings.Hosts); err != nil {
		slog.Error("mqtt discovery publish failed", "err", err)
	}

	for ctx.Err() == nil {
		monitor.ProbeAndPublish(settings, mqtt, m, settings.Hosts)
		poll.Wait(ctx, time.Duration(settings.IntervalSeconds)*time.Second)
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

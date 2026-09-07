package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/monitor"
	"github.com/resnostyle/mqttkit/logx"
	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/mqttkit/poll"
)

func main() {
	settings, err := monitor.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	logx.Configure(settings.LogLevel, false)

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

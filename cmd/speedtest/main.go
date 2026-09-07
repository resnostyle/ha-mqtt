package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/speedtest"
	"github.com/resnostyle/mqttkit/logx"
	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/mqttkit/poll"
)

func main() {
	settings, err := speedtest.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	logx.Configure(settings.LogLevel, false)

	slog.Info("starting ha-mqtt speedtest",
		"interval", settings.IntervalSeconds,
		"server_id", settings.ServerID,
		"mqtt", settings.MQTTHost,
		"port", settings.MQTTPort,
		"discovery", settings.MQTTDiscoveryEnabled,
	)

	ctx, cancel := poll.NotifyContext()
	defer cancel()

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

	if err := speedtest.PublishDiscovery(settings, mqtt); err != nil {
		slog.Error("mqtt discovery publish failed", "err", err)
	}

	runner := speedtest.NewLibRunner()
	for ctx.Err() == nil {
		speedtest.RunAndPublish(ctx, settings, mqtt, runner)
		poll.Wait(ctx, time.Duration(settings.IntervalSeconds)*time.Second)
	}
	slog.Info("exited")
}

package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/lib/ha"
	"github.com/resnostyle/ha-mqtt/internal/weather"
	"github.com/resnostyle/mqttkit/logx"
	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/mqttkit/poll"
)

func main() {
	settings, err := weather.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	logx.Configure(settings.LogLevel, false)

	sun := "off"
	if settings.SunEnabled {
		sun = settings.SunEntity
	}
	slog.Info("starting ha-mqtt weather",
		"sources", weather.SourceSummary(settings),
		"sun", sun,
		"interval", settings.PollIntervalSeconds,
		"mqtt", settings.MQTTHost,
		"port", settings.MQTTPort,
		"discovery", settings.MQTTDiscoveryEnabled,
	)

	ctx, cancel := poll.NotifyContext()
	defer cancel()

	client := ha.New(settings.HAURL, settings.HAToken, 30*time.Second)
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

	if err := weather.PublishDiscovery(settings, mqtt); err != nil {
		slog.Error("mqtt discovery publish failed", "err", err)
	}

	for ctx.Err() == nil {
		if err := weather.PublishOnce(ctx, settings, client, mqtt); err != nil {
			slog.Error("publish cycle failed", "err", err)
		}
		poll.Wait(ctx, time.Duration(settings.PollIntervalSeconds)*time.Second)
	}
	slog.Info("exited")
}

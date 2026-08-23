package speedtest

import (
	"context"
	"log/slog"

	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

func PublishDiscovery(settings Settings, mqtt mqttpub.Sink) error {
	if !settings.MQTTDiscoveryEnabled {
		slog.Info("MQTT discovery disabled")
		return nil
	}
	configs := BuildDiscoveryConfigs(settings.MQTTTopicPrefix)
	if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
		return err
	}
	slog.Info("published mqtt discovery configs", "count", len(configs))
	return nil
}

func RunAndPublish(ctx context.Context, settings Settings, mqtt mqttpub.Sink, runner Runner) Result {
	slog.Info("starting speedtest", "server_id", settings.ServerID)
	result := runner.Run(ctx, settings.ServerID)
	payload := BuildPayload(result)
	if err := mqtt.Publish("current", payload, true); err != nil {
		slog.Error("publish failed", "err", err)
		return result
	}
	if result.OK {
		dl, ul, ping := 0.0, 0.0, 0.0
		if result.DownloadMbps != nil {
			dl = *result.DownloadMbps
		}
		if result.UploadMbps != nil {
			ul = *result.UploadMbps
		}
		if result.PingMS != nil {
			ping = *result.PingMS
		}
		slog.Info("published speedtest results",
			"download_mbps", dl,
			"upload_mbps", ul,
			"ping_ms", ping,
			"server", result.ServerName,
		)
	} else {
		slog.Warn("speedtest failed", "error", result.Error)
	}
	return result
}

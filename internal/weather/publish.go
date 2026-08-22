package weather

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

type HAClient interface {
	GetState(ctx context.Context, entityID string) (map[string]any, error)
	GetForecasts(ctx context.Context, entityID, forecastType string) ([]map[string]any, error)
}

func SourceTopicPrefix(settings Settings, source Source) string {
	return settings.MQTTTopicPrefix + "/" + source.TopicSuffix()
}

func PublishDiscovery(settings Settings, mqtt mqttpub.Sink) error {
	if !settings.MQTTDiscoveryEnabled {
		slog.Info("MQTT discovery disabled")
		return nil
	}
	total := 0
	for _, source := range settings.WeatherSources {
		configs := BuildDiscoveryConfigs(
			SourceTopicPrefix(settings, source),
			source.Name,
			source.Label,
			settings.TemperatureUnit,
			settings.PressureUnit,
			settings.WindSpeedUnit,
			settings.VisibilityUnit,
		)
		if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
			return err
		}
		total += len(configs)
		slog.Info("published discovery configs", "count", len(configs), "source", source.Name, "entity", source.EntityID)
	}
	if settings.SunEnabled {
		sunConfigs := BuildSunDiscoveryConfigs(settings.MQTTSunTopicPrefix)
		if err := mqtt.PublishDiscovery(sunConfigs, settings.MQTTDiscoveryPrefix); err != nil {
			return err
		}
		total += len(sunConfigs)
		slog.Info("published discovery configs", "count", len(sunConfigs), "source", "sun", "entity", settings.SunEntity)
	}
	slog.Info("published mqtt discovery configs total", "count", total)
	return nil
}

func PublishSource(ctx context.Context, settings Settings, ha HAClient, mqtt mqttpub.Sink, source Source) error {
	entity := source.EntityID
	prefix := source.TopicSuffix()
	state, err := ha.GetState(ctx, entity)
	if err != nil {
		return err
	}
	daily, err := ha.GetForecasts(ctx, entity, "daily")
	if err != nil {
		return err
	}
	hourly := []map[string]any{}
	rawHourly, hourlyErr := ha.GetForecasts(ctx, entity, "hourly")
	if hourlyErr != nil {
		slog.Error("hourly forecast unavailable; continuing without it", "source", source.Name, "entity", entity, "err", hourlyErr)
	} else {
		hourly = TrimHourly(rawHourly, settings.HourlyForecastHours)
	}

	if err := mqtt.Publish(prefix+"/current", BuildCurrent(state), true); err != nil {
		return err
	}
	if err := mqtt.Publish(prefix+"/daily", BuildForecastPayload(daily, "daily", entity), true); err != nil {
		return err
	}
	if len(hourly) > 0 {
		if err := mqtt.Publish(prefix+"/hourly", BuildForecastPayload(hourly, "hourly", entity), true); err != nil {
			return err
		}
	}
	if err := mqtt.Publish(prefix+"/5day", Build5Day(daily, source.Name, settings.TemperatureUnit), true); err != nil {
		return err
	}
	slog.Info("published weather topics", "source", source.Name, "entity", entity)
	return nil
}

func PublishSun(ctx context.Context, settings Settings, ha HAClient, mqtt mqttpub.Sink) error {
	state, err := ha.GetState(ctx, settings.SunEntity)
	if err != nil {
		return err
	}
	topic := settings.MQTTSunTopicPrefix + "/current"
	if err := mqtt.PublishRaw(topic, BuildSun(state), true); err != nil {
		return err
	}
	slog.Info("published sun topic", "topic", topic, "entity", settings.SunEntity)
	return nil
}

func PublishOnce(ctx context.Context, settings Settings, ha HAClient, mqtt mqttpub.Sink) error {
	errors := 0
	for _, source := range settings.WeatherSources {
		if err := PublishSource(ctx, settings, ha, mqtt, source); err != nil {
			errors++
			slog.Error("publish failed", "source", source.Name, "entity", source.EntityID, "err", err)
		}
	}
	if settings.SunEnabled {
		if err := PublishSun(ctx, settings, ha, mqtt); err != nil {
			errors++
			slog.Error("publish failed", "source", "sun", "entity", settings.SunEntity, "err", err)
		}
	}
	total := len(settings.WeatherSources)
	if settings.SunEnabled {
		total++
	}
	if errors > 0 && errors == total {
		return fmt.Errorf("all publish targets failed")
	}
	return nil
}

func SourceSummary(settings Settings) string {
	parts := make([]string, 0, len(settings.WeatherSources))
	for _, s := range settings.WeatherSources {
		parts = append(parts, s.Name+"="+s.EntityID)
	}
	return strings.Join(parts, ", ")
}

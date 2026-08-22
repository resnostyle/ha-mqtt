package weather

import (
	"context"
	"testing"

	"github.com/resnostyle/ha-mqtt/internal/lib/env"
	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

type fakeHA struct {
	states    map[string]map[string]any
	forecasts map[string][]map[string]any
	hourlyErr bool
}

func (f fakeHA) GetState(ctx context.Context, entityID string) (map[string]any, error) {
	return f.states[entityID], nil
}

func (f fakeHA) GetForecasts(ctx context.Context, entityID, forecastType string) ([]map[string]any, error) {
	if forecastType == "hourly" && f.hourlyErr {
		return nil, context.Canceled
	}
	return f.forecasts[forecastType], nil
}

type fakeMQTT struct {
	topics []string
}

func (f *fakeMQTT) Publish(suffix string, payload any, retain bool) error {
	f.topics = append(f.topics, suffix)
	return nil
}
func (f *fakeMQTT) PublishRaw(topic string, payload any, retain bool) error {
	f.topics = append(f.topics, topic)
	return nil
}
func (f *fakeMQTT) PublishDiscovery(configs []mqttpub.Config, discoveryPrefix string) error {
	return nil
}

func TestPublishOnce(t *testing.T) {
	settings := Settings{
		Common: env.Common{MQTTTopicPrefix: "home/weather"},
		WeatherSources: []Source{{
			Name: "pirate", EntityID: "weather.pirateweather", Label: "Pirate Weather",
		}},
		SunEnabled:          true,
		SunEntity:           "sun.sun",
		MQTTSunTopicPrefix:  "home/sun",
		HourlyForecastHours: 48,
		TemperatureUnit:     "°F",
	}
	ha := fakeHA{
		states: map[string]map[string]any{
			"weather.pirateweather": {"entity_id": "weather.pirateweather", "state": "sunny", "attributes": map[string]any{"temperature": 70}},
			"sun.sun":               {"entity_id": "sun.sun", "state": "above_horizon", "attributes": map[string]any{}},
		},
		forecasts: map[string][]map[string]any{
			"daily":  {{"condition": "sunny"}},
			"hourly": {{"condition": "sunny"}},
		},
		hourlyErr: true,
	}
	mqtt := &fakeMQTT{}
	if err := PublishOnce(context.Background(), settings, ha, mqtt); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"pirate/current": true, "pirate/daily": true, "pirate/5day": true, "home/sun/current": true}
	for _, topic := range mqtt.topics {
		delete(want, topic)
	}
	if len(want) != 0 {
		t.Fatalf("missing topics %v have %v", want, mqtt.topics)
	}
}

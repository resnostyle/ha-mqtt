package weather

import (
	"testing"

	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

func TestNormalizeUnit(t *testing.T) {
	cases := map[string]string{
		"F":  "°F",
		"f":  "°F",
		"C":  "°C",
		"c":  "°C",
		"°F": "°F",
		"°C": "°C",
	}
	for in, want := range cases {
		if got := normalizeUnit(in); got != want {
			t.Fatalf("normalizeUnit(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseWeatherSourcesDefaultAndMulti(t *testing.T) {
	single, err := ParseWeatherSources("", "weather.pirateweather")
	if err != nil {
		t.Fatal(err)
	}
	if len(single) != 1 || single[0].Name != "pirate" || single[0].EntityID != "weather.pirateweather" {
		t.Fatalf("%v", single)
	}
	multi, err := ParseWeatherSources("pirate:weather.pirateweather,metno:weather.forecast_home", "")
	if err != nil {
		t.Fatal(err)
	}
	if multi[0].Name != "pirate" || multi[1].Name != "metno" {
		t.Fatalf("%v", multi)
	}
	if multi[1].EntityID != "weather.forecast_home" || multi[1].Label != "Met.no" {
		t.Fatalf("%v", multi[1])
	}
}

func TestBuildCurrent(t *testing.T) {
	payload := BuildCurrent(map[string]any{
		"entity_id":    "weather.pirateweather",
		"state":        "rainy",
		"last_updated": "2026-08-11T23:17:19+00:00",
		"attributes": map[string]any{
			"temperature":          81,
			"apparent_temperature": 82,
			"humidity":             64,
			"temperature_unit":     "°F",
		},
	})
	if payload["condition"] != "rainy" || payload["temperature"] != 81 {
		t.Fatalf("%v", payload)
	}
	if payload["entity_id"] != "weather.pirateweather" {
		t.Fatal(payload["entity_id"])
	}
	if _, ok := payload["published"]; !ok {
		t.Fatal("missing published")
	}
}

func TestTrimHourly(t *testing.T) {
	hours := make([]map[string]any, 60)
	for i := range hours {
		hours[i] = map[string]any{"datetime": i}
	}
	if len(TrimHourly(hours, 48)) != 48 {
		t.Fatal("expected 48")
	}
	if len(TrimHourly(hours, 0)) != 60 {
		t.Fatal("expected full list")
	}
}

func TestBuild5Day(t *testing.T) {
	payload := Build5Day([]map[string]any{{
		"datetime":    "2026-08-12T00:00:00+00:00",
		"condition":   "rainy",
		"temperature": 85,
		"templow":     70,
		"humidity":    60,
		"wind_speed":  10,
	}}, "metno", "°F")
	cfg := payload["config"].(map[string]any)
	if cfg["source"] != "metno" {
		t.Fatal(cfg)
	}
	forecast := payload["forecast"].([]map[string]any)
	if len(forecast) != 5 {
		t.Fatalf("len %d", len(forecast))
	}
	if forecast[0]["condition"] != "Rain" {
		t.Fatal(forecast[0])
	}
	if forecast[1]["condition"] != "No data" {
		t.Fatal(forecast[1])
	}
}

func TestBuildSun(t *testing.T) {
	payload := BuildSun(map[string]any{
		"entity_id":    "sun.sun",
		"state":        "below_horizon",
		"last_updated": "2026-08-12T03:24:30+00:00",
		"attributes": map[string]any{
			"next_rising": "2026-08-12T10:32:27+00:00",
			"elevation":   -32.32,
			"azimuth":     326.28,
			"rising":      false,
		},
	})
	if payload["state"] != "below_horizon" {
		t.Fatal(payload["state"])
	}
	if payload["daytime"] != false {
		t.Fatal(payload["daytime"])
	}
	if payload["elevation"] != -32.32 {
		t.Fatal(payload["elevation"])
	}
	day := BuildSun(map[string]any{
		"entity_id":  "sun.sun",
		"state":      "above_horizon",
		"attributes": map[string]any{"elevation": 40.0, "azimuth": 180.0, "rising": false},
	})
	if day["daytime"] != true {
		t.Fatal(day["daytime"])
	}
}

func byID(configs []mqttpub.Config) map[string]mqttpub.Config {
	out := map[string]mqttpub.Config{}
	for _, c := range configs {
		out[c.ObjectID] = c
	}
	return out
}

func TestDiscoveryIncludesDeviceAndCoreSensors(t *testing.T) {
	configs := BuildDiscoveryConfigs("home/weather/pirate", "pirate", "Pirate Weather", "°F", "inHg", "mph", "mi")
	got := byID(configs)
	temp := got["weather_mqtt_pirate_temperature"]
	if temp.Payload["state_topic"] != "home/weather/pirate/current" {
		t.Fatal(temp.Payload["state_topic"])
	}
	if temp.Payload["value_template"] != "{{ value_json.temperature }}" {
		t.Fatal(temp.Payload["value_template"])
	}
	if temp.Payload["device_class"] != "temperature" {
		t.Fatal(temp.Payload["device_class"])
	}
	if temp.Payload["unit_of_measurement"] != "°F" {
		t.Fatal(temp.Payload["unit_of_measurement"])
	}
	device := temp.Payload["device"].(map[string]any)
	ids := device["identifiers"].([]string)
	if ids[0] != "weather_mqtt_pirate" {
		t.Fatal(ids)
	}
	if device["name"] != "Weather MQTT (Pirate Weather)" {
		t.Fatal(device["name"])
	}
	daily := got["weather_mqtt_pirate_daily"]
	if daily.Payload["state_topic"] != "home/weather/pirate/daily" {
		t.Fatal(daily.Payload["state_topic"])
	}
	if daily.Payload["json_attributes_topic"] != "home/weather/pirate/daily" {
		t.Fatal(daily.Payload["json_attributes_topic"])
	}
	if len(configs) < 10 {
		t.Fatalf("len %d", len(configs))
	}
}

func TestDiscoveryMetnoUsesSeparateDevice(t *testing.T) {
	configs := BuildDiscoveryConfigs("home/weather/metno", "metno", "Met.no", "°C", "inHg", "km/h", "mi")
	temp := byID(configs)["weather_mqtt_metno_temperature"]
	if temp.Payload["unit_of_measurement"] != "°C" {
		t.Fatal(temp.Payload["unit_of_measurement"])
	}
	device := temp.Payload["device"].(map[string]any)
	if device["identifiers"].([]string)[0] != "weather_mqtt_metno" {
		t.Fatal(device["identifiers"])
	}
	if temp.Payload["state_topic"] != "home/weather/metno/current" {
		t.Fatal(temp.Payload["state_topic"])
	}
}

func TestSunDiscovery(t *testing.T) {
	got := byID(BuildSunDiscoveryConfigs("home/sun"))
	elev := got["weather_mqtt_sun_elevation"]
	if elev.Payload["state_topic"] != "home/sun/current" {
		t.Fatal(elev.Payload["state_topic"])
	}
	device := elev.Payload["device"].(map[string]any)
	if device["identifiers"].([]string)[0] != "weather_mqtt_sun" {
		t.Fatal(device["identifiers"])
	}
	if got["weather_mqtt_sun_next_rising"].Payload["device_class"] != "timestamp" {
		t.Fatal(got["weather_mqtt_sun_next_rising"].Payload["device_class"])
	}
}

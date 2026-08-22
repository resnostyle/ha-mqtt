package mqttpub

import "testing"

func TestJoin(t *testing.T) {
	if got := Join("home/weather", "pirate/current"); got != "home/weather/pirate/current" {
		t.Fatalf("got %q", got)
	}
	if got := Join("home/weather/", "/pirate/current"); got != "home/weather/pirate/current" {
		t.Fatalf("got %q", got)
	}
}

func TestDiscoveryTopic(t *testing.T) {
	got := DiscoveryTopic("homeassistant", "sensor", "weather_mqtt_pirate_temperature")
	want := "homeassistant/sensor/weather_mqtt_pirate_temperature/config"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = DiscoveryTopic("homeassistant", "binary_sensor", "pinger_mqtt_example_speaker_reachable")
	want = "homeassistant/binary_sensor/pinger_mqtt_example_speaker_reachable/config"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMarshalCompact(t *testing.T) {
	body, err := Marshal(map[string]any{"a": 1, "b": nil})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if s != `{"a":1,"b":null}` && s != `{"b":null,"a":1}` {
		t.Fatalf("unexpected json %s", s)
	}
}

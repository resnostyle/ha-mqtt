package env

import (
	"testing"
)

func TestRequire(t *testing.T) {
	t.Setenv("HA_URL", "http://example.test")
	got, err := Require("HA_URL")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://example.test" {
		t.Fatalf("got %q", got)
	}
	if _, err := Require("MISSING_REQUIRED_ENV"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBool(t *testing.T) {
	t.Setenv("FLAG", "yes")
	if !Bool("FLAG", false) {
		t.Fatal("expected true")
	}
	t.Setenv("FLAG", "off")
	if Bool("FLAG", true) {
		t.Fatal("expected false")
	}
}

func TestLoadCommon(t *testing.T) {
	t.Setenv("HA_URL", "http://ha.example.test/")
	t.Setenv("HA_TOKEN", "tok")
	t.Setenv("MQTT_HOST", "mqtt.example.test")
	t.Setenv("MQTT_PORT", "1884")
	t.Setenv("MQTT_DISCOVERY_ENABLED", "false")
	c, err := LoadCommon("home/weather", "weather-mqtt")
	if err != nil {
		t.Fatal(err)
	}
	if c.HAURL != "http://ha.example.test" {
		t.Fatalf("url %q", c.HAURL)
	}
	if c.MQTTPort != 1884 {
		t.Fatalf("port %d", c.MQTTPort)
	}
	if c.MQTTDiscoveryEnabled {
		t.Fatal("expected discovery disabled")
	}
	if c.MQTTTopicPrefix != "home/weather" {
		t.Fatalf("prefix %q", c.MQTTTopicPrefix)
	}
}

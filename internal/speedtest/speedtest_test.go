package speedtest

import (
	"context"
	"testing"

	"github.com/resnostyle/ha-mqtt/internal/lib/env"
	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

func fptr(v float64) *float64 { return &v }

type fakeMQTT struct {
	published []struct {
		topic   string
		payload any
	}
	discovery []mqttpub.Config
}

func (f *fakeMQTT) Publish(topicSuffix string, payload any, retain bool) error {
	f.published = append(f.published, struct {
		topic   string
		payload any
	}{topicSuffix, payload})
	return nil
}

func (f *fakeMQTT) PublishRaw(topic string, payload any, retain bool) error {
	return f.Publish(topic, payload, retain)
}

func (f *fakeMQTT) PublishDiscovery(configs []mqttpub.Config, discoveryPrefix string) error {
	f.discovery = append(f.discovery, configs...)
	return nil
}

type fakeRunner struct {
	result Result
}

func (f fakeRunner) Run(ctx context.Context, serverID string) Result {
	return f.result
}

func TestFromEnv(t *testing.T) {
	t.Setenv("SPEEDTEST_INTERVAL_SECONDS", "1800")
	t.Setenv("SPEEDTEST_SERVER_ID", "12345")
	t.Setenv("MQTT_HOST", "127.0.0.1")
	settings, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if settings.IntervalSeconds != 1800 {
		t.Fatalf("interval %d", settings.IntervalSeconds)
	}
	if settings.ServerID != "12345" {
		t.Fatalf("server %q", settings.ServerID)
	}
	if settings.MQTTTopicPrefix != "home/speedtest" {
		t.Fatal(settings.MQTTTopicPrefix)
	}
}

func TestFromEnvRejectsBadServerID(t *testing.T) {
	t.Setenv("SPEEDTEST_SERVER_ID", "not-a-number")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildPayloadSuccess(t *testing.T) {
	result := Result{
		OK:             true,
		DownloadMbps:   fptr(94.2),
		UploadMbps:     fptr(21.5),
		PingMS:         fptr(12.3),
		JitterMS:       fptr(1.1),
		ServerID:       "12345",
		ServerName:     "Example ISP",
		ServerLocation: "City, Country",
		TestedAt:       "2026-08-22T00:00:00Z",
	}
	payload := BuildPayload(result)
	if payload["ok"] != true {
		t.Fatal(payload["ok"])
	}
	if payload["download_mbps"] != result.DownloadMbps && payload["download_mbps"] != *result.DownloadMbps {
		t.Fatal(payload["download_mbps"])
	}
	if payload["error"] != nil {
		t.Fatal(payload["error"])
	}
	if payload["server_name"] != "Example ISP" {
		t.Fatal(payload["server_name"])
	}
}

func TestBuildPayloadFailure(t *testing.T) {
	payload := BuildPayload(Result{OK: false, Error: "timeout", TestedAt: "t"})
	if payload["ok"] != false {
		t.Fatal(payload["ok"])
	}
	if payload["error"] != "timeout" {
		t.Fatal(payload["error"])
	}
	if payload["download_mbps"] != nil {
		t.Fatal(payload["download_mbps"])
	}
}

func TestDiscoverySensors(t *testing.T) {
	configs := BuildDiscoveryConfigs("home/speedtest")
	if len(configs) != 4 {
		t.Fatalf("len %d", len(configs))
	}
	byID := map[string]mqttpub.Config{}
	for _, c := range configs {
		byID[c.ObjectID] = c
	}
	dl := byID["speedtest_mqtt_download"]
	if dl.Component != "sensor" {
		t.Fatal(dl.Component)
	}
	if dl.Payload["state_topic"] != "home/speedtest/current" {
		t.Fatal(dl.Payload["state_topic"])
	}
	if dl.Payload["unit_of_measurement"] != "Mbps" {
		t.Fatal(dl.Payload["unit_of_measurement"])
	}
	device := dl.Payload["device"].(map[string]any)
	if device["name"] != "Speedtest MQTT" {
		t.Fatal(device["name"])
	}
	if _, ok := byID["speedtest_mqtt_jitter"]; !ok {
		t.Fatal("missing jitter")
	}
}

func TestRunAndPublish(t *testing.T) {
	settings := Settings{
		MQTT:     env.MQTT{MQTTTopicPrefix: "home/speedtest", MQTTDiscoveryEnabled: true, MQTTDiscoveryPrefix: "homeassistant"},
		ServerID: "",
	}
	mqtt := &fakeMQTT{}
	runner := fakeRunner{result: Result{
		OK: true, DownloadMbps: fptr(100), UploadMbps: fptr(50), PingMS: fptr(10),
		ServerName: "ISP", TestedAt: "t",
	}}
	if err := PublishDiscovery(settings, mqtt); err != nil {
		t.Fatal(err)
	}
	if len(mqtt.discovery) != 4 {
		t.Fatalf("discovery %d", len(mqtt.discovery))
	}
	got := RunAndPublish(context.Background(), settings, mqtt, runner)
	if !got.OK {
		t.Fatal(got)
	}
	if len(mqtt.published) != 1 || mqtt.published[0].topic != "current" {
		t.Fatalf("%+v", mqtt.published)
	}
	payload := mqtt.published[0].payload.(map[string]any)
	if payload["ok"] != true {
		t.Fatal(payload)
	}
}

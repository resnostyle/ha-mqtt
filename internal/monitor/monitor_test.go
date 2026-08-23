package monitor

import (
	"testing"

	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
	"github.com/resnostyle/ha-mqtt/internal/lib/probe"
)

func fptr(v float64) *float64 { return &v }

func TestParseHosts(t *testing.T) {
	hosts, err := ParseHosts("router:192.168.1.1,nas:192.168.1.50")
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 {
		t.Fatalf("len %d", len(hosts))
	}
	if hosts[0].Name != "router" || hosts[0].Host != "192.168.1.1" || hosts[0].Slug != "router" {
		t.Fatalf("%+v", hosts[0])
	}
	if hosts[1].Name != "nas" || hosts[1].Host != "192.168.1.50" {
		t.Fatalf("%+v", hosts[1])
	}
}

func TestParseHostsRejectsInvalid(t *testing.T) {
	if _, err := ParseHosts(""); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := ParseHosts("badentry"); err == nil {
		t.Fatal("expected bad entry error")
	}
	if _, err := ParseHosts("1router:1.2.3.4"); err == nil {
		t.Fatal("expected name validation error")
	}
	if _, err := ParseHosts("router-gw:1.2.3.4"); err == nil {
		t.Fatal("expected name validation error for hyphen")
	}
	if _, err := ParseHosts("router:1.2.3.4,router:5.6.7.8"); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestSlugify(t *testing.T) {
	if Slugify("router") != "router" {
		t.Fatal(Slugify("router"))
	}
	if Slugify("my-nas") != "my_nas" {
		t.Fatal(Slugify("my-nas"))
	}
}

func TestBuildHostAndSummaryPayload(t *testing.T) {
	target := HostTarget{Name: "router", Slug: "router", Host: "192.168.1.1"}
	lat := 1.2
	result := probe.Result{Reachable: true, LatencyMS: &lat, ProbedAt: "2026-08-22T00:00:00Z"}
	stats := probe.Stats{Probes: 5, Successes: 5, SuccessRate: 1.0, AvgLatencyMS: fptr(1.5)}
	payload := BuildHostPayload(target, result, stats, "icmp")
	if payload["name"] != "router" || payload["host"] != "192.168.1.1" {
		t.Fatalf("%v", payload)
	}
	if payload["reachable"] != true || payload["method"] != "icmp" {
		t.Fatalf("%v", payload)
	}
	summary := BuildSummaryPayload([]ProbeOutcome{
		{Target: target, Result: result, Stats: stats},
		{Target: HostTarget{Name: "nas", Slug: "nas", Host: "192.168.1.50"}, Result: probe.Result{Reachable: false, Error: "unreachable"}, Stats: probe.Stats{Probes: 1, ConsecutiveFailures: 1}},
	}, "icmp")
	if summary["host_count"] != 2 || summary["reachable_count"] != 1 || summary["unreachable_count"] != 1 {
		t.Fatalf("%v", summary)
	}
}

func TestDiscoveryIncludesLatencyAndReachable(t *testing.T) {
	configs := BuildDiscoveryConfigs("home/monitor", HostTarget{Name: "router", Slug: "router", Host: "192.168.1.1"})
	if len(configs) != 2 {
		t.Fatalf("len %d", len(configs))
	}
	var latency, reachable mqttpub.Config
	for _, c := range configs {
		if c.Component == "sensor" {
			latency = c
		}
		if c.Component == "binary_sensor" {
			reachable = c
		}
	}
	if latency.ObjectID != "monitor_mqtt_router_latency" {
		t.Fatal(latency.ObjectID)
	}
	if latency.Payload["state_topic"] != "home/monitor/router/current" {
		t.Fatal(latency.Payload["state_topic"])
	}
	device := latency.Payload["device"].(map[string]any)
	if device["identifiers"].([]string)[0] != "monitor_mqtt_router" {
		t.Fatal(device["identifiers"])
	}
	if reachable.ObjectID != "monitor_mqtt_router_reachable" {
		t.Fatal(reachable.ObjectID)
	}
	if reachable.Payload["device_class"] != "connectivity" {
		t.Fatal(reachable.Payload["device_class"])
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("MONITOR_HOSTS", "router:192.168.1.1")
	t.Setenv("MONITOR_METHOD", "tcp")
	t.Setenv("MONITOR_INTERVAL_SECONDS", "30")
	t.Setenv("MQTT_HOST", "127.0.0.1")
	settings, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Method != "tcp" || settings.IntervalSeconds != 30 {
		t.Fatalf("%+v", settings)
	}
	if len(settings.Hosts) != 1 || settings.Hosts[0].Host != "192.168.1.1" {
		t.Fatalf("%+v", settings.Hosts)
	}
	if settings.MQTTTopicPrefix != "home/monitor" {
		t.Fatal(settings.MQTTTopicPrefix)
	}
}

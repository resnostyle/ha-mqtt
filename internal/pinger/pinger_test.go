package pinger

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/resnostyle/ha-mqtt/internal/lib/ha"
	"github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"
)

const (
	testHost         = "192.0.2.10"
	testHostAlt      = "192.0.2.20"
	testHostUpdated  = "192.0.2.30"
	testCastUUID     = "11111111-2222-3333-4444-555555555555"
	testCastUUIDComp = "11111111222233334444555555555555"
	testCastUUIDOther = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	testCastGroupUUID = "22222222-3333-4444-5555-666666666666"
	testTVCastUUID    = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	testDisabledUUID  = "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"
)

func testTarget(host string) PingTarget {
	return PingTarget{
		EntityID:     "media_player.example_speaker",
		DeviceID:     "dev1",
		Slug:         "example_speaker",
		FriendlyName: "Example speaker",
		CastUUID:     testCastUUID,
		Manufacturer: "Google Inc.",
		Model:        "Google Home",
		Host:         host,
	}
}

func testSettings() Settings {
	return Settings{
		PingMethod:             "tcp8008",
		PingTimeoutMS:          2000,
		PingTCPPort:            8008,
		PingStatsWindow:        5,
		PingManufacturerFilter: map[string]struct{}{"Google Inc.": {}},
		PingExcludeModels:      map[string]struct{}{"Google Cast Group": {}},
		PingHostOverrides:      map[string]string{},
	}
}

func TestSlugify(t *testing.T) {
	if Slugify("Example Speaker") != "example_speaker" {
		t.Fatal(Slugify("Example Speaker"))
	}
	if Slugify("media-player-2") != "media_player_2" {
		t.Fatal(Slugify("media-player-2"))
	}
}

func TestNormalizeCastUUID(t *testing.T) {
	if NormalizeCastUUID(testCastUUIDComp) != testCastUUID {
		t.Fatal(NormalizeCastUUID(testCastUUIDComp))
	}
}

func TestUUIDFromServiceName(t *testing.T) {
	name := "Example-speaker-" + testCastUUID + "._googlecast._tcp.local."
	if UUIDFromServiceName(name) != testCastUUID {
		t.Fatal(UUIDFromServiceName(name))
	}
}

func TestParseCastServiceFromProperties(t *testing.T) {
	uuid, host, ok := ParseCastService(
		"ignored._googlecast._tcp.local.",
		"example-speaker.local.",
		map[string]string{"id": testCastUUIDComp},
		[]string{testHost},
	)
	if !ok || uuid != testCastUUID || host != testHost {
		t.Fatalf("%s %s %v", uuid, host, ok)
	}
}

func TestParseCastServicePrefersParsedAddresses(t *testing.T) {
	uuid, host, ok := ParseCastService(
		"ignored._googlecast._tcp.local.",
		"example-speaker.local.",
		map[string]string{"id": testCastUUIDComp},
		[]string{testHostAlt, "2001:db8::1"},
	)
	if !ok || uuid != testCastUUID || host != testHostAlt {
		t.Fatalf("%s %s %v", uuid, host, ok)
	}
}

func TestReapplyFailedHostsUpdatesOnlyFailures(t *testing.T) {
	resolver := NewResolver()
	resolver.Cache = map[string]string{
		testCastUUID:      testHostUpdated,
		testCastUUIDOther: testHostAlt,
	}
	ok := testTarget(testHostAlt)
	ok.EntityID = "media_player.kitchen"
	ok.CastUUID = testCastUUIDOther
	failed := testTarget(testHost)
	updated := resolver.ReapplyFailedHosts(
		[]PingTarget{ok, failed},
		map[string]struct{}{"media_player.example_speaker": {}},
		map[string]string{},
	)
	if updated[0].Host != testHostAlt {
		t.Fatal(updated[0].Host)
	}
	if updated[1].Host != testHostUpdated {
		t.Fatal(updated[1].Host)
	}
}

func TestReapplyFailedHostsSkipsPinned(t *testing.T) {
	resolver := NewResolver()
	resolver.Cache = map[string]string{testCastUUID: testHostUpdated}
	failed := testTarget(testHost)
	updated := resolver.ReapplyFailedHosts(
		[]PingTarget{failed},
		map[string]struct{}{"media_player.example_speaker": {}},
		map[string]string{"media_player.example_speaker": testHost},
	)
	if updated[0].Host != testHost {
		t.Fatal(updated[0].Host)
	}
}

func TestDiscoverCastTargetsFiltersAndDedupes(t *testing.T) {
	disabled := "user"
	entities := []ha.EntityRegistryEntry{
		{EntityID: "media_player.example_speaker", Platform: "cast", DeviceID: "dev1", OriginalName: strPtr("Example speaker")},
		{EntityID: "media_player.example_speaker_2", Platform: "cast", DeviceID: "dev1", OriginalName: strPtr("Example speaker")},
		{EntityID: "media_player.cast_group", Platform: "cast", DeviceID: "dev2", OriginalName: strPtr("Example group")},
		{EntityID: "media_player.example_tv", Platform: "cast", DeviceID: "dev3", OriginalName: strPtr("Example TV")},
		{EntityID: "media_player.disabled_speaker", Platform: "cast", DeviceID: "dev4", DisabledBy: &disabled, OriginalName: strPtr("Disabled")},
	}
	devices := []ha.DeviceRegistryEntry{
		{ID: "dev1", Name: "Example speaker", Manufacturer: "Google Inc.", Model: "Google Home", Identifiers: ha.IdentifierPairs{{"cast", testCastUUID}}},
		{ID: "dev2", Name: "Example group", Manufacturer: "Google Inc.", Model: "Google Cast Group", Identifiers: ha.IdentifierPairs{{"cast", testCastGroupUUID}}},
		{ID: "dev3", Name: "Example TV", Manufacturer: "Example Corp", Model: "Example Streamer", Identifiers: ha.IdentifierPairs{{"cast", testTVCastUUID}}},
		{ID: "dev4", Name: "Disabled", Manufacturer: "Google Inc.", Model: "Google Home Mini", Identifiers: ha.IdentifierPairs{{"cast", testDisabledUUID}}},
	}
	targets := DiscoverCastTargets(entities, devices, testSettings())
	if len(targets) != 1 {
		t.Fatalf("len %d %+v", len(targets), targets)
	}
	if targets[0].EntityID != "media_player.example_speaker" {
		t.Fatal(targets[0].EntityID)
	}
	if targets[0].CastUUID != testCastUUID {
		t.Fatal(targets[0].CastUUID)
	}
}

func strPtr(s string) *string { return &s }

func fptr(v float64) *float64 { return &v }

func TestBuildDeviceAndSummaryPayload(t *testing.T) {
	target := testTarget(testHost)
	lat := 12.4
	result := ProbeResult{Reachable: true, LatencyMS: &lat, ProbedAt: "2026-08-17T22:30:00+00:00"}
	avg := 14.1
	stats := ProbeStats{Probes: 10, Successes: 9, SuccessRate: 0.9, AvgLatencyMS: &avg, ConsecutiveFailures: 0, LastSuccessAt: "2026-08-17T22:30:00+00:00"}
	payload := BuildDevicePayload(target, result, stats, "tcp8008")
	if payload["entity_id"] != "media_player.example_speaker" {
		t.Fatal(payload["entity_id"])
	}
	if payload["reachable"] != true {
		t.Fatal(payload["reachable"])
	}
	if payload["latency_ms"] != &lat && payload["latency_ms"] != result.LatencyMS {
		t.Fatal(payload["latency_ms"])
	}
	if payload["method"] != "tcp8008" {
		t.Fatal(payload["method"])
	}
	st := payload["stats"].(map[string]any)
	if st["success_rate"] != 0.9 {
		t.Fatal(st["success_rate"])
	}
	ok := ProbeResult{Reachable: true, LatencyMS: fptr(10.0), ProbedAt: "t1"}
	fail := ProbeResult{Reachable: false, Error: "timeout", ProbedAt: "t2"}
	summary := BuildSummaryPayload([]ProbeOutcome{
		{Target: target, Result: ok, Stats: ProbeStats{Probes: 1, Successes: 1, SuccessRate: 1.0, AvgLatencyMS: fptr(10.0)}},
		{Target: target.WithHost(testHostAlt), Result: fail, Stats: ProbeStats{Probes: 1, ConsecutiveFailures: 1}},
	}, "tcp8008")
	if summary["device_count"] != 2 || summary["reachable_count"] != 1 || summary["unreachable_count"] != 1 {
		t.Fatalf("%v", summary)
	}
}

func TestDiscoveryIncludesLatencyAndReachable(t *testing.T) {
	configs := BuildDiscoveryConfigs("home/ping", testTarget(testHost))
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
	if latency.ObjectID != "pinger_mqtt_example_speaker_latency" {
		t.Fatal(latency.ObjectID)
	}
	if latency.Payload["state_topic"] != "home/ping/example_speaker/current" {
		t.Fatal(latency.Payload["state_topic"])
	}
	if latency.Payload["value_template"] != "{{ value_json.latency_ms }}" {
		t.Fatal(latency.Payload["value_template"])
	}
	if latency.Payload["unit_of_measurement"] != "ms" {
		t.Fatal(latency.Payload["unit_of_measurement"])
	}
	device := latency.Payload["device"].(map[string]any)
	if device["identifiers"].([]string)[0] != "pinger_mqtt_example_speaker" {
		t.Fatal(device["identifiers"])
	}
	if device["name"] != "Pinger MQTT (Example speaker)" {
		t.Fatal(device["name"])
	}
	if reachable.ObjectID != "pinger_mqtt_example_speaker_reachable" {
		t.Fatal(reachable.ObjectID)
	}
	if reachable.Payload["device_class"] != "connectivity" {
		t.Fatal(reachable.Payload["device_class"])
	}
}

func TestProbeTCPSuccessAndTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	result := probeTCP("127.0.0.1", port, 2000)
	if !result.Reachable || result.LatencyMS == nil || result.Error != "" {
		t.Fatalf("%+v", result)
	}

	p := NewPinger(testSettings())
	p.SetProbeTCP(func(host string, port, timeoutMS int) ProbeResult {
		return ProbeResult{Reachable: false, Error: "timeout", ProbedAt: "t"}
	})
	res, _ := p.Probe(testTarget(testHost))
	if res.Reachable || res.Error != "timeout" {
		t.Fatalf("%+v", res)
	}
}

func TestPingerRollingStats(t *testing.T) {
	settings := testSettings()
	settings.PingStatsWindow = 3
	p := NewPinger(settings)
	seq := []ProbeResult{
		{Reachable: true, LatencyMS: fptr(10.0), ProbedAt: "t1"},
		{Reachable: true, LatencyMS: fptr(20.0), ProbedAt: "t2"},
		{Reachable: false, Error: "timeout", ProbedAt: "t3"},
	}
	i := 0
	p.SetProbeTCP(func(host string, port, timeoutMS int) ProbeResult {
		r := seq[i]
		i++
		return r
	})
	target := testTarget(testHost)
	p.Probe(target)
	p.Probe(target)
	_, stats := p.Probe(target)
	if stats.Probes != 3 {
		t.Fatal(stats.Probes)
	}
	if stats.SuccessRate != 0.667 {
		t.Fatal(stats.SuccessRate)
	}
	if stats.AvgLatencyMS == nil || *stats.AvgLatencyMS != 15.0 {
		t.Fatalf("%v", stats.AvgLatencyMS)
	}
	if stats.ConsecutiveFailures != 1 {
		t.Fatal(stats.ConsecutiveFailures)
	}
}

func TestUnresolvedHost(t *testing.T) {
	p := NewPinger(testSettings())
	result, stats := p.Probe(testTarget(""))
	if result.Reachable || result.Error != "unresolved_host" {
		t.Fatalf("%+v", result)
	}
	if stats.ConsecutiveFailures != 1 {
		t.Fatal(stats.ConsecutiveFailures)
	}
}

func TestResolveTargetsUsesCache(t *testing.T) {
	resolver := NewResolver()
	resolver.Browse = func(ctx context.Context) (map[string]string, error) {
		return map[string]string{testCastUUID: testHost}, nil
	}
	targets, err := resolver.ResolveTargets(context.Background(), []PingTarget{testTarget("")})
	if err != nil {
		t.Fatal(err)
	}
	if targets[0].Host != testHost {
		t.Fatal(targets[0].Host)
	}
	_ = time.Second
}

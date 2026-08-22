package pinger

import (
	"log/slog"
	"math"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ProbeResult struct {
	Reachable bool
	LatencyMS *float64
	Error     string
	ProbedAt  string
}

type ProbeStats struct {
	Probes               int
	Successes            int
	SuccessRate          float64
	AvgLatencyMS         *float64
	ConsecutiveFailures  int
	LastSuccessAt        string
}

func (s ProbeStats) ToMap() map[string]any {
	return map[string]any{
		"probes":               s.Probes,
		"success_rate":         s.SuccessRate,
		"avg_latency_ms":       s.AvgLatencyMS,
		"consecutive_failures": s.ConsecutiveFailures,
		"last_success_at":      nilIfEmpty(s.LastSuccessAt),
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type sample struct {
	ok  bool
	lat *float64
}

type targetTracker struct {
	target               PingTarget
	history              []sample
	consecutiveFailures  int
	lastSuccessAt        string
}

func (t *targetTracker) record(result ProbeResult, window int) ProbeStats {
	t.history = append(t.history, sample{ok: result.Reachable, lat: result.LatencyMS})
	if window > 0 && len(t.history) > window {
		t.history = t.history[len(t.history)-window:]
	}
	if result.Reachable {
		t.consecutiveFailures = 0
		t.lastSuccessAt = result.ProbedAt
	} else {
		t.consecutiveFailures++
	}
	probes := len(t.history)
	successes := 0
	var latencies []float64
	for _, s := range t.history {
		if s.ok {
			successes++
			if s.lat != nil {
				latencies = append(latencies, *s.lat)
			}
		}
	}
	stats := ProbeStats{
		Probes:              probes,
		Successes:           successes,
		ConsecutiveFailures: t.consecutiveFailures,
		LastSuccessAt:       t.lastSuccessAt,
	}
	if probes > 0 {
		stats.SuccessRate = math.Round((float64(successes)/float64(probes))*1000) / 1000
	}
	if len(latencies) > 0 {
		sum := 0.0
		for _, v := range latencies {
			sum += v
		}
		avg := math.Round((sum/float64(len(latencies)))*10) / 10
		stats.AvgLatencyMS = &avg
	}
	return stats
}

type ProbeOutcome struct {
	Target PingTarget
	Result ProbeResult
	Stats  ProbeStats
}

type Pinger struct {
	settings Settings
	trackers map[string]*targetTracker
	ProbeTCP func(host string, port, timeoutMS int) ProbeResult
	ProbeICMP func(host string, timeoutMS int) ProbeResult
}

func NewPinger(settings Settings) *Pinger {
	return &Pinger{
		settings:  settings,
		trackers:  map[string]*targetTracker{},
		ProbeTCP:  probeTCP,
		ProbeICMP: probeICMP,
	}
}

func utcNowISO() string {
	return time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

func probeTCP(host string, port, timeoutMS int) ProbeResult {
	probedAt := utcNowISO()
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), time.Duration(timeoutMS)*time.Millisecond)
	if err != nil {
		msg := err.Error()
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			msg = "timeout"
		} else if strings.Contains(msg, "connection refused") {
			msg = "refused"
		}
		return ProbeResult{Reachable: false, Error: msg, ProbedAt: probedAt}
	}
	_ = conn.Close()
	lat := math.Round(time.Since(start).Seconds()*1000*10) / 10
	return ProbeResult{Reachable: true, LatencyMS: &lat, ProbedAt: probedAt}
}

func probeICMP(host string, timeoutMS int) ProbeResult {
	probedAt := utcNowISO()
	timeoutSec := int(math.Round(float64(timeoutMS) / 1000))
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	cmd := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeoutSec), host)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return ProbeResult{Reachable: false, Error: "unreachable", ProbedAt: probedAt}
		}
		if strings.Contains(err.Error(), "executable file not found") {
			return ProbeResult{Reachable: false, Error: "ping_not_available", ProbedAt: probedAt}
		}
		return ProbeResult{Reachable: false, Error: err.Error(), ProbedAt: probedAt}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "time=") {
			part := strings.Split(strings.Split(line, "time=")[1], " ")[0]
			part = strings.TrimSuffix(part, "ms")
			v, convErr := strconv.ParseFloat(part, 64)
			if convErr == nil {
				lat := math.Round(v*10) / 10
				return ProbeResult{Reachable: true, LatencyMS: &lat, ProbedAt: probedAt}
			}
		}
	}
	return ProbeResult{Reachable: true, ProbedAt: probedAt}
}

func (p *Pinger) syncTrackers(targets []PingTarget) {
	active := map[string]struct{}{}
	for _, t := range targets {
		active[t.EntityID] = struct{}{}
		if tr, ok := p.trackers[t.EntityID]; ok {
			tr.target = t
		} else {
			p.trackers[t.EntityID] = &targetTracker{target: t}
		}
	}
	for id := range p.trackers {
		if _, ok := active[id]; !ok {
			delete(p.trackers, id)
		}
	}
}

func (p *Pinger) Probe(target PingTarget) (ProbeResult, ProbeStats) {
	var result ProbeResult
	if target.Host == "" {
		result = ProbeResult{Reachable: false, Error: "unresolved_host", ProbedAt: utcNowISO()}
	} else if p.settings.PingMethod == "icmp" {
		result = p.ProbeICMP(target.Host, p.settings.PingTimeoutMS)
	} else {
		result = p.ProbeTCP(target.Host, p.settings.PingTCPPort, p.settings.PingTimeoutMS)
	}
	tr, ok := p.trackers[target.EntityID]
	if !ok {
		tr = &targetTracker{target: target}
		p.trackers[target.EntityID] = tr
	}
	stats := tr.record(result, p.settings.PingStatsWindow)
	return result, stats
}

func (p *Pinger) ProbeAll(targets []PingTarget) []ProbeOutcome {
	p.syncTrackers(targets)
	out := make([]ProbeOutcome, 0, len(targets))
	for _, target := range targets {
		result, stats := p.Probe(target)
		out = append(out, ProbeOutcome{Target: target, Result: result, Stats: stats})
		if result.Reachable {
			lat := 0.0
			if result.LatencyMS != nil {
				lat = *result.LatencyMS
			}
			slog.Debug("probe ok", "name", target.FriendlyName, "host", target.Host, "ms", lat)
		} else {
			host := target.Host
			if host == "" {
				host = "unresolved"
			}
			slog.Warn("probe fail", "name", target.FriendlyName, "host", host, "error", result.Error)
		}
	}
	return out
}

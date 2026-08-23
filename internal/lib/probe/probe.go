// Package probe provides TCP/ICMP connectivity probes and rolling stats.
package probe

import (
	"log/slog"
	"math"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Reachable bool
	LatencyMS *float64
	Error     string
	ProbedAt  string
}

type Stats struct {
	Probes              int
	Successes           int
	SuccessRate         float64
	AvgLatencyMS        *float64
	ConsecutiveFailures int
	LastSuccessAt       string
}

func (s Stats) ToMap() map[string]any {
	return map[string]any{
		"probes":               s.Probes,
		"success_rate":         s.SuccessRate,
		"avg_latency_ms":       s.AvgLatencyMS,
		"consecutive_failures": s.ConsecutiveFailures,
		"last_success_at":      NilIfEmpty(s.LastSuccessAt),
	}
}

func NilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func UTCNowISO() string {
	return time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

type sample struct {
	ok  bool
	lat *float64
}

type tracker struct {
	history             []sample
	consecutiveFailures int
	lastSuccessAt       string
}

func (t *tracker) record(result Result, window int) Stats {
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
	stats := Stats{
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

// Config controls how a Prober reaches hosts and windows stats.
type Config struct {
	Method      string // "icmp" or anything else (treated as TCP)
	TimeoutMS   int
	TCPPort     int
	StatsWindow int
}

// Prober probes hosts by ID and keeps rolling reachability stats.
type Prober struct {
	config    Config
	trackers  map[string]*tracker
	ProbeTCP  func(host string, port, timeoutMS int) Result
	ProbeICMP func(host string, timeoutMS int) Result
}

func NewProber(config Config) *Prober {
	return &Prober{
		config:    config,
		trackers:  map[string]*tracker{},
		ProbeTCP:  TCP,
		ProbeICMP: ICMP,
	}
}

func TCP(host string, port, timeoutMS int) Result {
	probedAt := UTCNowISO()
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), time.Duration(timeoutMS)*time.Millisecond)
	if err != nil {
		msg := err.Error()
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			msg = "timeout"
		} else if strings.Contains(msg, "connection refused") {
			msg = "refused"
		}
		return Result{Reachable: false, Error: msg, ProbedAt: probedAt}
	}
	_ = conn.Close()
	lat := math.Round(time.Since(start).Seconds()*1000*10) / 10
	return Result{Reachable: true, LatencyMS: &lat, ProbedAt: probedAt}
}

func ICMP(host string, timeoutMS int) Result {
	probedAt := UTCNowISO()
	timeoutSec := int(math.Round(float64(timeoutMS) / 1000))
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	cmd := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeoutSec), host)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return Result{Reachable: false, Error: "unreachable", ProbedAt: probedAt}
		}
		if strings.Contains(err.Error(), "executable file not found") {
			return Result{Reachable: false, Error: "ping_not_available", ProbedAt: probedAt}
		}
		return Result{Reachable: false, Error: err.Error(), ProbedAt: probedAt}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "time=") {
			part := strings.Split(strings.Split(line, "time=")[1], " ")[0]
			part = strings.TrimSuffix(part, "ms")
			v, convErr := strconv.ParseFloat(part, 64)
			if convErr == nil {
				lat := math.Round(v*10) / 10
				return Result{Reachable: true, LatencyMS: &lat, ProbedAt: probedAt}
			}
		}
	}
	return Result{Reachable: true, ProbedAt: probedAt}
}

func (p *Prober) SyncIDs(ids []string) {
	active := map[string]struct{}{}
	for _, id := range ids {
		active[id] = struct{}{}
		if _, ok := p.trackers[id]; !ok {
			p.trackers[id] = &tracker{}
		}
	}
	for id := range p.trackers {
		if _, ok := active[id]; !ok {
			delete(p.trackers, id)
		}
	}
}

func (p *Prober) Probe(id, host, logName string) (Result, Stats) {
	var result Result
	if host == "" {
		result = Result{Reachable: false, Error: "unresolved_host", ProbedAt: UTCNowISO()}
	} else if p.config.Method == "icmp" {
		result = p.ProbeICMP(host, p.config.TimeoutMS)
	} else {
		result = p.ProbeTCP(host, p.config.TCPPort, p.config.TimeoutMS)
	}
	tr, ok := p.trackers[id]
	if !ok {
		tr = &tracker{}
		p.trackers[id] = tr
	}
	stats := tr.record(result, p.config.StatsWindow)
	if result.Reachable {
		lat := 0.0
		if result.LatencyMS != nil {
			lat = *result.LatencyMS
		}
		slog.Debug("probe ok", "name", logName, "host", host, "ms", lat)
	} else {
		displayHost := host
		if displayHost == "" {
			displayHost = "unresolved"
		}
		slog.Warn("probe fail", "name", logName, "host", displayHost, "error", result.Error)
	}
	return result, stats
}

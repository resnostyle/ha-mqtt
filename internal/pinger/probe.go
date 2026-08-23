package pinger

import (
	"github.com/resnostyle/ha-mqtt/internal/lib/probe"
)

type ProbeResult = probe.Result
type ProbeStats = probe.Stats

type ProbeOutcome struct {
	Target PingTarget
	Result ProbeResult
	Stats  ProbeStats
}

type Pinger struct {
	settings Settings
	prober   *probe.Prober
}

func NewPinger(settings Settings) *Pinger {
	return &Pinger{
		settings: settings,
		prober: probe.NewProber(probe.Config{
			Method:      settings.PingMethod,
			TimeoutMS:   settings.PingTimeoutMS,
			TCPPort:     settings.PingTCPPort,
			StatsWindow: settings.PingStatsWindow,
		}),
	}
}

// SetProbeTCP overrides the underlying TCP probe (tests).
func (p *Pinger) SetProbeTCP(fn func(host string, port, timeoutMS int) ProbeResult) {
	p.prober.ProbeTCP = fn
}

func (p *Pinger) Probe(target PingTarget) (ProbeResult, ProbeStats) {
	return p.prober.Probe(target.EntityID, target.Host, target.FriendlyName)
}

func (p *Pinger) ProbeAll(targets []PingTarget) []ProbeOutcome {
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.EntityID)
	}
	p.prober.SyncIDs(ids)

	out := make([]ProbeOutcome, 0, len(targets))
	for _, target := range targets {
		result, stats := p.Probe(target)
		out = append(out, ProbeOutcome{Target: target, Result: result, Stats: stats})
	}
	return out
}

func utcNowISO() string {
	return probe.UTCNowISO()
}

func nilIfEmpty(s string) any {
	return probe.NilIfEmpty(s)
}

func probeTCP(host string, port, timeoutMS int) ProbeResult {
	return probe.TCP(host, port, timeoutMS)
}

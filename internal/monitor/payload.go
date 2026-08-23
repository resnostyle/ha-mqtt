package monitor

import (
	"github.com/resnostyle/ha-mqtt/internal/lib/probe"
)

func BuildHostPayload(target HostTarget, result probe.Result, stats probe.Stats, method string) map[string]any {
	return map[string]any{
		"name":       target.Name,
		"slug":       target.Slug,
		"host":       target.Host,
		"reachable":  result.Reachable,
		"latency_ms": result.LatencyMS,
		"error":      probe.NilIfEmpty(result.Error),
		"method":     method,
		"stats":      stats.ToMap(),
		"probed_at":  result.ProbedAt,
		"published":  probe.UTCNowISO(),
	}
}

func BuildSummaryPayload(results []ProbeOutcome, method string) map[string]any {
	devices := make([]map[string]any, 0, len(results))
	reachable := 0
	for _, r := range results {
		if r.Result.Reachable {
			reachable++
		}
		devices = append(devices, map[string]any{
			"name":       r.Target.Name,
			"slug":       r.Target.Slug,
			"host":       r.Target.Host,
			"reachable":  r.Result.Reachable,
			"latency_ms": r.Result.LatencyMS,
			"error":      probe.NilIfEmpty(r.Result.Error),
			"stats":      r.Stats.ToMap(),
		})
	}
	return map[string]any{
		"method":            method,
		"host_count":        len(results),
		"reachable_count":   reachable,
		"unreachable_count": len(results) - reachable,
		"hosts":             devices,
		"published":         probe.UTCNowISO(),
	}
}

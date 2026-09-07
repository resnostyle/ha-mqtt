package pinger

import "github.com/resnostyle/mqttkit/payload"

func BuildDevicePayload(target PingTarget, result ProbeResult, stats ProbeStats, method string) map[string]any {
	return map[string]any{
		"entity_id":     target.EntityID,
		"friendly_name": target.FriendlyName,
		"slug":          target.Slug,
		"host":          payload.NilIfEmpty(target.Host),
		"cast_uuid":     target.CastUUID,
		"manufacturer":  target.Manufacturer,
		"model":         target.Model,
		"area_id":       payload.NilIfEmpty(target.AreaID),
		"reachable":     result.Reachable,
		"latency_ms":    result.LatencyMS,
		"error":         payload.NilIfEmpty(result.Error),
		"method":        method,
		"stats":         stats.ToMap(),
		"probed_at":     result.ProbedAt,
		"published":     payload.UTCNowISO(),
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
			"slug":          r.Target.Slug,
			"entity_id":     r.Target.EntityID,
			"friendly_name": r.Target.FriendlyName,
			"host":          payload.NilIfEmpty(r.Target.Host),
			"reachable":     r.Result.Reachable,
			"latency_ms":    r.Result.LatencyMS,
			"error":         payload.NilIfEmpty(r.Result.Error),
			"stats":         r.Stats.ToMap(),
		})
	}
	return map[string]any{
		"method":            method,
		"device_count":      len(results),
		"reachable_count":   reachable,
		"unreachable_count": len(results) - reachable,
		"devices":           devices,
		"published":         payload.UTCNowISO(),
	}
}

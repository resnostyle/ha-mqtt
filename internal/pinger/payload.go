package pinger

func BuildDevicePayload(target PingTarget, result ProbeResult, stats ProbeStats, method string) map[string]any {
	return map[string]any{
		"entity_id":     target.EntityID,
		"friendly_name": target.FriendlyName,
		"slug":          target.Slug,
		"host":          nilIfEmpty(target.Host),
		"cast_uuid":     target.CastUUID,
		"manufacturer":  target.Manufacturer,
		"model":         target.Model,
		"area_id":       nilIfEmpty(target.AreaID),
		"reachable":     result.Reachable,
		"latency_ms":    result.LatencyMS,
		"error":         nilIfEmpty(result.Error),
		"method":        method,
		"stats":         stats.ToMap(),
		"probed_at":     result.ProbedAt,
		"published":     utcNowISO(),
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
			"host":          nilIfEmpty(r.Target.Host),
			"reachable":     r.Result.Reachable,
			"latency_ms":    r.Result.LatencyMS,
			"error":         nilIfEmpty(r.Result.Error),
			"stats":         r.Stats.ToMap(),
		})
	}
	return map[string]any{
		"method":            method,
		"device_count":      len(results),
		"reachable_count":   reachable,
		"unreachable_count": len(results) - reachable,
		"devices":           devices,
		"published":         utcNowISO(),
	}
}

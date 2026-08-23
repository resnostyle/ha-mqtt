package speedtest

func BuildPayload(result Result) map[string]any {
	return map[string]any{
		"download_mbps":   floatOrNil(result.DownloadMbps),
		"upload_mbps":     floatOrNil(result.UploadMbps),
		"ping_ms":         floatOrNil(result.PingMS),
		"jitter_ms":       floatOrNil(result.JitterMS),
		"server_id":       nilIfEmpty(result.ServerID),
		"server_name":     nilIfEmpty(result.ServerName),
		"server_location": nilIfEmpty(result.ServerLocation),
		"ok":              result.OK,
		"error":           nilIfEmpty(result.Error),
		"tested_at":       result.TestedAt,
		"published":       utcNowISO(),
	}
}

func floatOrNil(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

package speedtest

import "github.com/resnostyle/mqttkit/payload"

func BuildPayload(result Result) map[string]any {
	return map[string]any{
		"download_mbps":   floatOrNil(result.DownloadMbps),
		"upload_mbps":     floatOrNil(result.UploadMbps),
		"ping_ms":         floatOrNil(result.PingMS),
		"jitter_ms":       floatOrNil(result.JitterMS),
		"server_id":       payload.NilIfEmpty(result.ServerID),
		"server_name":     payload.NilIfEmpty(result.ServerName),
		"server_location": payload.NilIfEmpty(result.ServerLocation),
		"ok":              result.OK,
		"error":           payload.NilIfEmpty(result.Error),
		"tested_at":       result.TestedAt,
		"published":       payload.UTCNowISO(),
	}
}

func floatOrNil(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

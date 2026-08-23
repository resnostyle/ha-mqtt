package speedtest

import "github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"

const (
	deviceManufacturer = "speedtest-mqtt"
	deviceUID          = "speedtest_mqtt"
)

func deviceBlock() map[string]any {
	return map[string]any{
		"identifiers":  []string{deviceUID},
		"name":         "Speedtest MQTT",
		"manufacturer": deviceManufacturer,
		"model":        "Internet",
	}
}

func BuildDiscoveryConfigs(topicPrefix string) []mqttpub.Config {
	current := topicPrefix + "/current"
	device := deviceBlock()

	download := map[string]any{
		"name":                  "Download",
		"unique_id":             deviceUID + "_download",
		"state_topic":           current,
		"value_template":        "{{ value_json.download_mbps }}",
		"device":                device,
		"object_id":             deviceUID + "_download",
		"state_class":           "measurement",
		"unit_of_measurement":   "Mbps",
		"icon":                  "mdi:download",
		"json_attributes_topic": current,
	}
	upload := map[string]any{
		"name":                  "Upload",
		"unique_id":             deviceUID + "_upload",
		"state_topic":           current,
		"value_template":        "{{ value_json.upload_mbps }}",
		"device":                device,
		"object_id":             deviceUID + "_upload",
		"state_class":           "measurement",
		"unit_of_measurement":   "Mbps",
		"icon":                  "mdi:upload",
		"json_attributes_topic": current,
	}
	ping := map[string]any{
		"name":                  "Ping",
		"unique_id":             deviceUID + "_ping",
		"state_topic":           current,
		"value_template":        "{{ value_json.ping_ms }}",
		"device":                device,
		"object_id":             deviceUID + "_ping",
		"state_class":           "measurement",
		"unit_of_measurement":   "ms",
		"icon":                  "mdi:timer-outline",
		"json_attributes_topic": current,
	}
	jitter := map[string]any{
		"name":                  "Jitter",
		"unique_id":             deviceUID + "_jitter",
		"state_topic":           current,
		"value_template":        "{{ value_json.jitter_ms }}",
		"device":                device,
		"object_id":             deviceUID + "_jitter",
		"state_class":           "measurement",
		"unit_of_measurement":   "ms",
		"icon":                  "mdi:chart-timeline-variant",
		"json_attributes_topic": current,
	}
	return []mqttpub.Config{
		{ObjectID: deviceUID + "_download", Component: "sensor", Payload: download},
		{ObjectID: deviceUID + "_upload", Component: "sensor", Payload: upload},
		{ObjectID: deviceUID + "_ping", Component: "sensor", Payload: ping},
		{ObjectID: deviceUID + "_jitter", Component: "sensor", Payload: jitter},
	}
}

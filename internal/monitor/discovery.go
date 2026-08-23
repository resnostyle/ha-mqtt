package monitor

import "github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"

const deviceManufacturer = "monitor-mqtt"

func deviceBlock(target HostTarget) map[string]any {
	return map[string]any{
		"identifiers":  []string{"monitor_mqtt_" + target.Slug},
		"name":         "Monitor MQTT (" + target.Name + ")",
		"manufacturer": deviceManufacturer,
		"model":        "Host",
	}
}

func BuildDiscoveryConfigs(topicPrefix string, target HostTarget) []mqttpub.Config {
	current := topicPrefix + "/" + target.Slug + "/current"
	device := deviceBlock(target)
	uid := "monitor_mqtt_" + target.Slug

	latency := map[string]any{
		"name":                  "Latency",
		"unique_id":             uid + "_latency",
		"state_topic":           current,
		"value_template":        "{{ value_json.latency_ms }}",
		"device":                device,
		"object_id":             uid + "_latency",
		"state_class":           "measurement",
		"unit_of_measurement":   "ms",
		"icon":                  "mdi:speedometer",
		"json_attributes_topic": current,
	}
	reachable := map[string]any{
		"name":                  "Reachable",
		"unique_id":             uid + "_reachable",
		"state_topic":           current,
		"value_template":        "{{ 'true' if value_json.reachable else 'false' }}",
		"device":                device,
		"object_id":             uid + "_reachable",
		"device_class":          "connectivity",
		"payload_on":            "true",
		"payload_off":           "false",
		"icon":                  "mdi:lan-connect",
		"json_attributes_topic": current,
	}
	return []mqttpub.Config{
		{ObjectID: uid + "_latency", Component: "sensor", Payload: latency},
		{ObjectID: uid + "_reachable", Component: "binary_sensor", Payload: reachable},
	}
}

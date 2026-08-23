package env

import (
	"os"
	"strings"
)

// MQTT is MQTT broker + discovery + logging configuration (no Home Assistant).
type MQTT struct {
	MQTTHost             string
	MQTTPort             int
	MQTTUsername         string
	MQTTPassword         string
	MQTTTopicPrefix      string
	MQTTClientID         string
	MQTTDiscoveryEnabled bool
	MQTTDiscoveryPrefix  string
	LogLevel             string
}

// LoadMQTT loads MQTT and logging settings without requiring HA credentials.
func LoadMQTT(defaultTopicPrefix, defaultClientID string) (MQTT, error) {
	port, err := Int("MQTT_PORT", 1883)
	if err != nil {
		return MQTT{}, err
	}
	return MQTT{
		MQTTHost:             Get("MQTT_HOST", "127.0.0.1"),
		MQTTPort:             port,
		MQTTUsername:         strings.TrimSpace(os.Getenv("MQTT_USERNAME")),
		MQTTPassword:         strings.TrimSpace(os.Getenv("MQTT_PASSWORD")),
		MQTTTopicPrefix:      strings.TrimRight(Get("MQTT_TOPIC_PREFIX", defaultTopicPrefix), "/"),
		MQTTClientID:         Get("MQTT_CLIENT_ID", defaultClientID),
		MQTTDiscoveryEnabled: Bool("MQTT_DISCOVERY_ENABLED", true),
		MQTTDiscoveryPrefix:  strings.TrimRight(Get("MQTT_DISCOVERY_PREFIX", "homeassistant"), "/"),
		LogLevel:             strings.ToUpper(Get("LOG_LEVEL", "INFO")),
	}, nil
}

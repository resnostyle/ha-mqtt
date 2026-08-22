// Package env loads common environment variables shared by services.
package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Common is HA + MQTT + logging configuration shared by every service.
type Common struct {
	HAURL                  string
	HAToken                string
	MQTTHost               string
	MQTTPort               int
	MQTTUsername           string
	MQTTPassword           string
	MQTTTopicPrefix        string
	MQTTClientID           string
	MQTTDiscoveryEnabled   bool
	MQTTDiscoveryPrefix    string
	LogLevel               string
}

func Require(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return value, nil
}

func Get(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func Int(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return n, nil
}

func Bool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func CSV(name string, fallback string) map[string]struct{} {
	raw := Get(name, fallback)
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out[part] = struct{}{}
	}
	return out
}

func LoadCommon(defaultTopicPrefix, defaultClientID string) (Common, error) {
	url, err := Require("HA_URL")
	if err != nil {
		return Common{}, err
	}
	token, err := Require("HA_TOKEN")
	if err != nil {
		return Common{}, err
	}
	port, err := Int("MQTT_PORT", 1883)
	if err != nil {
		return Common{}, err
	}
	return Common{
		HAURL:                strings.TrimRight(url, "/"),
		HAToken:              token,
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

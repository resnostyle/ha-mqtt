package pinger

import (
	"github.com/resnostyle/mqttkit/hadisc"
	"github.com/resnostyle/mqttkit/mqttpub"
)

const deviceManufacturer = "pinger-mqtt"

func deviceBlock(target PingTarget) map[string]any {
	model := target.Model
	if model == "" {
		model = "Google Cast"
	}
	return hadisc.Device([]string{"pinger_mqtt_" + target.Slug}, "Pinger MQTT ("+target.FriendlyName+")", deviceManufacturer, model)
}

func BuildDiscoveryConfigs(topicPrefix string, target PingTarget) []mqttpub.Config {
	current := topicPrefix + "/" + target.Slug + "/current"
	uid := "pinger_mqtt_" + target.Slug
	return hadisc.LatencyReachable(uid, current, deviceBlock(target))
}

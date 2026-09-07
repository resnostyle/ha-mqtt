package monitor

import (
	"github.com/resnostyle/mqttkit/hadisc"
	"github.com/resnostyle/mqttkit/mqttpub"
)

const deviceManufacturer = "monitor-mqtt"

func deviceBlock(target HostTarget) map[string]any {
	return hadisc.Device([]string{"monitor_mqtt_" + target.Slug}, "Monitor MQTT ("+target.Name+")", deviceManufacturer, "Host")
}

func BuildDiscoveryConfigs(topicPrefix string, target HostTarget) []mqttpub.Config {
	current := topicPrefix + "/" + target.Slug + "/current"
	uid := "monitor_mqtt_" + target.Slug
	return hadisc.LatencyReachable(uid, current, deviceBlock(target))
}

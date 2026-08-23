package speedtest

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/resnostyle/ha-mqtt/internal/lib/env"
)

type Settings struct {
	env.MQTT
	IntervalSeconds int
	ServerID        string // empty = auto-select nearest
}

func FromEnv() (Settings, error) {
	mqtt, err := env.LoadMQTT("home/speedtest", "speedtest-mqtt")
	if err != nil {
		return Settings{}, err
	}
	interval, err := env.Int("SPEEDTEST_INTERVAL_SECONDS", 3600)
	if err != nil {
		return Settings{}, err
	}
	if interval < 1 {
		return Settings{}, fmt.Errorf("SPEEDTEST_INTERVAL_SECONDS must be >= 1")
	}
	serverID := strings.TrimSpace(os.Getenv("SPEEDTEST_SERVER_ID"))
	if serverID != "" {
		if _, err := strconv.Atoi(serverID); err != nil {
			return Settings{}, fmt.Errorf("invalid SPEEDTEST_SERVER_ID %q; expected numeric id", serverID)
		}
	}
	return Settings{
		MQTT:            mqtt,
		IntervalSeconds: interval,
		ServerID:        serverID,
	}, nil
}

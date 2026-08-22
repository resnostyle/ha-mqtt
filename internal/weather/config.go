package weather

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/resnostyle/ha-mqtt/internal/lib/env"
)

var sourceNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type Source struct {
	Name     string
	EntityID string
	Label    string
}

func (s Source) TopicSuffix() string {
	return s.Name
}

type Settings struct {
	env.Common
	WeatherSources      []Source
	TemperatureUnit     string
	PressureUnit        string
	WindSpeedUnit       string
	VisibilityUnit      string
	SunEnabled          bool
	SunEntity           string
	MQTTSunTopicPrefix  string
	PollIntervalSeconds int
	HourlyForecastHours int
}

func FromEnv() (Settings, error) {
	common, err := env.LoadCommon("home/weather", "weather-mqtt")
	if err != nil {
		return Settings{}, err
	}
	sources, err := ParseWeatherSources(os.Getenv("HA_WEATHER_SOURCES"), os.Getenv("HA_WEATHER_ENTITY"))
	if err != nil {
		return Settings{}, err
	}
	poll, err := env.Int("POLL_INTERVAL_SECONDS", 900)
	if err != nil {
		return Settings{}, err
	}
	hourly, err := env.Int("HOURLY_FORECAST_HOURS", 48)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		Common:              common,
		WeatherSources:      sources,
		TemperatureUnit:     env.Get("TEMPERATURE_UNIT", "°F"),
		PressureUnit:        env.Get("PRESSURE_UNIT", "inHg"),
		WindSpeedUnit:       env.Get("WIND_SPEED_UNIT", "mph"),
		VisibilityUnit:      env.Get("VISIBILITY_UNIT", "mi"),
		SunEnabled:          env.Bool("SUN_ENABLED", true),
		SunEntity:           env.Get("HA_SUN_ENTITY", "sun.sun"),
		MQTTSunTopicPrefix:  strings.TrimRight(env.Get("MQTT_SUN_TOPIC_PREFIX", "home/sun"), "/"),
		PollIntervalSeconds: poll,
		HourlyForecastHours: hourly,
	}, nil
}

func ParseWeatherSources(raw, legacyEntity string) ([]Source, error) {
	raw = strings.TrimSpace(raw)
	legacyEntity = strings.TrimSpace(legacyEntity)
	if raw == "" {
		entity := legacyEntity
		if entity == "" {
			entity = "weather.pirateweather"
		}
		return []Source{{
			Name:     "pirate",
			EntityID: entity,
			Label:    defaultLabel("pirate"),
		}}, nil
	}

	var sources []Source
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, entityID, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("invalid HA_WEATHER_SOURCES entry %q; expected name:entity_id (e.g. metno:weather.forecast_home)", part)
		}
		name = strings.ToLower(strings.TrimSpace(name))
		entityID = strings.TrimSpace(entityID)
		if !sourceNameRE.MatchString(name) {
			return nil, fmt.Errorf("invalid weather source name %q; use lowercase letters, numbers, underscores", name)
		}
		if !strings.HasPrefix(entityID, "weather.") {
			return nil, fmt.Errorf("invalid weather entity %q; expected weather.*", entityID)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate weather source name: %s", name)
		}
		seen[name] = struct{}{}
		sources = append(sources, Source{
			Name:     name,
			EntityID: entityID,
			Label:    defaultLabel(name),
		})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("HA_WEATHER_SOURCES is empty")
	}
	return sources, nil
}

func defaultLabel(name string) string {
	labels := map[string]string{
		"pirate":        "Pirate Weather",
		"metno":         "Met.no",
		"met_no":        "Met.no",
		"forecast_home": "Met.no",
	}
	if label, ok := labels[name]; ok {
		return label
	}
	return titleWords(strings.ReplaceAll(name, "_", " "))
}

func titleWords(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		runes := []rune(strings.ToLower(p))
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToTitle(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

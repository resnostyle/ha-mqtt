package weather

import (
	"time"
)

var conditionLabels = map[string]string{
	"sunny":           "Sunny",
	"clear-night":     "Clear skies tonight",
	"partlycloudy":    "Partly cloudy",
	"cloudy":          "Cloudy",
	"rainy":           "Rain",
	"pouring":         "Heavy rain",
	"snowy":           "Snow",
	"snowy-rainy":     "Snow/rain",
	"fog":             "Foggy",
	"hail":            "Hail",
	"lightning":       "Lightning",
	"lightning-rainy": "Thunderstorms",
	"windy":           "Windy",
	"windy-variant":   "Windy",
	"exceptional":     "Exceptional",
}

func utcNowISO() string {
	return time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
}

func attrs(state map[string]any) map[string]any {
	if a, ok := state["attributes"].(map[string]any); ok && a != nil {
		return a
	}
	return map[string]any{}
}

func BuildCurrent(state map[string]any) map[string]any {
	a := attrs(state)
	updated := state["last_updated"]
	if updated == nil {
		updated = utcNowISO()
	}
	return map[string]any{
		"condition":            state["state"],
		"temperature":          a["temperature"],
		"apparent_temperature": a["apparent_temperature"],
		"dew_point":            a["dew_point"],
		"humidity":             a["humidity"],
		"ozone":                a["ozone"],
		"cloud_coverage":       a["cloud_coverage"],
		"pressure":             a["pressure"],
		"pressure_unit":        a["pressure_unit"],
		"wind_bearing":         a["wind_bearing"],
		"wind_speed":           a["wind_speed"],
		"wind_gust_speed":      a["wind_gust_speed"],
		"wind_speed_unit":      a["wind_speed_unit"],
		"visibility":           a["visibility"],
		"visibility_unit":      a["visibility_unit"],
		"temperature_unit":     a["temperature_unit"],
		"precipitation_unit":   a["precipitation_unit"],
		"attribution":          a["attribution"],
		"entity_id":            state["entity_id"],
		"updated":              updated,
		"published":            utcNowISO(),
	}
}

func BuildSun(state map[string]any) map[string]any {
	a := attrs(state)
	sunState := state["state"]
	updated := state["last_updated"]
	if updated == nil {
		updated = utcNowISO()
	}
	return map[string]any{
		"state":         sunState,
		"daytime":       sunState == "above_horizon",
		"rising":        a["rising"],
		"elevation":     a["elevation"],
		"azimuth":       a["azimuth"],
		"next_dawn":     a["next_dawn"],
		"next_dusk":     a["next_dusk"],
		"next_rising":   a["next_rising"],
		"next_setting":  a["next_setting"],
		"next_noon":     a["next_noon"],
		"next_midnight": a["next_midnight"],
		"entity_id":     state["entity_id"],
		"updated":       updated,
		"published":     utcNowISO(),
	}
}

func BuildForecastPayload(forecast []map[string]any, forecastType, entityID string) map[string]any {
	if forecast == nil {
		forecast = []map[string]any{}
	}
	return map[string]any{
		"type":      forecastType,
		"entity_id": entityID,
		"count":     len(forecast),
		"forecast":  forecast,
		"published": utcNowISO(),
	}
}

func TrimHourly(forecast []map[string]any, hours int) []map[string]any {
	if hours <= 0 || hours >= len(forecast) {
		return forecast
	}
	return forecast[:hours]
}

func Build5Day(forecast []map[string]any, source, unit string) map[string]any {
	if unit == "" {
		unit = "°F"
	}
	const maxDays = 5
	entries := make([]map[string]any, 0, maxDays)
	for i := 0; i < maxDays; i++ {
		if i < len(forecast) {
			day := forecast[i]
			condition, _ := day["condition"].(string)
			label := condition
			if mapped, ok := conditionLabels[condition]; ok {
				label = mapped
			}
			entries = append(entries, map[string]any{
				"day_index":                  i,
				"datetime":                   day["datetime"],
				"condition":                  label,
				"condition_raw":              condition,
				"temperature":                day["temperature"],
				"templow":                    day["templow"],
				"humidity":                   day["humidity"],
				"wind_speed":                 day["wind_speed"],
				"precipitation":              day["precipitation"],
				"precipitation_probability":  day["precipitation_probability"],
			})
			continue
		}
		entries = append(entries, map[string]any{
			"day_index":   i,
			"condition":   "No data",
			"temperature": nil,
			"templow":     nil,
			"humidity":    nil,
			"wind_speed":  nil,
		})
	}
	return map[string]any{
		"config": map[string]any{
			"unit":     unit,
			"max_days": maxDays,
			"fields":   []string{"condition", "temperature", "templow", "humidity", "wind_speed"},
			"source":   source,
		},
		"forecast":  entries,
		"published": utcNowISO(),
	}
}

package weather

import "github.com/resnostyle/ha-mqtt/internal/lib/mqttpub"

const deviceManufacturer = "weather-mqtt"

func deviceBlock(sourceName, sourceLabel string) map[string]any {
	return map[string]any{
		"identifiers":  []string{"weather_mqtt_" + sourceName},
		"name":         "Weather MQTT (" + sourceLabel + ")",
		"manufacturer": deviceManufacturer,
		"model":        sourceLabel + " bridge",
	}
}

func sensor(objectID, name, stateTopic, valueTemplate, uniqueID string, device map[string]any, extras map[string]any) mqttpub.Config {
	cfg := map[string]any{
		"name":           name,
		"unique_id":      uniqueID,
		"state_topic":    stateTopic,
		"value_template": valueTemplate,
		"device":         device,
		"object_id":      objectID,
	}
	for k, v := range extras {
		if v != nil && v != "" {
			cfg[k] = v
		}
	}
	return mqttpub.Config{ObjectID: objectID, Component: "sensor", Payload: cfg}
}

func BuildDiscoveryConfigs(topicPrefix, sourceName, sourceLabel, temperatureUnit, pressureUnit, windSpeedUnit, visibilityUnit string) []mqttpub.Config {
	current := topicPrefix + "/current"
	daily := topicPrefix + "/daily"
	hourly := topicPrefix + "/hourly"
	fiveDay := topicPrefix + "/5day"
	device := deviceBlock(sourceName, sourceLabel)
	uid := "weather_mqtt_" + sourceName

	return []mqttpub.Config{
		sensor(uid+"_condition", "Condition", current, "{{ value_json.condition }}", uid+"_condition", device, map[string]any{
			"icon":                  "mdi:weather-partly-cloudy",
			"json_attributes_topic": current,
		}),
		sensor(uid+"_temperature", "Temperature", current, "{{ value_json.temperature }}", uid+"_temperature", device, map[string]any{
			"device_class":        "temperature",
			"state_class":         "measurement",
			"unit_of_measurement": temperatureUnit,
		}),
		sensor(uid+"_apparent_temperature", "Apparent Temperature", current, "{{ value_json.apparent_temperature }}", uid+"_apparent_temperature", device, map[string]any{
			"device_class":        "temperature",
			"state_class":         "measurement",
			"unit_of_measurement": temperatureUnit,
		}),
		sensor(uid+"_dew_point", "Dew Point", current, "{{ value_json.dew_point }}", uid+"_dew_point", device, map[string]any{
			"device_class":        "temperature",
			"state_class":         "measurement",
			"unit_of_measurement": temperatureUnit,
		}),
		sensor(uid+"_humidity", "Humidity", current, "{{ value_json.humidity }}", uid+"_humidity", device, map[string]any{
			"device_class":        "humidity",
			"state_class":         "measurement",
			"unit_of_measurement": "%",
		}),
		sensor(uid+"_pressure", "Pressure", current, "{{ value_json.pressure }}", uid+"_pressure", device, map[string]any{
			"device_class":        "pressure",
			"state_class":         "measurement",
			"unit_of_measurement": pressureUnit,
		}),
		sensor(uid+"_wind_speed", "Wind Speed", current, "{{ value_json.wind_speed }}", uid+"_wind_speed", device, map[string]any{
			"device_class":        "wind_speed",
			"state_class":         "measurement",
			"unit_of_measurement": windSpeedUnit,
		}),
		sensor(uid+"_wind_gust", "Wind Gust", current, "{{ value_json.wind_gust_speed }}", uid+"_wind_gust", device, map[string]any{
			"device_class":        "wind_speed",
			"state_class":         "measurement",
			"unit_of_measurement": windSpeedUnit,
		}),
		sensor(uid+"_wind_bearing", "Wind Bearing", current, "{{ value_json.wind_bearing }}", uid+"_wind_bearing", device, map[string]any{
			"state_class":         "measurement",
			"unit_of_measurement": "°",
			"icon":                "mdi:compass",
		}),
		sensor(uid+"_cloud_coverage", "Cloud Coverage", current, "{{ value_json.cloud_coverage }}", uid+"_cloud_coverage", device, map[string]any{
			"state_class":         "measurement",
			"unit_of_measurement": "%",
			"icon":                "mdi:weather-cloudy",
		}),
		sensor(uid+"_visibility", "Visibility", current, "{{ value_json.visibility }}", uid+"_visibility", device, map[string]any{
			"device_class":        "distance",
			"state_class":         "measurement",
			"unit_of_measurement": visibilityUnit,
		}),
		sensor(uid+"_daily", "Daily Forecast", daily, "{{ value_json.forecast[0].condition if value_json.forecast else 'unknown' }}", uid+"_daily", device, map[string]any{
			"icon":                  "mdi:calendar-week",
			"json_attributes_topic": daily,
		}),
		sensor(uid+"_hourly", "Hourly Forecast", hourly, "{{ value_json.forecast[0].condition if value_json.forecast else 'unknown' }}", uid+"_hourly", device, map[string]any{
			"icon":                  "mdi:clock-outline",
			"json_attributes_topic": hourly,
		}),
		sensor(uid+"_5day", "5-Day Forecast", fiveDay, "{{ value_json.forecast[0].condition_raw if value_json.forecast and value_json.forecast[0].condition_raw else (value_json.forecast[0].condition if value_json.forecast else 'unknown') }}", uid+"_5day", device, map[string]any{
			"icon":                  "mdi:weather-cloudy-clock",
			"json_attributes_topic": fiveDay,
		}),
	}
}

func BuildSunDiscoveryConfigs(topicPrefix string) []mqttpub.Config {
	current := topicPrefix + "/current"
	device := map[string]any{
		"identifiers":  []string{"weather_mqtt_sun"},
		"name":         "Sun MQTT",
		"manufacturer": deviceManufacturer,
		"model":        "Sun day-context bridge",
	}
	uid := "weather_mqtt_sun"
	return []mqttpub.Config{
		sensor(uid+"_state", "State", current, "{{ value_json.state }}", uid+"_state", device, map[string]any{
			"icon":                  "mdi:theme-light-dark",
			"json_attributes_topic": current,
		}),
		sensor(uid+"_daytime", "Daytime", current, "{{ value_json.daytime }}", uid+"_daytime", device, map[string]any{
			"icon": "mdi:weather-sunny",
		}),
		sensor(uid+"_elevation", "Elevation", current, "{{ value_json.elevation }}", uid+"_elevation", device, map[string]any{
			"state_class":         "measurement",
			"unit_of_measurement": "°",
			"icon":                "mdi:angle-acute",
		}),
		sensor(uid+"_azimuth", "Azimuth", current, "{{ value_json.azimuth }}", uid+"_azimuth", device, map[string]any{
			"state_class":         "measurement",
			"unit_of_measurement": "°",
			"icon":                "mdi:compass",
		}),
		sensor(uid+"_next_rising", "Next Rising", current, "{{ value_json.next_rising }}", uid+"_next_rising", device, map[string]any{
			"device_class": "timestamp",
			"icon":         "mdi:weather-sunset-up",
		}),
		sensor(uid+"_next_setting", "Next Setting", current, "{{ value_json.next_setting }}", uid+"_next_setting", device, map[string]any{
			"device_class": "timestamp",
			"icon":         "mdi:weather-sunset-down",
		}),
		sensor(uid+"_next_dawn", "Next Dawn", current, "{{ value_json.next_dawn }}", uid+"_next_dawn", device, map[string]any{
			"device_class": "timestamp",
			"icon":         "mdi:weather-sunset-up",
		}),
		sensor(uid+"_next_dusk", "Next Dusk", current, "{{ value_json.next_dusk }}", uid+"_next_dusk", device, map[string]any{
			"device_class": "timestamp",
			"icon":         "mdi:weather-sunset-down",
		}),
	}
}

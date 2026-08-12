"""Tests for Home Assistant MQTT discovery configs."""

from src.config import parse_weather_sources
from src.discovery import build_discovery_configs


def test_discovery_includes_device_and_core_sensors():
    configs = build_discovery_configs(
        "home/weather/pirate",
        source_name="pirate",
        source_label="Pirate Weather",
    )
    by_id = {object_id: cfg for object_id, cfg in configs}

    assert "weather_mqtt_pirate_temperature" in by_id
    temp = by_id["weather_mqtt_pirate_temperature"]
    assert temp["state_topic"] == "home/weather/pirate/current"
    assert temp["value_template"] == "{{ value_json.temperature }}"
    assert temp["device_class"] == "temperature"
    assert temp["unit_of_measurement"] == "°F"
    assert temp["device"]["identifiers"] == ["weather_mqtt_pirate"]
    assert temp["device"]["name"] == "Weather MQTT (Pirate Weather)"

    daily = by_id["weather_mqtt_pirate_daily"]
    assert daily["state_topic"] == "home/weather/pirate/daily"
    assert daily["json_attributes_topic"] == "home/weather/pirate/daily"

    assert len(configs) >= 10


def test_discovery_metno_uses_separate_device():
    configs = build_discovery_configs(
        "home/weather/metno",
        source_name="metno",
        source_label="Met.no",
        temperature_unit="°C",
        wind_speed_unit="km/h",
    )
    by_id = {object_id: cfg for object_id, cfg in configs}
    temp = by_id["weather_mqtt_metno_temperature"]
    assert temp["unit_of_measurement"] == "°C"
    assert temp["device"]["identifiers"] == ["weather_mqtt_metno"]
    assert temp["state_topic"] == "home/weather/metno/current"


def test_parse_weather_sources_default_and_multi():
    single = parse_weather_sources("", legacy_entity="weather.pirateweather")
    assert len(single) == 1
    assert single[0].name == "pirate"
    assert single[0].entity_id == "weather.pirateweather"

    multi = parse_weather_sources(
        "pirate:weather.pirateweather,metno:weather.forecast_home"
    )
    assert [s.name for s in multi] == ["pirate", "metno"]
    assert multi[1].entity_id == "weather.forecast_home"
    assert multi[1].label == "Met.no"

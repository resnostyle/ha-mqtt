"""Home Assistant MQTT discovery configs for weather-mqtt sensors."""

from __future__ import annotations

from typing import Any


DEVICE_MANUFACTURER = "weather-mqtt"


def device_block(*, source_name: str, source_label: str) -> dict[str, Any]:
    return {
        "identifiers": [f"weather_mqtt_{source_name}"],
        "name": f"Weather MQTT ({source_label})",
        "manufacturer": DEVICE_MANUFACTURER,
        "model": f"{source_label} bridge",
    }


def _sensor(
    *,
    object_id: str,
    name: str,
    state_topic: str,
    value_template: str,
    unique_id: str,
    device: dict[str, Any],
    device_class: str | None = None,
    state_class: str | None = None,
    unit: str | None = None,
    icon: str | None = None,
    json_attributes_topic: str | None = None,
) -> tuple[str, dict[str, Any]]:
    """Return (discovery object_id, config payload)."""
    config: dict[str, Any] = {
        "name": name,
        "unique_id": unique_id,
        "state_topic": state_topic,
        "value_template": value_template,
        "device": device,
        "object_id": object_id,
    }
    if device_class:
        config["device_class"] = device_class
    if state_class:
        config["state_class"] = state_class
    if unit:
        config["unit_of_measurement"] = unit
    if icon:
        config["icon"] = icon
    if json_attributes_topic:
        config["json_attributes_topic"] = json_attributes_topic
    return object_id, config


def build_discovery_configs(
    topic_prefix: str,
    *,
    source_name: str,
    source_label: str,
    temperature_unit: str = "°F",
    pressure_unit: str = "inHg",
    wind_speed_unit: str = "mph",
    visibility_unit: str = "mi",
) -> list[tuple[str, dict[str, Any]]]:
    """Build MQTT sensor discovery configs for one weather source."""
    current = f"{topic_prefix}/current"
    daily = f"{topic_prefix}/daily"
    hourly = f"{topic_prefix}/hourly"
    five_day = f"{topic_prefix}/5day"
    device = device_block(source_name=source_name, source_label=source_label)
    uid = f"weather_mqtt_{source_name}"

    configs: list[tuple[str, dict[str, Any]]] = [
        _sensor(
            object_id=f"{uid}_condition",
            name="Condition",
            state_topic=current,
            value_template="{{ value_json.condition }}",
            unique_id=f"{uid}_condition",
            device=device,
            icon="mdi:weather-partly-cloudy",
            json_attributes_topic=current,
        ),
        _sensor(
            object_id=f"{uid}_temperature",
            name="Temperature",
            state_topic=current,
            value_template="{{ value_json.temperature }}",
            unique_id=f"{uid}_temperature",
            device=device,
            device_class="temperature",
            state_class="measurement",
            unit=temperature_unit,
        ),
        _sensor(
            object_id=f"{uid}_apparent_temperature",
            name="Apparent Temperature",
            state_topic=current,
            value_template="{{ value_json.apparent_temperature }}",
            unique_id=f"{uid}_apparent_temperature",
            device=device,
            device_class="temperature",
            state_class="measurement",
            unit=temperature_unit,
        ),
        _sensor(
            object_id=f"{uid}_dew_point",
            name="Dew Point",
            state_topic=current,
            value_template="{{ value_json.dew_point }}",
            unique_id=f"{uid}_dew_point",
            device=device,
            device_class="temperature",
            state_class="measurement",
            unit=temperature_unit,
        ),
        _sensor(
            object_id=f"{uid}_humidity",
            name="Humidity",
            state_topic=current,
            value_template="{{ value_json.humidity }}",
            unique_id=f"{uid}_humidity",
            device=device,
            device_class="humidity",
            state_class="measurement",
            unit="%",
        ),
        _sensor(
            object_id=f"{uid}_pressure",
            name="Pressure",
            state_topic=current,
            value_template="{{ value_json.pressure }}",
            unique_id=f"{uid}_pressure",
            device=device,
            device_class="pressure",
            state_class="measurement",
            unit=pressure_unit,
        ),
        _sensor(
            object_id=f"{uid}_wind_speed",
            name="Wind Speed",
            state_topic=current,
            value_template="{{ value_json.wind_speed }}",
            unique_id=f"{uid}_wind_speed",
            device=device,
            device_class="wind_speed",
            state_class="measurement",
            unit=wind_speed_unit,
        ),
        _sensor(
            object_id=f"{uid}_wind_gust",
            name="Wind Gust",
            state_topic=current,
            value_template="{{ value_json.wind_gust_speed }}",
            unique_id=f"{uid}_wind_gust",
            device=device,
            device_class="wind_speed",
            state_class="measurement",
            unit=wind_speed_unit,
        ),
        _sensor(
            object_id=f"{uid}_wind_bearing",
            name="Wind Bearing",
            state_topic=current,
            value_template="{{ value_json.wind_bearing }}",
            unique_id=f"{uid}_wind_bearing",
            device=device,
            state_class="measurement",
            unit="°",
            icon="mdi:compass",
        ),
        _sensor(
            object_id=f"{uid}_cloud_coverage",
            name="Cloud Coverage",
            state_topic=current,
            value_template="{{ value_json.cloud_coverage }}",
            unique_id=f"{uid}_cloud_coverage",
            device=device,
            state_class="measurement",
            unit="%",
            icon="mdi:weather-cloudy",
        ),
        _sensor(
            object_id=f"{uid}_visibility",
            name="Visibility",
            state_topic=current,
            value_template="{{ value_json.visibility }}",
            unique_id=f"{uid}_visibility",
            device=device,
            device_class="distance",
            state_class="measurement",
            unit=visibility_unit,
        ),
        _sensor(
            object_id=f"{uid}_daily",
            name="Daily Forecast",
            state_topic=daily,
            value_template=(
                "{{ value_json.forecast[0].condition "
                "if value_json.forecast else 'unknown' }}"
            ),
            unique_id=f"{uid}_daily",
            device=device,
            icon="mdi:calendar-week",
            json_attributes_topic=daily,
        ),
        _sensor(
            object_id=f"{uid}_hourly",
            name="Hourly Forecast",
            state_topic=hourly,
            value_template=(
                "{{ value_json.forecast[0].condition "
                "if value_json.forecast else 'unknown' }}"
            ),
            unique_id=f"{uid}_hourly",
            device=device,
            icon="mdi:clock-outline",
            json_attributes_topic=hourly,
        ),
        _sensor(
            object_id=f"{uid}_5day",
            name="5-Day Forecast",
            state_topic=five_day,
            value_template=(
                "{{ value_json.forecast[0].condition_raw "
                "if value_json.forecast and value_json.forecast[0].condition_raw "
                "else (value_json.forecast[0].condition if value_json.forecast else 'unknown') }}"
            ),
            unique_id=f"{uid}_5day",
            device=device,
            icon="mdi:weather-cloudy-clock",
            json_attributes_topic=five_day,
        ),
    ]
    return configs

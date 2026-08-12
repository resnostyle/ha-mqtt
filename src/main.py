"""Poll Home Assistant weather entities and republish to MQTT."""

from __future__ import annotations

import logging
import signal
import sys
import time

from .config import Settings, WeatherSource
from .discovery import build_discovery_configs
from .ha import HomeAssistantClient
from .mqtt_out import MqttPublisher
from .payload import (
    build_5day,
    build_current,
    build_forecast_payload,
    trim_hourly,
)

logger = logging.getLogger(__name__)


def configure_logging(level: str) -> None:
    logging.basicConfig(
        level=getattr(logging, level, logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
        stream=sys.stdout,
    )


def source_topic_prefix(settings: Settings, source: WeatherSource) -> str:
    return f"{settings.mqtt_topic_prefix}/{source.topic_suffix}"


def publish_discovery(settings: Settings, mqtt: MqttPublisher) -> None:
    if not settings.mqtt_discovery_enabled:
        logger.info("MQTT discovery disabled")
        return
    total = 0
    for source in settings.weather_sources:
        configs = build_discovery_configs(
            source_topic_prefix(settings, source),
            source_name=source.name,
            source_label=source.label,
            temperature_unit=settings.temperature_unit,
            pressure_unit=settings.pressure_unit,
            wind_speed_unit=settings.wind_speed_unit,
            visibility_unit=settings.visibility_unit,
        )
        mqtt.publish_discovery(
            configs,
            discovery_prefix=settings.mqtt_discovery_prefix,
        )
        total += len(configs)
        logger.info(
            "Published %s discovery configs for %s (%s)",
            len(configs),
            source.name,
            source.entity_id,
        )
    logger.info("Published %s MQTT discovery configs total", total)


def publish_source(
    settings: Settings,
    ha: HomeAssistantClient,
    mqtt: MqttPublisher,
    source: WeatherSource,
) -> None:
    entity = source.entity_id
    prefix = source.topic_suffix
    state = ha.get_state(entity)
    daily = ha.get_forecasts(entity, "daily")

    hourly: list = []
    try:
        hourly = trim_hourly(
            ha.get_forecasts(entity, "hourly"),
            settings.hourly_forecast_hours,
        )
    except Exception:
        logger.exception(
            "Hourly forecast unavailable for %s (%s); continuing without it",
            source.name,
            entity,
        )

    mqtt.publish(f"{prefix}/current", build_current(state))
    mqtt.publish(
        f"{prefix}/daily",
        build_forecast_payload(daily, forecast_type="daily", entity_id=entity),
    )
    if hourly:
        mqtt.publish(
            f"{prefix}/hourly",
            build_forecast_payload(hourly, forecast_type="hourly", entity_id=entity),
        )
    mqtt.publish(
        f"{prefix}/5day",
        build_5day(
            daily,
            source=source.name,
            unit=settings.temperature_unit,
        ),
    )
    logger.info("Published %s weather topics for %s", source.name, entity)


def publish_once(settings: Settings, ha: HomeAssistantClient, mqtt: MqttPublisher) -> None:
    errors = 0
    for source in settings.weather_sources:
        try:
            publish_source(settings, ha, mqtt, source)
        except Exception:
            errors += 1
            logger.exception(
                "Publish failed for %s (%s)", source.name, source.entity_id
            )
    if errors and errors == len(settings.weather_sources):
        raise RuntimeError("All weather sources failed to publish")


def main() -> None:
    settings = Settings.from_env()
    configure_logging(settings.log_level)
    stop = False

    def _handle_signal(signum: int, _frame: object) -> None:
        nonlocal stop
        logger.info("Received signal %s, shutting down", signum)
        stop = True

    signal.signal(signal.SIGINT, _handle_signal)
    signal.signal(signal.SIGTERM, _handle_signal)

    source_summary = ", ".join(
        f"{s.name}={s.entity_id}" for s in settings.weather_sources
    )
    logger.info(
        "Starting weather-mqtt (sources=[%s], interval=%ss, mqtt=%s:%s, discovery=%s)",
        source_summary,
        settings.poll_interval_seconds,
        settings.mqtt_host,
        settings.mqtt_port,
        settings.mqtt_discovery_enabled,
    )

    with HomeAssistantClient(settings.ha_url, settings.ha_token) as ha:
        mqtt = MqttPublisher(
            host=settings.mqtt_host,
            port=settings.mqtt_port,
            client_id=settings.mqtt_client_id,
            username=settings.mqtt_username,
            password=settings.mqtt_password,
            topic_prefix=settings.mqtt_topic_prefix,
        )
        try:
            try:
                publish_discovery(settings, mqtt)
            except Exception:
                logger.exception("MQTT discovery publish failed")

            while not stop:
                try:
                    publish_once(settings, ha, mqtt)
                except Exception:
                    logger.exception("Publish cycle failed")
                deadline = time.monotonic() + settings.poll_interval_seconds
                while not stop and time.monotonic() < deadline:
                    time.sleep(min(1.0, deadline - time.monotonic()))
        finally:
            mqtt.close()

    logger.info("Exited")


if __name__ == "__main__":
    main()

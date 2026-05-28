"""Kafka producer helper for AI services."""

import json
import logging
from typing import Optional

logger = logging.getLogger(__name__)


class KafkaProducer:
    """Simple Kafka producer wrapper."""

    def __init__(self, brokers: list[str]):
        self.brokers = brokers
        self._producer = None

    def connect(self):
        try:
            from kafka import KafkaProducer as KP

            self._producer = KP(
                bootstrap_servers=self.brokers,
                value_serializer=lambda v: json.dumps(v).encode("utf-8"),
            )
            logger.info(f"Kafka producer connected to {self.brokers}")
        except ImportError:
            logger.warning("kafka-python not installed, producer disabled")
        except Exception as e:
            logger.error(f"Kafka producer connect error: {e}")

    def send(self, topic: str, message: dict) -> bool:
        if not self._producer:
            logger.warning("Producer not connected, dropping message")
            return False
        try:
            future = self._producer.send(topic, message)
            future.get(timeout=10)
            return True
        except Exception as e:
            logger.error(f"Send to {topic} failed: {e}")
            return False

    def close(self):
        if self._producer:
            self._producer.close()

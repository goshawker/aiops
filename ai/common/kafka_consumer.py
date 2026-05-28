"""Kafka consumer helper for AI services."""

import json
import logging
from typing import Callable, Optional

logger = logging.getLogger(__name__)


class KafkaConsumer:
    """Simple Kafka consumer wrapper."""

    def __init__(self, brokers: list[str], topic: str, group_id: str):
        self.brokers = brokers
        self.topic = topic
        self.group_id = group_id
        self._consumer = None

    def start(self, handler: Callable[[dict], None]):
        """Start consuming messages."""
        try:
            from kafka import KafkaConsumer as KC

            self._consumer = KC(
                self.topic,
                bootstrap_servers=self.brokers,
                group_id=self.group_id,
                auto_offset_reset="latest",
                value_deserializer=lambda m: json.loads(m.decode("utf-8")),
            )

            logger.info(f"Consuming from {self.topic} (group={self.group_id})")
            for message in self._consumer:
                try:
                    handler(message.value)
                except Exception as e:
                    logger.error(f"Handler error: {e}")

        except ImportError:
            logger.warning("kafka-python not installed, running without Kafka")
        except Exception as e:
            logger.error(f"Kafka consumer error: {e}")

    def stop(self):
        if self._consumer:
            self._consumer.close()

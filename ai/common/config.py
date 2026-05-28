"""Common configuration for AI services."""

import os
import logging

# Kafka
KAFKA_BROKERS = os.getenv("KAFKA_BROKERS", "localhost:9092").split(",")
TOPIC_METRICS = os.getenv("TOPIC_METRICS", "aiops.metrics")
TOPIC_LOGS = os.getenv("TOPIC_LOGS", "aiops.logs")
TOPIC_ALERTS = os.getenv("TOPIC_ALERTS", "aiops.alerts")
TOPIC_INCIDENTS = os.getenv("TOPIC_INCIDENTS", "aiops.incidents")

# Service
SERVICE_NAME = os.getenv("SERVICE_NAME", "ai-service")
PORT = int(os.getenv("PORT", "5001"))
LOG_LEVEL = os.getenv("LOG_LEVEL", "info").upper()


def setup_logging():
    logging.basicConfig(
        level=getattr(logging, LOG_LEVEL, logging.INFO),
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    )

"""Anomaly Detection Service.

Phase 2: River online learning with rule engine fallback.
"""

import threading
import logging
from flask import Flask, jsonify, request

from ai.common.config import (
    setup_logging, KAFKA_BROKERS, TOPIC_METRICS, TOPIC_ALERTS, SERVICE_NAME, PORT,
)
from ai.common.kafka_consumer import KafkaConsumer
from ai.common.kafka_producer import KafkaProducer
from ai.anomaly.river_detector import RiverDetector, MetricPoint

setup_logging()
logger = logging.getLogger(SERVICE_NAME)

app = Flask(__name__)

# Components — River detector with 3-sigma fallback
detector = RiverDetector(
    n_trees=25,
    height=15,
    window_size=250,
    warmup_points=50,
    sigma_multiplier=3.0,
)
producer = KafkaProducer(KAFKA_BROKERS)


def handle_metric(msg: dict):
    """Process incoming metric from Kafka."""
    point = MetricPoint(
        timestamp=msg.get("timestamp", 0),
        value=float(msg.get("value", 0)),
        labels=msg.get("labels", {}),
    )
    point.metric_name = msg.get("metric_name", "")

    result = detector.detect(point)
    if result:
        alert = {
            "source_type": "metric",
            "source": result.metric_name,
            "severity": result.severity,
            "title": f"Anomaly: {result.metric_name}",
            "message": result.message,
            "value": str(result.value),
            "threshold": str(result.threshold),
            "labels": result.labels,
            "anomaly_type": result.anomaly_type,
            "confidence": result.confidence,
        }
        producer.send(TOPIC_ALERTS, alert)
        logger.warning(f"Anomaly detected: {result.message}")


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "anomaly"})


@app.route("/api/v1/anomaly/detect", methods=["POST"])
def detect():
    """Manual anomaly detection endpoint."""
    data = request.json or {}
    point = MetricPoint(
        timestamp=data.get("timestamp", 0),
        value=float(data.get("value", 0)),
        labels=data.get("labels", {}),
    )
    point.metric_name = data.get("metric_name", "")

    result = detector.detect(point)
    if result:
        return jsonify({
            "anomaly": True,
            "result": {
                "metric_name": result.metric_name,
                "value": result.value,
                "threshold": result.threshold,
                "anomaly_type": result.anomaly_type,
                "severity": result.severity,
                "confidence": result.confidence,
                "message": result.message,
            },
        })
    return jsonify({"anomaly": False})


@app.route("/api/v1/anomaly/thresholds", methods=["POST"])
def set_threshold():
    """Set a static threshold for a metric."""
    data = request.json or {}
    metric = data.get("metric_name", "")
    labels = data.get("labels", {})
    op = data.get("op", ">")
    value = float(data.get("value", 0))

    if not metric:
        return jsonify({"error": "metric_name required"}), 400

    detector.set_threshold(metric, labels, op, value)
    return jsonify({"status": "ok", "message": f"Threshold set: {metric} {op} {value}"})


@app.route("/api/v1/anomaly/status")
def status():
    """Return model status and diagnostics."""
    return jsonify(detector.get_status())


def start_kafka():
    """Start Kafka consumer in background thread."""
    consumer = KafkaConsumer(KAFKA_BROKERS, TOPIC_METRICS, "anomaly-service")
    t = threading.Thread(target=consumer.start, args=(handle_metric,), daemon=True)
    t.start()


if __name__ == "__main__":
    producer.connect()
    start_kafka()
    app.run(host="0.0.0.0", port=PORT)

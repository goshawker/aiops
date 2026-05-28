"""Alert Aggregation Service.

Groups similar alerts into incidents using label matching.
"""

import threading
import logging
from flask import Flask, jsonify, request

from ai.common.config import (
    setup_logging, KAFKA_BROKERS, TOPIC_ALERTS, TOPIC_INCIDENTS, SERVICE_NAME, PORT,
)
from ai.common.kafka_consumer import KafkaConsumer
from ai.common.kafka_producer import KafkaProducer
from ai.alert_agg.aggregator import LabelMatchAggregator, Alert

setup_logging()
logger = logging.getLogger(SERVICE_NAME)

app = Flask(__name__)

# Components
aggregator = LabelMatchAggregator(window_seconds=300)
producer = KafkaProducer(KAFKA_BROKERS)


def handle_alert(msg: dict):
    """Process incoming alert from Kafka."""
    alert = Alert(
        source_type=msg.get("source_type", "metric"),
        source=msg.get("source", ""),
        host=msg.get("host", ""),
        service=msg.get("service", ""),
        severity=msg.get("severity", "warning"),
        title=msg.get("title", ""),
        message=msg.get("message", ""),
        value=msg.get("value", ""),
        threshold=msg.get("threshold", ""),
        labels=msg.get("labels", {}),
    )

    incident = aggregator.process(alert)
    if incident:
        incident_msg = {
            "id": incident.id,
            "title": incident.title,
            "description": incident.description,
            "severity": incident.severity,
            "affected_services": incident.affected_services,
            "affected_hosts": incident.affected_hosts,
            "event_count": incident.event_count,
            "status": incident.status,
        }
        producer.send(TOPIC_INCIDENTS, incident_msg)
        logger.info(f"Incident created: {incident.id}")


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "alert_agg"})


@app.route("/api/v1/alert/aggregate", methods=["POST"])
def aggregate():
    """Manual alert aggregation endpoint."""
    data = request.json or {}
    alert = Alert(
        source_type=data.get("source_type", "manual"),
        source=data.get("source", ""),
        host=data.get("host", ""),
        service=data.get("service", ""),
        severity=data.get("severity", "warning"),
        title=data.get("title", ""),
        message=data.get("message", ""),
    )

    incident = aggregator.process(alert)
    if incident:
        return jsonify({
            "incident_created": True,
            "incident": {
                "id": incident.id,
                "title": incident.title,
                "severity": incident.severity,
                "event_count": incident.event_count,
            },
        })
    return jsonify({"incident_created": False, "message": "Alert merged into existing incident"})


@app.route("/api/v1/alert/incidents", methods=["GET"])
def list_incidents():
    """List open incidents."""
    incidents = []
    for inc in aggregator.open_incidents.values():
        incidents.append({
            "id": inc.id,
            "title": inc.title,
            "severity": inc.severity,
            "affected_services": inc.affected_services,
            "affected_hosts": inc.affected_hosts,
            "event_count": inc.event_count,
            "status": inc.status,
        })
    return jsonify({"data": incidents, "count": len(incidents)})


@app.route("/api/v1/alert/incidents/<incident_id>/resolve", methods=["POST"])
def resolve_incident(incident_id):
    """Resolve an incident."""
    if aggregator.resolve(incident_id):
        return jsonify({"status": "ok"})
    return jsonify({"error": "incident not found"}), 404


def start_kafka():
    """Start Kafka consumer in background thread."""
    consumer = KafkaConsumer(KAFKA_BROKERS, TOPIC_ALERTS, "alert-agg-service")
    t = threading.Thread(target=consumer.start, args=(handle_alert,), daemon=True)
    t.start()


if __name__ == "__main__":
    producer.connect()
    start_kafka()
    app.run(host="0.0.0.0", port=PORT)

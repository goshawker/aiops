"""Root Cause Analysis Service.

Phase 2: PC algorithm with correlation fallback.
"""

import logging
from flask import Flask, jsonify, request

from ai.common.config import setup_logging, SERVICE_NAME, PORT
from ai.rca.engine import RCAEngine, MetricTimeSeries

setup_logging()
logger = logging.getLogger(SERVICE_NAME)

app = Flask(__name__)

# Components
rca = RCAEngine()


@app.route("/health")
def health():
    return jsonify({
        "status": "ok",
        "service": "rca",
        "graph_nodes": len(rca.get_graph()["nodes"]),
        "graph_edges": len(rca.get_graph()["edges"]),
    })


@app.route("/api/v1/rca/ingest", methods=["POST"])
def ingest():
    """Ingest time series data for causal discovery."""
    data = request.json or {}
    count = 0

    for m in data.get("metrics", []):
        ts = MetricTimeSeries(
            name=m.get("name", ""),
            timestamps=m.get("timestamps", []),
            values=m.get("values", []),
            labels=m.get("labels", {}),
        )
        if ts.name and ts.values:
            rca.add_time_series(ts)
            count += 1

    return jsonify({"status": "ok", "ingested": count})


@app.route("/api/v1/rca/discover", methods=["POST"])
def discover():
    """Trigger causal graph discovery."""
    edges = rca.discover_causal_graph()
    return jsonify({
        "status": "ok",
        "edge_count": len(edges),
        "edges": [
            {"source": e.source, "target": e.target, "confidence": e.confidence, "lag": e.lag}
            for e in edges
        ],
    })


@app.route("/api/v1/rca/analyze", methods=["POST"])
def analyze():
    """Analyze root cause for an incident."""
    data = request.json or {}
    affected = data.get("affected_metrics", [])

    if not affected:
        return jsonify({"error": "affected_metrics required"}), 400

    results = rca.analyze_root_cause(affected)

    return jsonify({
        "root_causes": [
            {
                "metric_name": rc.metric_name,
                "score": rc.score,
                "reason": rc.reason,
                "related_metrics": rc.related_metrics,
                "evidence": rc.evidence,
            }
            for rc in results
        ],
    })


@app.route("/api/v1/rca/graph")
def graph():
    """Return the current causal graph for visualization."""
    return jsonify(rca.get_graph())


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=PORT)

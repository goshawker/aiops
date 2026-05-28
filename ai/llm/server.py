"""LLM Health Summary Service.

Phase 2: Qwen2-1.5B INT4 with rule engine fallback.
Phase 3: Multi-tier model support with hot-swap.
"""

import logging
from flask import Flask, jsonify, request

from ai.common.config import setup_logging, SERVICE_NAME, PORT
from ai.llm.summary_engine import RuleEngineSummary, MetricSnapshot, LogSummary
from ai.llm.qwen_engine import (
    generate_summary as qwen_summary,
    chat as qwen_chat,
    get_model_info,
    switch_model_tier,
)

setup_logging()
logger = logging.getLogger(SERVICE_NAME)

app = Flask(__name__)

# Components
rule_engine = RuleEngineSummary()


@app.route("/health")
def health():
    info = get_model_info()
    return jsonify({
        "status": "ok",
        "service": "llm",
        "model": "qwen2" if info["loaded"] else "rule_engine",
        "model_info": info,
    })


@app.route("/api/v1/llm/summary", methods=["POST"])
def summary():
    """Generate health summary.

    Strategy: try Qwen2 first, fall back to rule engine.
    """
    data = request.json or {}

    # Parse metrics
    metrics = []
    for m in data.get("metrics", []):
        metrics.append(MetricSnapshot(
            name=m.get("name", ""),
            value=float(m.get("value", 0)),
            unit=m.get("unit", ""),
            change_pct=float(m.get("change_pct", 0)),
            host=m.get("host", ""),
            service=m.get("service", ""),
        ))

    # Parse log summary
    log_data = data.get("log_summary")
    log_summary = None
    if log_data:
        log_summary = LogSummary(
            total_count=log_data.get("total_count", 0),
            error_count=log_data.get("error_count", 0),
            warn_count=log_data.get("warn_count", 0),
            top_errors=log_data.get("top_errors", []),
        )

    # Parse incidents
    incidents = data.get("incidents", [])

    # Always generate rule engine report (fast, always available)
    rule_report = rule_engine.generate(metrics, log_summary, incidents)

    # Try Qwen2 for richer summary
    metrics_text = "\n".join(
        f"- {m.name}: {m.value}{m.unit}" + (f" (变化 {m.change_pct:+.0f}%)" if m.change_pct else "")
        for m in metrics
    ) or "无指标数据"

    logs_text = (
        f"总日志 {log_summary.total_count} 条，错误 {log_summary.error_count} 条"
        + (f"，高频错误：{log_summary.top_errors[0]}" if log_summary and log_summary.top_errors else "")
    ) if log_summary else "无日志数据"

    incidents_text = "\n".join(
        f"- [{i.get('severity', 'info')}] {i.get('title', '未知')}"
        for i in incidents
    ) or "无活跃事件"

    qwen_result = qwen_summary(metrics_text, logs_text, incidents_text)

    if qwen_result:
        return jsonify({
            "status": rule_report.status,
            "summary": qwen_result,
            "details": rule_report.details,
            "recommendations": rule_report.recommendations,
            "source": "qwen2",
        })

    # Fallback to rule engine
    return jsonify({
        "status": rule_report.status,
        "summary": rule_report.summary,
        "details": rule_report.details,
        "recommendations": rule_report.recommendations,
        "source": "rule_engine",
    })


@app.route("/api/v1/llm/chat", methods=["POST"])
def chat():
    """Chat endpoint with Qwen2 + rule engine fallback."""
    data = request.json or {}
    message = data.get("message", "")
    history = data.get("history", [])

    # Try Qwen2
    result = qwen_chat(message, history)
    if result:
        return jsonify({
            "response": result,
            "source": "qwen2",
        })

    # Rule engine fallback
    if "告警" in message or "alert" in message.lower():
        response = "告警查询功能开发中，请使用告警管理页面查看。"
    elif "指标" in message or "metric" in message.lower():
        response = "指标查询功能开发中，请使用指标监控页面查看。"
    else:
        response = "我是 AIOps AI 助手，目前处于基础模式。完整 AI 对话功能将在下一版本上线。"

    return jsonify({
        "response": response,
        "source": "rule_engine",
    })


@app.route("/api/v1/llm/model")
def model_info():
    """Return model status."""
    return jsonify(get_model_info())


@app.route("/api/v1/llm/model/switch", methods=["POST"])
def model_switch():
    """Switch model tier (e.g. 1.5b → 7b-int4).

    Body: {"tier": "7b-int4"}
    """
    data = request.json or {}
    tier = data.get("tier", "")

    if not tier:
        return jsonify({"error": "tier is required"}), 400

    result = switch_model_tier(tier)
    if not result["success"]:
        return jsonify(result), 400

    return jsonify(result)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=PORT)

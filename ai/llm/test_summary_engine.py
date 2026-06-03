"""Tests for RuleEngineSummary."""

import pytest
from llm.summary_engine import RuleEngineSummary, MetricSnapshot, LogSummary, HealthReport


@pytest.fixture
def engine():
    return RuleEngineSummary()


class TestRuleEngineSummary:
    def test_normal_metrics(self, engine):
        metrics = [MetricSnapshot(name="cpu_usage", value=50)]
        report = engine.generate(metrics)
        assert report.status == "normal"
        assert "正常" in report.summary

    def test_cpu_warning(self, engine):
        metrics = [MetricSnapshot(name="cpu_usage", value=85)]
        report = engine.generate(metrics)
        assert report.status == "warning"
        assert "CPU" in report.summary

    def test_memory_critical(self, engine):
        metrics = [MetricSnapshot(name="memory_usage", value=96)]
        report = engine.generate(metrics)
        assert report.status == "critical"

    def test_disk_warning(self, engine):
        metrics = [MetricSnapshot(name="disk_usage", value=95)]
        report = engine.generate(metrics)
        assert report.status == "warning"
        assert len(report.recommendations) > 0

    def test_error_rate_critical(self, engine):
        metrics = [MetricSnapshot(name="error_rate", value=10)]
        report = engine.generate(metrics)
        assert report.status == "critical"

    def test_log_errors_warning(self, engine):
        log = LogSummary(error_count=20, top_errors=["connection refused"])
        report = engine.generate([], log_summary=log)
        assert report.status == "warning"
        assert "错误日志" in report.summary

    def test_log_no_errors(self, engine):
        log = LogSummary(error_count=0)
        report = engine.generate([], log_summary=log)
        assert report.status == "normal"
        assert "无错误" in report.details[0]

    def test_critical_incident(self, engine):
        incidents = [{"severity": "critical"}]
        report = engine.generate([], incidents=incidents)
        assert report.status == "critical"

    def test_escalate_never_downgrades(self, engine):
        assert engine._escalate("critical", "warning") == "critical"
        assert engine._escalate("critical", "normal") == "critical"
        assert engine._escalate("warning", "normal") == "warning"

    def test_escalate_upgrades(self, engine):
        assert engine._escalate("normal", "warning") == "warning"
        assert engine._escalate("warning", "critical") == "critical"
        assert engine._escalate("normal", "critical") == "critical"

    def test_empty_metrics(self, engine):
        report = engine.generate([])
        assert report.status == "normal"
        assert "正常" in report.details[0]

    def test_source_is_rule_engine(self, engine):
        report = engine.generate([])
        assert report.source == "rule_engine"

    def test_cpu_with_change_pct(self, engine):
        metrics = [MetricSnapshot(name="cpu_usage", value=85, change_pct=15)]
        report = engine.generate(metrics)
        assert len(report.recommendations) > 0

"""Tests for RuleEngineDetector."""

import pytest
from anomaly.detector import RuleEngineDetector, MetricPoint


@pytest.fixture
def detector():
    return RuleEngineDetector(window_size=100, sigma_multiplier=3.0)


def _make_point(name="cpu", value=50.0, labels=None):
    return MetricPoint(metric_name=name, timestamp=1000.0, value=value, labels=labels or {})


class TestRuleEngineDetector:
    def test_no_data_returns_none(self, detector):
        result = detector.detect(_make_point(value=100))
        assert result is None

    def test_warmup_returns_none(self, detector):
        """With < 10 points, should return None."""
        for i in range(9):
            result = detector.detect(_make_point(value=float(i)))
            assert result is None

    def test_detects_spike(self, detector):
        """After warmup with normal values, a spike should be detected."""
        for i in range(20):
            detector.detect(_make_point(value=50.0 + (i % 3)))
        result = detector.detect(_make_point(value=200.0))
        assert result is not None
        assert result.anomaly_type == "spike"

    def test_detects_drop(self, detector):
        """After warmup with normal values, a drop should be detected."""
        for i in range(20):
            detector.detect(_make_point(value=50.0 + (i % 3)))
        result = detector.detect(_make_point(value=-100.0))
        assert result is not None
        assert result.anomaly_type == "drop"

    def test_normal_value_returns_none(self, detector):
        """A value within normal range should not trigger."""
        for i in range(20):
            detector.detect(_make_point(value=50.0 + (i % 5)))  # Add variance
        result = detector.detect(_make_point(value=51.0))
        assert result is None

    def test_static_threshold_gt(self, detector):
        detector.set_threshold("cpu", {}, ">", 80.0)
        result = detector.detect(_make_point(value=90.0))
        assert result is not None
        assert result.anomaly_type == "threshold"
        assert result.severity == "warning"

    def test_static_threshold_lt(self, detector):
        detector.set_threshold("mem", {}, "<", 10.0)
        result = detector.detect(_make_point(name="mem", value=5.0))
        assert result is not None
        assert result.anomaly_type == "threshold"

    def test_static_threshold_not_triggered(self, detector):
        detector.set_threshold("cpu", {}, ">", 80.0)
        result = detector.detect(_make_point(value=70.0))
        # No sigma data yet, so returns None
        assert result is None

    def test_static_threshold_critical_severity(self, detector):
        detector.set_threshold("cpu", {}, ">", 80.0)
        result = detector.detect(_make_point(value=130.0))
        assert result is not None
        assert result.severity == "critical"

    def test_static_threshold_priority_over_sigma(self, detector):
        """Static threshold should be checked before sigma."""
        for i in range(20):
            detector.detect(_make_point(value=50.0))
        detector.set_threshold("cpu", {}, ">", 80.0)
        result = detector.detect(_make_point(value=90.0))
        assert result is not None
        assert result.anomaly_type == "threshold"

    def test_make_key_sorted_labels(self, detector):
        k1 = detector._make_key("cpu", {"b": "2", "a": "1"})
        k2 = detector._make_key("cpu", {"a": "1", "b": "2"})
        assert k1 == k2

    def test_window_size_respected(self, detector):
        detector.window_size = 5
        for i in range(10):
            detector.detect(_make_point(value=float(i)))
        key = detector._make_key("cpu", {})
        assert len(detector.windows[key]) == 5

    def test_severity_levels(self, detector):
        """z-score > 5 -> critical, > 4 -> warning, else info."""
        for i in range(30):
            detector.detect(_make_point(value=50.0))
        # Extreme spike should be critical
        result = detector.detect(_make_point(value=500.0))
        assert result is not None
        assert result.severity == "critical"

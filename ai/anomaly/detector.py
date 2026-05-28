"""Anomaly Detection - Rule Engine (3-sigma threshold).

Phase 1 implementation: static threshold + 3-sigma detection.
Phase 2 will replace this with River online learning.
"""

import logging
import time
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger(__name__)


@dataclass
class MetricPoint:
    """A single metric data point."""
    timestamp: float
    value: float
    labels: dict = field(default_factory=dict)


@dataclass
class AnomalyResult:
    """Result of anomaly detection."""
    metric_name: str
    labels: dict
    value: float
    threshold: float
    anomaly_type: str  # "spike", "drop", "high_variance"
    severity: str  # "critical", "warning", "info"
    confidence: float  # 0.0 - 1.0
    message: str


class RuleEngineDetector:
    """3-sigma based anomaly detection.

    Maintains a rolling window of metric values and detects anomalies
    using standard deviation from the mean.
    """

    def __init__(self, window_size: int = 100, sigma_multiplier: float = 3.0):
        self.window_size = window_size
        self.sigma_multiplier = sigma_multiplier
        # metric_key -> list of values
        self.windows: dict[str, list[float]] = {}
        # metric_key -> static thresholds (from alert rules)
        self.static_thresholds: dict[str, dict] = {}

    def set_threshold(self, metric_name: str, labels: dict, op: str, value: float):
        """Set a static threshold for a metric."""
        key = self._make_key(metric_name, labels)
        self.static_thresholds[key] = {"op": op, "value": value}

    def detect(self, point: MetricPoint) -> Optional[AnomalyResult]:
        """Detect if a metric point is anomalous."""
        key = self._make_key(point.metric_name, point.labels)

        # Check static threshold first
        if key in self.static_thresholds:
            result = self._check_static(point, self.static_thresholds[key])
            if result:
                return result

        # Check 3-sigma
        return self._check_sigma(point)

    def _check_static(self, point: MetricPoint, threshold: dict) -> Optional[AnomalyResult]:
        """Check against static threshold."""
        op = threshold["op"]
        value = threshold["value"]
        triggered = False

        if op == ">" and point.value > value:
            triggered = True
        elif op == ">=" and point.value >= value:
            triggered = True
        elif op == "<" and point.value < value:
            triggered = True
        elif op == "<=" and point.value <= value:
            triggered = True

        if not triggered:
            return None

        severity = "warning"
        if point.value > value * 1.5:
            severity = "critical"

        return AnomalyResult(
            metric_name=point.metric_name,
            labels=point.labels,
            value=point.value,
            threshold=value,
            anomaly_type="threshold",
            severity=severity,
            confidence=0.95,
            message=f"{point.metric_name} {op} {value} (current: {point.value:.2f})",
        )

    def _check_sigma(self, point: MetricPoint) -> Optional[AnomalyResult]:
        """Check using 3-sigma rule."""
        key = self._make_key(point.metric_name, point.labels)

        # Add to window
        if key not in self.windows:
            self.windows[key] = []
        window = self.windows[key]
        window.append(point.value)
        if len(window) > self.window_size:
            window.pop(0)

        # Need at least 10 data points
        if len(window) < 10:
            return None

        mean = sum(window) / len(window)
        variance = sum((x - mean) ** 2 for x in window) / len(window)
        std = variance ** 0.5

        if std == 0:
            return None

        z_score = abs(point.value - mean) / std
        if z_score < self.sigma_multiplier:
            return None

        # Determine type
        if point.value > mean:
            anomaly_type = "spike"
        else:
            anomaly_type = "drop"

        # Severity based on z-score
        if z_score > 5:
            severity = "critical"
        elif z_score > 4:
            severity = "warning"
        else:
            severity = "info"

        return AnomalyResult(
            metric_name=point.metric_name,
            labels=point.labels,
            value=point.value,
            threshold=mean + self.sigma_multiplier * std,
            anomaly_type=anomaly_type,
            severity=severity,
            confidence=min(z_score / 10.0, 1.0),
            message=f"{point.metric_name} {anomaly_type}: {point.value:.2f} "
                    f"(mean={mean:.2f}, std={std:.2f}, z={z_score:.1f})",
        )

    def _make_key(self, metric_name: str, labels: dict) -> str:
        label_str = ",".join(f"{k}={v}" for k, v in sorted(labels.items()))
        return f"{metric_name}{{{label_str}}}"

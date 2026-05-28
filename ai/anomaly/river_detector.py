"""Anomaly Detection - River Online Learning.

Phase 2 implementation: HalfSpaceTrees for online anomaly detection.
Falls back to 3-sigma rule engine if River is not installed.
"""

import logging
import time
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger(__name__)

# Try importing River; fall back gracefully
try:
    from river import anomaly as river_anomaly
    from river import preprocessing as river_preprocessing
    HAS_RIVER = True
    logger.info("River library loaded — online learning enabled")
except ImportError:
    HAS_RIVER = False
    logger.warning("River not installed — falling back to 3-sigma rule engine")

from ai.anomaly.detector import RuleEngineDetector, MetricPoint, AnomalyResult


class RiverDetector:
    """Online anomaly detection using River HalfSpaceTrees.

    Maintains a separate model per metric series (keyed by metric name + labels).
    HalfSpaceTrees is an online algorithm that:
    - Learns the data distribution incrementally (no batch training)
    - O(n) memory per tree, fast inference
    - Works well for detecting point anomalies in streaming data

    Falls back to the Phase 1 rule engine for:
    - First N warmup points (not enough data for River)
    - Static threshold rules
    - When River is not installed
    """

    def __init__(
        self,
        n_trees: int = 25,
        height: int = 15,
        window_size: int = 250,
        warmup_points: int = 50,
        sigma_multiplier: float = 3.0,
    ):
        self.n_trees = n_trees
        self.height = height
        self.window_size = window_size
        self.warmup_points = warmup_points

        # metric_key -> River model
        self.models: dict = {}
        # metric_key -> point count
        self.point_counts: dict[str, int] = {}
        # metric_key -> running stats for confidence calibration
        self.stats: dict[str, dict] = {}

        # Fallback rule engine for static thresholds and warmup
        self.rule_engine = RuleEngineDetector(window_size=100, sigma_multiplier=sigma_multiplier)

    def set_threshold(self, metric_name: str, labels: dict, op: str, value: float):
        """Set a static threshold (delegated to rule engine)."""
        self.rule_engine.set_threshold(metric_name, labels, op, value)

    def _get_or_create_model(self, key: str):
        """Get or create a River model for a metric series."""
        if key not in self.models:
            if HAS_RIVER:
                model = river_anomaly.HalfSpaceTrees(
                    n_trees=self.n_trees,
                    height=self.height,
                    window_size=self.window_size,
                    seed=42,
                )
                self.models[key] = model
                self.point_counts[key] = 0
                self.stats[key] = {"values": [], "anomaly_scores": []}
                logger.debug(f"Created River model for {key}")
            else:
                # No River — will use rule engine fallback
                self.models[key] = None
                self.point_counts[key] = 0
        return self.models.get(key)

    def detect(self, point: MetricPoint) -> Optional[AnomalyResult]:
        """Detect if a metric point is anomalous.

        Strategy:
        1. Check static thresholds (always, via rule engine)
        2. If River available and warmup complete: use HalfSpaceTrees score
        3. Otherwise: fall back to 3-sigma rule engine
        """
        key = self._make_key(point.metric_name, point.labels)

        # 1. Static threshold check (always)
        static_result = self.rule_engine._check_static(point, self.rule_engine.static_thresholds.get(key, {}))
        if static_result:
            return static_result

        # Get or create model
        model = self._get_or_create_model(key)
        self.point_counts[key] = self.point_counts.get(key, 0) + 1

        # 2. River-based detection
        if HAS_RIVER and model is not None:
            return self._detect_river(key, model, point)

        # 3. Fallback: 3-sigma
        return self.rule_engine._check_sigma(point)

    def _detect_river(self, key: str, model, point: MetricPoint) -> Optional[AnomalyResult]:
        """Detect using River HalfSpaceTrees."""
        count = self.point_counts[key]

        # During warmup, just learn — don't detect
        if count <= self.warmup_points:
            model.learn_one({"value": point.value})
            return None

        # Score the point (higher = more anomalous)
        score = model.score_one({"value": point.value})

        # Learn from the point (online learning)
        model.learn_one({"value": point.value})

        # Update running stats for calibration
        stats = self.stats[key]
        stats["values"].append(point.value)
        stats["anomaly_scores"].append(score)
        if len(stats["values"]) > 1000:
            stats["values"] = stats["values"][-500:]
            stats["anomaly_scores"] = stats["anomaly_scores"][-500:]

        # Adaptive threshold: use the 95th percentile of recent scores
        if len(stats["anomaly_scores"]) < 20:
            # Not enough data for adaptive threshold
            return None

        sorted_scores = sorted(stats["anomaly_scores"])
        p95_idx = int(len(sorted_scores) * 0.95)
        adaptive_threshold = sorted_scores[p95_idx]

        # Also check if score is significantly above the adaptive threshold
        if score < adaptive_threshold:
            return None

        # Determine severity based on how far above threshold
        ratio = score / max(adaptive_threshold, 0.001)
        if ratio > 3.0:
            severity = "critical"
        elif ratio > 2.0:
            severity = "warning"
        else:
            severity = "info"

        # Confidence based on score distribution
        mean_score = sum(stats["anomaly_scores"]) / len(stats["anomaly_scores"])
        std_score = (sum((s - mean_score) ** 2 for s in stats["anomaly_scores"]) / len(stats["anomaly_scores"])) ** 0.5
        if std_score > 0:
            z = (score - mean_score) / std_score
            confidence = min(z / 5.0, 1.0)
        else:
            confidence = 0.5

        # Determine anomaly type from recent values
        values = stats["values"]
        if len(values) >= 10:
            recent_mean = sum(values[-10:]) / 10
            older_mean = sum(values[-50:-10]) / max(len(values[-50:-10]), 1) if len(values) > 10 else recent_mean
            if point.value > recent_mean:
                anomaly_type = "spike"
            else:
                anomaly_type = "drop"
        else:
            anomaly_type = "anomaly"

        return AnomalyResult(
            metric_name=point.metric_name,
            labels=point.labels,
            value=point.value,
            threshold=adaptive_threshold,
            anomaly_type=anomaly_type,
            severity=severity,
            confidence=max(confidence, 0.1),
            message=(
                f"{point.metric_name} {anomaly_type}: {point.value:.4f} "
                f"(score={score:.4f}, threshold={adaptive_threshold:.4f}, ratio={ratio:.1f})"
            ),
        )

    def _make_key(self, metric_name: str, labels: dict) -> str:
        label_str = ",".join(f"{k}={v}" for k, v in sorted(labels.items()))
        return f"{metric_name}{{{label_str}}}"

    def get_status(self) -> dict:
        """Return status of all models."""
        models_info = {}
        for key, model in self.models.items():
            count = self.point_counts.get(key, 0)
            stats = self.stats.get(key, {})
            models_info[key] = {
                "point_count": count,
                "warmup_done": count > self.warmup_points,
                "has_river_model": model is not None,
                "recent_score": stats.get("anomaly_scores", [0])[-1] if stats.get("anomaly_scores") else 0,
            }
        return {
            "river_available": HAS_RIVER,
            "model_count": len(self.models),
            "models": models_info,
        }

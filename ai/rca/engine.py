"""Root Cause Analysis Engine.

Uses the PC algorithm to discover causal relationships between metrics,
then traces back through the causal graph to identify root causes.

Phase 2 implementation with pgmpy.
Falls back to correlation-based analysis if pgmpy is not available.
"""

import logging
import time
from dataclasses import dataclass, field
from typing import Optional

import numpy as np

logger = logging.getLogger(__name__)

# Try importing pgmpy for PC algorithm
try:
    from pgmpy.estimators import PC
    from pgmpy.models import BayesianNetwork
    HAS_PGMPY = True
    logger.info("pgmpy loaded — PC algorithm available")
except ImportError:
    HAS_PGMPY = False
    logger.warning("pgmpy not installed — falling back to correlation-based RCA")


@dataclass
class MetricTimeSeries:
    """Time series data for a metric."""
    name: str
    timestamps: list[float]
    values: list[float]
    labels: dict = field(default_factory=dict)


@dataclass
class CausalEdge:
    """A causal relationship between two metrics."""
    source: str
    target: str
    confidence: float
    lag: int = 0  # time lag in seconds


@dataclass
class RootCause:
    """A identified root cause."""
    metric_name: str
    score: float  # 0-1, higher = more likely root cause
    reason: str
    related_metrics: list[str] = field(default_factory=list)
    evidence: list[str] = field(default_factory=list)


class RCAEngine:
    """Root Cause Analysis engine.

    Discovers causal relationships using PC algorithm and identifies
    root causes when incidents occur.
    """

    def __init__(self):
        # metric_name -> time series data
        self.time_series: dict[str, MetricTimeSeries] = {}
        # Discovered causal graph edges
        self.causal_edges: list[CausalEdge] = []
        # Last graph discovery time
        self.last_discovery: float = 0
        # Minimum data points for graph discovery
        self.min_points = 50

    def add_time_series(self, ts: MetricTimeSeries):
        """Add or update time series data for a metric."""
        existing = self.time_series.get(ts.name)
        if existing:
            # Merge, keep last 1000 points
            existing.timestamps.extend(ts.timestamps)
            existing.values.extend(ts.values)
            if len(existing.timestamps) > 1000:
                existing.timestamps = existing.timestamps[-1000:]
                existing.values = existing.values[-1000:]
        else:
            self.time_series[ts.name] = ts

    def discover_causal_graph(self) -> list[CausalEdge]:
        """Discover causal relationships using PC algorithm.

        This should be called periodically (e.g., every hour) with fresh data.
        """
        if len(self.time_series) < 2:
            logger.warning("Need at least 2 metrics for causal discovery")
            return []

        # Check if we have enough data
        names = []
        data_matrix = []
        min_len = min(len(ts.values) for ts in self.time_series.values())

        if min_len < self.min_points:
            logger.warning(f"Need at least {self.min_points} data points, have {min_len}")
            return []

        # Build aligned data matrix
        for name, ts in self.time_series.items():
            if len(ts.values) >= self.min_points:
                names.append(name)
                data_matrix.append(ts.values[-min_len:])

        if len(names) < 2:
            return []

        data = np.array(data_matrix).T  # shape: (samples, metrics)

        if HAS_PGMPY:
            edges = self._pc_discovery(names, data)
        else:
            edges = self._correlation_discovery(names, data)

        self.causal_edges = edges
        self.last_discovery = time.time()
        logger.info(f"Causal graph discovered: {len(edges)} edges from {len(names)} metrics")
        return edges

    def _pc_discovery(self, names: list[str], data: np.ndarray) -> list[CausalEdge]:
        """Run PC algorithm for causal discovery."""
        try:
            import pandas as pd
            df = pd.DataFrame(data, columns=names)

            pc = PC(df)
            model = pc.estimate(variant="stable", max_cond_vars=4)

            edges = []
            for src, dst in model.edges():
                edges.append(CausalEdge(
                    source=src,
                    target=dst,
                    confidence=0.8,  # PC doesn't give edge weights directly
                ))

            return edges
        except Exception as e:
            logger.error(f"PC algorithm failed: {e}")
            return self._correlation_discovery(names, data)

    def _correlation_discovery(self, names: list[str], data: np.ndarray) -> list[CausalEdge]:
        """Fallback: use correlation to approximate causal relationships."""
        n = len(names)
        corr = np.corrcoef(data.T)

        edges = []
        for i in range(n):
            for j in range(i + 1, n):
                r = abs(corr[i, j])
                if r > 0.7:  # Strong correlation threshold
                    # Use Granger-like lag test for direction
                    lag = self._estimate_lag(data[:, i], data[:, j])
                    if lag >= 0:
                        edges.append(CausalEdge(source=names[i], target=names[j], confidence=r, lag=lag))
                    else:
                        edges.append(CausalEdge(source=names[j], target=names[i], confidence=r, lag=-lag))

        return edges

    def _estimate_lag(self, x: np.ndarray, y: np.ndarray, max_lag: int = 5) -> int:
        """Estimate time lag between two series using cross-correlation."""
        best_lag = 0
        best_corr = 0

        for lag in range(-max_lag, max_lag + 1):
            if lag == 0:
                continue
            if lag > 0:
                c = np.corrcoef(x[:-lag], y[lag:])[0, 1]
            else:
                c = np.corrcoef(x[-lag:], y[:lag])[0, 1]

            if abs(c) > abs(best_corr):
                best_corr = c
                best_lag = lag

        return best_lag

    def analyze_root_cause(self, incident_metrics: list[str]) -> list[RootCause]:
        """Analyze root cause for an incident.

        Given a list of affected metrics, trace back through the causal graph
        to find the most likely root cause.
        """
        if not self.causal_edges:
            logger.warning("No causal graph available, using simple analysis")
            return self._simple_analysis(incident_metrics)

        # Build adjacency list (reversed for root cause tracing)
        parents: dict[str, list[str]] = {}
        for edge in self.causal_edges:
            if edge.target not in parents:
                parents[edge.target] = []
            parents[edge.target].append(edge.source)

        # Trace back from affected metrics
        candidates: dict[str, float] = {}
        evidence: dict[str, list[str]] = {}

        for metric in incident_metrics:
            visited = set()
            queue = [metric]
            depth = 0

            while queue and depth < 5:
                next_queue = []
                for node in queue:
                    if node in visited:
                        continue
                    visited.add(node)

                    node_parents = parents.get(node, [])
                    if not node_parents:
                        # This is a root node — likely root cause
                        candidates[node] = candidates.get(node, 0) + 1.0 / (depth + 1)
                        if node not in evidence:
                            evidence[node] = []
                        evidence[node].append(f"影响 {metric} (深度 {depth})")
                    else:
                        next_queue.extend(node_parents)

                queue = next_queue
                depth += 1

        # Rank candidates
        results = []
        for metric, score in sorted(candidates.items(), key=lambda x: -x[1]):
            results.append(RootCause(
                metric_name=metric,
                score=min(score, 1.0),
                reason=f"因果链追溯指向 {metric}",
                related_metrics=incident_metrics,
                evidence=evidence.get(metric, []),
            ))

        return results[:5]  # Top 5

    def _simple_analysis(self, incident_metrics: list[str]) -> list[RootCause]:
        """Simple correlation-based root cause when no graph is available."""
        if not incident_metrics:
            return []

        # Find the metric with highest correlation to others
        scores: dict[str, float] = {}
        for target in incident_metrics:
            ts_target = self.time_series.get(target)
            if not ts_target:
                continue

            for other_name, other_ts in self.time_series.items():
                if other_name == target:
                    continue
                min_len = min(len(ts_target.values), len(other_ts.values))
                if min_len < 10:
                    continue

                corr = np.corrcoef(
                    ts_target.values[-min_len:],
                    other_ts.values[-min_len:]
                )[0, 1]

                if abs(corr) > 0.5:
                    scores[other_name] = scores.get(other_name, 0) + abs(corr)

        results = []
        for metric, score in sorted(scores.items(), key=lambda x: -x[1]):
            results.append(RootCause(
                metric_name=metric,
                score=min(score / len(incident_metrics), 1.0),
                reason=f"与受影响指标高度相关",
                related_metrics=incident_metrics,
            ))

        return results[:5]

    def get_graph(self) -> dict:
        """Return the current causal graph for visualization."""
        nodes = set()
        edges = []
        for e in self.causal_edges:
            nodes.add(e.source)
            nodes.add(e.target)
            edges.append({
                "source": e.source,
                "target": e.target,
                "confidence": e.confidence,
                "lag": e.lag,
            })

        return {
            "nodes": list(nodes),
            "edges": edges,
            "last_discovery": self.last_discovery,
            "metric_count": len(self.time_series),
        }

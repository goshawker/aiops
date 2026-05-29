---
title: "Metrics Collection Test Plan Design"
last_updated: "2026-05-29"
---

## Overview
This design document outlines the test strategy for the metrics collection subsystem. It covers unit, integration, load, and end‑to‑end tests that validate the Agent’s `/metrics` endpoint, heartbeat logic, and Collector persistence.

## Architecture
- **Agent** – runs on each host, collects system metrics via `collectMetrics`, stores them in `lastMetrics`, exposes `/metrics` and sends heartbeats to the Collector.
- **Collector** – receives heartbeats, persists metrics in a time‑series store, exposes `/metrics/query`.
- **Test harness** – orchestrates agents, collectors, and queries.

## Metrics Collection Flow
1. Agent starts `collectAndSend` loop.
2. `collectAndSend` calls `collectMetrics` → updates `lastMetrics`.
3. `serveMetrics` exposes `lastMetrics` in Prometheus format.
4. Heartbeat JSON is POSTed to Collector.
5. Collector stores metrics and serves `/metrics/query`.

## Testing Approach
1. **Unit tests** – mock `collectMetrics`, call `serveMetrics`, assert Prometheus output.
2. **Integration tests** – run Agent + Collector in containers, verify heartbeat persistence and query correctness.
3. **Load tests** – spin up many Agents, monitor `/metrics` latency and value stability.
4. **End‑to‑end tests** – simulate a real deployment, run full pipeline, generate a test report.

## Acceptance Criteria
- `/metrics` returns valid Prometheus text with all expected labels.
- Heartbeats are persisted and queryable via `/metrics/query`.
- Load tests show < 200 ms latency under 100 concurrent Agents.
- End‑to‑end report shows 100 % pass for all test cases.

## Dependencies
- Collector service running on `collector.yaml`.
- Docker for integration and load tests.
- Prometheus client library for parsing metrics.

## Risks & Mitigations
- **Race conditions** in `lastMetrics` → protected by `metricsMu`.
- **Collector downtime** → test harness retries heartbeats.
- **Metric drift** → use tolerance thresholds in tests.

---

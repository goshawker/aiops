# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0.0] - 2026-05-28

### Added

**Core Platform**
- Go API Gateway with routing and health checks
- Query service: PromQL proxy + ClickHouse log/trace queries
- Alert engine: Kafka pipeline, event ingestion, rule evaluation
- Admin service: RBAC, user management, audit logs
- Collector service: agent registration, heartbeat, config management
- Job service: shell/http job execution with cron scheduling
- Host agent: lightweight CGO_ENABLED=0 binary for metric collection

**AI Services**
- Anomaly detection: 3-sigma rule engine + River HalfSpaceTrees online learning
- Alert aggregation: label-matching deduplication via Kafka
- LLM service: Qwen2 multi-tier model (1.5B/7B INT4/INT8) with rule engine fallback
- Root cause analysis: PC algorithm causal discovery via pgmpy
- Model manager: hot-swap model tiers, offline download script

**Frontend**
- React 18 + TypeScript + Vite + Ant Design 5
- Dashboard, Metrics, Logs, Alerts, Anomaly, RCA, Topology, Traces, Jobs pages
- Zustand state management, Axios API clients

**Infrastructure**
- Docker Compose: VictoriaMetrics, ClickHouse, Kafka, ZooKeeper, NebulaGraph
- Helm chart: full K8s deployment with StatefulSets, Ingress, PVCs
- NebulaGraph schema: topology graph (hosts, services, components, edges)
- ClickHouse schema: logs table with partitioning and TTL
- SQLite schema: users, tenants, alert rules, jobs, audit logs

**Enterprise Features**
- Multi-tenant isolation (X-Tenant-ID header, plan-based limits)
- Xinchuang compatibility: DM (达梦) and KingBase (金仓) DB adapters
- Offline installer: air-gapped deployment package with Docker image tarballs
- APM trace ingestion: OTLP-like JSON format with Gantt chart visualization

**Deployment**
- 7 Go services, 4 Python AI services, 1 frontend
- Production-ready Helm chart with configurable replicas and resources

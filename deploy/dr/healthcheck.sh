#!/bin/bash
# AIOps Platform - Health Check & Auto-Failover Script
#
# Monitors all services and performs automatic recovery actions.
# Usage:
#   bash deploy/dr/healthcheck.sh              # Check all services
#   bash deploy/dr/healthcheck.sh --fix        # Auto-fix unhealthy services
#   bash deploy/dr/healthcheck.sh --watch 30   # Continuous monitoring (every 30s)

set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-/opt/aiops/deploy/docker-compose/docker-compose.yml}"
ALERT_WEBHOOK="${ALERT_WEBHOOK:-}"
FIX_MODE=false
WATCH_INTERVAL=0

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --fix) FIX_MODE=true; shift ;;
    --watch) WATCH_INTERVAL="$2"; shift 2 ;;
    *) shift ;;
  esac
done

# ── Service Health Checks ──────────────────────────────

check_service() {
  local name="$1"
  local url="$2"
  local container="$3"

  # Check container status
  local status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "not_found")
  if [ "$status" != "running" ]; then
    echo "FAIL  $name - container $status"
    return 1
  fi

  # Check HTTP health (if URL provided)
  if [ -n "$url" ]; then
    local http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 400 ]; then
      echo "OK    $name (HTTP $http_code)"
      return 0
    else
      echo "WARN  $name - HTTP $http_code"
      return 1
    fi
  else
    echo "OK    $name (container running)"
    return 0
  fi
}

# ── Auto-Fix Actions ───────────────────────────────────

fix_service() {
  local container="$1"
  local name="$2"

  echo "FIX   Restarting $name ($container)..."
  docker restart "$container" 2>/dev/null || {
    echo "FIX   Container restart failed, trying compose..."
    docker compose -f "$COMPOSE_FILE" restart "$name" 2>/dev/null || true
  }
  sleep 3

  # Verify recovery
  local status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "not_found")
  if [ "$status" = "running" ]; then
    echo "FIX   $name recovered"
    return 0
  else
    echo "FIX   $name FAILED to recover"
    return 1
  fi
}

# ── Alert Notification ─────────────────────────────────

send_alert() {
  local message="$1"

  if [ -n "$ALERT_WEBHOOK" ]; then
    curl -s -X POST "$ALERT_WEBHOOK" \
      -H "Content-Type: application/json" \
      -d "{\"text\": \"[AIOps DR] $message\"}" \
      --max-time 5 2>/dev/null || true
  fi
}

# ── Main Check ─────────────────────────────────────────

run_checks() {
  local failures=0
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

  echo "========================================"
  echo " AIOps Health Check - $timestamp"
  echo "========================================"

  # Storage Layer
  echo ""
  echo "[Storage]"
  check_service "VictoriaMetrics" "http://localhost:8428/health" "aiops-vm" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-vm" "victoria-metrics"; }
  check_service "ClickHouse" "http://localhost:8123/ping" "aiops-clickhouse" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-clickhouse" "clickhouse"; }

  # Data Pipeline
  echo ""
  echo "[Pipeline]"
  check_service "Zookeeper" "" "aiops-zookeeper" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-zookeeper" "zookeeper"; }
  check_service "Kafka" "" "aiops-kafka" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-kafka" "kafka"; }

  # Metrics Collection
  echo ""
  echo "[Metrics]"
  check_service "Node Exporter" "http://localhost:9100/metrics" "aiops-node-exporter" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-node-exporter" "node-exporter"; }
  check_service "VMAgent" "http://localhost:8429/health" "aiops-vmagent" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-vmagent" "vmagent"; }

  # Go Services
  echo ""
  echo "[Go Services]"
  check_service "Gateway" "http://localhost:8080/health" "aiops-gateway" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-gateway" "gateway"; }
  check_service "Query" "" "aiops-query" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-query" "query-service"; }
  check_service "Alert" "" "aiops-alert" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-alert" "alert-engine"; }
  check_service "Admin" "" "aiops-admin" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-admin" "admin-service"; }
  check_service "Collector" "" "aiops-collector" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-collector" "collector-service"; }
  check_service "Job" "" "aiops-job" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-job" "job-engine"; }

  # AI Services
  echo ""
  echo "[AI Services]"
  check_service "Anomaly" "http://localhost:5001/health" "aiops-anomaly" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-anomaly" "anomaly-service"; }
  check_service "LLM" "http://localhost:5004/health" "aiops-llm" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-llm" "llm-service"; }

  # Frontend
  echo ""
  echo "[Frontend]"
  check_service "Web (Nginx)" "http://localhost:80" "aiops-web" || { failures=$((failures+1)); $FIX_MODE && fix_service "aiops-web" "web"; }

  # Summary
  echo ""
  echo "========================================"
  if [ $failures -eq 0 ]; then
    echo " ALL SERVICES HEALTHY"
  else
    echo " FAILURES: $failures"
    send_alert "$failures service(s) unhealthy on $(hostname)"
  fi
  echo "========================================"

  return $failures
}

# ── Watch Mode ─────────────────────────────────────────

if [ "$WATCH_INTERVAL" -gt 0 ]; then
  echo "Monitoring every ${WATCH_INTERVAL}s (Ctrl+C to stop)"
  while true; do
    run_checks || true
    sleep "$WATCH_INTERVAL"
  done
else
  run_checks
fi

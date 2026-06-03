#!/bin/bash
# AIOps Platform - Automated Backup Script
# Backs up: SQLite, ClickHouse, VictoriaMetrics
#
# Usage:
#   bash deploy/backup/backup.sh              # Full backup
#   bash deploy/backup/backup.sh --sqlite     # SQLite only
#   bash deploy/backup/backup.sh --clickhouse # ClickHouse only
#   bash deploy/backup/backup.sh --vm         # VictoriaMetrics only
#
# Cron example (daily at 2:00 AM, keep 30 days):
#   0 2 * * * /opt/aiops/deploy/backup/backup.sh --retention 30 >> /var/log/aiops-backup.log 2>&1

set -euo pipefail

# ── Configuration ──────────────────────────────────────
BACKUP_DIR="${BACKUP_DIR:-/opt/aiops/backups}"
RETENTION_DAYS="${1:-30}"
DOCKER_COMPOSE="${DOCKER_COMPOSE:-/opt/aiops/deploy/docker-compose/docker-compose.yml}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE_TAG=$(date +%Y%m%d)

# Parse args
DO_ALL=true
DO_SQLITE=false
DO_CLICKHOUSE=false
DO_VM=false

for arg in "$@"; do
  case "$arg" in
    --sqlite) DO_ALL=false; DO_SQLITE=true ;;
    --clickhouse) DO_ALL=false; DO_CLICKHOUSE=true ;;
    --vm) DO_ALL=false; DO_VM=true ;;
    --retention) ;;  # next arg is retention days
    --retention=*) RETENTION_DAYS="${arg#*=}" ;;
    [0-9]*) RETENTION_DAYS="$arg" ;;
  esac
done

echo "========================================"
echo " AIOps Backup - $(date)"
echo " Backup dir: $BACKUP_DIR"
echo " Retention:  ${RETENTION_DAYS} days"
echo "========================================"

mkdir -p "$BACKUP_DIR"/{sqlite,clickhouse,victoriametrics}

# ── SQLite Backup ──────────────────────────────────────
backup_sqlite() {
  echo ""
  echo "[SQLite] Starting backup..."

  # Copy SQLite database from Docker volume
  local container="aiops-sqlite-init"
  local backup_file="$BACKUP_DIR/sqlite/aiops_${TIMESTAMP}.db"

  # Method 1: Copy from running admin container's mounted volume
  if docker exec aiops-admin test -f /app/aiops.db 2>/dev/null; then
    docker cp aiops-admin:/app/aiops.db "$backup_file"
    echo "[SQLite] Copied from admin container -> $backup_file"
  # Method 2: Use sqlite3 to dump from volume
  elif docker ps --format '{{.Names}}' | grep -q sqlite; then
    docker exec "$container" sh -c "sqlite3 /data/aiops.db '.backup /data/backup.db'" 2>/dev/null
    docker cp "$container:/data/backup.db" "$backup_file"
    echo "[SQLite] Copied from volume -> $backup_file"
  # Method 3: Direct file copy if running outside Docker
  elif [ -f "aiops.db" ]; then
    cp aiops.db "$backup_file"
    echo "[SQLite] Copied from local file -> $backup_file"
  else
    echo "[SQLite] WARNING: Could not find SQLite database, skipping"
    return 1
  fi

  # Verify backup
  if [ -f "$backup_file" ] && [ -s "$backup_file" ]; then
    local size=$(du -h "$backup_file" | cut -f1)
    echo "[SQLite] Backup completed: $size"
  else
    echo "[SQLite] ERROR: Backup file is empty or missing"
    return 1
  fi
}

# ── ClickHouse Backup ──────────────────────────────────
backup_clickhouse() {
  echo ""
  echo "[ClickHouse] Starting backup..."

  local backup_dir="$BACKUP_DIR/clickhouse/aiops_${TIMESTAMP}"
  mkdir -p "$backup_dir"

  # Use clickhouse-client to export tables
  local container="aiops-clickhouse"

  if ! docker ps --format '{{.Names}}' | grep -q "$container"; then
    echo "[ClickHouse] WARNING: Container not running, skipping"
    return 1
  fi

  # Export logs table (last 7 days for manageable size)
  echo "[ClickHouse] Exporting aiops.logs (last 7 days)..."
  docker exec "$container" clickhouse-client \
    --query "SELECT * FROM aiops.logs WHERE timestamp > now() - INTERVAL 7 DAY FORMAT Native" \
    > "$backup_dir/logs.native" 2>/dev/null || true

  # Export traces table (last 3 days)
  echo "[ClickHouse] Exporting aiops.traces (last 3 days)..."
  docker exec "$container" clickhouse-client \
    --query "SELECT * FROM aiops.traces WHERE timestamp > now() - INTERVAL 3 DAY FORMAT Native" \
    > "$backup_dir/traces.native" 2>/dev/null || true

  # Export schema
  echo "[ClickHouse] Exporting schema..."
  docker exec "$container" clickhouse-client \
    --query "SHOW CREATE TABLE aiops.logs" > "$backup_dir/logs_schema.sql" 2>/dev/null || true
  docker exec "$container" clickhouse-client \
    --query "SHOW CREATE TABLE aiops.traces" > "$backup_dir/traces_schema.sql" 2>/dev/null || true

  # Compress
  echo "[ClickHouse] Compressing..."
  tar czf "$BACKUP_DIR/clickhouse/aiops_${TIMESTAMP}.tar.gz" -C "$BACKUP_DIR/clickhouse" "aiops_${TIMESTAMP}"
  rm -rf "$backup_dir"

  local size=$(du -h "$BACKUP_DIR/clickhouse/aiops_${TIMESTAMP}.tar.gz" | cut -f1)
  echo "[ClickHouse] Backup completed: $size"
}

# ── VictoriaMetrics Backup ─────────────────────────────
backup_victoriametrics() {
  echo ""
  echo "[VictoriaMetrics] Starting backup..."

  local backup_file="$BACKUP_DIR/victoriametrics/vm_${TIMESTAMP}.tar.gz"
  local container="aiops-vm"

  if ! docker ps --format '{{.Names}}' | grep -q "$container"; then
    echo "[VictoriaMetrics] WARNING: Container not running, skipping"
    return 1
  fi

  # Create snapshot via API
  echo "[VictoriaMetrics] Creating snapshot..."
  local snapshot_name="backup_${TIMESTAMP}"
  docker exec "$container" wget -q -O - "http://localhost:8428/snapshot/create" > /dev/null 2>&1 || true

  # Copy snapshot data
  echo "[VictoriaMetrics] Copying snapshot data..."
  docker cp "$container:/victoria-metrics-data/snapshots" "/tmp/vm_snapshots_${TIMESTAMP}" 2>/dev/null || true

  if [ -d "/tmp/vm_snapshots_${TIMESTAMP}" ]; then
    tar czf "$backup_file" -C "/tmp" "vm_snapshots_${TIMESTAMP}"
    rm -rf "/tmp/vm_snapshots_${TIMESTAMP}"

    local size=$(du -h "$backup_file" | cut -f1)
    echo "[VictoriaMetrics] Backup completed: $size"
  else
    # Fallback: copy data directory directly (requires stopping VM)
    echo "[VictoriaMetrics] Snapshot not available, copying data directory..."
    docker run --rm -v aiops_vm-data:/data -v "$BACKUP_DIR/victoriametrics":/backup alpine \
      tar czf "/backup/vm_data_${TIMESTAMP}.tar.gz" -C /data .

    local size=$(du -h "$BACKUP_DIR/victoriametrics/vm_data_${TIMESTAMP}.tar.gz" | cut -f1)
    echo "[VictoriaMetrics] Backup completed (data copy): $size"
  fi
}

# ── Cleanup Old Backups ────────────────────────────────
cleanup_old_backups() {
  echo ""
  echo "[Cleanup] Removing backups older than ${RETENTION_DAYS} days..."

  local count=0
  for dir in sqlite clickhouse victoriametrics; do
    if [ -d "$BACKUP_DIR/$dir" ]; then
      local deleted=$(find "$BACKUP_DIR/$dir" -type f -mtime +"$RETENTION_DAYS" -delete -print | wc -l)
      count=$((count + deleted))
    fi
  done

  echo "[Cleanup] Removed $count old backup files"
}

# ── Main ───────────────────────────────────────────────
main() {
  local start_time=$(date +%s)
  local errors=0

  if $DO_ALL || $DO_SQLITE; then
    backup_sqlite || errors=$((errors + 1))
  fi

  if $DO_ALL || $DO_CLICKHOUSE; then
    backup_clickhouse || errors=$((errors + 1))
  fi

  if $DO_ALL || $DO_VM; then
    backup_victoriametrics || errors=$((errors + 1))
  fi

  cleanup_old_backups

  local end_time=$(date +%s)
  local duration=$((end_time - start_time))

  echo ""
  echo "========================================"
  echo " Backup completed in ${duration}s"
  echo " Errors: $errors"
  echo " Location: $BACKUP_DIR"
  echo "========================================"

  # Summary
  echo ""
  echo "Backup inventory:"
  du -sh "$BACKUP_DIR"/*/* 2>/dev/null || echo "  (no backups found)"

  return $errors
}

main

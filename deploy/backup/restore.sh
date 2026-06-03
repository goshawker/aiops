#!/bin/bash
# AIOps Platform - Backup Restore Script
#
# Usage:
#   bash deploy/backup/restore.sh --sqlite /path/to/backup.db
#   bash deploy/backup/restore.sh --clickhouse /path/to/backup.tar.gz
#   bash deploy/backup/restore.sh --vm /path/to/backup.tar.gz
#   bash deploy/backup/restore.sh --list              # List available backups

set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/opt/aiops/backups}"

echo "========================================"
echo " AIOps Restore - $(date)"
echo "========================================"

# ── List backups ───────────────────────────────────────
list_backups() {
  echo ""
  echo "Available backups in $BACKUP_DIR:"
  echo ""

  for dir in sqlite clickhouse victoriametrics; do
    echo "[$dir]"
    if [ -d "$BACKUP_DIR/$dir" ]; then
      ls -lh "$BACKUP_DIR/$dir"/*.{db,tar.gz} 2>/dev/null | awk '{print "  " $NF " (" $5 ")"}' || echo "  (none)"
    else
      echo "  (directory not found)"
    fi
    echo ""
  done
}

# ── Restore SQLite ─────────────────────────────────────
restore_sqlite() {
  local backup_file="$1"

  if [ ! -f "$backup_file" ]; then
    echo "ERROR: Backup file not found: $backup_file"
    exit 1
  fi

  echo "[SQLite] Restoring from $backup_file..."

  # Stop admin service
  echo "[SQLite] Stopping admin service..."
  docker compose -f deploy/docker-compose/docker-compose.yml stop admin 2>/dev/null || true

  # Copy backup to container
  docker cp "$backup_file" aiops-admin:/app/aiops.db 2>/dev/null || {
    echo "[SQLite] ERROR: Could not copy to admin container"
    echo "[SQLite] Trying direct file copy..."
    cp "$backup_file" aiops.db
  }

  # Restart admin service
  echo "[SQLite] Starting admin service..."
  docker compose -f deploy/docker-compose/docker-compose.yml start admin 2>/dev/null || true

  echo "[SQLite] Restore completed"
}

# ── Restore ClickHouse ─────────────────────────────────
restore_clickhouse() {
  local backup_file="$1"

  if [ ! -f "$backup_file" ]; then
    echo "ERROR: Backup file not found: $backup_file"
    exit 1
  fi

  echo "[ClickHouse] Restoring from $backup_file..."

  # Extract backup
  local tmp_dir="/tmp/ch_restore_$$"
  mkdir -p "$tmp_dir"
  tar xzf "$backup_file" -C "$tmp_dir"

  # Find native files
  local container="aiops-clickhouse"

  if [ -f "$tmp_dir"/*/logs.native ]; then
    echo "[ClickHouse] Restoring logs table..."
    cat "$tmp_dir"/*/logs.native | docker exec -i "$container" clickhouse-client \
      --query "INSERT INTO aiops.logs FORMAT Native" 2>/dev/null || true
  fi

  if [ -f "$tmp_dir"/*/traces.native ]; then
    echo "[ClickHouse] Restoring traces table..."
    cat "$tmp_dir"/*/traces.native | docker exec -i "$container" clickhouse-client \
      --query "INSERT INTO aiops.traces FORMAT Native" 2>/dev/null || true
  fi

  rm -rf "$tmp_dir"
  echo "[ClickHouse] Restore completed"
}

# ── Restore VictoriaMetrics ────────────────────────────
restore_vm() {
  local backup_file="$1"

  if [ ! -f "$backup_file" ]; then
    echo "ERROR: Backup file not found: $backup_file"
    exit 1
  fi

  echo "[VictoriaMetrics] Restoring from $backup_file..."

  # Stop VM
  echo "[VictoriaMetrics] Stopping VictoriaMetrics..."
  docker compose -f deploy/docker-compose/docker-compose.yml stop victoria-metrics 2>/dev/null || true

  # Extract and copy data
  local tmp_dir="/tmp/vm_restore_$$"
  mkdir -p "$tmp_dir"
  tar xzf "$backup_file" -C "$tmp_dir"

  # Restore via Docker volume
  docker run --rm -v aiops_vm-data:/data -v "$tmp_dir":/backup alpine \
    sh -c "rm -rf /data/* && cp -r /backup/*/snapshots/* /data/ 2>/dev/null || cp -r /backup/*/* /data/" 2>/dev/null || true

  rm -rf "$tmp_dir"

  # Restart VM
  echo "[VictoriaMetrics] Starting VictoriaMetrics..."
  docker compose -f deploy/docker-compose/docker-compose.yml start victoria-metrics 2>/dev/null || true

  echo "[VictoriaMetrics] Restore completed"
}

# ── Main ───────────────────────────────────────────────
case "${1:-}" in
  --list)
    list_backups
    ;;
  --sqlite)
    restore_sqlite "${2:?Usage: restore.sh --sqlite /path/to/backup.db}"
    ;;
  --clickhouse)
    restore_clickhouse "${2:?Usage: restore.sh --clickhouse /path/to/backup.tar.gz}"
    ;;
  --vm)
    restore_vm "${2:?Usage: restore.sh --vm /path/to/backup.tar.gz}"
    ;;
  *)
    echo "Usage:"
    echo "  restore.sh --list                          List available backups"
    echo "  restore.sh --sqlite /path/to/backup.db     Restore SQLite"
    echo "  restore.sh --clickhouse /path/to/backup.tar.gz  Restore ClickHouse"
    echo "  restore.sh --vm /path/to/backup.tar.gz    Restore VictoriaMetrics"
    exit 1
    ;;
esac

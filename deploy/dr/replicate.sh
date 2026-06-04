#!/bin/bash
# AIOps Platform - Cross-Node Data Replication
#
# Replicates backups from primary node to standby node.
# Usage:
#   bash deploy/dr/replicate.sh --to standby@192.168.1.200
#   bash deploy/dr/replicate.sh --to standby@192.168.1.200 --dry-run

set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/opt/aiops/backups}"
REMOTE_PATH="/opt/aiops/backups"
DRY_RUN=false
TARGET=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --to) TARGET="$2"; shift 2 ;;
    --dry-run) DRY_RUN=true; shift ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [ -z "$TARGET" ]; then
  echo "Usage: replicate.sh --to user@host [--dry-run]"
  echo ""
  echo "This script replicates AIOps backups to a standby node."
  echo "The standby node should have AIOps installed but stopped."
  exit 1
fi

echo "========================================"
echo " AIOps Data Replication"
echo " From: $(hostname)"
echo " To:   $TARGET"
echo " Time: $(date)"
echo "========================================"

# Verify SSH connectivity
echo ""
echo "[Check] Testing SSH connectivity..."
if ! ssh -o ConnectTimeout=5 "$TARGET" "echo ok" >/dev/null 2>&1; then
  echo "ERROR: Cannot connect to $TARGET via SSH"
  echo "Ensure SSH key authentication is configured."
  exit 1
fi
echo "[Check] SSH connection OK"

# Ensure remote backup directory exists
echo ""
echo "[Setup] Ensuring remote backup directory..."
ssh "$TARGET" "mkdir -p $REMOTE_PATH/{sqlite,clickhouse,victoriametrics}"

# Sync latest backups
echo ""
echo "[Sync] Replicating SQLite backups..."
if $DRY_RUN; then
  rsync -avn "$BACKUP_DIR/sqlite/" "$TARGET:$REMOTE_PATH/sqlite/"
else
  rsync -avz --progress "$BACKUP_DIR/sqlite/" "$TARGET:$REMOTE_PATH/sqlite/"
fi

echo ""
echo "[Sync] Replicating ClickHouse backups..."
if $DRY_RUN; then
  rsync -avn --max-size=500M "$BACKUP_DIR/clickhouse/" "$TARGET:$REMOTE_PATH/clickhouse/"
else
  rsync -avz --progress --max-size=500M "$BACKUP_DIR/clickhouse/" "$TARGET:$REMOTE_PATH/clickhouse/"
fi

echo ""
echo "[Sync] Replicating VictoriaMetrics backups..."
if $DRY_RUN; then
  rsync -avn --max-size=1G "$BACKUP_DIR/victoriametrics/" "$TARGET:$REMOTE_PATH/victoriametrics/"
else
  rsync -avz --progress --max-size=1G "$BACKUP_DIR/victoriametrics/" "$TARGET:$REMOTE_PATH/victoriametrics/"
fi

# Verify remote state
echo ""
echo "[Verify] Remote backup inventory:"
ssh "$TARGET" "du -sh $REMOTE_PATH/*/* 2>/dev/null || echo '  (no backups)'"

echo ""
echo "========================================"
echo " Replication completed"
echo " Standby node: $TARGET"
echo " Next: Run backup.sh on primary, then this script to sync"
echo "========================================"

#!/bin/bash
# VigilOps - Invisible Code Watermark System
# Embeds unique identifiers for source code leak tracking
#
# Usage:
#   bash scripts/add-watermark.sh              # Add watermarks
#   bash scripts/add-watermark.sh --verify     # Verify watermarks exist

set -euo pipefail
cd "$(dirname "$0")/.."

ORG_ID="vigilops"
BUILD_ID=$(date +%Y%m%d)

verify_watermark() {
    local file="$1"
    grep -q "VigilOps.*build:" "$file" 2>/dev/null
}

echo "Adding watermarks (org: ${ORG_ID}, build: ${BUILD_ID})..."
count=0

# Go files - add watermark after package declaration
echo "Go files..."
for f in $(find cmd internal -name "*.go" ! -name "*_test.go" 2>/dev/null); do
    if ! verify_watermark "$f"; then
        # Insert watermark line after package declaration
        awk -v wm="// [${ORG_ID}] build:${BUILD_ID}" '
            /^package / { print; print wm; next }
            { print }
        ' "$f" > "${f}.tmp" && mv "${f}.tmp" "$f"
        count=$((count + 1))
    fi
done

# TypeScript files - add watermark as first line
echo "TypeScript files..."
for f in $(find web/src -name "*.ts" -o -name "*.tsx" 2>/dev/null); do
    if ! verify_watermark "$f"; then
        echo "// [${ORG_ID}] build:${BUILD_ID}" | cat - "$f" > "${f}.tmp" && mv "${f}.tmp" "$f"
        count=$((count + 1))
    fi
done

# Python files - add watermark (preserve shebang)
echo "Python files..."
for f in $(find ai -name "*.py" ! -path "*__pycache__*" 2>/dev/null); do
    if ! verify_watermark "$f"; then
        if head -1 "$f" | grep -q "^#!"; then
            awk -v wm="# [${ORG_ID}] build:${BUILD_ID}" 'NR==1{print;print wm;next}{print}' "$f" > "${f}.tmp" && mv "${f}.tmp" "$f"
        else
            echo "# [${ORG_ID}] build:${BUILD_ID}" | cat - "$f" > "${f}.tmp" && mv "${f}.tmp" "$f"
        fi
        count=$((count + 1))
    fi
done

echo ""
echo "Done. Added watermarks to $count files."

# Verify
echo ""
echo "Verification (sample):"
for f in cmd/gateway/main.go web/src/pages/Dashboard.tsx ai/anomaly/detector.py; do
    if [ -f "$f" ]; then
        if verify_watermark "$f"; then
            echo "  OK   $f"
        else
            echo "  MISS $f"
        fi
    fi
done

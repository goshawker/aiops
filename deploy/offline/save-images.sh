#!/bin/bash
# Save Docker images for offline deployment
# Run this on a machine with internet access

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT_DIR="${1:-$SCRIPT_DIR/docker-images}"

mkdir -p "$OUTPUT_DIR"

IMAGES=(
    "victoriametrics/victoria-metrics:v1.106.1"
    "victoriametrics/vmagent:v1.106.1"
    "clickhouse/clickhouse-server:24.10"
    "bitnami/zookeeper:3.9"
    "bitnami/kafka:3.8"
    "prom/node-exporter:v1.8.2"
    "nginx:1.27"
    "vesoft/nebula-metad:v3.8.0"
    "vesoft/nebula-storaged:v3.8.0"
    "vesoft/nebula-graphd:v3.8.0"
)

echo "=== Saving Docker images for offline deployment ==="
echo "Output: $OUTPUT_DIR"
echo ""

for img in "${IMAGES[@]}"; do
    filename=$(echo "$img" | tr '/:' '_')
    echo "Pulling and saving: $img"
    docker pull "$img"
    docker save "$img" -o "$OUTPUT_DIR/${filename}.tar"
    echo "  -> ${filename}.tar"
done

echo ""
echo "=== Done. ${#IMAGES[@]} images saved to $OUTPUT_DIR ==="
echo "Copy this directory to the offline machine before running install.sh"

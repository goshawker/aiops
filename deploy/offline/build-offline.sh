#!/bin/bash
# AIOps Platform - Offline Installation Package Builder
# Usage: ./build-offline.sh [output_dir]
#
# Builds a self-contained installation package for environments
# without internet access (信创/内网部署).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUTPUT_DIR="${1:-$PROJECT_ROOT/dist/offline}"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

echo "=== AIOps Offline Package Builder ==="
echo "Version: $VERSION"
echo "Output:  $OUTPUT_DIR"
echo ""

# Clean
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"/{bin,web,ai,config,sql,docker,scripts}

# ──────────────────────────────────────
# 1. Build Go binaries (linux/amd64 for 信创 x86)
# ──────────────────────────────────────
echo "[1/6] Building Go services..."
cd "$PROJECT_ROOT"
for svc in gateway query alert collector job admin agent; do
    echo "  - $svc"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn,direct \
        go build -ldflags="-s -w" -o "$OUTPUT_DIR/bin/$svc" "./cmd/$svc" 2>/dev/null || \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 GOPROXY=https://goproxy.cn,direct \
        go build -o "$OUTPUT_DIR/bin/$svc" "./cmd/$svc"
done
echo "  Done."

# ──────────────────────────────────────
# 2. Build frontend
# ──────────────────────────────────────
echo "[2/6] Building frontend..."
cd "$PROJECT_ROOT/web"
npm run build 2>/dev/null
cp -r dist/* "$OUTPUT_DIR/web/"
echo "  Done."

# ──────────────────────────────────────
# 3. Copy Python AI services
# ──────────────────────────────────────
echo "[3/6] Copying AI services..."
cd "$PROJECT_ROOT"
cp -r ai/* "$OUTPUT_DIR/ai/"
# Pre-download Python wheels if possible
if command -v pip3 &> /dev/null; then
    echo "  Downloading Python wheels..."
    pip3 download -d "$OUTPUT_DIR/ai/wheels" \
        flask kafka-python pyyaml numpy scipy 2>/dev/null || \
    echo "  Warning: Could not download wheels (pip download failed)"
fi
echo "  Done."

# ──────────────────────────────────────
# 4. Copy configs and SQL
# ──────────────────────────────────────
echo "[4/6] Copying configs and SQL..."
cp -r "$PROJECT_ROOT/configs/"* "$OUTPUT_DIR/config/"
cp -r "$PROJECT_ROOT/deploy/sql/"* "$OUTPUT_DIR/sql/"
cp -r "$PROJECT_ROOT/deploy/clickhouse/"* "$OUTPUT_DIR/sql/"
cp -r "$PROJECT_ROOT/deploy/nebula/"* "$OUTPUT_DIR/sql/"
echo "  Done."

# ──────────────────────────────────────
# 5. Copy Docker assets
# ──────────────────────────────────────
echo "[5/6] Copying Docker assets..."
cp "$PROJECT_ROOT/deploy/docker-compose/docker-compose.yml" "$OUTPUT_DIR/docker/"
cp "$PROJECT_ROOT/deploy/docker-compose/Dockerfile.go" "$OUTPUT_DIR/docker/"
cp "$PROJECT_ROOT/deploy/docker-compose/Dockerfile.python" "$OUTPUT_DIR/docker/"
[ -f "$PROJECT_ROOT/deploy/docker-compose/nginx.conf" ] && \
    cp "$PROJECT_ROOT/deploy/docker-compose/nginx.conf" "$OUTPUT_DIR/docker/"
[ -f "$PROJECT_ROOT/deploy/docker-compose/vmagent.yml" ] && \
    cp "$PROJECT_ROOT/deploy/docker-compose/vmagent.yml" "$OUTPUT_DIR/docker/"

# Save Docker images list for offline pull
cat > "$OUTPUT_DIR/docker/images.txt" <<'EOF'
victoriametrics/victoria-metrics:v1.106.1
victoriametrics/vmagent:v1.106.1
clickhouse/clickhouse-server:24.10
bitnami/zookeeper:3.9
bitnami/kafka:3.8
prom/node-exporter:v1.8.2
nginx:1.27
vesoft/nebula-metad:v3.8.0
vesoft/nebula-storaged:v3.8.0
vesoft/nebula-graphd:v3.8.0
EOF
echo "  Done."

# ──────────────────────────────────────
# 6. Create install script
# ──────────────────────────────────────
echo "[6/6] Creating install script..."
cat > "$OUTPUT_DIR/install.sh" <<'INSTALL_EOF'
#!/bin/bash
# AIOps Platform - Offline Installer
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="${AIOPS_HOME:-/opt/aiops}"

echo "=== AIOps Offline Installer ==="
echo "Install directory: $INSTALL_DIR"
echo ""

# Check prerequisites
command -v docker &> /dev/null || { echo "Error: docker not found"; exit 1; }
command -v docker compose &> /dev/null || { echo "Error: docker compose not found"; exit 1; }

# Create directories
mkdir -p "$INSTALL_DIR"/{bin,web,ai,config,sql}

# Copy binaries
echo "[1/4] Installing Go services..."
cp "$SCRIPT_DIR/bin/"* "$INSTALL_DIR/bin/"
chmod +x "$INSTALL_DIR/bin/"*
echo "  Done."

# Copy frontend
echo "[2/4] Installing frontend..."
cp -r "$SCRIPT_DIR/web/"* "$INSTALL_DIR/web/"
echo "  Done."

# Copy AI services
echo "[3/4] Installing AI services..."
cp -r "$SCRIPT_DIR/ai/"* "$INSTALL_DIR/ai/"
# Install Python deps from local wheels if available
if [ -d "$SCRIPT_DIR/ai/wheels" ]; then
    pip3 install --no-index --find-links="$SCRIPT_DIR/ai/wheels" \
        flask kafka-python pyyaml 2>/dev/null || \
    echo "  Warning: Python wheel install failed, install manually"
fi
echo "  Done."

# Copy configs
echo "[4/4] Installing configs..."
cp -r "$SCRIPT_DIR/config/"* "$INSTALL_DIR/config/"
cp -r "$SCRIPT_DIR/sql/"* "$INSTALL_DIR/sql/"
echo "  Done."

# Load Docker images (if tarballs exist)
if ls "$SCRIPT_DIR/docker/"*.tar &> /dev/null; then
    echo "Loading Docker images..."
    for img in "$SCRIPT_DIR/docker/"*.tar; do
        docker load -i "$img"
    done
    echo "  Done."
fi

# Create systemd service files
cat > /etc/systemd/system/aiops-gateway.service <<EOF
[Unit]
Description=AIOps Gateway
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/bin/gateway -config $INSTALL_DIR/config/gateway.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit configs in $INSTALL_DIR/config/"
echo "  2. Initialize databases: cd $INSTALL_DIR && ./bin/admin -config config/admin.yaml"
echo "  3. Start services:"
echo "     - Docker:  cd $SCRIPT_DIR/docker && docker compose up -d"
echo "     - Systemd: systemctl start aiops-gateway"
echo "  4. Open http://localhost:3000"
echo ""
INSTALL_EOF

chmod +x "$OUTPUT_DIR/install.sh"
echo "  Done."

# ──────────────────────────────────────
# Package
# ──────────────────────────────────────
echo ""
echo "=== Creating archive..."
cd "$(dirname "$OUTPUT_DIR")"
tar czf "aiops-offline-${VERSION}.tar.gz" "$(basename "$OUTPUT_DIR")"
echo "Package: $(pwd)/aiops-offline-${VERSION}.tar.gz"
echo ""
echo "=== Build complete ==="

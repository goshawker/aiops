#!/bin/bash
# AIOps Agent Installer
# Usage (online):  curl -sSL http://<collector>:8084/install.sh | bash -s -- --collector http://<collector>:8084
# Usage (offline): bash install.sh --local-bin ./aiops-agent --collector http://<collector>:8084
set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────
COLLECTOR_URL="http://localhost:8084"
DOWNLOAD_BASE_URL=""  # Default: derived from COLLECTOR_URL host, port 3000
AGENT_NAME=""
TAGS="{}"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/aiops"
DATA_DIR="/var/lib/aiops-agent"
VERSION="latest"
LOCAL_BIN=""  # offline mode: path to pre-downloaded agent binary

# ── Parse args ────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --collector) COLLECTOR_URL="$2"; shift 2 ;;
    --download-url) DOWNLOAD_BASE_URL="$2"; shift 2 ;;
    --name) AGENT_NAME="$2"; shift 2 ;;
    --tags) TAGS="$2"; shift 2 ;;
    --version) VERSION="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --local-bin) LOCAL_BIN="$2"; shift 2 ;;  # offline install
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [ -z "$AGENT_NAME" ]; then
  AGENT_NAME="$(hostname)"
fi

# Derive DOWNLOAD_BASE_URL from COLLECTOR_URL if not explicitly set
# Replaces port 8084 with 3000 (Nginx serves binaries)
if [ -z "$DOWNLOAD_BASE_URL" ]; then
  DOWNLOAD_BASE_URL="$(echo "$COLLECTOR_URL" | sed 's|:[0-9]\+|:3000|')"
fi

# ── Detect arch ───────────────────────────────────────────────
ARCH="$(uname -m)"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

case "$ARCH" in
  x86_64|amd64)  BIN_ARCH="amd64" ;;
  aarch64|arm64) BIN_ARCH="arm64" ;;
  loongarch64)   BIN_ARCH="loong64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="agent-${OS}-${BIN_ARCH}"
if [ "$VERSION" != "latest" ]; then
  BINARY="agent-${OS}-${BIN_ARCH}-${VERSION}"
fi

echo "========================================"
echo " AIOps Agent Installer"
echo "========================================"
echo "  OS:          $OS"
echo "  Arch:        $ARCH"
echo "  Collector:   $COLLECTOR_URL"
echo "  Agent Name:  $AGENT_NAME"
echo "  Install Dir: $INSTALL_DIR"
echo "========================================"

# ── Create directories ─────────────────────────────────────
mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR"

# ── Install agent binary ──────────────────────────────────
if [ -n "$LOCAL_BIN" ]; then
  # Offline mode: use pre-downloaded binary
  if [ ! -f "$LOCAL_BIN" ]; then
    echo "ERROR: local binary not found: $LOCAL_BIN"
    exit 1
  fi
  cp "$LOCAL_BIN" "${INSTALL_DIR}/aiops-agent"
  echo "  -> copied from $LOCAL_BIN"
else
  # Online mode: download from collector service
  DOWNLOAD_URL="${DOWNLOAD_BASE_URL}/agent-bin/${BINARY}"
  echo "Downloading agent from $DOWNLOAD_URL ..."
  if command -v curl &>/dev/null; then
    curl -sSL "$DOWNLOAD_URL" -o "${INSTALL_DIR}/aiops-agent"
  elif command -v wget &>/dev/null; then
    wget -q "$DOWNLOAD_URL" -O "${INSTALL_DIR}/aiops-agent"
  else
    echo "ERROR: curl or wget required"
    exit 1
  fi
  echo "  -> downloaded from $DOWNLOAD_URL"

  # Validate the downloaded file is a binary (not HTML/error page)
  if head -c 4 "${INSTALL_DIR}/aiops-agent" 2>/dev/null | grep -q $'^\x7fELF'; then
    echo "  -> valid ELF binary"
  elif head -c 2 "${INSTALL_DIR}/aiops-agent" 2>/dev/null | grep -q $'^MZ'; then
    echo "  -> valid PE binary (Windows)"
  else
    echo "  -> WARNING: downloaded file does not appear to be a valid binary, retrying with amd64..."
    BINARY_ALT="agent-${OS}-$(echo "$ARCH" | sed 's/aarch64/amd64/;s/arm64/amd64/;s/x86_64/arm64/')"
    DOWNLOAD_URL_ALT="${DOWNLOAD_BASE_URL}/agent-bin/${BINARY_ALT}"
    if command -v curl &>/dev/null; then
      curl -sSL "$DOWNLOAD_URL_ALT" -o "${INSTALL_DIR}/aiops-agent"
    elif command -v wget &>/dev/null; then
      wget -q "$DOWNLOAD_URL_ALT" -O "${INSTALL_DIR}/aiops-agent"
    fi
    echo "  -> retried with $DOWNLOAD_URL_ALT"
  fi
fi

chmod +x "${INSTALL_DIR}/aiops-agent"
echo "  installed: ${INSTALL_DIR}/aiops-agent"

# ── Write config ──────────────────────────────────────────
cat > "${CONFIG_DIR}/agent.yaml" << EOF
collector_url: "${COLLECTOR_URL}"
agent_name: "${AGENT_NAME}"
interval: 30s
tags: '${TAGS}'
EOF
echo "  -> ${CONFIG_DIR}/agent.yaml"

# ── Create systemd service ────────────────────────────────
if command -v systemctl &>/dev/null; then
  cat > /etc/systemd/system/aiops-agent.service << EOF
[Unit]
Description=AIOps Agent
Documentation=https://github.com/aiops/agent
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/aiops-agent --collector ${COLLECTOR_URL} --name ${AGENT_NAME} --tags '${TAGS}'
Restart=always
RestartSec=10
StartLimitInterval=300
StartLimitBurst=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable aiops-agent
  systemctl restart aiops-agent
  echo "  -> systemd service installed and started"

  # Show status
  sleep 2
  systemctl status aiops-agent --no-pager
fi

echo ""
echo "========================================"
echo " Agent installed successfully!"
echo " Name:    $AGENT_NAME"
echo " Service: aiops-agent"
echo " Config:  ${CONFIG_DIR}/agent.yaml"
echo " Logs:    journalctl -u aiops-agent -f"
echo "========================================"

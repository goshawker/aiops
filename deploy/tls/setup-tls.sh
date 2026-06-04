#!/bin/bash
# VigilOps TLS Certificate Setup
# Supports: Let's Encrypt (domain) / Self-signed (IP-only)
#
# Usage:
#   bash deploy/tls/setup-tls.sh --domain aiops.example.com
#   bash deploy/tls/setup-tls.sh --ip 192.168.1.100
#   bash deploy/tls/setup-tls.sh --self-signed

set -euo pipefail

CERT_DIR="$(cd "$(dirname "$0")" && pwd)/certs"
DOMAIN=""
IP=""
SELF_SIGNED=false
EMAIL="admin@$(hostname -f 2>/dev/null || echo 'localhost')"

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain) DOMAIN="$2"; shift 2 ;;
    --ip) IP="$2"; shift 2 ;;
    --self-signed) SELF_SIGNED=true; shift ;;
    --email) EMAIL="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

mkdir -p "$CERT_DIR"

echo "========================================"
echo " VigilOps TLS Certificate Setup"
echo "========================================"

# ── Option 1: Let's Encrypt (with domain) ──────────────
if [ -n "$DOMAIN" ]; then
  echo "Mode: Let's Encrypt (domain: $DOMAIN)"
  echo ""

  # Check if certbot is installed
  if ! command -v certbot &>/dev/null; then
    echo "Installing certbot..."
    if command -v apt-get &>/dev/null; then
      apt-get update && apt-get install -y certbot python3-certbot-nginx
    elif command -v yum &>/dev/null; then
      yum install -y certbot python3-certbot-nginx
    elif command -v brew &>/dev/null; then
      brew install certbot
    else
      echo "ERROR: Cannot install certbot. Please install manually."
      exit 1
    fi
  fi

  # Stop nginx temporarily for standalone verification
  echo "Stopping nginx for certificate verification..."
  docker compose -f "$(dirname "$0")/../docker-compose/docker-compose.yml" stop web 2>/dev/null || true

  # Request certificate
  echo "Requesting Let's Encrypt certificate..."
  certbot certonly --standalone \
    -d "$DOMAIN" \
    --email "$EMAIL" \
    --agree-tos \
    --non-interactive \
    --cert-path "$CERT_DIR/server.crt" \
    --key-path "$CERT_DIR/server.key" \
    --fullchain-path "$CERT_DIR/fullchain.pem"

  # Copy certificates
  cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "$CERT_DIR/server.crt"
  cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" "$CERT_DIR/server.key"

  # Setup auto-renewal cron
  echo "Setting up auto-renewal..."
  cat > /etc/cron.d/aiops-certbot-renew << 'EOF'
0 3 * * 1 root certbot renew --quiet --deploy-hook "docker compose -f /opt/aiops/deploy/docker-compose/docker-compose.yml restart web"
EOF

  echo ""
  echo "Let's Encrypt certificate installed!"
  echo "Auto-renewal configured (weekly check, every Monday 3:00 AM)"

# ── Option 2: Self-signed (IP or no domain) ────────────
else
  if [ -n "$IP" ]; then
    echo "Mode: Self-signed (IP: $IP)"
    SUBJECT="/CN=$IP"
    SAN="subjectAltName=IP:$IP,IP:127.0.0.1,DNS:localhost"
  else
    echo "Mode: Self-signed (generic)"
    SUBJECT="/CN=$(hostname)"
    SAN="subjectAltName=DNS:$(hostname),DNS:localhost,IP:127.0.0.1"
  fi

  # Generate self-signed certificate (valid 10 years)
  echo "Generating self-signed certificate..."
  openssl req -x509 -nodes -days 3650 \
    -newkey rsa:2048 \
    -keyout "$CERT_DIR/server.key" \
    -out "$CERT_DIR/server.crt" \
    -subj "$SUBJECT" \
    -addext "$SAN" \
    2>/dev/null

  # Generate DH parameters for perfect forward secrecy
  echo "Generating DH parameters (this may take a minute)..."
  openssl dhparam -out "$CERT_DIR/dhparam.pem" 2048 2>/dev/null

  echo ""
  echo "Self-signed certificate installed!"
  echo "NOTE: Browsers will show a security warning for self-signed certificates."
  echo "      For production, use a domain name with Let's Encrypt."
fi

# Set permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"
[ -f "$CERT_DIR/dhparam.pem" ] && chmod 644 "$CERT_DIR/dhparam.pem"

echo ""
echo "========================================"
echo " Certificate files:"
echo "   Cert: $CERT_DIR/server.crt"
echo "   Key:  $CERT_DIR/server.key"
[ -f "$CERT_DIR/dhparam.pem" ] && echo "   DH:   $CERT_DIR/dhparam.pem"
echo ""
echo " Next steps:"
echo "   1. Restart nginx: docker compose restart web"
echo "   2. Access: https://${DOMAIN:-$IP:-localhost}"
echo "========================================"

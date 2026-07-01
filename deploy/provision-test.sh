#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# One-box test deployment of the Xray panel on a clean Ubuntu 22.04 server.
#
# Topology (single host):
#   - Native Xray (systemd service `xray`) — the node the panel manages.
#   - Panel backend + frontend + Caddy (self-signed TLS) via Docker Compose,
#     using host networking so the panel can SSH to this same box and so the
#     subscription links use the real public IP.
#
# The panel is served on https://<IP>:8443 (self-signed → browser warning).
# Xray REALITY inbounds use 443. You create the server/inbound/client in the UI.
#
# Run as root, FROM THE REPO ROOT:
#   sudo bash deploy/provision-test.sh
#
# Idempotent: safe to re-run. Prints the panel URL and admin credentials at the
# end. This is a TEST setup — tear it down and rotate the root password after.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

if [[ $EUID -ne 0 ]]; then echo "Run as root (sudo)." >&2; exit 1; fi
if [[ ! -f backend/cmd/panel/main.go ]]; then
  echo "Run from the repository root (backend/, frontend/ must be here)." >&2
  exit 1
fi

# Public IP the panel and clients use (auto-detected, override with PUBLIC_IP=…).
PUBLIC_IP="${PUBLIC_IP:-$(curl -fsS4 https://api.ipify.org || true)}"
if [[ -z "$PUBLIC_IP" ]]; then echo "Could not detect public IP; set PUBLIC_IP=…" >&2; exit 1; fi
PANEL_PORT="${PANEL_PORT:-8443}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-$(openssl rand -base64 12 | tr -d '/+=' | cut -c1-16)}"

echo "==> Public IP: $PUBLIC_IP   Panel port: $PANEL_PORT"

# ── 1. Base packages ────────────────────────────────────────────────────────
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq curl ca-certificates openssl git >/dev/null

# ── 1b. Swap (guard against OOM during the Go build on small VPSes) ──────────
# The pure-Go SQLite driver compiles with a large memory footprint; on a 1–2 GB
# box the build can trigger the OOM killer (which may take out sshd). Ensure at
# least ~2 GB of swap exists.
if [[ "$(free -m | awk '/^Swap:/{print $2}')" -lt 1024 ]]; then
  if [[ ! -f /swapfile ]]; then
    echo "==> Adding 2G swap to avoid OOM during build…"
    fallocate -l 2G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=2048
    chmod 600 /swapfile
    mkswap /swapfile >/dev/null
  fi
  swapon /swapfile 2>/dev/null || true
  grep -q '/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

# ── 2. Docker (+ compose plugin) ────────────────────────────────────────────
if ! command -v docker >/dev/null 2>&1; then
  echo "==> Installing Docker…"
  curl -fsSL https://get.docker.com | sh
fi
docker compose version >/dev/null 2>&1 || { echo "docker compose plugin missing" >&2; exit 1; }

# ── 3. Native Xray (the managed node) ───────────────────────────────────────
if ! command -v xray >/dev/null 2>&1; then
  echo "==> Installing Xray-core…"
  bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install
fi
# Minimal valid starting config so the service is active before the first sync.
install -d /usr/local/etc/xray
if [[ ! -s /usr/local/etc/xray/config.json ]]; then
  cat > /usr/local/etc/xray/config.json <<'JSON'
{ "log": {"loglevel":"warning"}, "inbounds": [], "outbounds": [{"protocol":"freedom","tag":"direct"}] }
JSON
fi
systemctl enable --now xray >/dev/null 2>&1 || true
systemctl restart xray || true

# ── 4. Panel secrets (.env) ─────────────────────────────────────────────────
if [[ ! -f .env ]]; then
  echo "==> Generating .env (secrets)…"
  cat > .env <<EOF
PANEL_ENCRYPTION_KEY=$(openssl rand -base64 32)
PANEL_JWT_SECRET=$(openssl rand -base64 48)
PANEL_ADMIN_USERNAME=${ADMIN_USER}
PANEL_ADMIN_PASSWORD=${ADMIN_PASS}
PANEL_DOMAIN=${PUBLIC_IP}:${PANEL_PORT}
DB_PATH=/data/panel.db
PANEL_HTTP_ADDR=127.0.0.1:8080
LOG_LEVEL=info
EOF
  chmod 600 .env
else
  echo "==> .env already exists — keeping it."
  ADMIN_USER=$(grep -E '^PANEL_ADMIN_USERNAME=' .env | cut -d= -f2-)
  ADMIN_PASS='(unchanged — see existing .env)'
fi

# ── 5. Caddy config: self-signed TLS on :PANEL_PORT ─────────────────────────
cat > deploy/Caddyfile.test <<EOF
{
    # Self-signed internal CA (no ACME, no domain needed).
    auto_https disable_redirects
}
https://${PUBLIC_IP}:${PANEL_PORT} {
    tls internal
    encode gzip
    @backend path /api/* /sub/* /swagger/* /healthz /readyz
    handle @backend {
        reverse_proxy 127.0.0.1:8080
    }
    handle {
        root * /srv/www
        try_files {path} /index.html
        file_server
    }
}
EOF

# ── 6. Test compose (host networking) ───────────────────────────────────────
cat > docker-compose.test.yml <<EOF
services:
  backend:
    build: ./backend
    env_file: .env
    network_mode: host
    volumes:
      - panel-data:/data
    restart: unless-stopped

  frontend:
    build: ./frontend
    command: sh -c "cp -r /usr/share/nginx/html/* /srv/www/ && echo 'frontend published'"
    volumes:
      - web-root:/srv/www

  caddy:
    image: caddy:2-alpine
    network_mode: host
    depends_on: [backend, frontend]
    volumes:
      - ./deploy/Caddyfile.test:/etc/caddy/Caddyfile:ro
      - web-root:/srv/www:ro
      - caddy-data:/data
      - caddy-config:/config
    restart: unless-stopped

volumes:
  panel-data:
  web-root:
  caddy-data:
  caddy-config:
EOF

# ── 7. Build & start ────────────────────────────────────────────────────────
echo "==> Building and starting the panel (first build takes a few minutes)…"
docker compose -f docker-compose.test.yml up -d --build

echo
echo "──────────────────────────────────────────────────────────────────────"
echo " Panel URL : https://${PUBLIC_IP}:${PANEL_PORT}   (self-signed → accept the browser warning)"
echo " Login     : ${ADMIN_USER}"
echo " Password  : ${ADMIN_PASS}"
echo "──────────────────────────────────────────────────────────────────────"
echo
echo "Next, in the panel UI:"
echo "  1) Servers → Add server:"
echo "       host=${PUBLIC_IP}  ssh_user=root  auth=password  secret=<root password>"
echo "       (xray service name: xray, config path: /usr/local/etc/xray/config.json)"
echo "  2) Open the server → Check (should go online), then Add inbound (VLESS Reality, port 443)."
echo "  3) Clients → Add client → open it → grant the inbound → Save access."
echo "     The panel syncs config.json to Xray automatically (xray -test + restart)."
echo "  4) Copy the subscription link / QR and connect."
echo
echo "Make sure your cloud firewall allows inbound TCP ${PANEL_PORT} and 443."
echo "This is a TEST deployment: tear down with"
echo "  docker compose -f docker-compose.test.yml down -v"
echo "and ROTATE the root password afterwards."

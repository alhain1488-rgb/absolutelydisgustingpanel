#!/usr/bin/env bash
# Smoke test for a running panel backend (SPEC §12).
#
# Exercises the full happy path against a live backend using only its HTTP API:
#   healthz -> login -> create server (mock) -> create inbound -> create client
#   -> grant inbound -> GET /sub/{token} returns a non-empty valid subscription.
#
# It uses a mock server entry (no real SSH), so operations that require a live
# node (check/restart/stats/sync) are not exercised here.
#
# Usage:
#   BASE_URL=http://localhost:8080 \
#   ADMIN_USER=admin ADMIN_PASS=secret \
#   ./scripts/smoke.sh
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-}"

if [[ -z "$ADMIN_PASS" ]]; then
  echo "ADMIN_PASS is required" >&2
  exit 1
fi

json() { python3 -c 'import sys,json;print(json.load(sys.stdin)['"$1"'])'; }

echo "==> healthz"
code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/healthz")
[[ "$code" == "200" ]] || { echo "healthz failed: $code"; exit 1; }

echo "==> login"
TOKEN=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" | json '"token"')
[[ -n "$TOKEN" && "$TOKEN" != "None" ]] || { echo "login failed (2FA enabled?)"; exit 1; }
AUTH=(-H "Authorization: Bearer $TOKEN")

echo "==> create server (mock)"
SID=$(curl -s "${AUTH[@]}" -X POST "$BASE_URL/api/servers" \
  -d '{"name":"smoke","host":"smoke.example.com","ssh_user":"root","ssh_auth_method":"password","ssh_secret":"x"}' \
  | json '"id"')
echo "    server id=$SID"

echo "==> create inbound (VLESS Reality)"
IID=$(curl -s "${AUTH[@]}" -X POST "$BASE_URL/api/servers/$SID/inbounds" \
  -d '{"tag":"smoke-reality","protocol":"vless","port":443,"settings_json":{"flow":"xtls-rprx-vision"},"stream_settings_json":{"network":"tcp","security":"reality","reality":{"dest":"www.microsoft.com:443","server_names":["www.microsoft.com"]}}}' \
  | json '"id"')
echo "    inbound id=$IID"

echo "==> create client"
CLIENT=$(curl -s "${AUTH[@]}" -X POST "$BASE_URL/api/clients" -d '{"name":"smoke-user"}')
CID=$(echo "$CLIENT" | json '"id"')
SUBTOK=$(echo "$CLIENT" | json '"subscription_token"')
echo "    client id=$CID"

echo "==> grant inbound"
curl -s "${AUTH[@]}" -X PUT "$BASE_URL/api/clients/$CID/inbounds" -d "{\"inbound_ids\":[$IID]}" >/dev/null

echo "==> fetch subscription"
SUB=$(curl -s "$BASE_URL/sub/$SUBTOK")
DECODED=$(echo "$SUB" | base64 -d)
if [[ -z "$DECODED" ]]; then
  echo "subscription is empty" >&2
  exit 1
fi
echo "$DECODED" | grep -q '^vless://' || { echo "subscription missing vless URI: $DECODED"; exit 1; }

echo
echo "SMOKE OK — subscription:"
echo "$DECODED"

#!/usr/bin/env bash

set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_info() {
  printf "%b[INFO]%b %s\n" "$GREEN" "$NC" "$1"
}

print_warn() {
  printf "%b[WARN]%b %s\n" "$YELLOW" "$NC" "$1"
}

print_error() {
  printf "%b[ERROR]%b %s\n" "$RED" "$NC" "$1"
}

SSH_TARGET="${SSH_TARGET:-codex-web3}"
REMOTE_BASE="${REMOTE_BASE:-/home/forge/codexsecure.web3connected.com}"
REMOTE_PATH="${REMOTE_BASE}/current"
COMPOSE_FILE="docker-compose.prod.yml"

print_info "CodexSecure API deployment starting"
print_info "SSH target: ${SSH_TARGET}"
print_info "Remote path: ${REMOTE_PATH}"

if [[ ! -f "Dockerfile" ]]; then
  print_error "Run this script from Docker/codexsecure_api"
  exit 1
fi

if [[ ! -f "${COMPOSE_FILE}" ]]; then
  print_error "Missing ${COMPOSE_FILE}"
  exit 1
fi

print_info "Creating remote deployment directory"
ssh "${SSH_TARGET}" "mkdir -p '${REMOTE_PATH}'"

print_info "Syncing CodexSecure API files"
rsync -az --delete \
  --exclude '.git/' \
  --exclude 'vendor/' \
  --exclude 'bin/' \
  --exclude 'logs/' \
  --exclude '.env' \
  ./ "${SSH_TARGET}:${REMOTE_PATH}/"

print_info "Syncing codex-zone-mcp sidecar"
ssh "${SSH_TARGET}" "mkdir -p '${REMOTE_PATH}/../codex_zone_mcp'"
rsync -az --delete \
  --exclude '.git/' \
  --exclude 'node_modules/' \
  --exclude 'src/' \
  ../codex_identity/codex_zone_mcp/ "${SSH_TARGET}:${REMOTE_PATH}/../codex_zone_mcp/"

print_info "Starting CodexSecure API container on CodexWeb3"
ssh "${SSH_TARGET}" <<'ENDSSH'
set -euo pipefail
REMOTE_PATH="/home/forge/codexsecure.web3connected.com/current"
cd "${REMOTE_PATH}"

if [[ ! -f .env ]]; then
  if [[ -f .env.example ]]; then
    cp .env.example .env
    echo "[WARN] Remote .env did not exist; copied .env.example. Update secrets before live traffic."
  else
    echo "[ERROR] No remote .env and no .env.example available."
    exit 1
  fi
fi

# Ensure codex-net network exists
docker network create codex-net 2>/dev/null || true

docker compose -f docker-compose.prod.yml build codexsecure-api
docker compose -f docker-compose.prod.yml up -d

echo
echo "--- CONTAINER STATUS ---"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "codexsecure|NAMES" || true

echo
echo "--- HEALTH CHECK ---"
sleep 3
curl -sS http://127.0.0.1:8084/health || true
echo
ENDSSH

print_info "CodexSecure API deployment finished"
print_warn "Ensure Nginx is configured to proxy codexsecure.web3connected.com -> 127.0.0.1:8084"

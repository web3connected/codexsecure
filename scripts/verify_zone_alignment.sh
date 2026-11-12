#!/usr/bin/env bash
# Verify that a running CodexSecure API's zone registry aligns with an expected registry.
#
# Usage:
#   scripts/verify_zone_alignment.sh [registry.json]
#
# Environment:
#   ALIGNMENT_BASE_URL   API base URL (default http://localhost:8084).
#                        Through the gateway use e.g. http://localhost:8090/secure
#   CODEX_API_KEY        Sent as X-API-Key (required by the gateway)
#
# Exit codes: 0 aligned, 1 mismatch, 2 API unavailable or request rejected.
set -euo pipefail

BASE_URL="${ALIGNMENT_BASE_URL:-http://localhost:8084}"
ENDPOINT="${BASE_URL%/}/api/v1/policy/alignment"
REGISTRY_FILE="${1:-$(cd "$(dirname "$0")/.." && pwd)/testdata/zones/canonical_registry.json}"

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required to run the zone alignment check" >&2
  exit 2
fi

auth_args=()
if [[ -n "${CODEX_API_KEY:-}" ]]; then
  auth_args=(-H "X-API-Key: ${CODEX_API_KEY}")
fi

printf 'Zone alignment against %s\n' "$ENDPOINT"
printf 'Expected registry: %s\n\n' "$REGISTRY_FILE"

body_file="$(mktemp)"
trap 'rm -f "$body_file"' EXIT

if ! http_code="$(curl -sS -m 10 -o "$body_file" -w '%{http_code}' -X POST "$ENDPOINT" \
  -H 'Content-Type: application/json' "${auth_args[@]}" --data @"$REGISTRY_FILE")"; then
  echo "UNAVAILABLE: could not reach ${ENDPOINT}" >&2
  exit 2
fi

if [[ "$http_code" != "200" ]]; then
  echo "UNAVAILABLE: HTTP ${http_code}" >&2
  cat "$body_file" >&2
  echo >&2
  exit 2
fi

# The gateway wraps upstream responses as {status, service, data}.
result="$(jq 'if has("data") and (.data | type == "object") and (.data | has("aligned")) then .data else . end' "$body_file")"
echo "$result" | jq .

if [[ "$(echo "$result" | jq -r '.aligned')" == "true" ]]; then
  echo "ALIGNED"
  exit 0
fi

echo "MISMATCH"
exit 1

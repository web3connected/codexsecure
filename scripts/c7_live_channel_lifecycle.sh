#!/usr/bin/env bash
# C7 live lifecycle via the gateway. Reads the application key from Laravel's .env; never prints it.
# Usage: scripts/c7_live_channel_lifecycle.sh <output file>   (local development only; needs the running gateway and Go)
set -uo pipefail
GW=http://127.0.0.1:8090/secure/api/v1/sites
SITE=${SITE:-web3codex-local}
KEY=$(grep -E '^CODEX_GATEWAY_API_KEY=' /mnt/c/laragon/www/web3codex/.env | head -1 | cut -d= -f2- | tr -d '"\r')
OUT=$1; : > "$OUT"
call() { # label method path [body]
  local body=${4:-}
  local args=(-s -o "$OUT.body" -w '%{http_code}' -X "$2" -H "X-API-Key: $KEY")
  [ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
  local code; code=$(curl "${args[@]}" "$GW/$3")
  printf '%s %s %s -> %s %s\n' "$1" "$2" "$3" "$code" "$(cat "$OUT.body")" >> "$OUT"
}
call open POST "$SITE/channels" '{"zone":"Z7"}'
ID=$(tail -1 "$OUT" | grep -oE '"channel":"[0-9a-f-]{36}"' | head -1 | cut -d'"' -f4)
echo "$ID" > "$OUT.id"
call get_pending GET "$SITE/channels/$ID"
call activate POST "$SITE/channels/$ID/activate"
call get_active GET "$SITE/channels/$ID"
call close POST "$SITE/channels/$ID/close"
call get_closed GET "$SITE/channels/$ID"
rm -f "$OUT.body"

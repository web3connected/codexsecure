#!/usr/bin/env bash
# C10 live renewal via the gateway (local development only). Reads the keys from the
# .env files and never prints them. Usage: scripts/c10_live_channel_renewal.sh <output file>
set -uo pipefail
GW=http://127.0.0.1:8090/secure/api/v1/sites
GO=http://127.0.0.1:8084/api/v1/sites
SITE=web3codex-local
KEY=$(grep -E '^CODEX_GATEWAY_API_KEY=' /mnt/c/laragon/www/web3codex/.env | head -1 | cut -d= -f2- | tr -d '"\r')
SVC=$(grep -E '^SERVICE_API_KEY=' /home/web3codex/projects/Docker/CodexWeb3/codexsecure_api/.env | cut -d= -f2-)
OUT=$1; : > "$OUT"; T=$(mktemp)
req() { # label method url [curl args...] ; prints the body to $T
  local label=$1 method=$2 url=$3; shift 3
  local code; code=$(curl -s -o "$T" -w '%{http_code}' -X "$method" "$@" "$url")
  printf '%-40s %s -> %s %s\n' "$label" "$method" "$code" "$(head -c 900 "$T")" >> "$OUT"
}
field() { python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); c=(d.get("data") or {}).get("channel") or {}; print(json.dumps(c.get(sys.argv[2])) if sys.argv[2]=="protection" else c.get(sys.argv[2]))' "$T" "$1"; }
sign() { docker exec -e SITE="$1" -e M="$2" -e P="$3" codex-gateway node -e '
  const g=require("/app/dist/auth/gatewayAssertion.js"); const k=g.loadAssertionKey(process.env.GATEWAY_ASSERTION_KID,process.env.GATEWAY_ASSERTION_PRIVATE_KEY);
  process.stdout.write(g.issueAssertion(k,{appId:"web3connected-demo",site:process.env.SITE,method:process.env.M,path:process.env.P,requestId:require("crypto").randomUUID()}))'; }
A=(-H "X-API-Key: $KEY")
req "1 open"                        POST "$GW/$SITE/channels" "${A[@]}" -H 'Content-Type: application/json' -d '{"zone":"Z7"}'; ID=$(field channel)
req "2 activate"                    POST "$GW/$SITE/channels/$ID/activate" "${A[@]}"; E0=$(field expires_at); L0=$(field last_activity_at); P0=$(field protection)
sleep 2
req "3 get (status check)"          GET  "$GW/$SITE/channels/$ID" "${A[@]}"; E1=$(field expires_at)
sleep 2
req "4 renew"                       POST "$GW/$SITE/channels/$ID/renew" "${A[@]}"; E2=$(field expires_at); L2=$(field last_activity_at); P2=$(field protection); ID2=$(field channel)
req "5 get after renew"             GET  "$GW/$SITE/channels/$ID" "${A[@]}"; E3=$(field expires_at)
req "6 renew, body field refused"   POST "$GW/$SITE/channels/$ID/renew" "${A[@]}" -H 'Content-Type: application/json' -d '{"expires_at":"2030-01-01T00:00:00Z"}'
req "7 renew, invalid app key"      POST "$GW/$SITE/channels/$ID/renew" -H 'X-API-Key: w3c_app_invalidinvalidinvalidinvalid00000000000'
req "8 renew, unauthorized site"    POST "$GW/other-site/channels/$ID/renew" "${A[@]}"
req "9 renew, unknown channel"      POST "$GW/$SITE/channels/$(cat /proc/sys/kernel/random/uuid)/renew" "${A[@]}"
req "10 renew, invalid identifier"  POST "$GW/$SITE/channels/not-a-uuid/renew" "${A[@]}"
P="/api/v1/sites/$SITE/channels/$ID"
req "11 go: renew, no assertion"    POST "$GO/$SITE/channels/$ID/renew" -H "X-Service-Key: $SVC"
S=$(sign $SITE POST "$P/activate")
req "12 go: activate assertion on renew" POST "$GO/$SITE/channels/$ID/renew" -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $S"
S=$(sign other-site POST "/api/v1/sites/other-site/channels/$ID/renew")
req "13 go: other-site grant, own path" POST "$GO/other-site/channels/$ID/renew" -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $S"
S=$(sign $SITE POST "$P/renew")
req "14 go: valid (control)"        POST "$GO/$SITE/channels/$ID/renew" -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $S"; E4=$(field expires_at)
req "15 go: replayed"               POST "$GO/$SITE/channels/$ID/renew" -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $S"
S=$(sign $SITE POST "$P/renew"); sleep 37
req "16 go: expired assertion"      POST "$GO/$SITE/channels/$ID/renew" -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $S"
req "17 get after refusals"         GET  "$GW/$SITE/channels/$ID" "${A[@]}"; E5=$(field expires_at)
req "18 open second (pending)"      POST "$GW/$SITE/channels" "${A[@]}" -H 'Content-Type: application/json' -d '{"zone":"Z1"}'; PID=$(field channel); PE=$(field expires_at)
req "19 renew pending"              POST "$GW/$SITE/channels/$PID/renew" "${A[@]}"; PE2=$(field expires_at)
req "20 close"                      POST "$GW/$SITE/channels/$ID/close" "${A[@]}"; EC=$(field expires_at)
req "21 renew closed"               POST "$GW/$SITE/channels/$ID/renew" "${A[@]}"; EC2=$(field expires_at)
req "22 close pending"              POST "$GW/$SITE/channels/$PID/close" "${A[@]}"
{
  echo "--- summary"
  echo "channel=$ID renewed_channel=$ID2 pending=$PID"
  echo "activate: last_activity=$L0 expires=$E0"
  echo "get (status check) expires=$E1  (unchanged: $([ "$E1" = "$E0" ] && echo yes || echo NO))"
  echo "renew:    last_activity=$L2 expires=$E2  (extended: $([[ "$E2" > "$E0" ]] && echo yes || echo NO))"
  echo "protection unchanged by renew: $([ "$P0" = "$P2" ] && echo yes || echo NO)"
  echo "get after renew expires=$E3 (= renew: $([ "$E3" = "$E2" ] && echo yes || echo NO))"
  echo "go valid renew expires=$E4; after refusals expires=$E5 (= last valid renew: $([ "$E5" = "$E4" ] && echo yes || echo NO))"
  echo "pending: expires before=$PE after refused renew=$PE2 (unchanged: $([ "$PE" = "$PE2" ] && echo yes || echo NO))"
  echo "closed: expires at close=$EC after refused renew=$EC2 (unchanged: $([ "$EC" = "$EC2" ] && echo yes || echo NO))"
} >> "$OUT"
rm -f "$T"

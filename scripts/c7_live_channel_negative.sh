#!/usr/bin/env bash
# C7 negative tests. Secrets are read into variables and never printed.
# Usage: scripts/c7_live_channel_negative.sh <closed channel uuid> <output file>   (local development only)
set -uo pipefail
GW=http://127.0.0.1:8090/secure/api/v1/sites
GO=http://127.0.0.1:8084/api/v1/sites
KEY=$(grep -E '^CODEX_GATEWAY_API_KEY=' /mnt/c/laragon/www/web3codex/.env | head -1 | cut -d= -f2- | tr -d '"\r')
SVC=$(grep -E '^SERVICE_API_KEY=' /home/web3codex/projects/Docker/CodexWeb3/codexsecure_api/.env | cut -d= -f2-)
CLOSED=$1; OUT=$2; : > "$OUT"
T=$(mktemp)
req() { # label url method [curl args...]
  local label=$1 url=$2 method=$3; shift 3
  local code; code=$(curl -s -o "$T" -w '%{http_code}' -X "$method" "$@" "$url")
  printf '%-44s %s -> %s %s\n' "$label" "$method" "$code" "$(head -c 260 "$T")" >> "$OUT"
}
# A gateway-signed assertion from inside the gateway container (its own C4 signer and key).
sign() { # app site method path
  docker exec -e A="$1" -e SITE="$2" -e M="$3" -e P="$4" codex-gateway node -e '
    const g=require("/app/dist/auth/gatewayAssertion.js");
    const k=g.loadAssertionKey(process.env.GATEWAY_ASSERTION_KID,process.env.GATEWAY_ASSERTION_PRIVATE_KEY);
    process.stdout.write(g.issueAssertion(k,{appId:process.env.A,site:process.env.SITE,method:process.env.M,path:process.env.P,requestId:require("crypto").randomUUID()}))'
}
P="web3codex-local/channels/$CLOSED"
# Gateway: application authentication and site authorization
req "gw: no application key"                 "$GW/$P" GET
req "gw: invalid application key"            "$GW/$P" GET -H "X-API-Key: w3c_app_invalidinvalidinvalidinvalid00000000000"
req "gw: unauthorized site"                  "$GW/other-site/channels" POST -H "X-API-Key: $KEY" -H 'Content-Type: application/json' -d '{"zone":"Z1"}'
req "gw: cross-site read of channel"         "$GW/other-site/channels/$CLOSED" GET -H "X-API-Key: $KEY"
req "gw: forged client assertion+service key" "$GW/$P" GET -H "X-API-Key: $KEY" -H 'X-CodexSecure-Gateway-Assertion: a.b.c' -H 'X-Service-Key: forged'
req "gw: reactivate closed channel"          "$GW/$P/activate" POST -H "X-API-Key: $KEY"
req "gw: close closed channel"               "$GW/$P/close" POST -H "X-API-Key: $KEY"
req "gw: unknown channel"                    "$GW/web3codex-local/channels/$(cat /proc/sys/kernel/random/uuid)" GET -H "X-API-Key: $KEY"
req "gw: invalid channel identifier"         "$GW/web3codex-local/channels/not-a-uuid" GET -H "X-API-Key: $KEY"
req "gw: refused body field"                 "$GW/web3codex-local/channels" POST -H "X-API-Key: $KEY" -H 'Content-Type: application/json' -d '{"zone":"Z1","codexhash":"x"}'
# Go directly: the gateway assertion is required in addition to the service key
req "go: service key, no assertion"          "$GO/$P" GET -H "X-Service-Key: $SVC"
req "go: service key, malformed assertion"   "$GO/$P" GET -H "X-Service-Key: $SVC" -H 'X-CodexSecure-Gateway-Assertion: a.b.c'
A=$(sign web3connected-demo web3codex-local GET "/api/v1/sites/$P")
req "go: valid assertion, no service key"    "$GO/$P" GET -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo web3codex-local GET "/api/v1/sites/$P")
TAMPER="${A%?}"; [ "${A: -1}" = A ] && TAMPER+=Q || TAMPER+=A
req "go: tampered signature"                 "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $TAMPER"
req "go: valid assertion (control)"          "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
req "go: replayed assertion"                 "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo web3codex-local POST "/api/v1/sites/$P")
req "go: assertion for another method"       "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo web3codex-local GET "/api/v1/sites/$P/close")
req "go: assertion for another path"         "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo web3codex-local GET "/api/v1/sites/other-site/channels/$CLOSED")
req "go: web3codex-local assertion, other path" "$GO/other-site/channels/$CLOSED" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo other-site GET "/api/v1/sites/other-site/channels/$CLOSED")
req "go: other-site grant, its own path"     "$GO/other-site/channels/$CLOSED" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo other-site GET "/api/v1/sites/$P")
req "go: other-site grant on web3codex path" "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
A=$(sign web3connected-demo web3codex-local GET "/api/v1/sites/$P")
sleep 37
req "go: expired assertion (37 s)"           "$GO/$P" GET -H "X-Service-Key: $SVC" -H "X-CodexSecure-Gateway-Assertion: $A"
rm -f "$T"

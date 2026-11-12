# Communication Channel C6: Go Gateway Assertion Verifier

**Date:** 2026-10-30
**Side:** CodexSecure Go API (`codexsecure_api`, port 8084)
**Status:** The implementation is complete and tested in isolation. **Nothing is deployed:** no container was rebuilt or restarted, no `.env` was changed and no credential was generated or rotated. The live lifecycle `Laravel → Gateway → Go → codex_zones` waits on the approvals in §8.
**Backup before the change:** `~/codex_backups/codexsecure_api-pre-C6-20261030-104356.tgz`

The Go API now verifies the C4 gateway's signed Ed25519 assertion and installs the `CommunicationPrincipal` that `ChannelAuth` already required. It implements the existing C4 contract exactly, with no new field, header or protocol. `ChannelAuth`, the handlers, the manager and the channel lifecycle are unchanged.

## 1. Files

### Reviewed (read-only)

| Area | Files |
| --- | --- |
| Go channels | `internal/handlers/channels.go`, `internal/middleware/channel_auth.go`, `communication_auth.go`, `service_key.go`, `debug.go` (`RequestIDMiddleware`), `internal/server/routes/api.go`, `internal/channels/*`, `internal/config/config.go`, the existing channel unit and integration tests, `scripts/test_communication_contract.sh` |
| Gateway (C4) | `src/auth/gatewayAssertion.ts`, `src/auth/applicationSites.ts`, `src/auth/applicationKeys.ts`, `src/channels.ts`, `src/response.ts` (`attachRequestId`), `dist/auth/gatewayAssertion.js` (checked to match `src`), `test/channels.test.js`, `CHANNEL_ROUTES.md`, `APPLICATION_KEYS.md`, `.env.example`, `docker-compose.yml` |
| Web3Codex | `docs/codexsecure/COMMUNICATION_CHANNEL_C4_IMPLEMENTATION.md` (§3.3 Go requirement), `COMMUNICATION_CHANNEL_C5_IMPLEMENTATION.md`, `COMMUNICATION_CHANNEL_C5B_SETUP.md` (§7) |
| Deployment | the root `docker-compose.yml`, `codexsecure_api/docker-compose.yml`, the running containers' environment (variable **names** only) |

### Changed (all in `codexsecure_api`)

| File | Change |
| --- | --- |
| `internal/middleware/gateway_assertion.go` | **New.** `GatewayVerifier`, `GatewayAssertion()`, `ParseGatewayAssertionKeys` |
| `internal/handlers/channels.go` | The channel group is now `GatewayAssertion(), ChannelAuth()`. Only `RegisterRoutes` changed: 1 line plus a comment |
| `internal/middleware/gateway_assertion_test.go` | **New.** Verifier unit tests, including the real-signer fixture |
| `internal/middleware/testdata/c4_gateway_assertions.json` | **New.** Assertions produced by the real C4 signer (§5.1) |
| `internal/handlers/channels_gateway_test.go` | **New.** Route tests through `RegisterRoutes` |
| `internal/handlers/channels_gateway_integration_test.go` | **New.** Signed lifecycle against disposable PostgreSQL (`integration` tag) |
| `docs/codexsecure/COMMUNICATION_CHANNEL_C6_GO_VERIFIER.md` | This report |

Not changed: Web3Codex, the gateway, migrations and schemas, CodexHash, CodexTime, `ChannelAuth`, `CommunicationAuth`, the channel manager and handlers, and the non-channel routes.

## 2. The C4 assertion contract as implemented

The source is `codex_gateway/src/auth/gatewayAssertion.ts` (`issueAssertion`).

- **Header:** `X-CodexSecure-Gateway-Assertion`, exactly once.
- **Format:** a compact JWS with three base64url segments, without padding (strict decoding).
- **Protected header:** `{"alg":"EdDSA","kid":<kid>,"typ":"codexsecure-gateway+jwt"}`
- **Claims:** `{"app","aud":"codexsecure-api","channel_sites":[<site>],"exp","iat","iss":"codex-gateway","method","path","rid","sub":"application:<app>"}`
- **Signature:** Ed25519 over the ASCII bytes `<header segment>.<claims segment>`, exactly as received. Go never re-serializes before verifying.

The Go checks, in order (C4 §3.3 and C5B §7):

| # | Check | Rejected when |
| --- | --- | --- |
| 1 | Size and shape | over 4 KiB, not 3 segments, padded or non-base64url segment |
| 2 | Header | not a JSON object with only `alg`, `kid` and `typ`; `alg` ≠ `EdDSA`; `typ` ≠ `codexsecure-gateway+jwt` |
| 3 | Key | `kid` not configured |
| 4 | Signature | not 64 bytes or `ed25519.Verify` fails (any modified header, claims or signature) |
| 5 | Claims | not a JSON object of exactly the 10 C4 members (unknown members are refused, so no read, admin or discovery grant can be smuggled in) |
| 6 | `iss` and `aud` | ≠ `codex-gateway` / `codexsecure-api` |
| 7 | Time | `iat` or `exp` missing or not an integer; `exp < iat`; `exp − iat > 60 s`; `iat > now + 5 s`; `exp < now − 5 s` |
| 8 | Application | `app` does not match the gateway's `APP_ID_PATTERN` (`^[a-z0-9][a-z0-9-]{1,62}$`), or `sub` ≠ `application:` + `app` |
| 9 | Method | `method` ≠ the request method (exact, case-sensitive) |
| 10 | Path | `path` ≠ `c.Request.URL.EscapedPath()`. The query is not bound, as the gateway drops it. Alternative encodings such as `%2D` do not match |
| 11 | Site | `channel_sites` is not exactly one entry, the gateway's site pattern, equal to `c.Param("site")` |
| 12 | Replay | `rid` empty, non-printable or over 128 characters, or already accepted within its lifetime |

**Expiration contract.** The gateway issues `exp = iat + 30`. Go applies the C4 Go requirement: a 5 s clock skew either side and at most 60 s of lifetime. A gateway assertion is therefore accepted from `iat − 5 s` to `iat + 35 s`. The fixture test pins this boundary.

**Replay (optional in C4, implemented).** Accepted `rid`s are held in memory until `exp + 5 s` and then pruned. The gateway issues a fresh UUIDv4 per request (`attachRequestId`), so legitimate traffic never repeats one. The cache is per process: with several Go replicas, an assertion could be replayed once against another replica within its lifetime, still only for the same method, path and site.

## 3. Authentication and site authorization

Order on the channel group: `GatewayAssertion` → `ChannelAuth` → handler.

- **The verifier never answers a request.** On any failure it installs nothing, logs a warning (reason, path, remote address, never the token) and continues. `ChannelAuth` then answers with its existing contract, so no new error code reaches the gateway:
  - missing or wrong `X-Service-Key` (constant-time, unchanged) → `401 SERVICE_KEY_INVALID`, **even with a valid assertion**;
  - no verified assertion → `403 ADMIN_UNAUTHORIZED` "trusted gateway identity is required".
- **On success** it calls `SetCommunicationPrincipal(c, {GatewayAuthenticated: true, Actor: sub, ChannelSites: {<site>: true}})`. `<site>` is `c.Param("site")`, which the assertion must name exactly. Read, admin and discovery grants are never set. `SetCommunicationPrincipal` already freezes the maps.
- **Site isolation.** The only channel grant is the asserted site, and the assertion is bound to the exact path, so a client cannot choose another site. An assertion for `web3codex-local` presented on `/sites/other-site/...` fails the path and site checks, and `ChannelAuth` returns 403. A channel UUID of site A requested with a valid assertion for site B reaches the manager as `(B, uuid)` and returns `404 CHANNEL_NOT_FOUND`, as before. Both cases are tested against PostgreSQL.
- **Scope.** The verifier runs only on `/api/v1/sites/:site/channels/*`. On the communication routes the header is ignored and `CommunicationAuth` still refuses (tested).
- **Application identity.** Go accepts any application the gateway signs for, provided `app` and `sub` agree. Site authority stays where C4 placed it (gateway `GATEWAY_APP_SITES`). Go has no second application list, so `web3connected-demo` needs no Go configuration. The running gateway's `GATEWAY_APP_KEYS` holds exactly one application, `web3connected-demo`, which matches the intended `GATEWAY_APP_SITES=web3connected-demo:web3codex-local`.

## 4. Configuration

Go, in `codexsecure_api/.env`, which the container already loads through `env_file`:

```env
# Already required by ChannelAuth; equal to the gateway's CODEXSECURE_SERVICE_KEY.
SERVICE_API_KEY=<shared service key>
# Gateway public key(s): comma-separated <kid>:<x>, where x is the "x" member of
# the gateway's public Ed25519 JWK (base64url, 32 bytes). List the old and new
# kid during a rotation.
CODEXSECURE_GATEWAY_ASSERTION_KEYS=gateway-2026-10:<x>
```

- Only the public key is configured in Go, so a leak on the Go side cannot mint gateway identities.
- `CODEXSECURE_GATEWAY_ASSERTION_KEYS` is read once, when the routes are registered (the convention `ChannelAuth` uses for `SERVICE_API_KEY`). If it is missing or invalid, Go logs a warning at startup and refuses every channel request (`403`). On success it logs the loaded `kid`s.
- `x` is exactly what the gateway's key-generation command in `CHANNEL_ROUTES.md` prints (`publicKey.export({format:'jwk'}).x`).

The matching gateway configuration, in `codex_gateway/.env`, is unchanged here:

```env
GATEWAY_APP_SITES=web3connected-demo:web3codex-local
CODEXSECURE_SERVICE_KEY=<same value as Go SERVICE_API_KEY>
GATEWAY_ASSERTION_KID=gateway-2026-10
GATEWAY_ASSERTION_PRIVATE_KEY=<base64 PKCS#8 DER Ed25519 private key>
```

### Current development state (inspected, not changed)

| Item | State |
| --- | --- |
| `codexsecure-api` container | Image `codexsecure-api:latest`, built 2026-10-25, **before C2B**, so it has no channel routes. No `SERVICE_API_KEY` and no `CODEXSECURE_GATEWAY_ASSERTION_KEYS` in its environment or in `codexsecure_api/.env` |
| `codex-gateway` container | Image of 2026-10-11, **before C4**. `GATEWAY_APP_KEYS=web3connected-demo:<digest>` only. No `GATEWAY_APP_SITES`, `CODEXSECURE_SERVICE_KEY` or `GATEWAY_ASSERTION_*` in the environment or in `codex_gateway/.env` |
| Gateway assertion key pair | **Does not exist yet.** This would be the first issuance, not a rotation |
| `SERVICE_API_KEY` | **Does not exist yet** on either side |

## 5. Test results

All tests ran on 2026-10-30 with Go 1.27.3 and `GOFLAGS=-buildvcs=false`.

| Command | Result |
| --- | --- |
| `go vet ./internal/middleware ./internal/handlers` | clean |
| `go build ./...` | ok |
| `go test -count=1 -race ./...` | all packages `ok` |
| `go test -count=1 -v -run 'GatewayAssertion\|ChannelGateway\|ParseGateway'` | 77 new tests and subtests pass |
| `scripts/test_communication_contract.sh` on a fresh disposable PG16 cluster (127.0.0.1:55439, `-race -tags integration`) | `communications`, `channels` and `handlers` `ok`; 339 PASS, 0 SKIP (338 before the new integration test) |

A mutation check confirmed that the tests detect each binding: disabling the signature, method, path or site check made 7–8 tests fail. The source was restored afterwards and compared byte for byte.

### 5.1 Real C4 signer fixture

`testdata/c4_gateway_assertions.json` holds Open, Get, Activate and Close assertions for `web3connected-demo` / `web3codex-local`. They were produced by the gateway's own `issueAssertion` (`codex_gateway/dist/auth/gatewayAssertion.js`, confirmed to match `src`) with a throwaway key whose private half was discarded. Only the public `x`, the `kid` (`gw-test-c6`) and the tokens are stored. Go verifies them byte for byte, so Go and the gateway agree on the encoding, not only on a Go re-implementation. The generator script was a one-off and is not kept. The JSON's `generator` field describes it.

### 5.2 Coverage against the C6 test list

| Requirement | Test |
| --- | --- |
| Valid signed Open, Get, Activate and Close | `TestGatewayAssertionVerifiesRealC4Signer` (real signer); `TestChannelGatewayAssertionAccepted` (route); `TestChannelGatewayAssertionPostgresIntegration` (database) |
| Correct application and site context | `TestGatewayAssertionMiddleware`: Actor `application:web3connected-demo`, `ChannelSites == {web3codex-local}`, no other grants; the manager receives the asserted site |
| Missing assertion | route and middleware tests → 403 |
| Invalid signature | fixture (flipped signature) and route test |
| Expired assertion | fixture window (−6 s / +36 s rejected, −5 s / +35 s accepted), route test, future `iat` |
| Modified site | re-signed for another site; claims tampered without re-signing; `two sites`, `no sites` |
| Modified application identity | `app` tampered (signature fails); `sub` ≠ `app`; invalid `app` pattern; `administrator:` subject |
| Modified method | route test (all 4 operations), unit `method` and `lowercase method` |
| Modified path | route test, fixture (`activate` → `close`, `%2D` encoding, another operation's path), path with query |
| Untrusted gateway key | another key under the same `kid`; unknown `kid`; no keys configured; invalid key configuration |
| Missing or invalid service key | valid assertion with a missing or wrong `X-Service-Key` → `401 SERVICE_KEY_INVALID`, manager not called |
| Cross-site channel access | route tests (assertion on another site's path) and PostgreSQL (site B's assertion → 404 for site A's channel; site A's assertion on site B's path → 403) |
| Also covered | `alg`/`typ`/`iss`/`aud`, extra header or claim members, fractional `iat`, lifetime over 60 s, malformed segments, oversized token, duplicate header, replayed `rid`, key parsing, header ignored on communication routes |

## 6. Compatibility with the existing channel handlers

- `ChannelAuth`, `ChannelHandler` (Open/Get/Activate/Close), `channels.Manager`, the error mapping and the response envelopes are unchanged. All pre-existing channel unit and integration tests pass without modification. They still inject a principal directly, which keeps working because the verifier installs nothing when no assertion is present.
- The lifecycle is unchanged and verified through the verifier against PostgreSQL. Open returns `201` `pending` with `not_protected` and does not activate. Get does not renew. Activate goes through the real CodexHash and CodexTime clients (stub servers) to `active` and `protected`. Close returns `closed` with `closed_at`. Activating a closed channel returns `409 CHANNEL_CLOSED`.
- The gateway's response handling (`projectResult` and `projectError` in `src/channels.ts`) needs no change. Go still returns the same envelopes, and a refused assertion is still a 401 or 403 that the gateway maps to `502 UPSTREAM_ERROR`.
- Non-channel routes are unchanged (`TestGatewayAssertionIgnoredOutsideChannels`, full suite).

## 7. Integration results and remaining blockers

The live `Laravel → Gateway → Go → codex_zones` test was **not run**. Every remaining step needs approval (§8). Blockers, in order:

1. **The Go container predates C2B.** It has no channel routes, so it must be rebuilt.
2. **The gateway container predates C4.** It must be rebuilt with the C4 source and configured (`GATEWAY_APP_SITES`, service key, assertion key).
3. **No credentials exist yet.** Both the assertion key pair and `SERVICE_API_KEY` must be generated. This is the first issuance, not a rotation of an existing development credential.
4. **The Laravel flag.** `CODEX_SECURE_CHANNELS_API` is still unset on the Web3Codex side. Enabling it is a Web3Codex step, outside this phase.
5. **Known CodexTime finding (unchanged, not redesigned).** `CurrentTIUExact` rejects `cached:true`, and the legacy `get_current_tiu` has a 1 s Redis cache that every reader fills, including the gateway's public `GET /time/api/time/current-tiu`. Activation can therefore return `503 CODEXTIME_UNAVAILABLE` under concurrent TIU reads. If the live test hits this, it is a CodexTime-owner decision (C4 §8 options), not a verifier defect.
6. **Side effect of setting `SERVICE_API_KEY`.** The legacy `ServiceKeyMiddleware` on `/api/v1/zones` (`GET ""`, `POST /detect`, `GET /:code`) is a no-op while the key is unset, and it starts enforcing `X-Service-Key` once the key is set. No caller of these routes was found in Web3Codex, the gateway or the CodexWeb3 services, but any unlisted client would start receiving `401 SERVICE_KEY_INVALID`. `/zones/health`, policy and the other routes are unaffected.

## 8. Changes requiring approval

None of these has been done. They are proposed in this order, and each runs only when explicitly approved.

1. **Generate the gateway assertion key pair** using the command in `CHANNEL_ROUTES.md`, without printing the private key:
   - `codex_gateway/.env`: `GATEWAY_ASSERTION_KID=gateway-2026-10` and `GATEWAY_ASSERTION_PRIVATE_KEY=<base64 PKCS#8>`;
   - `codexsecure_api/.env`: `CODEXSECURE_GATEWAY_ASSERTION_KEYS=gateway-2026-10:<x>`.
2. **Generate the shared service key** (32 random bytes, without printing it):
   - `codexsecure_api/.env`: `SERVICE_API_KEY=<key>`;
   - `codex_gateway/.env`: `CODEXSECURE_SERVICE_KEY=<same key>`.
   - Accept the `/zones` side effect (§7.6).
3. **Grant the site** in `codex_gateway/.env`: `GATEWAY_APP_SITES=web3connected-demo:web3codex-local`.
4. **Back up first:** copy both `.env` files to `~/codex_backups/2026-10-30/`, then `docker tag codexsecure-api:latest codexsecure-api:pre-C6-20261030` and `docker tag codexweb3-codex-gateway:latest codexweb3-codex-gateway:pre-C4-20261030`.
5. **Rebuild and recreate only these two services**, from `Docker/CodexWeb3`:
   - `docker compose up -d --build --no-deps codexsecure-api`
   - `docker compose up -d --build --no-deps codex-gateway`
   - Then confirm `/health` on both, and check that the Go log shows `gateway assertion keys loaded [gateway-2026-10]` and the gateway log shows no `Channel routes are not configured`.
6. **Run the lifecycle through the gateway** (Open → Get pending → Activate → Get active → Close → Get closed) on `web3codex-local`. This can be a direct gateway call with the application key, or Laravel's `CodexSecureChannelGatewayLiveTest` once the Web3Codex side sets `CODEX_SECURE_CHANNELS_API=true`.

**Rollback:** restore the two `.env` backups, retag the `pre-*` images to `latest`, then `docker compose up -d --no-deps --no-build codexsecure-api codex-gateway`. To roll back the source, restore `~/codex_backups/codexsecure_api-pre-C6-20261030-104356.tgz`.

Stopped after the Go verifier, tests and report. The next phase has not been started, and Web3Codex and the gateway were not modified.

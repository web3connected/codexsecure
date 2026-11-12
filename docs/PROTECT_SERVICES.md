# Protect Services — CodexHash and TIU

**Date:** 2026-10-11 (CodexHash made stateless the same day)
**Status:** Deployed 2026-10-11 and verified end to end through the gateway with the Laravel demo application.

Stage 4 (Protect) of the Laravel SDK requests two independent values for one communication: a CodexHash and a TIU. CodexSecure serves both by delegating to the existing services — it does not implement a second hash or time algorithm, run another Sevenfold Process, or record the communication itself.

| Operation | Endpoint (API) | Through gateway | Backed by |
|---|---|---|---|
| CodexHash | `POST /api/v1/protect/codexhash` | `POST /secure/api/v1/protect/codexhash` | codexhash-api `POST /api/hash/payload` (stateless, SHA-512, `hash_size` 512) via `codexhash.Client.PayloadHash` |
| TIU | `POST /api/v1/protect/tiu` | `POST /secure/api/v1/protect/tiu` | codextime-api `GET /api/time/current-tiu` via `codextime.Client.CurrentTIU` |

## Authentication

Laravel calls the gateway with `X-API-Key` (`CODEX_GATEWAY_API_KEY`, a Web3Codex application key — see `codex_gateway/APPLICATION_KEYS.md`); the gateway authenticates the application and forwards to the API. The Go routes add no authentication of their own — the same convention as `/api/v1/policy/*`. Port 8084 is published on `127.0.0.1` only, so the internal path is reachable without a key from the host machine but not from other machines.

## Request

```json
{ "payload": { "communication_id": "…", "request": { "method": "GET", "path": "/" }, "…": "…" } }
```

- `payload` must be a non-empty JSON object; body limit 1 MiB.
- Top-level `codexhash` and `tiu` are rejected (`422 reserved_field`): neither value may be part of the hash input. The SDK keeps returned security values outside `data`, so its payload never contains them.
- Duplicate object keys are rejected (`422 duplicate_key`).
- Both endpoints validate the same envelope. The TIU does not depend on the payload contents.

## CodexHash input representation — `codexsecure-protect-payload-v1`

CodexHash hashes the canonical form of `payload` (not the raw request bytes and not Laravel's `submitted_json`):

1. Object keys sorted by byte order at every depth; array order preserved.
2. Number tokens kept exactly as written in the request.
3. Compact JSON, no insignificant whitespace.
4. Strings re-encoded without escaping `/`, `&`, `<`, `>` or non-ASCII; U+2028/U+2029 escaped.
   Equivalent: Python `json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False)`; PHP `json_encode` of a recursively `ksort`ed array with `JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE` (number formatting aside).
5. CodexSecure sends this string as `data` to codexhash-api `POST /api/hash/payload` with `algo: "sha512"`, `hash_size: 512`.
6. `codexhash` = codexhash-api's `payload_hash` = lowercase hex SHA-512 of the UTF-8 canonical string (128 characters).

Verification can therefore recompute the value from the payload alone.

`POST /api/hash/payload` is stateless: it computes the payload hash with the same function (`compute_payload_hash`) that the chaining endpoint `POST /api/hash/` uses, but creates no event, reads no previous event and writes nothing to `codexhash_events`. The chaining endpoint is unchanged and remains available for workflows that need a chained, persisted event. (codexhash-api's service-wide usage tracker still attempts a best-effort `request_logs` row for every request, as it does for `/health`; that is unrelated to hashing and does not affect the result.)

Test vector: `testdata/protect/laravel_get_payload.json` → SHA-512 `7d225e21ff80847f…8f4d628b0` (computed independently in Python; see `internal/protect/payload_test.go`).

CodexSecure fails closed: if codexhash-api is unavailable, rejects the request or returns an unexpected algorithm, no value is returned. The legacy local SHA-256 fallback in `codexhash.Client.Hash` is not used.

## TIU

`tiu` is CodexTime's current Time Index Unit: seconds since the CodexTime epoch (`2020-01-01 00:00:00` UTC), a float with microsecond precision, returned as a JSON number exactly as CodexTime reports it. `GET /api/time/current-tiu` has no side effects; no synchronization or stamp event is created (CodexTime's `POST /api/time/stamp` was not used because it also records a payload digest in CodexHash). Each request reads the TIU once; there is no idempotency store, so repeated requests return the then-current TIU. CodexSecure does not persist the TIU or its association with the communication — the SDK keeps both values in its communication envelope.

The CodexTime client fails open with `tiu: 0` / `source: "unavailable"`; the endpoint treats that, any client error and any non-positive TIU as `503 tiu_unavailable`.

## Responses

The API returns a bare object; the gateway wraps it as `{status, service, tiu, request_id, data}`, so the SDK reads `data.codexhash` / `data.tiu`.

```json
{ "codexhash": "<128 hex>", "algorithm": "sha512", "representation": "codexsecure-protect-payload-v1" }
```

`codexhash_event_id` was removed when hashing became stateless: no event exists, so none is returned.

```json
{ "tiu": 213913067.109267, "timestamp": "2026-10-11T20:17:47.109278Z", "epoch": "2020-01-01 00:00:00", "source": "codextime" }
```

| Status | `error` | Cause |
|---|---|---|
| 400 | `invalid_request` | Body not JSON, not an object, or `payload` absent |
| 413 | `payload_too_large` | Body over 1 MiB |
| 422 | `invalid_payload` / `reserved_field` / `duplicate_key` | Payload not a non-empty object, contains top-level `codexhash`/`tiu`, or duplicate keys |
| 502 | `invalid_service_response` | CodexHash returned an unexpected algorithm or hash |
| 503 | `codexhash_unavailable` / `tiu_unavailable` | Backing service unavailable or failed |

Error bodies never echo payload contents or internal addresses. Logs record the operation and status only.

## Laravel SDK configuration

```env
CODEX_GATEWAY_URL=http://localhost:8090
CODEX_GATEWAY_API_KEY=<provisioned tenant key>
CODEX_SECURE_PROTECT_HASH_ENDPOINT=/secure/api/v1/protect/codexhash
CODEX_SECURE_PROTECT_TIU_ENDPOINT=/secure/api/v1/protect/tiu
```

## Deployment notes (2026-10-11)

- codexhash-api `POST /api/hash/payload` is live (bind-mounted source, auto-reload). Rollback: restore `~/codex_backups/2026-10-11/codexhash_live_backup_20261011/hash_tracking.py` and remove `tests/test_stateless_payload_hash.py`.
- codexsecure-api rollback image: `codexsecure-api:pre-protect-20261011`.
- Remaining exposure: codextime-api still listens on `0.0.0.0:50002` (VS Code holds `127.0.0.1:50002` on Windows, so a loopback-only binding could not be published); codexhash-api and codextime-api routes have no authentication, and Laravel's CodexAuth package calls both directly on localhost.

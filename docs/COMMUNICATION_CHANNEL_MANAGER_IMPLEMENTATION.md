# Communication Channels — Phase C2B: Go CommunicationChannelManager

Status: **implemented and tested locally, 2026-10-27. Not deployed, not committed.**
Contract: Web3Codex `docs/codexsecure/api/CODEXSECURE_COMMUNICATION_CHANNELS_GO_REQUIREMENTS.md`
(C1 + C2A) and migration `2026_10_27_000000_communication_channels`, both read only.

## 1. Files inspected

- Go API: `internal/communications/{errors,store,postgres,configuration}.go`, `internal/handlers/{communications,protect}.go`,
  `internal/middleware/{communication_auth,service_key}.go`, `internal/server/routes/{api,dependencies}.go`, `cmd/server/main.go`,
  `internal/database/postgres.go`, `pkg/codexhash/client.go`, `pkg/codextime/client.go`, existing integration tests and
  `scripts/test_communication_contract.sh`, `docs/COMMUNICATION_CONFIGURATION_IMPLEMENTATION.md`, `docs/DISCOVERY_COVERAGE_IMPLEMENTATION.md`.
- Web3Codex (read only): the Go requirements document above, the channel migration `.php`, `.up.sql`, `.down.sql`.

## 2. Files created or modified

Created:

| File | Purpose |
| --- | --- |
| `internal/channels/channel.go` | `Channel`, `Protection`, `Result`, result/status constants |
| `internal/channels/manager.go` | `Manager` (Open, Get, Close, Expire, unexported `renew`), response contract validation |
| `internal/channels/errors.go` | Channel-specific `MapError`, `OutcomeError` |
| `internal/channels/manager_test.go` | Contract validation and error mapping unit tests |
| `internal/channels/postgres_integration_test.go` | Manager against the real functions as `codex_zones_runtime` |
| `internal/handlers/channels.go` | `ChannelHandler` and the three routes |
| `internal/handlers/channels_test.go` | Handler, authentication, validation, mapping and route tests (stub manager) |
| `internal/handlers/channels_integration_test.go` | Gin → ChannelAuth → Manager → disposable PostgreSQL |
| `internal/middleware/channel_auth.go` | `ChannelAuth` |
| this report | |

Modified (backups in `~/codex_backups/codexsecure_api_c2b_20261027_182126/`):

| File | Change |
| --- | --- |
| `internal/middleware/communication_auth.go` | `CommunicationPrincipal.ChannelSites` grant, copied like the other grants. `CommunicationAuth` unchanged |
| `internal/server/routes/dependencies.go` | `ChannelHandler` dependency |
| `internal/server/routes/api.go` | Registers channel routes when the handler is present |
| `cmd/server/main.go` | `handlers.NewChannelHandler(channels.NewManager(db.Pool))` |
| `scripts/test_communication_contract.sh` | Applies the unchanged channel migration; tests `./internal/channels` |

No Web3Codex, Laravel, SDK, Seven-Fold or schema file was changed.

## 3. Manager

`internal/channels.Manager` holds the existing `pgxpool.Pool` (the API connects as `codex_zones_runtime`). Every method is
one autocommit `SELECT public.codex_channels_*(…)` statement — its own short transaction, deciding on database time. The
manager contains no table SQL; the runtime role has no table privilege (verified), so it cannot bypass the functions.

| Method | Function | Accepted results | Notes |
| --- | --- | --- | --- |
| `Open(ctx, site, zone)` | `codex_channels_open` | `opened` | No identity, lifetime or time argument exists. Also requires `pending` + `not_protected` back |
| `Get(ctx, site, channel)` | `codex_channels_get` | `found`, `not_found` | Never renews; lazily expires a lapsed channel |
| `Close(ctx, site, channel)` | `codex_channels_close` | `closed`, `already_closed`, `expired`, `not_found` | Not activity |
| `Expire(ctx, site, channel)` | `codex_channels_expire` | `expired`, `already_expired`, `already_closed`, `not_due`, `not_found` | Manager-internal; not over HTTP; no sweeper yet |
| `Renew(ctx, site, channel)` | `codex_channels_renew` | `renewed` (others become errors) | Exported in C10 for `POST …/renew` — see §5 and `docs/codexsecure/COMMUNICATION_CHANNEL_C10_RENEWAL.md` |

Every response is decoded strictly (unknown or duplicate fields fail) and validated before use: result is in the
function's set; `channel` is null exactly for `not_found`; site and channel identity match the request; status is known;
protection is all-or-none and consistent with status (`pending` unprotected, `active` protected); `closed_at` iff closed;
results naming a state carry that state. A violation is an internal error (`500 INTERNAL_ERROR`), never passed to a
client. A nil UUID is rejected before any database call. `tiu` is kept as `json.Number` (exact decimal, no float).

## 4. API routes

Registered on the existing `v1` group (`/api/v1`, exposed by the gateway as `/secure/api/v1`):

| Route | Success | Other results |
| --- | --- | --- |
| `POST /sites/{site}/channels` body exactly `{"zone":"Z1"}` | `201 {"data":{"result":"opened","channel":{…}}}` | `400`/`422 CHANNEL_REQUEST_INVALID`, `413`, `404 SITE_NOT_FOUND` |
| `GET /sites/{site}/channels/{channel}` | `200 {"data":{"result":"found",…}}` (closed/expired reported by `status`) | `404 CHANNEL_NOT_FOUND`, `422 CHANNEL_IDENTIFIER_INVALID` |
| `POST /sites/{site}/channels/{channel}/close` (empty body or `{}`) | `200 {"data":{"result":"closed",…}}` | `409 CHANNEL_CLOSED`/`CHANNEL_EXPIRED` with `data`, `404 CHANNEL_NOT_FOUND`, `422 CHANNEL_IDENTIFIER_INVALID`, `400` for any body field |

- Open rejects any other field (`channel`, `lifetime_seconds`, `expires_at`, …), duplicate keys and non-objects. The zone
  is validated by the database (`CX422`), so Go and schema cannot disagree.
- Channel identifiers must be the canonical 36-character UUID form (either case); nil, braced, URN and unhyphenated forms
  get `422 CHANNEL_IDENTIFIER_INVALID` without a database call (avoids the `22P02` cast failure).
- Error mapping: `CX404` → `404 SITE_NOT_FOUND`; `CX422` → `422 CHANNEL_REQUEST_INVALID`; everything else (including
  `CX409`, `23514`, `42501`, connection errors) → `500 INTERNAL_ERROR` with no database detail. Envelope is the existing
  `{"error","code"}` (`communications.APIError`), plus `data` on 409.
- There is no expire, list, update or delete route (tested: 404). Activate (C3B) and renew (C10) have their own routes.

**Authentication** (`ChannelAuth`): the same constant-time `X-Service-Key` check as `CommunicationAuth` (unset key denies),
plus the trusted gateway `CommunicationPrincipal` with a **`ChannelSites` grant for `{site}`**. No `X-CodexSecure-Actor`
and no `Idempotency-Key` (`CommunicationAuth` requires both on POST, so it could not be reused unchanged). Read, admin and
discovery grants do not authorize channels. Failures: `401 SERVICE_KEY_INVALID`, `403 ADMIN_UNAUTHORIZED` (the existing
code for a missing gateway identity or site grant). As with communications, the gateway does not yet install a principal,
so in any deployment these routes currently **fail closed**.

## 5. Renewal preparation (`codex_channels_renew`)

Inspected behaviour: locks the row; expires a lapsed channel first (deadline counts as passed) and returns `expired`;
returns `pending`/`closed`/`expired` unchanged for non-active channels; for `active` sets `last_activity_at = now()`,
`expires_at = now() + 30 min` unless a concurrent renewal already advanced it (then `renewed` without moving backwards).
The guard trigger independently forbids any other activity change.

Prepared in Go: `Manager.renew` calls it, validates the response, and turns every non-`renewed` outcome into an error
(`pending` → `409 CHANNEL_NOT_ACTIVE`, `closed` → `CHANNEL_CLOSED`, `expired` → `CHANNEL_EXPIRED`, `not_found` → `404`), so a
communication on a non-active channel fails rather than continuing. It is unexported and has no caller: Open, Get, Close,
handlers, heartbeats and polling cannot reach it. The integration phase adds the single caller after a communication passes
the Seven-Fold Process for an approved Communication Object (requirements open decision 4). Its compatibility, concurrency
and refusal paths are tested against an active schema-owner fixture.

## 6. Protection and activation boundary (proposed, not implemented)

Genuine sources already in this API:

- CodexHash `Client.PayloadHash(ctx, data, "sha512", 512)` — stateless, **fails closed**, 128 lowercase hex; matches
  `protection_codexhash ~ '^[0-9a-f]{128}$'`. `Client.Hash` must never be used: it falls back to a local sha256.
- CodexTime `Client.CurrentTIU` — usable only when `Source == "api"` and `TIU > 0`; the client otherwise fails open
  (`Source: "unavailable"`, TIU 0), exactly as `ProtectHandler.IssueTIU` already rejects.
- `protect.CanonicalPayload` for a deterministic binding representation.

Proposed interface (next phase):

```go
// internal/channels
type ChannelProtector interface { // CodexHash + CodexTime, fail closed
    Protect(ctx context.Context, binding Binding) (VerifiedProtection, error)
}
type Binding struct{ Channel uuid.UUID; Site, Zone string; OpenedAt time.Time } // canonicalized before hashing
type VerifiedProtection struct{ CodexHash string; TIU string /* exact decimal */ }

func (m *Manager) Activate(ctx context.Context, site string, channel uuid.UUID) (Result, error)
// 1. Get: must be found, pending, not lapsed.
// 2. Protect(binding from the database's channel row) — any CodexHash/CodexTime failure aborts; channel stays pending.
// 3. One call to a new schema function codex_channels_activate(site, channel, codexhash, tiu) that re-locks the row,
//    re-checks pending/not lapsed, sets protected_at = now() and status = active.
// 4. Verify: recompute PayloadHash over the same canonical binding and compare before reporting success.
```

Required from Web3Codex before this can be built: an activation function (the runtime role has no `UPDATE` privilege, and
no function writes protection today), and decisions on the TIU representation (§8.6) and requirements open decision 3.

## 7. Test results

Disposable PostgreSQL **16.15** cluster, freshly initialized under the session scratchpad at `127.0.0.1:55439`,
database `codexsecure_configuration_test`, bootstrapped by `scripts/test_communication_contract.sh`, which applied the four
earlier migrations and the **unchanged** channel migration from the Web3Codex tree. Tests run as `codex_zones_runtime`
through the same functions the API uses; fixtures only as the schema owner. The cluster was stopped and deleted afterwards.
No live `codex_zones` connection was made.

| Suite | Result |
| --- | --- |
| `go test -race -tags integration ./internal/communications ./internal/channels ./internal/handlers` | **218 PASS, 0 FAIL, 0 SKIP** |
| `go test ./...` (unit, full repository) | all packages `ok` |
| `go vet` (changed packages, with and without `integration` tag) | clean |

Channel coverage:

| Area | Tests |
| --- | --- |
| Open pending | Server-generated identity (distinct per open), `pending`, `not_protected`, `last_activity_at = opened_at`, deadline exactly +30 min |
| Retrieve | `found`; does not move activity or deadline; closed/expired reported by status |
| Close | `closed` + `closed_at`, activity unchanged; repeat → `already_closed`/409; lapsed → `expired`/409 |
| Expiration | Lazy expiry on Get/Close; explicit `Expire` → `expired` then `already_expired`; `not_due` at 0 and 29 min; closed → `already_closed`; lapsed active channel is expired, never renewed |
| Renewal compatibility | Active fixture renews to now + 30 min; 16 concurrent renewals succeed and never move activity back; pending/closed/expired/not-found refused with the mapped codes |
| Site isolation | Another site's channel is `not_found` for get/close/expire/renew and unchanged; HTTP: no grant → 403, grant for site B → 404 for A's channel |
| Invalid identifiers / inputs | Nil UUID rejected before the database; non-canonical forms → 422; zones `''`, `Z0`, `Z13`, `z1`, `Z01` → `CHANNEL_REQUEST_INVALID`; unknown site → `SITE_NOT_FOUND`; extra body fields rejected |
| Authentication | Missing/wrong/unset service key → 401; no principal, unauthenticated principal, wrong-site grant, admin/read/discovery-only grants → 403; none reach the manager |
| Error mapping | `CX404`, `CX422`, `CX409`, `23514`, `42501`, connection errors; no database detail in responses |
| Concurrency | 20 parallel opens → 20 unique identities; 12 parallel closes → exactly 1 `closed`; 12 parallel expiries → exactly 1 `expired`; Close/Expire race on a lapsed channel never closes it; Get/Close/Expire race on a live channel → exactly one close, no expiry |
| Database compatibility | Exact signatures, `jsonb` returns, `SECURITY DEFINER`, runtime `EXECUTE` on the five functions only; runtime refused `SELECT/INSERT/UPDATE/DELETE`, sequence, `codex_channels_locate`; 3-argument `open` (lifetime/identity) does not exist; timeout is `00:30:00`; malformed UUID is `22P02`; raw output decodes strictly |

Rerun (fresh disposable cluster required; the script refuses existing roles/databases):

```sh
COMMUNICATION_TEST_PORT=55439 COMMUNICATION_TEST_USER=codex_test_admin \
WEB3CODEX_REFERENCE_ROOT=/mnt/c/laragon/www/web3codex GOFLAGS=-buildvcs=false \
  scripts/test_communication_contract.sh
```

## 8. Discrepancies and decisions for review

1. **`Renew` visibility.** The requirements sketch an exported `Renew`; it is unexported `renew` with no caller until the
   qualifying-activity path exists, so nothing can call it by accident. Exported when the integration phase adds its caller.
2. **Authentication scope (requirements open decision 1), provisional.** A new `ChannelSites` grant, required for all three
   routes. `CommunicationAuth` could not be reused unchanged (requires actor + `Idempotency-Key`). The gateway must
   eventually supply this grant; until then the routes fail closed. 403s reuse `ADMIN_UNAUTHORIZED`; a dedicated
   `CHANNEL_UNAUTHORIZED` code is an easy change if preferred.
3. **Timestamp format.** The database emits `+00:00` with six fractional digits; Go re-encodes the same instants as
   RFC 3339 UTC (`Z`, trailing zeros trimmed). Equal instants, different text; SDKs must parse RFC 3339, not compare strings.
4. **Unknown site on Get/Close.** The functions return `not_found` (not `CX404`), so these routes report
   `CHANNEL_NOT_FOUND`. Only Open reports `SITE_NOT_FOUND`. This avoids site enumeration; it matches the SQL, but the
   requirements table does not state it.
5. **Close body.** The requirements say "empty body"; `{}` is also accepted for JSON clients; any field is rejected.
6. **TIU representation (blocks activation).** `protection_tiu` is `NUMERIC(20,6) > 0`. The Go CodexTime client returns a
   `float64`, and CodexTime V1 (`codextime-tiu-api/1`) publishes exact decimal strings on a 1e-50 s base, which
   `NUMERIC(20,6)` cannot store without rounding. The column type or the TIU form to store must be decided before activation.
7. **No activation function.** `pending → active` is allowed by the guard but no function performs it, and the runtime
   cannot `UPDATE`. A `codex_channels_activate` function is required (Web3Codex). Correct per C2A ("no channel can become
   active"), recorded here as the next schema dependency.
8. **CodexHash constraint vs fallback.** The column accepts only 128-hex (SHA-512). `codexhash.Client.Hash`'s sha256 fallback
   would be rejected by the constraint, but activation must use `PayloadHash` and never rely on the constraint to catch it.
9. **PostgreSQL version.** C2A was verified on PostgreSQL 17; this run used 16.15 (no 17 available locally). The migration
   uses nothing version-specific (`gen_random_uuid` is core since 13). A PG17 rerun is advisable before deployment.
10. **Existing script guards.** The `: "VAR:?…}"` lines at the top of `scripts/test_communication_contract.sh` are malformed
    (missing `${`), so they do not produce their messages; `set -u` still stops on unset variables. Pre-existing, not changed.

## 9. Remaining work

- Activation with genuine CodexHash/CodexTime protection and verification (§6), after Web3Codex adds the activation
  function and the TIU decision.
- Qualifying-activity renewal from the Seven-Fold Process (open decision 4), the only caller of `renew`.
- Gateway delivery of the trusted principal and `ChannelSites` grants; SDK channel support.
- Optional expiry sweeper (open decision 6), retention (5), maximum lifetime (2).

# Communication Channels — Phase C3B: Protected Channel Activation (Go)

Date: 2026-10-28. Scope: the CodexSecure Go API (`codexsecure_api`) only. Nothing was committed or deployed. No Web3Codex migration, Laravel, SDK, gateway, CodexHash or CodexTime code changed. No persistent database was touched. All PostgreSQL work ran on a disposable PostgreSQL 16 cluster on 127.0.0.1:55439, which was deleted afterwards.

Predecessors: [C2B implementation](COMMUNICATION_CHANNEL_MANAGER_IMPLEMENTATION.md), [C2B→C3A handoff](COMMUNICATION_CHANNEL_C3A_HANDOFF.md), and the Web3Codex C3A report (`/mnt/c/laragon/www/web3codex/docs/codexsecure/architecture/history/CODEXSECURE_COMMUNICATION_CHANNEL_C3A_REPORT.md`).

Backup before editing: `~/codex_backups/codexsecure_api-pre-C3B-20261028-185802.tgz` (this directory is not a Git repository).

## 1. Files inspected

**Go (this repository):**

- `internal/channels/` (`channel.go`, `errors.go`, `manager.go` and both tests);
- `internal/handlers/channels.go` and its unit and integration tests;
- `internal/handlers/protect.go`;
- `internal/middleware/channel_auth.go`, `communication_auth.go` and `tiu.go`;
- `internal/protect/payload.go`;
- `internal/communications/configuration.go` (`DecodeStrict`) and `errors.go` (`APIError`);
- `pkg/codexhash/client.go`, `pkg/codextime/client.go` and their tests;
- `cmd/server/main.go`, `internal/server/routes/`, `docker-compose.yml`;
- `scripts/test_communication_contract.sh`;
- the C2B report and the C3A handoff.

**Web3Codex (read only):**

- `database/migrations/codex_zones/sql/2026_10_28_000000_communication_channel_activation.up.sql`;
- the C2A channel migration (`2026_10_27_000000…up.sql`);
- the `sites` definition (`2026_10_20_235826…up.sql`);
- the `codex_zones` README;
- `docs/codexsecure/api/CODEXSECURE_COMMUNICATION_CHANNELS_GO_REQUIREMENTS.md` (sections C3A and C3B);
- the C3A report.

**Services (read only, wire format only):**

- `codextime_api/src/services/time_service.py` (`get_current_tiu`, the Redis cache);
- `codexhash_api/src/routes/hash_tracking.py` (`POST /api/hash/payload`).

**Contract check.** The implemented C3A SQL matches the requirements document: the function signature, the result set, `CX422` on invalid input, the binding definition, the TIU schemes and the representation of `tiu` as a string. I found no discrepancy that needed a different design. The Go-side choices that differ from the requirements document's sketch are listed in §11.

## 2. Files created or modified

| File | Change |
| --- | --- |
| `pkg/codextime/exact.go` | **New.** `CurrentTIUExact`: a lossless, fail-closed read of the legacy service |
| `pkg/codextime/exact_test.go` | **New.** Wire preservation, invalid and cached responses, fail-closed behaviour |
| `internal/channels/activation.go` | **New.** Binding, `Activate`, CodexHash and TIU validation, the `codex_channels_activate` call |
| `internal/channels/activation_integration_test.go` | **New.** PostgreSQL activation tests with controlled service fixtures |
| `internal/channels/channel.go` | C3A `Protection` (`binding`, string `tiu`, `tiu_scheme`, `tiu_unit`); `activated` / `already_active` / `protection_mismatch` |
| `internal/channels/manager.go` | `NewManager(pool, hasher, clock)`; shared `decode`; validation of all six protection fields and the new result/status pairs |
| `internal/channels/errors.go` | Activation errors and their API codes |
| `internal/channels/manager_test.go` | C3A fixture; golden vectors; TIU, CodexHash and CodexTime validation; error mapping |
| `internal/channels/postgres_integration_test.go` | Active fixtures use a genuine binding hash and a scheme (C3A refuses the old placeholder); compatibility checks for the activation function |
| `internal/handlers/channels.go` | `POST …/channels/{channel}/activate` |
| `internal/handlers/channels_test.go` | Activate in authentication, identifier, outcome, body and error tests; header impersonation |
| `internal/handlers/channels_integration_test.go` | HTTP activation through `ChannelAuth` against PostgreSQL |
| `cmd/server/main.go` | Passes the existing CodexHash client and a CodexTime client to the manager |
| `scripts/test_communication_contract.sh` | Applies the C3A migration unchanged, in its own transaction. Also fixes the three required-variable guards, which were no-ops (`: "VAR:?…"` without `${}`) |

Unchanged: `codexhash.Client`, `codextime.Client.CurrentTIU`, `TIUMiddleware`, `ProtectHandler` and all middleware.

## 3. Manager activation interface

```go
func NewManager(pool *pgxpool.Pool, hasher PayloadHasher, clock TIUSource) *Manager
func (m *Manager) Activate(ctx context.Context, site string, channel uuid.UUID) (Result, error)

type PayloadHasher interface { // codexhash.Client
    PayloadHash(ctx context.Context, data, algo string, hashSize int) (codexhash.PayloadHashResult, error)
}
type TIUSource interface {     // codextime.Client
    CurrentTIUExact(ctx context.Context) (codextime.ExactTIU, error)
}

const BindingRepresentation = "codexsecure-channel-binding-v1"
```

`Activate` works in this order:

1. `Get` the channel within `{site}`. `not_found` is returned as is. An `active`, `closed` or `expired` channel returns `already_active`, `closed` or `expired` without contacting any service. This is a fast path only: the database decides again.
2. Build the binding from that database row only.
3. Call CodexHash, then CodexTime, both outside any transaction. Validate each response.
4. Call `codex_channels_activate` once, as an autocommit statement.
5. Decode strictly, then validate:
   - the result is one of the six allowed;
   - the result matches the channel status (`activated`/`already_active` → active, `protection_mismatch` → pending);
   - the identity and site match;
   - protection is either complete and well formed, or entirely absent.

   For `activated`, the stored hash, TIU, scheme and unit must equal the supplied values byte for byte, `protected_at` must equal `last_activity_at`, and the deadline must be later than both.

| Database result | `Activate` returns | HTTP |
| --- | --- | --- |
| `activated` | result | `200 {"data":{"result":"activated",…}}` |
| `already_active` | result (first protection) | `200 {"data":{"result":"already_active",…}}` |
| `closed` / `expired` | result | `409 CHANNEL_CLOSED` / `CHANNEL_EXPIRED` with `data` |
| `not_found` | result | `404 CHANNEL_NOT_FOUND` |
| `protection_mismatch` | logged (site and channel only), error | `500 INTERNAL_ERROR`, no data, never retried |
| `CX422` | error (Go built the input, so this is internal) | `500 INTERNAL_ERROR` |

`renew` stays unexported, with no caller and no route.

## 4. API route changes

One route, registered in the existing `ChannelAuth` group:

```
POST /api/v1/sites/{site}/channels/{channel}/activate     (gateway: /secure/api/v1/…)
```

- The body must be empty or `{}`. Any field, including `codexhash`, `tiu`, `tiu_scheme` or `site`, is refused with `400 CHANNEL_REQUEST_INVALID` before the manager is called.
- The channel identifier uses the existing canonical-UUID check (`422 CHANNEL_IDENTIFIER_INVALID`).
- Open does **not** activate. The requirements recommend that it should, but the contract does not require it. A separate route is the smaller change, keeps Open's behaviour unchanged, and lets a caller retry activation after a service outage. A channel that is never activated lapses after 30 minutes.

## 5. CodexHash integration

- It uses only the existing `codexhash.Client.PayloadHash(ctx, binding, "sha512", 512)`, which calls `POST /api/hash/payload`. `Hash`/`QuickHash` are not part of the `PayloadHasher` interface, so they cannot be used, and their local SHA-256 fallback is unreachable from activation.
- The **binding** is built from the row returned by `Get`: `{"channel","opened_at","representation","site","zone"}`, with `opened_at` formatted with `2006-01-02T15:04:05.000000Z07:00` in UTC. It is serialized with `json.Marshal` and canonicalized with the unchanged `protect.CanonicalPayload`.
- The **response** must have a nil error, `algo == "sha512"`, `hash_size == 512` and `^[0-9a-f]{128}$`. Otherwise:
  - a transport error, non-2xx status or decode failure gives `503 CODEXHASH_UNAVAILABLE`;
  - a decodable but wrong response gives `502 CODEXHASH_INVALID_RESPONSE`.

  This holds with `CODEXHASH_FAIL_OPEN=true` (the compose default) and with the client disabled.
- The database recomputes SHA-512 over its own binding. A well-formed hash of some other text is refused (`protection_mismatch` → 500), and nothing is written. Go does not recompute the hash locally.

## 6. CodexTime integration

- **Service used.** The existing service is the interim legacy `codextime_api`, `GET /api/time/current-tiu`. It returns `{"tiu": <JSON number>, "timestamp", "epoch": "2020-01-01 00:00:00", "cached": <bool>}`. CodexTime V1 is not cut over, and no Go client exists for it. Activation therefore uses the C3A scheme `codextime-legacy-seconds-2020/us`, unit `s`.
- **The new read.** `Client.CurrentTIUExact` was added beside `CurrentTIU`; `CurrentTIU` is unchanged. The new read:
  - is fail-closed: it ignores `CODEXTIME_FAIL_OPEN`, and a disabled client is an error;
  - has no fallback, zero or default value;
  - reads only the raw JSON number text, and never decodes into `float64`;
  - rejects duplicate keys and trailing data.
- **Required fields.** It requires:
  - `epoch == "2020-01-01 00:00:00"` (otherwise the scheme does not apply);
  - a non-empty `timestamp`;
  - `cached` as a JSON boolean.
- **Cached TIUs are refused.** `cached: true` gives `ErrTIUCached` → `503 CODEXTIME_UNAVAILABLE`. There is no retry, and no other value is substituted (handoff decision D5).
- **Errors.**
  - Transport, non-2xx, disabled or cached: `503 CODEXTIME_UNAVAILABLE`.
  - A malformed or unsupported body (a string `tiu`, zero, negative, exponent, more than six decimals, leading zeros, more than 14 integer digits, missing fields, the fail-open `{"tiu":0,"source":"unavailable"}` shape, or a V1 snapshot): `502 CODEXTIME_INVALID_RESPONSE`.
- **Manager check.** The manager re-checks the returned value, scheme and unit against the same rules as `codex_channels_tiu_valid`, for both schemes.
- **Unchanged.** CodexTime's calculation, clock rate, CUT Equation and mesh were not touched, and the TIU's value is not re-evaluated.

## 7. Exact TIU representation

| Step | Form |
| --- | --- |
| CodexTime wire | JSON number text, as Python writes it: `214612574.695765`, or `214612574.5` with trailing zeros dropped |
| `CurrentTIUExact` | The same digits. Only trailing zeros are appended to reach exactly six decimals (`214612574.500000`), which C3A allows because the value is unchanged. More than six decimals are rejected, never rounded |
| PostgreSQL call | Text: `$4::text`, cast by the function to unconstrained `NUMERIC` after validation |
| Database response / Go `Protection.TIU` | JSON string, Go `*string` |
| API response | JSON string (`"tiu":"214612574.500000"`), never a number |

The integration tests store values that `float64` cannot represent, such as `99999999999999.999999` and `12345678901234.123457`. They also store V1 integers of 18, 52 and 80 digits through a stub TIU source. `protection_tiu::text`, the function result and the API all return the original text.

## 8. Database activation function compatibility

- Go calls `public.codex_channels_activate($1::text,$2::uuid,$3::text,$4::text,$5::text,$6::text)`. The test `TestChannelDatabaseCompatibilityPostgres` now asserts:
  - the six-argument identity signature, `jsonb` return, `SECURITY DEFINER` and runtime `EXECUTE`;
  - that the runtime can execute exactly six channel functions;
  - that `codex_channels_binding`, `_tiu_valid` and `_site_key_guard` are not executable by the runtime.
- **The binding is identical in Go and PostgreSQL.** This was checked three ways:
  - both C3A golden vectors (including every escaping rule and a `-05:00` opening time) reproduce byte for byte in Go;
  - a full activation succeeds through a runtime pool whose session `TimeZone` is `America/New_York`, for a site key containing `" \ / < & > é U+2028 U+2029 \b \f \n \r \t \x01 \x1f DEL 😀`;
  - the text CodexHash received equals `codex_channels_binding(…)` computed by the database.
- **Strict decoding.** The C3A representation decodes strictly; unknown keys are still refused. A C2A database still decodes for pending channels, but `Activate` fails there (`42883` → 500), because the function does not exist.
- **Lifecycle.** The database remains the final authority. Go writes no tables and has no other activation path.

## 9. Authentication and security checks

- The route uses the existing `ChannelAuth`: a constant-time `X-Service-Key` check and a trusted `CommunicationPrincipal` with `GatewayAuthenticated` and `ChannelSites[{site}]`. Read, admin and discovery grants are not accepted.
- Tests cover a missing or wrong key, an unset `SERVICE_API_KEY`, a missing or unauthenticated principal, another site's grant and non-channel grants. They also cover a request carrying `X-Gateway`, `X-CodexSecure-Actor`, `X-Api-Key`, `Authorization`, `X-Forwarded-For` and a forged principal header. Every case is denied before the manager runs.
- Channel identity alone grants nothing. Another site's channel is `not_found`, both through `Activate` and directly through `codex_channels_activate`, even with that channel's genuine hash. Neither service is contacted for a foreign or unknown channel.
- **Not exposed to clients:**
  - database errors and codes (`CX409`, `23514`, `42501` and raw `CX422` all give 500 with a generic message);
  - service URLs and status codes;
  - the binding text and the rejected protection values.

  Logs record the operation, site, channel and the service error, never the TIU, hash or binding.
- Client input never supplies protection or lifecycle values. No renewal, expiry or other lifecycle route exists.

## 10. Test results

| Run | Result |
| --- | --- |
| Baseline before changes: `go vet ./...`, `go test -race ./...` | Clean; all packages pass |
| Baseline before changes: integration script (C2A schema, PG 16) | Pass (`communications`, `channels`, `handlers`) |
| After: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...` | Clean |
| After: `go test -count=1 -race ./...` | All 11 test packages pass |
| After: integration script on a fresh PG 16 cluster with C2A **and C3A** applied unchanged | Pass |
| After: `go test -count=1 -race -tags integration` for `communications`, `channels` and `handlers` on that cluster | **96 passed, 0 skipped, 0 failed** |
| Mutation checks | A serializer-style trimmed timestamp in the binding fails `TestBindingGoldenVectors`. Accepting `cached: true` fails `TestCurrentTIUExactRejectsCached` and the `codextime cached` activation case. Both sources were restored byte for byte, and the suites re-run green |

New activation coverage:

- success, persistence, the exact binding sent and stored TIU text;
- repeated activation: the fast path contacts no service, the database path keeps the first protection, and `renew` and `close` keep protection;
- the deadline reset on a 20-minute-old pending channel;
- a lapsed channel, a channel that lapses while CodexHash answers, and a channel closed while CodexHash answers;
- 21 CodexHash/CodexTime failure and malformed-response cases plus missing services. Each fails with its code and leaves the channel unchanged and unprotected;
- exact TIU preservation, as in §7;
- 16 concurrent activations, held until all pass the fast path: exactly one `activated`, fifteen `already_active`, one protection;
- site isolation and the cross-site hash;
- binding compatibility, as in §8;
- protection immutability: the schema owner gets `CX409`/`23514`, the runtime gets `42501`;
- the HTTP path: authorization, refused client protection, 200/409/404/422, and the TIU as a JSON string.

Existing Open, Get, Close, Expire, renewal, concurrency and isolation tests pass against the C3A schema.

**Fixture limits.** CodexHash and CodexTime are controlled `httptest` services. They speak the real wire formats and are reached through the real Go clients. They show that the Go and database contracts agree. They do **not** show that CodexTime's mathematics is correct, and they do not show either production service's availability or behaviour. Neither real service was run.

**Baseline failures:** none. `gofmt -l` lists about 35 unrelated files that were already unformatted; none of them was touched by C3B.

## 11. Remaining dependencies before end-to-end operation

1. **Gateway identity (blocking).** No production code installs a `CommunicationPrincipal`, and the gateway forwards neither `X-Service-Key` nor an identity. Every channel route, including activate, is unreachable in deployment. This is unchanged from C2B and is outside this phase. The route was tested through the authorized test harness only.
2. **Gateway error rewriting.** The gateway replaces non-2xx bodies with `UPSTREAM_ERROR`, so `CHANNEL_*` and `CODEX*_UNAVAILABLE` codes do not reach clients.
3. **Deployment order.** Neither channel migration is applied to the persistent `codex_zones` (C3A report §2). Apply C3A with or before this build, because `Activate` needs the function. C3B still reads C2A pending channels.
4. **Cached TIU policy (CodexTime owner).** The legacy service caches the TIU in Redis for 1 s, and `TIUMiddleware` on zone routes reads the same cache. Activation within about 1 s of another TIU read is therefore refused with `503 CODEXTIME_UNAVAILABLE`. Options:
   - accept that the caller retries;
   - add a cache-bypass read on the CodexTime side;
   - move to V1.

   C3B does not change CodexTime.
5. **V1 cut-over.** Production activation uses the interim legacy scheme. Using `codextime-tiu-2020/1` needs a V1 `TIUSource` (snapshot endpoint, API key, decimal strings). The manager and database already accept that scheme.
6. **Trust boundaries (carried over, not addressed):**
   - the CodexHash and CodexTime calls are unauthenticated HTTP;
   - the TIU is unattested and not covered by the binding;
   - `protected_at` (PostgreSQL) and the TIU (CodexTime host clock) are not cross-checked;
   - `zone` is a client declaration.
7. **Activation trigger.** If the protocol later requires Open to activate immediately, `Open` can call `Activate` after a successful open. The route can stay for retries.

**Differences from the Go sketch in the requirements document:**

- `Protection.TIU` is `*string`, not `*json.Number`. `json.Number` re-encodes as a bare JSON number, which would undo C3A's decision to return the TIU as a string. The database contract is unaffected.
- Activation has its own route instead of running inside Open (§4).
- `already_active` is `200`.

Stopped after C3B, as instructed.

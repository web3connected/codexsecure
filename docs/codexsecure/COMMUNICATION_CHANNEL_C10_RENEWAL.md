# Communication Channel C10: Request-Driven Channel Renewal

**Date:** 2026-10-30
**Sides:** CodexSecure Go API (primary), Codex Gateway (secondary), Web3Codex Laravel (final integration)
**Status:** **Complete in local development.** An active channel now renews after each visitor request that completes its Seven-Fold Process. The deadline becomes 30 minutes after that communication. Status checks, failed or halted communications, server errors, HEAD and OPTIONS never renew, and nothing renews without a request.

- **Added:** one Go route, one gateway route and one SDK method, plus one call site in each of the two existing completion stages.
- **Not added:** polling, timers, AJAX, background jobs, tracking, new stages, new lifecycle states and database migrations.
- **Not done:** nothing was committed or deployed beyond the two local development containers rebuilt in §8.3.

## 1. Go renewal implementation

The schema already had what renewal needs. `codex_channels_renew(site, uuid)` (C2A migration `2026_10_27_000000_communication_channels.up.sql`):

- locks the row through `codex_channels_locate`;
- expires a lapsed channel first and returns `expired`, so a lapsed channel is never revived;
- returns `pending`, `closed` or `expired` unchanged for a channel that isn't active;
- for an active channel, sets `last_activity_at = now()` and `expires_at = now() + 30 min`. If a concurrent renewal already advanced it, it returns `renewed` without moving the deadline backwards.

The `codex_channels_guard` trigger independently forbids any other change to activity or protection, and `codex_zones_runtime` already has `EXECUTE` on the function.

**C2B had prepared `Manager.renew` with no caller. C10 exposes it:**

| File | Change |
| --- | --- |
| `internal/channels/manager.go` | `renew` → exported `Renew`; same call and validation. Every non-`renewed` outcome is returned with its result and as an error: `pending` → `409 CHANNEL_NOT_ACTIVE`, `closed` → `409 CHANNEL_CLOSED`, `expired` → `409 CHANNEL_EXPIRED`, `not_found` → `404 CHANNEL_NOT_FOUND` |
| `internal/handlers/channels.go` | `POST /api/v1/sites/{site}/channels/{channel}/renew` on the existing channel group, so it is behind the same `GatewayAssertion` → `ChannelAuth` chain (service key, signed assertion bound to app, site, method and path). Canonical UUID only; empty body or `{}` only (any field → `400 CHANNEL_REQUEST_INVALID`). The response uses the existing envelopes: `200 {data: {result: "renewed", channel}}`, lifecycle refusals with the channel's state in `data`, other failures `{error, code}` without detail |

**Guarantees:**

- **Identity and protection.** The UUID, site, zone, `opened_at` and protection (binding, CodexHash, TIU, scheme, unit, `protected_at`) are unchanged. The trigger enforces this, and the integration tests assert it.
- **No reactivation or creation.** A lapsed channel is recorded as `expired`, and renewal never creates a channel.
- **Site isolation.** Lookups are scoped to the site, so another site's UUID returns `404`.
- **Concurrency.** Concurrent renewals serialize on the row lock, and the deadline never moves backwards (16 concurrent renewals, existing test).

Not changed: Open, Get, Activate, Close, `ChannelAuth`, the assertion verifier, expiry (still manager-internal, with no route) and every other route.

## 2. Database

**No migration and no schema change.** The existing protected function `codex_channels_renew` and its existing grant are reused as they are.

Note: the C1 requirements document (`CODEXSECURE_COMMUNICATION_CHANNELS_GO_REQUIREMENTS.md`) says there is "no HTTP endpoint for renewal". C10 deliberately replaces that decision with one narrowly scoped endpoint. The document itself was not edited, and should be updated by its owner.

## 3. Gateway integration

`codex_gateway/src/channels.ts`:

- The channel route allowlist gains `POST …/channels/{channel}/renew`. `GET`/`PUT` on it → `405`, and `…/renew/extra` → `404`.
- Renew follows exactly the same path as Activate and Close:
  - application key authentication;
  - the `GATEWAY_APP_SITES` site grant;
  - canonical UUID and no body fields;
  - a fresh Ed25519 assertion bound to `POST` and `/api/v1/sites/{site}/channels/{id}/renew`;
  - `X-Service-Key` forwarding;
  - no client header forwarded and no query string;
  - the same response projection, which forwards `CHANNEL_*` refusals with validated state.

There is no new authentication mechanism, and other routes are unchanged. `CHANNEL_ROUTES.md` was updated.

## 4. Laravel client integration

| File | Change |
| --- | --- |
| `packages/web3codex/codexsecure/src/Contracts/CommunicationChannelStore.php` | `renew(string $site, string $channel): array`, result `renewed` |
| `packages/web3codex/codexsecure/src/Clients/CodexSecureChannelClient.php` | `renew()` on the existing gateway transport: `POST …/renew` with no body. The response must be `renewed` with an `active`, protected channel of this site and UUID. Refusals use the existing `ChannelServiceException` codes (`not_active`, `closed`, `expired`, `not_found`, `unavailable`, `unauthorized`) |
| `packages/web3codex/codexsecure/src/Services/Channels/ChannelContinuity.php` | `renew(Request, requestId)` (§5); the context gains a `renewal` key (`renewed` \| `failed` \| null) |

There is no second client.

## 5. Seven-Fold renewal placement

`ChannelContinuity::renew()` is the only renewal path. It renews only when all of these hold:

- the request's channel context exists and channels are enabled;
- Protect verified the channel this request;
- the channel was **already active when the request arrived** (`activation: not_required`);
- there is no failure reason;
- it has not already renewed in this request.

That means:

- **A new or just-activated channel is not renewed:** Open, then Activate, already set its first deadline.
- **At most one renewal per request.**
- **Failures:** a renewal refused as `closed`, `expired` or `not_found` forgets the reference, so the next request's Initialize opens a new channel (C9). A transient failure (`unavailable`) keeps it, and the next qualifying request renews.
- **Stage results and the HTTP response never change.**

| Method | Completion point (existing stage) | Renews when |
| --- | --- | --- |
| GET | `Get\Respond::complete()`, right after it passes. That is the existing "lifecycle completed" point: all six earlier stages completed, and the response exists and is not 5xx | Respond passed |
| POST | `Post\Record`, after `$next()` returned. POST Process and Respond are placeholders, so Record is POST's last implemented stage. The handler's response is checked with the same rule as GET Respond | Record passed (`attached`), and the handler response is not 5xx and carries no rendered exception |
| PUT, PATCH, DELETE | **None.** Their Record, Process and Respond stages are placeholders | Never. They still share and activate the channel (C9). **Limitation**, documented rather than worked around |
| HEAD, OPTIONS | None (no Protect) | Never; no call at all |

**Not renewed:**

- a failed or halted stage (Z0 Boundary);
- a POST whose Laravel validation fails (the redirect carries the `ValidationException`);
- a handler 5xx or exception;
- `Get` calls (including C8's `establish()` and Record's fallback check);
- requests with channels disabled.

There is no eighth stage, and the stage order and responsibilities are unchanged.

## 6. API calls per request

These are channel calls through the gateway, compared with C9.

| Request | Open | Get | Activate | Renew | Total (C9) |
| --- | --- | --- | --- | --- | --- |
| First request, Protect passes | 1 | 0 | 1 | 0 | **2** (2) |
| First request, Protect fails or never runs | 1 | 0 | 0 | 0 | 1 (1) |
| Later completed GET or POST, channel active | 0 | 1 | 0 | 1 | **2** (1) |
| Later protected request that does not complete (validation failure, 5xx, Respond failed) | 0 | 1 | 0 | 0 | 1 (1) |
| Later protected request, channel still pending | 0 | 1 | 1 | 0 | 2 (2) |
| PUT, PATCH, DELETE with an active channel | 0 | 1 | 0 | 0 | 1 (1) |
| Halted GET (before Protect), HEAD, OPTIONS | 0 | 0 | 0 | 0 | 0 (0) |
| Channels disabled | 0 | 0 | 0 | 0 | 0 (0) |

**Cost:** one extra call per completed request on an active channel. In the live walk-through (§8.3), the POST that renewed took 163 ms in total, against 156 ms for a POST without renewal.

**Possible later saving (not done):** for POST, Record's association could use the Renew result in place of Protect's Get. That would change the C9 Protect contract, so it was left out.

## 7. Inactivity deadline behaviour

The deadline is always `last qualifying communication + 30 min`, computed from database time.

| Time | Event | Calls | Deadline |
| --- | --- | --- | --- |
| 10:00 | First request: Open, Protect activates | Open, Activate | 10:30 |
| 10:12 | Completed GET | Get, Renew | 10:42 |
| 10:25 | Completed POST | Get, Renew | 10:55 |
| 10:30 | POST failing validation, or a HEAD | Get / none | 10:55 (unchanged) |
| 10:40 | Completed GET | Get, Renew | 11:10 |
| 11:10 | No further requests | none | the channel lapses |
| later | Next request: Protect's Get finds `expired` and forgets the reference; the following request opens a new channel (C9) | Get, then Open, Activate | new channel |

Nothing runs between requests. Expiry is recorded lazily by the database when the channel is next used.

## 8. Test results

### 8.1 Go (`codexsecure_api`)

| Run | Result |
| --- | --- |
| `go vet ./internal/...` (+ `-tags integration` for handlers and channels) | clean |
| `go test -count=1 -race ./...` | all packages `ok` |
| `scripts/test_communication_contract.sh` on a fresh disposable PG16 (`-race -tags integration`) | `ok`: **361 PASS, 0 SKIP** |

**Coverage:**

- **`TestChannelRenew`:**
  - `renewed` for an empty body, `{}` and padded `{}`;
  - lifecycle bodies (`expires_at`, `last_activity_at`, `tiu`, `[]`, `null`) → `400`, with no manager call;
  - invalid UUID → `422`;
  - pending, closed and expired → `409` with state, not found → `404`;
  - a database error → `500 INTERNAL_ERROR` without detail.
- **Authentication tables:** the renew route was added to every case (no service key, no principal, another site's grant, admin grants only). None reaches the manager.
- **Gateway-assertion route tests:** renew added to the accepted set and to all 14 rejection cases (missing, invalid, expired, modified site/app/method/path, untrusted key, missing configuration, service key, cross-site), plus replay.
- **PostgreSQL, through the signed gateway route** (`TestChannelGatewayAssertionPostgresIntegration`):
  - renewing a pending channel → `CHANNEL_NOT_ACTIVE`;
  - another site → `404`, an unknown UUID → `404`;
  - an active channel → `renewed`, with `last_activity_at` later, `expires_at − last_activity_at = 30 min`, and the same UUID, zone, `opened_at`, CodexHash, TIU and `protected_at`;
  - a Get afterwards returns the same deadline (Get never renews);
  - renewing after close → `409 CHANNEL_CLOSED`, deadline unchanged.
- **Manager-level PostgreSQL tests**, existing and now calling `Renew`: active renewal, 16 concurrent renewals, pending, closed and lapsed-active (expired, not revived) rejection, and cross-site `not_found`.

### 8.2 Gateway (`codex_gateway`)

`npm test` passes: **29 tests, 0 failures**.

The new test, `renew (C10) …`:

- **Forwarding:** with no body or `{}`, it forwards `POST /api/v1/sites/site-a/channels/{id}/renew` with no query string, no body and no content type, and returns the renewed channel.
- **Credentials:** it sends the gateway's service key and assertion, which binds app `demo-app`, `POST`, the renew path, `channel_sites: ["site-a"]` and a 30 s lifetime. The client's forged service key, assertion and actor headers are dropped.

Renew was also added to the existing tests:

- unauthorized application, unknown key and missing key (`403`/`401`), and tenant keys;
- ungranted and encoded sites;
- malformed identifiers (`422`);
- refused bodies, including `expires_at` and `last_activity_at` (`400`);
- route shapes (`GET`/`PUT` → `405`, `/renew/extra` → `404`);
- forwarding of `CHANNEL_NOT_ACTIVE`, `CHANNEL_CLOSED` and `CHANNEL_EXPIRED` with state, and `CHANNEL_NOT_FOUND`.

Rejecting forged, expired, replayed and mismatched assertions is Go's job (§8.1, §8.3).

### 8.3 Laravel (`web3codex`, Laragon PHP 8.3.23)

| Run | Result |
| --- | --- |
| `CodexSecureChannelRenewalTest` (new) | **10 passed** |
| `CodexSecureChannelInitializeTest` (C9, fixture moved to `tests/Concerns/FakesChannelGateway.php`; 1 assertion updated) | 20 passed |
| `CodexSecurePostRecordChannelTest` (C8; 2 assertions updated) | 17 passed |
| `CodexSecureChannelClientTest` + `CodexSecureChannelContinuityTest` (renew cases added) | 72 passed |
| `CodexSecureGetRespondTest` (test double updated for `Respond`'s new constructor dependency) | 20 passed |
| `CodexSecureChannelGatewayLiveTest` (opt-in, live; renewal test added) | **6 passed** |
| Pint on every changed PHP file | passed |
| Full suite | see §8.4 |

**What the new suite covers:**

- a completed GET renews the active channel with Get + Renew, keeping the UUID, zone, `opened_at` and protection, while GET 7FP still has seven passes;
- GET, POST, GET move the deadline to 14:31, 14:32 and 14:33;
- POST renews only after Record passed and the handler succeeded. A validation failure, a handler 500 or an exception gets Get only;
- no renewal immediately after activation, nor for a channel left pending after a failed activation;
- no renewal for a page halted before Protect, or for a failed Protect (GET and POST);
- HEAD and OPTIONS make no call; PUT, PATCH and DELETE make Get only;
- status checks (`establish()`) never renew, and repeated `renew()` or Record in one request makes no second call;
- an `expired` refusal of the renewal is recorded (`CHANNEL_EXPIRED`, 409), the cookie is cleared, the stages and response are unchanged, and the next request opens a new channel;
- an unavailable renewal (`UPSTREAM_UNAVAILABLE`, 503) keeps the reference, and the next completed request renews;
- with channels off, nothing happens.

**Updated earlier assertions, each following from C10:**

- C9 "three later GETs, one Get each" is now Get + Renew each.
- C8 "attached with exactly one Get" is now one Get plus one Renew after completion.
- C8 "Record runs once" now counts two channel calls (Get, Renew) and still no repetition.

**Live integration**, against the rebuilt development services:

- `codexsecure-api` `bbbf22aa30a3` and `codex-gateway` `678c5de8fb38` were built from the C10 source with the existing C7 credentials.
- Only these two containers were recreated. The rollback images are `codexsecure-api:pre-C10-20261030` and `codexweb3-codex-gateway:pre-C10-20261030`.

*A. Through the gateway (`codexsecure_api/scripts/c10_live_channel_renewal.sh`):*

| Step | Result |
| --- | --- |
| Open, Activate | `active`; deadline `18:24:55.716924Z` |
| Get after 2 s | Deadline unchanged (a status check does not renew) |
| Renew after 2 more s | `renewed`. Activity `17:54:59.841403Z`, deadline `18:24:59.841403Z`; same UUID; protection byte-identical |
| Get | Same deadline as the renewal |
| Refused renewals: body field (`400`), invalid application key (`401`), unauthorized site (`403`), unknown channel (`404`), invalid identifier (`422`) | Refused |
| Go directly: no assertion (`403`), Activate's assertion presented on renew (`403`), replayed assertion (`403`), assertion 37 s old (`403`), another site's grant (`404`) | Refused. A valid control renewal → `200` |
| Deadline after all refusals | Equal to the last valid renewal |
| Pending channel renew | `409 CHANNEL_NOT_ACTIVE`, deadline unchanged |
| Closed channel renew | `409 CHANNEL_CLOSED`, deadline unchanged |

*B. Seven-Fold walk-through:*

- real requests through Laravel's HTTP kernel with the real `.env`, gateway, Go, CodexHash and CodexTime;
- a fresh application per request, carrying the encrypted cookie between requests;
- CSRF bypassed in the script only;
- the `log` mailer, so the contact form sent no email.

| # | Request | HTTP | Channel | Calls | Deadline |
| --- | --- | --- | --- | --- | --- |
| 1 | GET `/pages/home` | 200 | `3c308b27…` opened, pending | Open | — |
| 2 | POST contact, valid | 302 | Activated by this request: **not renewed** | Get, Activate | `18:28:35.384128Z` |
| 3 | GET home (7FP stops at Receive, see §9) | 200 | — | none | unchanged (checked) |
| 4 | POST contact, valid | 302 | **Renewed**, same UUID and protection | Get, Renew | `18:28:39.075213Z` |
| 5 | POST contact, failing validation | 302 | Not renewed (rejected communication) | Get | unchanged (checked) |
| 6 | HEAD home | 200 | Not renewed | none | unchanged (checked) |

Every test channel was closed afterwards.

### 8.4 Full Laravel suite

**First run:** 1052 tests, 17 failures + 1 error, 119 skipped. Two failures were new, both in `CodexSecureGetRespondTest`. Its test double subclasses `Respond` with its own constructor, so the new `ChannelContinuity` dependency was never set (500). The test double now passes the dependency to the parent constructor, and `CodexSecureGetRespondTest` passes (20/20).

**Final run** (`php vendor/bin/phpunit`, Laragon PHP 8.3.23): **1052 tests, 917 passed, 119 skipped** (opt-in live and PostgreSQL), **15 failed + 1 error, all pre-existing. No new failures.**

**Pre-existing failures,** separate from C10: the same nine files and counts as C8/C9.

- `CodexSecureDebuggerTest`, `CodexSecureFoundationTest`, `CodexSecureRequestAnalyzerTest` (1 each);
- `CodexSecureRouteAnalyzerTest` ×3, `CodexSecureZoneBarTest` ×1, `CodexSecureZoneDefinitionsTest` ×2;
- `CodexSecureZoneGuardTest` ×3, `CodexSecureZoneStrategiesTest` ×3 (one is the error), `PublicPagesTest` ×1.

**Test count:** 1052 = 1034 (C9) + 10 (renewal suite) + 7 (client) + 1 (live).

## 9. Remaining limitations

1. **PUT, PATCH and DELETE never renew.** They have no implemented Record, Process or Respond stage, so there is no completion point. When those stages exist, each needs one `ChannelContinuity::renew($request, $id)` call after it passes.
2. **Live GET pages do not renew in this development environment.** GET 7FP stops at Receive (`communications_configuration_api_disabled`) because `CODEX_SECURE_CONFIGURATION_API` is off in `.env` (pre-existing, C9 §8.2). Live GETs therefore neither activate nor renew; automated tests show both with the configuration API on. A visitor who only reads pages in this environment keeps a channel for 30 minutes after their last successful POST.
3. **POST renewal requires a successful handler.** Validation failures, 5xx and exceptions do not renew. That matches "after successful processing", but a visitor whose submissions keep failing validation does not keep the channel alive.
4. **Existing app defect found during the walk-through, not fixed (out of scope):** `App\Mail\ContactMessage` reads `$data['subject']`, which validation allows to be absent. A valid contact submission without a subject therefore returns 500 (and does not renew).
5. **Renewal is one extra gateway call** per completed request on an active channel (§6).
6. **Carried over:**
   - cookie-less clients open one channel per request (C9 §10.2);
   - concurrent first requests (C9 §10.3);
   - the per-process Go replay cache (C6);
   - the CodexTime cache risk if a caching CodexTime build returns (C7).

## 10. Files

**Go (`codexsecure_api`):**

- source: `internal/channels/manager.go`, `internal/handlers/channels.go`;
- tests: `internal/handlers/channels_test.go`, `channels_gateway_test.go`, `channels_integration_test.go`, `channels_gateway_integration_test.go`, `internal/channels/postgres_integration_test.go`, `activation_integration_test.go`;
- `scripts/c10_live_channel_renewal.sh` (new);
- `docs/COMMUNICATION_CHANNEL_MANAGER_IMPLEMENTATION.md` (renewal lines).

**Gateway (`codex_gateway`):** `src/channels.ts`, `test/channels.test.js`, `CHANNEL_ROUTES.md`.

**Laravel (`web3codex`):**

- source: `CommunicationChannelStore.php`, `CodexSecureChannelClient.php`, `ChannelContinuity.php`, `SevenFold/Get/Respond.php`, `SevenFold/Post/Record.php`;
- new tests: `tests/Concerns/FakesChannelGateway.php` (fixture moved from the C9 suite, with renew added), `tests/Feature/CodexSecureChannelRenewalTest.php`;
- updated tests: `CodexSecureChannelInitializeTest.php`, `CodexSecurePostRecordChannelTest.php`, `CodexSecureChannelClientTest.php`, `CodexSecureChannelGatewayLiveTest.php`, `CodexSecureGetRespondTest.php`.

**Backups before C10:** `~/codex_backups/C10-20261030-134022/` holds the Go and gateway source and the changed Laravel files.

Stopped after implementation, testing and this report. The next phase has not been started.

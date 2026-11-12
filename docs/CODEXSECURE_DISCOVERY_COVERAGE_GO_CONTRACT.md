# CodexSecure Go API: discovery, coverage, draft and replay contract

Status (2026-10-21): **required in Go, not implemented there.** Web3Codex prepared migration `2026_10_21_220000_communication_observation_coverage` (Pending on live `codex_zones`, validated on the disposable cluster only) and the PHP SDK side. The Go project (`/home/web3codex/projects/Docker/CodexWeb3/codexsecure_api`) was read for reference and **not modified**. Supersedes the schema-limitation sections of the Go reports `COMMUNICATION_CONFIGURATION_IMPLEMENTATION.md` §7–§10 and `AUTHORITY_DISCOVERY_ASSESSMENT_INTEGRATION.md` §B–§C once the migration is applied.

Everything below is additive or backward compatible with the deployed Go API **except** where marked *Go change required*. Until Go changes, the deployed API keeps working on the new schema (verified: same functions/signatures, legacy observation writers still accepted).

## 1. Database contract after the migration

| Object | Purpose | Runtime (`codex_zones_runtime`) |
|---|---|---|
| `communication_objects` | Registered identity. `object_type`, `detected_*`, `canonical_destination` are the immutable **first-seen baseline**, not current behavior. FK to pages is now `(page_id, site_id)`. | unchanged |
| `communication_observations` | Append-only instances. New `observed_type` (link/form/fetch, NOT NULL), `discovery_batch_id`, `batch_sequence`. Destination/method/type/zone are independent evidence. `observed_zone` = **communication zone**; source-page zone stays `pages.zone`. FK `(page_id, site_id)`. UPDATE only `last_seen_at`; DELETE refused for everyone. `(site_id, observed_instance_key)` unique. | + INSERT `(observed_type, discovery_batch_id, batch_sequence)` |
| `communication_discovery_batches` | Discovery generations: scope `site` or `page`, `generation` monotonic per (site, scope, page), status `open/complete/incomplete/failed/stale`, counts, `completeness_issues` (JSON array of `{code, …}`), `coverage_at`, `invalidated_at`. | SELECT |
| `communication_discovery_batch_pages` | Site-generation page manifest (page keys). | SELECT |
| `communication_discovery_batch_items` | Per-sequence ledger: `processed` (with `observation_id`) / `rejected` / `failed` (with `failure_code`), `attempts`. Processed is final. | SELECT |
| `communication_discovery_absences` | For **complete** page generations: `page_object_missing` (page-owned registration not observed) and `universal_not_observed` (site-owned registration seen on this page by the previous complete generation, or ever, and not now). Never deletes or changes a registration/approval. | SELECT |
| `communication_draft_generations` | Lifetime draft version counter per object (`last_draft_version`, only increases). | SELECT |
| view `communication_page_coverage` | Per page: latest generation/status, latest complete uninvalidated generation, `current` (= latest generation is that complete one). | SELECT |
| view `communication_current_observations` | Observations of each page's `current` complete generation. | SELECT |

Insert-time observation guard enforces only ownership: object and page belong to the site (FKs) and a page-scoped object is observed on its own page (`23514 Observation source page does not own this page-scoped communication`). A NULL `observed_type` is filled from `object_type` (compatibility with the deployed writer, which only inserts matching types).

Out-of-batch evidence: inserting an observation with `discovery_batch_id IS NULL` permanently sets `invalidated_at` on that page's complete generations and their parent site generations.

### New SECURITY DEFINER functions (runtime has EXECUTE)

| Function | Returns |
|---|---|
| `codex_communications_start_discovery(site_key text, batch_key uuid, scope text, page_key text, page_url text, page_zone text, parent_batch_key uuid, expected_pages text[], actor text)` | coverage summary JSON (below); `"replayed": true` for an identical start |
| `codex_communications_record_discovery_failure(site_key text, batch_key uuid, sequence int, outcome text /* rejected\|failed */, code text, registration_key uuid DEFAULT NULL)` | `{batch_key, sequence, outcome}` |
| `codex_communications_finalize_discovery(site_key text, batch_key uuid, expected_observations int, outcome text /* complete\|failed */, actor text)` | coverage summary JSON; `"replayed": true` for an identical finalize |
| `codex_communications_canonical_json(jsonb)`, `codex_communications_request_digest(jsonb)` | canonical text / SHA-256 hex (see §5) |

Coverage summary: `{batch_key, scope, generation, status, complete, page (page_key|null), parent_batch_key, expected_observations, processed_observations, rejected_observations, failed_observations, missing_objects, completeness_issues, started_at, finalized_at, coverage_at}`; a site start also returns `expected_pages`.

Rules enforced by the database:

- Start: page scope needs `page_key/url/zone` (page created if new; existing page identity must match, else `CX409`) and no manifest; site scope needs a non-empty manifest and no page. A child page generation needs an **open** parent site generation whose manifest contains the page. Starting a generation marks older **open** generations of the same scope `stale` (site: also their open children). Same `batch_key` + same canonical request → replay; different → `CX410`.
- Observation into a batch: batch must be `open`, `page` scope and the observation's page (`CX409` / `CX422`); the ledger row is written by trigger; a sequence already processed → `CX409 … already processed`.
- Finalize (page): `complete` only when every sequence `1..expected_observations` is `processed`, no sequence > expected exists, and no rejected/failed rows. Otherwise `incomplete` with issues `observations_missing {sequences}`, `unexpected_sequences`, `observations_rejected {count}`, `observations_failed {count}`; `outcome = failed` → `failed` with `discovery_reported_failed`. Absences are derived only for `complete`.
- Finalize (site): `complete` only when every manifest page has a `complete`, uninvalidated child generation that is that page's latest generation, **and** every registered page of the site is in the manifest. Issues: `page_not_covered {page}`, `registered_page_not_declared {page}`. One page never completes a site.
- Stale batch finalize → `CX409`; finalized batch with a different finalize request → `CX410`. Everything is immutable once finalized except the one-way `invalidated_at`.

SQLSTATE mapping to add in Go (`internal/communications/errors.go`): `CX404` → 404 `DISCOVERY_BATCH_NOT_FOUND` when raised by coverage functions (message `Discovery batch not found` / `Site not found` / `Parent site generation not found`); `CX409` → 409 `DISCOVERY_BATCH_CLOSED` for messages beginning `Discovery batch` (stale/closed/already processed), else existing `STALE_REVISION`; `CX410` → 409 `REPLAY_CONFLICT` for messages beginning `Replay conflict`, else `ALREADY_PROCESSED`; `CX422` → 422 `CONFIGURATION_INVALID`. The SDK maps unknown codes to "unavailable", so new codes degrade safely.

## 2. Discovery ingestion — *Go change required*

`internal/communications/discovery.go`:

1. INSERT `observed_type = d.Detected.Type` with every observation.
2. Delete the `kind != d.Detected.Type` → `DISCOVERY_SCHEMA_CONFLICT` branch and the `d.Detected.Zone != d.Page.Zone` preflight; keep validating type/zone against the registry. Keep the 23514 mapping only for the new ownership message.
3. Read observed type from `observed_type` (not `object_type`) in `findTx`/evaluation; evaluate each current observation individually (method ∉ approved methods, destination/type/zone mismatch → `unauthorized`; approval untouched).
4. Accept an optional `"batch": {"key": "<uuid>", "sequence": <int ≥ 1>}` in the discovery body (keep strict decoding for everything else). Resolve the batch id with `SELECT id FROM communication_discovery_batches b JOIN sites s … WHERE s.site_key=$1 AND b.batch_key=$2` and insert `discovery_batch_id`/`batch_sequence` in the same transaction as the observation. Without `batch` behave as today (the database invalidates coverage of that page).
5. When a discovery transaction fails *after* parsing a valid batch reference, roll it back, then call `codex_communications_record_discovery_failure` in a **new** transaction with outcome `rejected` (validation/schema/identity/stale conflicts) or `failed` (processing error) and the API error code. Then return the original error.
6. Replay: the existing request fingerprint must include `batch` (it does if `batch` is part of the decoded struct).
7. Current-observation reads: prefer `communication_current_observations` for assessment evidence; fall back to "newest per (object, page, browser identifier)" only for display, never for completeness.

### Proposed batch endpoints (same group, `DiscoverySites` grant, `Idempotency-Key` required)

| Method | Path | Body | Function |
|---|---|---|---|
| POST | `/sites/{site}/communications/discovery/batches` | `{scope, page: {key,url,zone}\|null, parent_batch_key\|null, expected_pages: [..]\|null}`; batch key = `Idempotency-Key` | `start_discovery(…, actor = principal service identity)` |
| POST | `/sites/{site}/communications/discovery/batches/{batch}/failures` | `{sequence, outcome, code, registration_key\|null}` | `record_discovery_failure` |
| POST | `/sites/{site}/communications/discovery/batches/{batch}/finalize` | `{outcome: complete\|failed, expected_observations\|null}` | `finalize_discovery` |

Responses: `{"data": <coverage summary>}` (failures: `{"data": {batch_key, sequence, outcome}}`). The PHP SDK (`CodexSecureDiscoveryClient`) already implements these calls behind `CODEX_SECURE_DISCOVERY_BATCHES=false` and validates that `complete` ⇔ `status = complete` with no issues.

## 3. Assessments — *Go change required*

`complete: true` is permitted only when all of these hold inside one repeatable-read snapshot:

- **Page scope** (`page` = absolute URL): `communication_page_coverage.current` is true for that page (latest generation complete, not invalidated); every registered communication in scope was evaluated from `communication_current_observations` of that generation; no evaluation failed. Include absences of that generation as gaps/results (`page_object_missing`, `universal_not_observed`) — they are findings, not completeness failures.
- **Site scope**: the site's latest site generation is `complete`, not invalidated, and no newer site generation exists; each of its child page generations is still that page's `current` coverage; every registered communication was evaluated.
- Otherwise `complete: false` with existing `completeness_issues`/`coverage_gaps` plus: `discovery_generation_missing`, `discovery_generation_open`, `discovery_generation_incomplete`, `discovery_generation_stale`, `coverage_invalidated`, `page_not_covered`.

Set `coverage_basis: "finalized_generation"` and add (additive) `coverage: {generation, batch_key, coverage_at}`. Drop `registry_restricts_observed_intent` once the migration is applied. Keep non-2xx for processing failures; the SDK never uses error bodies.

## 4. Draft generations — no API change

`draft_version` is now drawn from `communication_draft_generations` and never reissued for an object: after approval/rejection of draft *n*, the next draft is *n+1*, not 1. Request/response shapes are unchanged (`expected_version` is still the current draft's version, or 0 when there is none). Clients must not assume a new draft starts at 1. Existing data was backfilled from current drafts and recorded `draft_saved` results.

## 5. Request-bound replay — no API change

All five decision functions now compute a request digest internally and bind the idempotency key to object, decision, actor and digest. Identical retry → original result + `"replayed": true`, no new revision/event. Same key with a different request → `CX410 Replay conflict: … different request`; with another actor → `CX410 Replay conflict: … another administrator`; with a key recorded before this migration (no digest) → `CX410 … predates request binding` (fail closed). Go must pass the **verified principal's** actor (it already does).

Canonical request (keys sorted by UTF-8 bytes, no whitespace, integers only, strings escaped exactly as PostgreSQL `to_json(text)`: `\"` `\\` `\b` `\f` `\n` `\r` `\t`, other U+0000–U+001F as lowercase `\u00xx`, everything else literal — Go must use an encoder with `SetEscapeHTML(false)` that does not escape U+2028/U+2029). Digest = lowercase hex SHA-256 of the canonical text of the request object plus `"v": 1`:

| Decision | Request object |
|---|---|
| draft | `{action:"draft_saved", site, registration_key (lowercase UUID), actor, document (normalized v1), expected_version}` |
| approve | `{action:"approve", site, registration_key, actor, draft_version, structure_revision, reason (string\|null)}` |
| reject | `{action:"reject", site, registration_key, actor, draft_version, reason}` |
| disable / revoke | `{action:"disable"\|"revoke", site, registration_key, actor, reason}` |
| discovery start / finalize | `{action:"discovery_start", site, batch_key, scope, page:{key,url,zone}\|null, parent_batch_key, expected_pages (sorted, unique)\|null, actor}` / `{action:"discovery_finalize", site, batch_key, expected_observations, outcome, actor}` |

PHP: `Web3Codex\CodexSecure\Support\CanonicalJson::digest()`; parity with SQL is tested (`CommunicationObservationCoverageTest::test_canonical_json_is_identical_in_php_and_postgresql` and the stored-digest assertions). Go only needs this if it pre-checks replays; the database is authoritative.

## 6. Trusted gateway authority — Web3Codex proposal, gateway/Go implementation required

Audit (Web3Codex side): the SDK sends only `X-API-Key` (application credential from `CODEX_GATEWAY_API_KEY`), plus `X-CodexSecure-Actor` and `Idempotency-Key` on administrator mutations; it never sends `X-Service-Key` and never exposes any credential to the browser (requests originate from Laravel server code). Laravel authenticates administrators with CodexAuth (`auth:admin` guard); the actor string is `web3codex-admin:{id}`. There are no per-site administrator grants in Web3Codex today: one installation manages exactly one site (`CODEX_SECURE_SITE_KEY`). None of this is proof to Go, and the gateway currently drops actor/replay headers and re-wraps the envelope (Go report §"Gateway authority audit").

Proposed contract (requires gateway + Go work; not implemented in Web3Codex):

1. **Site binding** — the gateway maps each application key to exactly one CodexSecure site key; a request whose `{site}` differs is rejected (403) before proxying. This gives `ReadSites`/`DiscoverySites` for that application.
2. **Administrator assertion** — for mutations Laravel signs a short-lived assertion (Ed25519/JWS; key pair per installation, public key registered with the gateway) with claims `iss` (installation), `aud: codexsecure-communications`, `sub` (actor), `site`, `op` (draft/approve/reject/disable/revoke), `key` (registration UUID), `idem` (Idempotency-Key), `digest` (§5 request digest), `iat`/`exp` (≤ 60 s). The gateway verifies signature, audience, expiry, site binding and that `op/key/idem/digest` match the request, strips every client-supplied identity header, and forwards a verified principal to Go over the internal channel (mTLS or a gateway-only network plus `X-Service-Key` injected by the gateway). Go installs `CommunicationPrincipal{GatewayAuthenticated, Actor: sub, AdminSites: {site}}` only from that channel.
3. **Envelope** — pass Go's `{data}` / `{error, code, errors}` bodies and status codes through unchanged on `/communications/*` routes (no `wrapResponse`, no `UPSTREAM_ERROR` substitution).
4. **Tests** — missing/invalid application key, wrong site, missing/forged/expired/replayed assertion, altered body (digest mismatch), each decision endpoint, identical retries.

Web3Codex will add the assertion signer to the SDK once the gateway's verifier and key registration exist; until then `CODEX_SECURE_CONFIGURATION_API` stays `false` and Go keeps denying production requests (fail closed).

## 7. Go tests required

Against a disposable PostgreSQL with all four Web3Codex `codex_zones` migrations: changed method/destination/type/zone discovery persisted under one UUID and evaluated `unauthorized` (approval unchanged) and `approved` again when matching; observed type read from `observed_type`; batch start/observe/failure/finalize through HTTP; sequence gaps, rejections, duplicate sequences, stale and invalidated generations; empty complete page; site completeness only with all manifest pages and no undeclared registered pages; page/site assessments returning `complete: true` only on current coverage; absences reported; draft ABA (stale `draft_version` after approval → 409); replay conflict for different payload/actor; legacy idempotency key; error-code mapping for the new SQLSTATE messages; race tests for concurrent observations into one batch vs. finalize.

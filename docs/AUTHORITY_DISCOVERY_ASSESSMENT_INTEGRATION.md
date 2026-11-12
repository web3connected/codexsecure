# Authority, discovery and assessment integration

October 21, 2026 follow-up: [Discovery coverage implementation](DISCOVERY_COVERAGE_IMPLEMENTATION.md) supersedes the observation schema restrictions, draft reset limitations, and always-incomplete assessment behavior described below. Gateway requirements remain separate.


October 21, 2026. Follow-up to COMMUNICATION_CONFIGURATION_IMPLEMENTATION.md.

Completed Go-only investigation, changes and verification are described below. Gateway
administrator assertions, independent observation type/zone persistence, and complete
scan coverage remain external contract work. No identity verifier, scan proof or complete
assessment is fabricated in their absence. No live database access, schema changes,
credential/connection changes, sibling-project edits, activation, commits or pushes.

## Technical assessment and evidence

References reviewed read-only: the prior report; configuration/discovery/assessment
handlers and repositories; authorization evaluator; service-key and administrator/site
middleware; all existing tests; Web3Codex's registry/configuration migration SQL and
requirements; sibling gateway auth, application keys, proxy and response code.

The schema assertions below were verified on disposable local PostgreSQL initialized
from the unchanged migration sources. They are not claims of live catalog inspection.
The original model intentionally treated detected intent as an immutable definition.
Its guards protect that model correctly. The new requirement separates stable identity
from changing detected behavior; satisfying it needs an explicit schema evolution,
not removal of protections because an insert happened to fail.

## Gateway authority audit and Go boundary

`codex_gateway/src/middleware/auth.ts` authenticates X-API-Key as either an application
identity (`type`, `appId`) or a tenant identity (`type`, `tenantId`, `keyId`). Optional
HMAC/TIU authenticate qualifying tenant requests. None of these includes an administrator
or verified site grants. `authContext` stays in the gateway process.

The proxy in `src/index.ts` forwards content type, request ID, forwarded IP and a static
X-Gateway label. It does not forward a verifiable administrator/site assertion, service
key, actor or idempotency key. A label, application ID, tenant ID, shared service key or
client actor header cannot establish administrator authority. There is no existing
issuer, signature/channel verification contract or trusted administrator-to-site grant
source from which Go can implement a secure production verifier.

Go therefore continues to require all of these separate checks:

| Boundary | Evidence / enforcement |
|---|---|
| Gateway authenticated | Trusted server-installed principal attests that existing gateway auth succeeded; raw X-API-Key or X-Gateway headers cannot install one |
| Calling service authenticated | Independently configured SERVICE_API_KEY checked against X-Service-Key; missing configuration always denies |
| Administrator verified | Principal supplies actor; a request header must match it and cannot establish it |
| Administrator manages site | Principal's AdminSites must grant the route's site; ReadSites/DiscoverySites do not grant mutation rights |
| Object belongs to site | Site-and-registered-UUID lookup/lock in repository and authoritative decision functions |
| Mutation replay | UUID Idempotency-Key, with existing database decision replay/locking |

All five decisions (draft, approve, reject, disable, revoke) use the same boundary.
Production callers remain denied because no trusted principal provider is installed.
Tests install a principal as an explicit trusted fixture, not as a header-based shortcut.
No replacement gateway key validation, administrator authentication, signing scheme,
issuer or identity source was invented.

This phase hardens the Go boundary by copying principal grant maps when installed,
preventing later mutation of a provider's maps from changing request-local grants.
Audit actor identity is now read from that verified principal, rather than re-read from
the client header after authentication. Control characters in actor headers are rejected.

A further gateway contract mismatch was found: `wrapResponse` nests the Go response in
another data envelope, while non-2xx proxy handling discards the backend's stable error
code/body. The configuration SDK expects the Go data/error envelope directly. This also
needs agreement in the owning gateway/SDK integration; Go's existing SDK response shape
was preserved.

## Discovery: exact rejection and Go changes

| Observed change | Existing schema behavior | Classification |
|---|---|---|
| GET → DELETE or POST | `codex_communications_observation_guard()` requires observed_method = object's detected_method; raises 23514, even if POST is approved |
| Destination `/a` → `/changed` | Same trigger requires observation canonical_destination = registered canonical_destination; raises 23514 |
| Zone Z1 → Z2 | Same trigger requires observed_zone = detected_zone; additionally the page/site/observed_zone FK binds it to source-page zone |
| fetch → form | `communication_observations` has no per-observation type column; type can only be read from immutable object_type |
| Registered scope/component/page/item changes | Stable ownership/identity conflict; not a policy mismatch and must not be merged silently |

The first four are behavior/configuration differences when ownership and declarations
remain valid. They do not imply a new registered UUID or approval. The current legacy
intent constraints cannot represent all of them under that identity.

Go never tries to UPDATE immutable detected_url, detected_method, detected_zone or
object_type to record a new observation. It only INSERTs an observation and updates
permitted structural state/revision and last-seen columns. Approval fields/documents
are never written by discovery.

Changes made:

- Observed behavior validation no longer constructs an ApprovedConfiguration document.
  Recognized observed methods/types are retained as evidence even when that combination
  could not be approved (for example form + HEAD). Administrator policy validation and
  its type/method restrictions remain unchanged.
- Removed Go's blanket early method/destination/baseline-zone equality rejection. An
  INSERT now exercises the supported observation path and the actual database guard.
  The exact expected 23514/message/function is mapped to 409 DISCOVERY_SCHEMA_CONFLICT
  with field-specific diagnostics. Unrelated constraint failures are not mislabeled.
- Type changes remain an explicit field-specific conflict because no observation type
  column exists. Source-page/observed-zone inconsistency remains a specific FK conflict.
  There is no metadata fallback, approval rewrite, replacement identity or guard bypass.
- Conflicts roll back the entire ingestion, including tentative structural changes.
  They do not consume its observation replay key. A corrected retry with that key can
  succeed. Responses say that the observation was not persisted and approval is unchanged.
- Stable item lookup now selects the earliest valid item identifier, rather than allowing
  an earlier malformed historical identifier to hide a later established identity.

When the owning migration eventually permits independent method/destination observations,
the INSERT path can use that support without another method/destination equality check
in Go. Type and zone schema changes still require coordinated field/reader changes.

## Structural versus configuration behavior and ownership

Structural declaration validation is separate from observed behavior validation. Loss
of a required declaration with representable baseline intent records invalid structure,
increments structural revision and suspends an approved object. Correcting it advances
the revision again and requires explicit reapproval. Existing integration tests cover
this sequence. If both an unrepresentable behavior change and structural change occur,
the whole transaction fails; no partial structural result is claimed as persisted.

Configuration evaluation tests exercise changed method, destination, type and zone as
Unauthorized, then a matching observation as Approved under the same original approval.
They verify that neither the method list nor any document field changes. Disabled,
revoked, invalid structure, corrected structure, and universal observations on multiple
pages retain their existing states and precedence. No new authorization state was added.

Universal objects retain one site-owned UUID and approval across source pages. Separate
page objects with identical component/item/destination are registered independently on
their respective source pages. Site/page assessment tests verify inclusion of universal
objects only on pages where observed and no inclusion of unrelated-page observations.

## Assessment completeness and response compatibility

Completeness is still false under the supplied schema. It cannot prove which required
observations were expected, whether some ingestion failed/was rejected, whether absent
items were removed, or whether an entire site/page scan finished. Counts of rows and a
successful HTTP response do not prove any of these facts. No client complete flag,
process-memory ledger or borrowed page coverage is treated as proof.

The existing fields and SDK data envelope are preserved. Additive fields now make
incomplete results actionable:

```json
{
  "data": {
    "site": "example",
    "scope": {"type":"page","page":"https://example.test/contact"},
    "complete": false,
    "coverage_basis": "persisted_registry",
    "completeness_issues": ["discovery_snapshot_not_available"],
    "coverage_gaps": [{
      "code":"discovery_snapshot_not_available",
      "action":"Persist a finalized discovery generation for this exact scope",
      "page":"https://example.test/contact"
    }],
    "progress": {
      "registered_communications": 1,
      "evaluated_communications": 1,
      "evaluated_observations": 1
    },
    "communications": []
  }
}
```

This is an abbreviated example, not an exhaustive response. Existing baseline limitations
remain in completeness_issues; coverage_gaps explains each with an action and, where
applicable, page and registration_key. Actual missing observations, unknown structure
and evaluation failures add specific gaps. Progress counts describe only the registered
snapshot read, not the unknown expected scan population. Scope remains explicit and page
results never imply other-page coverage.

A database/processing failure remains non-2xx with the existing safe error/code envelope.
For assessment failures only, it also includes `data` containing incomplete/partial
assessment progress and `assessment_processing_failed` (plus an object-specific gap if
one evaluation failed). Raw database error details are not exposed. This additive error
data is informational; callers must not use error responses for notification resolution.
Validation and authorization failures keep their existing error response shape.

Complete site/page integration tests are **pending the external coverage contract**.
There are no tests that manufacture complete=true from a fake manifest or a caller claim.

## Required Web3Codex / Claude Changes

### A. Trusted gateway administrator/site contract

The owning integration must define and implement:

1. The authoritative administrator authentication source and lookup of active site
   management grants. Existing application or tenant authentication alone is insufficient.
2. A verifiable transport/assertion contract to Go: trusted issuer/channel and validation
   material, calling-service identity, administrator actor, authorized site and operation,
   audience, bounded validity and binding to the request/object/replay identity. Select
   the mechanism within the existing architecture; the current sources do not specify one.
3. Gateway-side stripping/replacement of all client-supplied identity assertions; an
   internal authenticated channel and verification before installing the Go principal.
   Never promote X-CodexSecure-Actor or X-Gateway into authority.
4. Continue existing X-API-Key validation. Inject the configured service credential over
   the internal channel and preserve actor and Idempotency-Key metadata. Do not disclose
   the shared service key to browsers. Preserve the principal's actor in approval history.
5. Preserve the Go configuration data/error contract on these routes, or explicitly
   coordinate gateway-envelope decoding with the SDK. Keep stable CX-derived codes
   rather than replacing them with generic UPSTREAM_ERROR.
6. End-to-end tests: missing/invalid gateway and service keys, unverified/forged administrator,
   expired or altered assertion, cross-site/object/body/replay substitution, valid grants,
   every decision endpoint and identical retries. These are not implemented by Go fixture tests.

Once that contract exists, implement its Go verifier and install the principal before
CommunicationAuth. Until then, keep unverified mutations unavailable.

### B. Independent observed intent: schema migration requirements

Current definitions are in Web3Codex's
`database/migrations/codex_zones/sql/2026_10_20_235826_communications.up.sql`, with the object
guard replaced by `2026_10_21_200000_communication_configuration.up.sql`.

**Table/function:** communication_observations and codex_communications_observation_guard.
**Existing behavior:** INSERT compares destination/method/zone with immutable registered
baseline and raises SQLSTATE 23514. UPDATE permits last_seen_at only.
**Failing example:** UUID R belongs to site S, valid universal component/item, baseline
GET https://example.test/api/products Z1, approved methods [GET,POST]. INSERT for the same
site/page/object using DELETE raises "Observation does not match its registered definition".
Even observed POST fails under the current baseline GET.
**Expected behavior:** append a distinct immutable DELETE observation under R; preserve
approval/document/structural revision; evaluator returns Unauthorized for DELETE. A later
matching observation may restore Approved under the existing valid approval.
**Proposed correction:** evolve the INSERT guard so site, registered object and page
ownership remain enforced, while method/destination/communication zone are independent
recognized observed values. Keep immutable observation history and the last-seen-only
update rule. Do not weaken the object guard or approval functions to implement observation
changes. Runtime needs only narrowly scoped INSERT privileges for observation fields.

**Type representation:** add a typed, constrained per-observation communication type
(e.g. observed_type: link/form/fetch), with owner-controlled backfill from legacy
object_type and atomic restoration/verification of immutable-observation protections.
Grant runtime INSERT on it. Coordinate Go INSERT/SELECT and SDK changes; do not stash
actual type in unvalidated metadata or mutate registration object_type.

**Zone representation and FK:** the current `(page_id,site_id,observed_zone)` FK requires
observed communication zone to equal source-page zone. A Z2 observation on a Z1 source
page fails; a separate Z2 page does not fix the baseline-zone trigger either. Decide and
document whether observed_zone means communication zone or source-page zone. If it is
the communication zone, retain page/site ownership via `(page_id,site_id)` and validate
communication zone independently against the approved zone registry. Review the analogous
object `(page_id,site_id,detected_zone)` FK for initial page-owned registrations. If the
legacy column must retain source-zone semantics, add a distinct typed communication-zone
field and coordinate its reader/writer contract instead. Never mislabel the source zone.

**Security implications:** observed values are untrusted evidence and may be unauthorized;
they must never update approved policy, expand methods, change ownership, or grant approval.
All site/object/page foreign keys and append-only history protections remain. A new
observation/replacement must not hide an independent still-current instance or another
page's behavior. Define supersession/order and stale-scan rejection explicitly.

**Required migration/regression tests:** runtime-role INSERT for GET/POST/DELETE under
one universal UUID; independent destinations/types/zones; legacy backfill; correct and
cross-site/page ownership; unauthorized-method/destination/type/zone evaluation; immutable
old observations; unchanged approval document/events and structural revision for pure
behavior changes; matching correction restores approval; declaration invalidation suspends
and structural correction requires approval; universal reuse and separate page identities;
concurrent observations, stale generations, failed transaction retry and rollback.

No existing alternative supported operation can satisfy these requirements: changing an
object baseline is guarded, recording another instance still invokes the equality guard,
metadata does not provide typed observed intent to the evaluator, and creating another
object would violate stable registration identity. Approval functions are administrator
decisions and are not observation ingestion operations.

### C. Durable coverage/failure contract

The schema has no authoritative discovery generation/expected-observation manifest or
failure ledger. The owning migration/SDK contract needs durable scope (site or exact
page), trusted generation identity/order, expected object/instance membership, processing
outcomes including rejections, missing/removed-item semantics, and atomic finalization.
Failed observation transactions must not erase evidence of an incomplete batch. Counts
alone cannot distinguish duplicate submissions from missing required observations.

Site finalization must prove coverage of the declared site's page population; a page
finalization must not imply other pages were scanned. Universal observations participate
in each relevant page without duplicating registrations. The assessment snapshot must
bind to a finalized generation and current consistent evaluation, not merely a historical
successful scan. Ingestion after finalization must invalidate or advance the generation.

Only after the agreed manifest is processed, required failures are absent, relevant
communications are evaluated, and the assessment transaction succeeds may Go emit
complete=true. Required tests include complete site/page batches, empty but explicitly
complete scopes, partial/failed/rejected batches, interrupted finalization, missing
records, concurrent/stale generations, corrected retries, current-instance replacement,
page isolation, changed policy/structure during evaluation and notification closure
limited to the exact complete scope. These complete-path tests remain pending.

The earlier draft-generation ABA finding remains documented in the previous report;
no unrelated approval contract or schema change was made in this phase.

## Files changed

Production: `internal/communications/discovery.go`, `assessment.go`, `store.go`;
`internal/middleware/communication_auth.go`; `internal/handlers/communications.go`.

Tests: `internal/communications/discovery_test.go`,
`database_contract_integration_test.go`, new
`authority_discovery_assessment_integration_test.go`, new
`evaluation_integration_phase_test.go`; `internal/middleware/communication_auth_test.go`;
`internal/handlers/communications_test.go`, `communications_integration_test.go`.

Documentation: this report and the previous report's follow-up pointer.

## Verification and completion status

Completed and tested in Go/disposable PostgreSQL:

- All 24 shared PHP validation fixtures and unchanged approval-policy restrictions.
- Observed behavior validation separated from approved policy validation.
- Actual method/destination/zone SQL rejections, immutable registered type, classified
  field-specific conflicts, no partial ingestion/approval changes, successful safe retries.
- Structural suspension/reapproval, universal observations across pages, independent
  similar page objects, mismatch classification and no evaluator policy mutation.
- All five administrator boundaries, forged/unverified identities, wrong site, service
  keys, authority-map snapshots, verified audit actor, replay and concurrent retry tests.
- Explicit scope, progress, missing records/structure gaps, database failure and safe
  non-2xx partial assessment data; original page-isolation coverage retained.

| Verification | Result | Scope / limitation |
|---|---|---|
| Shared PHP fixtures | PASS | Six valid normalized documents and eighteen rejected inputs; approval rules unchanged |
| Go unit tests | PASS | Discovery validation, evaluator, error responses, authority and coverage diagnostics |
| Communication HTTP integration | PASS | Gin → runtime-role repository → disposable PostgreSQL, including real database failure |
| Disposable PostgreSQL | PASS | Actual guard/FK failures, ownership, rollback/retry, lifecycle, missing evidence and page isolation |
| Race detector | PASS | `go test -race -tags integration ./internal/communications ./internal/handlers ./internal/middleware -v` |
| Concurrent retries | PASS | First-draft conflict and duplicate approval produce one accepted decision/revision |
| Full Go regression | PASS | `go test ./...` across all default-build packages |
| Real gateway / PHP SDK deployment | PENDING | No administrator/site assertion transport exists; no verifier fabricated |
| Complete site and page scans | BLOCKED | No durable scope manifest/finalization/failure contract exists |
| Durable changed-intent observations | BLOCKED | Required SQL guard/type/zone corrections are documented; expected rejection tests pass |

Unit/shared-fixture tests, HTTP tests, disposable PostgreSQL tests, race detection and
concurrent retries pass. Full `go test ./...` passes. PostgreSQL tests use only the dedicated
loopback database `codexsecure_configuration_test` on port 55439. No live tests were run.

Blocked/documented: actual gateway assertion verification; independently persisted
changed-intent type/zone and behavior accepted by the owning schema; complete site/page
coverage; deployed gateway/PHP SDK end-to-end tests. This phase does not declare those
paths working and does not activate live enforcement.

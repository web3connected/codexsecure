# CodexSecure communication configuration implementation

October 21, 2026 follow-up: [Discovery coverage implementation](DISCOVERY_COVERAGE_IMPLEMENTATION.md) supersedes the observation schema restrictions, draft reset limitations, and always-incomplete assessment behavior described below. Gateway requirements remain separate.


Follow-up: [Authority, discovery and assessment integration](AUTHORITY_DISCOVERY_ASSESSMENT_INTEGRATION.md) documents subsequent Go changes, precise schema/gateway requirements, and additive assessment fields. It supersedes discovery preflight and assessment failure details in this report.

October 21, 2026. Implemented in the Go project using Web3Codex's migration
sources and SDK contract. No further live database access is required or attempted.
No schema, credentials, connection settings, Laravel files, or gateway files changed.
No commits, pushes, activation, notification delivery, or live enforcement.

## 1. Architecture findings

The service uses Gin, pgx v5, Go's standard testing framework, and httptest. Routes
are registered by `internal/server/routes/api.go` with handlers supplied through
`Dependencies` by `cmd/server/main.go`. Go serves `/api/v1`; the gateway strips
`/secure`, producing the documented `/secure/api/v1` public paths. Administrative
communication routes use their own middleware group and never enter zone chains.

The existing `database.DB` owns the PostgreSQL pool. `PostgresStore` receives that
same pool; no new connection settings, framework, or migration mechanism was added.
Zone membership comes from the existing `policy.Registry`, which is copied into the
store at construction. Neither clients nor discoveries can introduce zone definitions.
Site enrollment remains administrative; runtime can read sites and register pages.

The legacy optional service-key middleware remains unchanged for existing routes.
The new communication group always requires a configured service key. Gateway
application/tenant authentication does not itself establish administrator site grants.
The current gateway proxy does not forward service credentials, actor/replay headers,
or verified administrator authority. The production communication group therefore
fails closed pending that integration, as detailed below.

## 2. Files created and modified

New files:

- `internal/communications/configuration.go`, `configuration_test.go`: schema-v1 validation and shared fixtures.
- `internal/communications/testdata/approved-configuration-v1.json`: unchanged shared PHP fixture copy.
- `internal/communications/evaluation.go`: individual-observation authorization.
- `internal/communications/errors.go`, `errors_test.go`: safe API errors and SQLSTATE mapping.
- `internal/communications/store.go`: SDK-compatible read models, decisions, coverage, store interface.
- `internal/communications/postgres.go`: snapshot reads, revision history, atomic database decisions.
- `internal/communications/discovery.go`, `discovery_test.go`: registration, declaration validation, replay and structural revisions.
- `internal/communications/assessment.go`: scoped persisted-registry assessments with explicit completeness limitations.
- `internal/communications/database_contract_integration_test.go`: unchanged SQL function/permission/constraint verification.
- `internal/communications/postgres_integration_test.go`: Go repository lifecycle, discovery, assessment and concurrent retry tests.
- `internal/middleware/communication_auth.go`, `communication_auth_test.go`: strict service authentication and trusted per-site authority.
- `internal/handlers/communications.go`, `communications_test.go`, `communications_integration_test.go`: all HTTP routes, request parsing, security/contract and PostgreSQL-backed HTTP tests.
- `scripts/test_communication_contract.sh`: unchanged Laravel SQL applied only to a fresh dedicated disposable cluster.
- This report.

Modified:

- `cmd/server/main.go`: constructs the store from the existing pool and zone registry.
- `internal/server/routes/dependencies.go`: communication handler dependency.
- `internal/server/routes/api.go`: registers the administrative communication group.

The earlier temporary live-catalog inspection command was removed. No live catalog
verification is a prerequisite for this implementation, per the user's instruction.

## 3. Database functions used

All configuration mutations use the functions defined by Web3Codex:

| Operation | Function signature |
|---|---|
| Draft | `codex_communications_save_draft(text, uuid, jsonb, bigint, text, uuid)` |
| Approve | `codex_communications_approve(text, uuid, bigint, bigint, text, uuid, text)` |
| Reject | `codex_communications_reject(text, uuid, bigint, text, uuid, text)` |
| Disable | `codex_communications_disable(text, uuid, text, uuid, text)` |
| Revoke | `codex_communications_revoke(text, uuid, text, uuid, text)` |

`codex_communications_configuration_valid(jsonb)` is also exercised by contract tests
and the database's existing constraints/functions. Go validates before invoking the
mutation functions. The repository never directly writes approvals, configuration
revisions, drafts, or approval events.

Each mutation runs in one transaction. Go locks the object's row using the same
site/UUID lookup as the database function, so approval's Go draft validation sees the
same locked object and draft as the decision. Replayed approvals still reach the
function after draft deletion and return its recorded result. Reads use repeatable-read,
read-only transactions across configuration, observations, drafts, and history.

SQLSTATE mapping: CX404 → 404 COMMUNICATION_NOT_FOUND; CX409 → 409 STALE_REVISION;
CX410 → 409 ALREADY_PROCESSED; CX422 → 422 CONFIGURATION_INVALID with errors;
CX423 → 422 STRUCTURE_INELIGIBLE. Privilege failures remain 500. A concurrent collision
on the specific global approval-event idempotency constraint maps to ALREADY_PROCESSED;
other uniqueness errors are not mislabeled. Database messages and credentials are not
returned in API responses.

## 4. Endpoints and request/response contract

All routes below are implemented and registered. Prefix:
`/secure/api/v1/sites/{site}/communications` through the existing gateway.

| Method | Suffix | Request / successful data |
|---|---|---|
| GET | `/{key}/configuration` | registration, structure, approval_status, effective, approved, draft, history, observations |
| GET | `/{key}/configuration/history` | history array plus immutable revision documents/statuses |
| PUT | `/{key}/configuration/draft` | `{document,expected_version}` → `{draft_version,based_on_revision}` |
| POST | `/{key}/configuration/approve` | `{draft_version,structure_revision,reason?}` → `{revision,structure_revision}` |
| POST | `/{key}/configuration/reject` | `{draft_version,reason?}` → `{revision}` |
| POST | `/{key}/disable` | `{reason?}` → `{approval_status:"disabled"}` |
| POST | `/{key}/revoke` | `{reason?}` → `{approval_status:"revoked"}` |
| POST | `/discovery` | documented below → registration, structure, observed_instance_key, replayed? |
| GET | `/assessments?scope=site\|page&page={absolute-url}` | site, scope, complete, assessed_at, coverage_basis, completeness_issues, communications |

Success uses `{"data":{...}}`. Errors use `{"error":"...","code":"...","errors"?:{field:[messages]}}`.
Mutation responses retain the SDK's compact function results rather than substituting
an incompatible read model. Revision/version values are JSON integers. Missing approved
and draft values are null. History entries include decision, actor, decided_at, revision,
previous/resulting statuses, structural revision and reason. Configuration reads also
return per-observation mismatches without modifying approved policy.

Bodies are limited to 64 KiB. Unknown fields, duplicate keys, multiple JSON values,
wrong types and malformed JSON are rejected. Missing/null optimistic versions are
rejected rather than silently converted to zero. Reasons are limited to 2000 characters.

## 5. Authentication and authorization

`CommunicationAuth` requires a nonempty configured SERVICE_API_KEY and a matching
X-Service-Key. It never becomes a no-op. It also requires a trusted in-process
`CommunicationPrincipal` installed by verified authentication middleware, with:

- `GatewayAuthenticated`: the existing gateway credential was verified.
- `ReadSites` or `AdminSites` for reads.
- `AdminSites` for administrator mutations and an actor matching the trusted principal.
- `DiscoverySites` for ingestion.

Mutations require a nonzero UUID Idempotency-Key. Administrator actor headers must be
nonempty, unpadded and at most 128 characters. A client-supplied actor, API-key string,
or gateway label never creates a principal or grants access. Site grants and site/UUID
repository predicates both enforce ownership.

**Production integration limitation:** no current middleware supplies this verified
principal, and the existing proxy drops the required headers. Production requests are
therefore denied until the existing gateway/authentication architecture supplies the
trusted context. HTTP integration tests inject it explicitly; they do not pretend to
verify real gateway keys. This is a fail-closed integration seam, not a new identity
provider or a header-based authorization shortcut.

## 6. JSON validation compatibility

All six valid shared PHP fixtures normalize identically; all eighteen invalid fixtures
are rejected. Methods normalize to canonical GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS
order; duplicates after normalization fail. ajax normalizes to fetch. Type/method
compatibility, exact-only matching, registry zone membership, strict boolean enabled,
required fields and unsupported fields are checked. Destination validation rejects
protocol-relative URLs, credentials, fragments, whitespace, controls and backslashes.
Stored approved/draft/history documents are validated too, including original unknown
fields before typed decoding could discard them.

This proves agreement on the supplied fixtures, not exhaustive equivalence of all PHP,
Go and PostgreSQL URL-parser edge cases or execution of the PHP test suite itself.

## 7. Discovery registration and observations

Example body:

```json
{
  "registration_key": "d9fe68fe-b0dc-4aca-b625-56c3514479f2",
  "scope": "universal",
  "component": "main-navigation",
  "page": {"key":"home","url":"https://example.test/","zone":"Z1"},
  "detected": {
    "destination":"https://example.test/products",
    "method":"GET", "transport_method":"GET", "type":"link", "zone":"Z1"
  },
  "declaration": {
    "scope":"universal", "component":"main-navigation",
    "item":"link", "item_identifier":"products-link"
  },
  "browser_object_identifier":"main-navigation-products",
  "original_url":"/products",
  "original_method":null,
  "expected_structure_revision":0
}
```

The site must already be enrolled. Page URLs must belong to the registered site origin.
Go creates or verifies the source page and enforces immutable page identity, registered
scope, component and page ownership. Universal objects have no owning page; observations
retain source pages. Page objects remain bound to their source page.

The caller supplies the authoritative registered UUID; Go does not hash a destination,
method or declaration into another identity algorithm. It compares the PHP declaration
ownership fields directly: site, scope, component/item and source page for page objects.
A site advisory lock serializes creation and duplicate checks. Another UUID for the same
universal declaration is rejected with REGISTRATION_IDENTITY_CONFLICT and its registered
UUID, preventing duplicates across pages. Registered item identifiers cannot be reassigned
under the same UUID. UUID assignment/resolution in the Laravel adapter remains required.

Declaration fields are checked against the browser model: scope and slug identifiers,
required universal component, declared item/actual type agreement, ajax alias, optional
item identifier validity. Missing/invalid declarations produce server-derived structural
issues and invalid state; browser approval eligibility/effective results are never accepted.
Ownership fields and detected values must still be well-formed to persist a registration.

Initial expected structural revision is 0. New valid/invalid discovery advances it to 1.
Later state, issue or declaration changes increase it; unchanged rediscovery does not.
Approval, policy and method arrays are never written by discovery. Raw URL/method attributes
are preserved. Idempotency-Key is the immutable observed-instance UUID; identical retries
return their original structural result. Reuse with a different request payload fails.

**Schema limitation:** existing guards reject changed destination, method, type or zone
under the same object. The API returns 409 DISCOVERY_SCHEMA_CONFLICT and rolls back the
entire discovery instead of creating a new identity, disguising intent in metadata, or
rewriting policy. Multiple methods remain valid in approved documents and the evaluator,
but the supplied observation schema cannot persist them under one registered object.
The page/observation zone foreign key also requires their zones to agree.

## 8. Configuration revision and decision lifecycle

Draft saving leaves approval unchanged. Draft updates use the database's optimistic
version checks. Approval validates the draft again under lock and delegates structural
eligibility, structural binding, revision creation, supersession and event recording to
the authoritative function. Rejection retains the current approved revision. Disable
retains its history/reference while denying authorization. Revoke clears the effective
reference; new explicit approval is required to restore authorization. Replays return
the database's recorded results without another revision or event.

Known source-contract issue: approval/rejection deletes a draft; a subsequent draft
starts at version 1 again. Old version numbers can therefore identify a later draft
when the structural revision is unchanged (ABA risk). Go preserves the SDK/function
contract and does not invent a new version parameter or bypass functions to fix this.
The database also replays the original same-object/action result even if a reused key's
payload differs. Approval functions do not bind replay identity to a request digest.

## 9. Effective authorization

Evaluation is server-side and individual to each observation. Precedence:

1. No approved revision → unapproved, or stored rejected/disabled/revoked.
2. Disabled/revoked status or enabled=false denies.
3. Non-valid structure → suspended.
4. Destination, method membership, type or zone mismatch → unauthorized.
5. Structural revision differs from the approval's binding → reapproval_required.
6. Otherwise approved.

Application-relative approved paths resolve against the site origin for exact destination
comparison. Each method is tested against the full approved method array. Aggregate
configuration status retains the strongest denial among its observations. A corrected
configuration mismatch can match again without a new policy; corrected structure still
needs explicit reapproval. Discovery never expands approved methods or grants approval.

## 10. Assessment coverage

Assessments use a consistent snapshot of registered communications and persisted
observations. Site scope returns registered site objects. Page scope returns page-owned
objects and universal objects observed on that page; returned observations are restricted
to that page. Cross-origin and unknown page requests are rejected. Structural issues,
effective state, current revision/draft and observation mismatches are included.

All current responses deliberately declare:

```json
{
  "complete": false,
  "coverage_basis": "persisted_registry",
  "completeness_issues": [
    "discovery_snapshot_not_available",
    "registry_restricts_observed_intent"
  ]
}
```

The schema records neither complete scan generations nor missing/removed items, and it
rejects changed-intent observations. Claiming complete coverage would allow incorrect
notification resolution. Clients cannot set completeness. Current observation candidates
are the newest persisted instance per browser identifier and source page; historical
instances remain preserved. Full complete assessment/notification closure awaits the
owning schema's observation/coverage contract. No Go notification system was added.

## 11. Database and HTTP integration results

Verified on isolated PostgreSQL 16.15 at loopback port 55439, database
`codexsecure_configuration_test`, using unchanged Laravel registry/configuration SQL.
A minimal PageMeta UUID table supplies only the FK prerequisite. The SQL contract suite
rolls back its rows. Repository/HTTP suites commit solely in the disposable database to
exercise real transactions and permanent audit history. Runtime pools SET ROLE
codex_zones_runtime; setup uses only the disposable cluster administrator.

Passing checks include:

- First/update/stale drafts; first/replacement approval; old revision retention.
- Reject preserving approval; disable; revoke; immutable history and direct-write denial.
- Invalid configurations, structural ineligibility and structural revision conflicts.
- Replayed approval after draft deletion; cross-action replay errors.
- Concurrent first-draft saves: one succeeds, one returns stale.
- Concurrent duplicate approvals: one revision/event, one replay result.
- Universal/page registration, repeated universal observations across pages, duplicate
  declaration rejection, preserved approval during rediscovery, structural suspension
  and corrected-structure reapproval.
- Site/page assessments, universal inclusion on observed pages, exclusion of other-page
  observations, unknown/cross-origin ownership failures, explicit incomplete coverage.
- Real Gin handlers → runtime-role repository → database functions for all seven SDK
  endpoints, discovery and assessments; missing/null version fields; service key, actor
  and replay failures; unknown communication and schema-conflict responses.
- Shared fixture, malformed JSON, safe error envelope, and trusted-context unit tests.

Schema defects are reproduced as expected conflicts, not hidden by weakened constraints.

## 12. Full Go regression results

`go test ./...` passes with loopback networking available for existing httptest servers.
The PostgreSQL repository and HTTP integration suites also pass with `-race`, including
concurrent draft saves and duplicate approval retries. No live database tests or writes, real
PHP SDK HTTP execution, deployed gateway tests or live enforcement tests were performed.

Example isolated rerun (local disposable server must be running):

```sh
COMMUNICATION_TEST_DSN='host=127.0.0.1 port=55439 dbname=codexsecure_configuration_test user=codex_test_admin sslmode=disable' \
  go test -tags integration ./internal/communications ./internal/handlers -run 'Postgres|Communication' -v
go test ./...
```

The bootstrap script reads migrations from WEB3CODEX_REFERENCE_ROOT, accepts only a
nonstandard local disposable port, fails if its roles/database already exist, and never
drops/replaces a database. Existing DB_* settings are not used by write tests.

## 13. Remaining Web3Codex/gateway integration

- Keep CODEX_SECURE_CONFIGURATION_API=false until authenticated gateway integration is
  verified. Go does not modify Laravel activation settings.
- The gateway must validate its existing X-API-Key, establish administrator identity and
  site-specific grants through the existing trusted authentication context, and securely
  deliver that verified context to Go. It must inject its configured service credential
  and preserve actor/replay metadata while stripping spoofed identity assertions.
  An authenticated application alone must not become an administrator.
- Add a server-side SDK discovery method for the body above. Persist/resolve the registered
  UUID from stable declaration ownership and use it in the editor/configuration client;
  do not send cs-item1 strings or sample demo identifiers into UUID endpoints.
- Send stable browser item identifiers, source page ownership, raw attributes, canonical
  detected intent, declarations, expected structural revision and a fresh replay UUID for
  each new ingestion. Reuse that UUID only for identical retries.
- Add SDK assessment retrieval and feed declared scope/completeness into the existing
  Security Notification System. Incomplete responses must never resolve coverage.
- Run the actual PHP SDK through the authenticated gateway after the trusted-context
  bridge exists. Current Go tests verify its observed wire contract, not a deployed SDK.

## 14. Corrections owned by Web3Codex schema/migrations

Proposals only; no Go-owned DDL or constraint bypass:

1. Permit independent observed destination/method/type/zone under a stable registered
   object, while retaining site/page ownership and immutable approval protections.
   Clarify observation-type storage and the current-instance/replacement contract.
2. Define durable discovery batch coverage, missing/removed items, completion and
   stale-generation rejection so assessments can safely become complete.
3. Retain a persistent draft generation/version beyond approval/rejection to avoid ABA.
   Consider binding administrative replay records to actor and canonical request digest.

The reference requirements document's old "migration not applied" note is superseded
by the user's report that Claude applied it. This implementation uses its definitions
as instructed and makes no claims about inspecting the live installation.

## 15. Completion classification

**Implemented and integration-tested:** Go configuration repositories and all SDK routes;
registration/observations permitted by the supplied schema; structural revision handling;
revision/decision lifecycle and replay; authorization evaluator; scoped incomplete
assessments; real runtime-role PostgreSQL and Gin handler paths.

**Implemented and unit-tested, awaiting production integration:** strict middleware's
trusted principal boundary, gateway absence/spoof rejection, validation and error handling.
Actual gateway credential validation remains in the existing gateway and was not replaced.

**Documented but not implemented in this Go-only scope:** gateway trusted-context bridge,
Laravel SDK discovery/assessment additions, schema corrections, complete discovery-snapshot
assessments and deployed end-to-end verification. No live request enforcement was attempted.

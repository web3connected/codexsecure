# Stage 2 — Receive: Communication Object retrieval

Verified locally on 2026-10-26. Source changes only; not deployed or committed.

## A. Existing Go implementation

Laravel binds `CommunicationAssessmentStore` to `CodexSecureAssessmentClient`.
`PageCommunications` calls its `assess(siteKey, absolutePageURL)` method.
The existing endpoint is:

```text
GET /api/v1/sites/{site}/communications/assessments?scope=page&page={absolute URL}
```

The gateway path adds `/secure`. There is no new retrieval endpoint. The same
handler/store still supports `scope=site` for administrative assessments.
`internal/handlers/communications.go` handles HTTP, `internal/communications/assessment.go`
reads a repeatable-read, read-only snapshot, and `postgres.go` loads configuration evidence.

Stage 2 collects facts. Existing Go `effective`, `mismatches`, completeness, history,
and draft fields remain available to other consumers. Laravel Receive drops evaluation
findings, history and drafts; it does not turn incomplete discovery or unapproved
objects into retrieval failures. No Stage 3 behavior was added.

## B. Data contract

The site path parameter is the exact enrolled `sites.site_key`. Query parameters:

| Parameter | Meaning |
| --- | --- |
| `scope` | Required: `page` or `site` |
| `page` | Required absolute HTTP(S) URL for page scope; no page filter for site scope |

Repeated or unsupported query parameters return 422. There is no page-key,
relative-path, site-URL, or zone request parameter.

The direct Go success envelope is `{"data": assessment}`. The gateway wraps it
as `{"status":"success","service":"codexsecure",...,"data":{"data":assessment}}`;
the current Laravel transport explicitly unwraps this shape.

| Assessment field | Meaning |
| --- | --- |
| `site` | Exact site key string (not a new nested site schema) |
| `scope` | `{type, page?}`; page is the requested absolute URL |
| `page` | Additive page-scope field `{key,url,zone}` from the registered page; omitted for site scope |
| `assessed_at` | UTC RFC3339 timestamp |
| `communications` | Always an array on success, including `[]` |
| `complete` | Discovery assessment completeness, **not** retrieval success |
| `coverage_basis` | `finalized_generation` |
| `coverage` | Optional `{generation,batch_key,coverage_at}`; omitted if no generation exists |
| `completeness_issues`, `coverage_gaps`, `progress` | Existing assessment evidence and counts |

Communication fields retain the current schema:

- `registration`: `registration_key` UUID, `scope`, nullable `component`, nullable
  `page` (owner URL for page scope; null for universal objects).
- `structure`: `state`, `revision`, `issues`. These describe the stored structural
  assessment; this API does not introduce a new raw declaration schema.
- `approval_status`: authoritative status on the registered object.
- `approved`: null or `{revision,document,structure_revision,approved_by,approved_at}`.
  `document` contains schema v1 destination, match rule, methods, type, zone and enabled flag.
- `draft`, `history`, `effective`: existing administrative fields, retained for compatibility.
- `observations`: page-filtered authoritative finalized observations with `id`, `page`,
  `destination`, `method`, `type`, `zone`, `effective`, `mismatches`.

No `id`, `type`, or `approved_configuration` aliases were invented: use
`registration.registration_key`, observation/configuration `type`, and `approved.document`.
Missing approval and missing finalized observations remain null/empty facts, not inferred approvals.
Historical observation associations can select a universal object while its finalized
`observations` is empty; this retains existing assessment semantics.

Example populated response (illustrative registered fixture with no finalized discovery):

```json
{
  "data": {
    "site": "example-site",
    "scope": {"type": "page", "page": "https://example.test/pages/home"},
    "page": {"key": "home", "url": "https://example.test/pages/home", "zone": "Z1"},
    "complete": false,
    "assessed_at": "2026-10-26T14:00:00Z",
    "coverage_basis": "finalized_generation",
    "completeness_issues": ["discovery_generation_missing", "communication_observations_missing"],
    "communications": [{
      "registration": {"registration_key": "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "scope": "page", "component": null, "page": "https://example.test/pages/home"},
      "structure": {"state": "valid", "revision": 1, "issues": []},
      "approval_status": "unapproved",
      "effective": "unapproved",
      "approved": null,
      "draft": null,
      "history": [],
      "observations": []
    }],
    "coverage_gaps": [
      {"code": "discovery_generation_missing", "page": "https://example.test/pages/home", "action": "Finalize discovery for this scope"},
      {"code": "communication_observations_missing", "registration_key": "d9fe68fe-b0dc-4aca-b625-56c3514479f2", "page": "https://example.test/pages/home", "action": "Ingest observations for this communication within the declared source-page scope"}
    ],
    "progress": {"registered_communications": 1, "evaluated_communications": 1, "evaluated_observations": 0}
  }
}
```

Successful empty response for a registered page without a discovery generation:

```json
{
  "data": {
    "site": "example-site",
    "scope": {"type": "page", "page": "https://example.test/pages/empty"},
    "page": {"key": "empty", "url": "https://example.test/pages/empty", "zone": "Z1"},
    "complete": false,
    "assessed_at": "2026-10-26T14:00:00Z",
    "coverage_basis": "finalized_generation",
    "completeness_issues": ["discovery_generation_missing"],
    "communications": [],
    "coverage_gaps": [{"code": "discovery_generation_missing", "page": "https://example.test/pages/empty", "action": "Finalize discovery for this scope"}],
    "progress": {"registered_communications": 0, "evaluated_communications": 0, "evaluated_observations": 0}
  }
}
```

`complete:false` does not invalidate successful empty retrieval.

| HTTP | Code | Outcome |
| --- | --- | --- |
| 200 | — | Retrieved registered page; populated or empty collection |
| 401 | `SERVICE_KEY_INVALID` | Missing/wrong service key, or service key not configured |
| 403 | `ADMIN_UNAUTHORIZED` | Missing trusted gateway principal or site grant |
| 404 | `SITE_NOT_FOUND` | Site not enrolled |
| 404 | `PAGE_NOT_FOUND` | Site exists but exact page URL is unregistered |
| 422 | `CONFIGURATION_INVALID` | Invalid scope/URL/origin/query parameters |
| 500 | `INTERNAL_ERROR` | Database/processing failure; never empty success |

Direct 404 example: `{"error":"page has not been registered","code":"PAGE_NOT_FOUND"}`.
Direct 500 includes `error`, `code`, and partial `data` with `complete:false` and an
`assessment_processing_failed` gap. Do not use partial communications as success.
The gateway currently replaces upstream error bodies with `UPSTREAM_ERROR`, preserving
HTTP status. Laravel still fails safely, but loses specific not-found reasons.

## C. Database relationships and identity

`sites.id → pages.site_id → communication_objects.page_id`. Universal objects have
no owner page; `communication_observations` associates them with source pages.
Selection starts from the site's objects, returns page-owned objects only for their
owner URL, and includes universal objects only when an observation associates them
with the requested page. `EXISTS` predicates prevent multiple observations from
duplicating objects. Observation evidence is filtered again to the requested page.

Approvals come from `communication_objects.approved_revision_id` joined to
`communication_configuration_revisions`, drafts from `communication_configuration_drafts`,
and events from `communication_approval_events`. Assessments use the existing
`communication_current_observations` view and discovery coverage records.
No tables, registries, or migrations were created in the real database.

Site key and page URL resolution use exact stored values. Origin comparison checks
scheme/host/port compatibility first; it does not normalize the stored page URL.
Relative paths are rejected. Different case, trailing slashes, query strings or
alternate paths do not silently resolve the original page. Page keys are returned
as facts, not accepted as alternate lookup inputs. An unrelated origin returns 422.
Laravel builds `site_url + request.getPathInfo()`, excluding query strings.

`page.zone` comes from `pages.zone`, never from a communication's destination zone.
The endpoint accepts no physical zone to overwrite and does not mutate page identity.
Laravel currently carries Stage 1's zone in its own payload and ignores the additive
registered-page field. A later mapping can compare registered identity with Stage 1;
it must not silently replace the physical zone. See [Zone Alignment](ZONE_ALIGNMENT.md).

## D. Local integration

Real local container: `codexsecure-api`, port `127.0.0.1:8084`, healthy, policy 3.1.0.
The existing assessment route responds 401 without authentication.
A temporary read-only Go probe executed inside its network environment successfully
connected as the existing runtime user to `codex_zones` at the configured PostgreSQL
host. Counts visible to that account: `sites`, `pages`, `communication_objects`,
`communication_observations`, `communication_configuration_revisions`, and
`communication_page_coverage` all **0**. Store lookup for `web3codex` returns
`SITE_NOT_FOUND`. No real Web3Codex origin or `/pages/home` registration is established.

Required Laravel settings for the next integration pass (not applied here):

```dotenv
CODEX_SECURE_CONFIGURATION_API=true
CODEX_SECURE_SITE_KEY=web3codex
CODEX_SECURE_SITE_URL=http://localhost:8000
CODEX_GATEWAY_URL=http://localhost:8090
CODEX_GATEWAY_API_KEY=<existing server-side gateway credential>
CODEX_SECURE_CONFIGURATION_TIMEOUT=5
```

The inspected Laravel `.env` has `APP_URL=http://localhost:8000` and
`CODEX_GATEWAY_URL=http://localhost:8090`; the three configuration/site settings
are absent, leaving the API disabled by default. The proposed site URL above follows
that current local origin; it is **not** a registered database value.

`web3codex` is the intended key, **not an existing verified enrollment**. Register the
site and pages through existing authorized enrollment/discovery tooling first; their
URLs must match Laravel's configured origin and route paths. From a Laravel container,
use the gateway's reachable internal address instead of localhost. Refresh Laravel's
configuration cache after the later configuration change. No secret values are recorded.

These flags alone cannot enable the current transport. Go requires both configured
`SERVICE_API_KEY` / `X-Service-Key` and a trusted server-installed
`CommunicationPrincipal` with `GatewayAuthenticated` plus `ReadSites[site]` or
`AdminSites[site]`. The running container has no `SERVICE_API_KEY` configured. The
current gateway forwards neither service key nor verifiable principal, and Go has
no production principal provider. Raw client headers cannot install a principal.
This task preserves fail-closed authentication; it does not invent a header bypass.
See [Authority integration](AUTHORITY_DISCOVERY_ASSESSMENT_INTEGRATION.md).

The next gateway/authentication integration must supply a verified site-scoped read
identity before this SDK can work end to end. Admin assertions are a separate
unfinished transport and enabling their flag alone cannot solve Receive authentication.
Laravel timeouts are 5 seconds by default for connect/request, minimum 1 second;
connection errors/timeouts become retrieval failures. The gateway defaults to a
10-second upstream timeout. Go propagates request cancellation through PostgreSQL calls.

## E. Live verification

| Scenario | Real running container / codex_zones | Isolated real HTTP + PostgreSQL |
| --- | --- | --- |
| Populated | Cannot verify: no enrolled site or objects | 200, one correct page object |
| Empty registered page | Cannot verify: no registered pages | 200, `communications:[]`, registered Z1 |
| Unknown | Store: `SITE_NOT_FOUND` for intended Web3Codex key | HTTP 404 `PAGE_NOT_FOUND` for unknown URL under enrolled site |
| Database failure | Healthy database left running | Closed pool: HTTP 500 with incomplete failure data |
| Unauthorized | Actual assessment endpoint: HTTP 401 | HTTP 401 without service key |

Isolated HTTP tests use a real TCP `httptest.Server`, real PostgreSQL, and an explicit
trusted **test** principal. They do not establish gateway authentication. Existing
unit tests cover missing principal, wrong key, site grants, and privilege boundaries.
No failure was injected into the real database or running service.

## F. Laravel compatibility

Read-only inspection of Web3Codex's actual `CodexSecureAssessmentClient`,
`CodexSecureServiceClient`, `CommunicationAssessment`, and `PageCommunications` shows
compatible request parameters, envelopes, collections, approval documents and failure
handling. `page` is additive; current strict parsing ignores it and retains the
requested URL. No response mapping change is required for existing Receive behavior.
Registered page key/zone would require an additive SDK DTO/mapping change if it must
be retained for later comparison. No Laravel code was changed or PHP test suite run.

Receive uses found/empty based on the retrieved collection, independently of discovery
completeness; it caches successes for 60 seconds by default and never caches failures.
Transport authentication and absent enrollment prevent unchanged Laravel from consuming
the real endpoint successfully today. Payload compatibility is not end-to-end readiness.

## G. Changes

- `internal/communications/store.go`: optional resolved `page` identity in assessments.
- `internal/communications/assessment.go`: read exact registered key/URL/zone in the
  existing snapshot; restrict observation-based inclusion to universal objects.
- `internal/communications/stage2_integration_test.go`: identity, zone, empty arrays,
  URL mismatch, unknown site/page, and duplicate universal association checks.
- `internal/handlers/stage2_integration_test.go`: real TCP HTTP populated/empty/unknown,
  unauthorized and database-failure scenarios using disposable PostgreSQL.
- This document and README cross-reference: contract, examples, evidence and blockers.

## H. Tests and remaining work

Passed with Go 1.27.3:

```text
go test ./...
go test -race -tags integration ./internal/communications ./internal/handlers
```

Integration tests used a temporary PostgreSQL 16 cluster at loopback port 55439 and
database `codexsecure_configuration_test`, initialized using unchanged Laravel registry,
configuration and coverage migrations through `scripts/test_communication_contract.sh`.
The temporary cluster was stopped after verification; the probe binary was removed
from the running container.

The runtime-role tests exercise existing ownership, approvals/revisions, universal
associations, finalized coverage and processing failures. The new HTTP test uses the
disposable cluster administrator to create fixtures; authorization is a test principal.
Unit tests use stubs; these are distinct from the PostgreSQL/HTTP verification above.

Remaining: authorized real enrollment and gateway → Go authentication integration,
then live SDK verification. Go source retrieval checks pass, but Stage 2 end-to-end
completion criteria are **not yet met** in the current local installation. The running
container was not rebuilt/restarted, and does not serve the additive source change.
No deployment, commit, Laravel edit, production setting change, or Stage 3 work.

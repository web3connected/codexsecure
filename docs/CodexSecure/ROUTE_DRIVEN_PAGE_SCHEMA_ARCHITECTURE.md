# CodexSecure Route-Driven Page Schema Architecture

**Last Updated:** April 21, 2026  
**Purpose:** Define how CodexSecure can trust and enforce page intent by loading application-generated route schemas before runtime enforcement.

---

## Core Premise

If CodexSecure is operating as an API security service, it cannot wait until enforcement time to discover what a page or route is supposed to be.

The trusted structure must already exist.

That structure should be generated from the application's route system, because every valid page, action, or API entrypoint must declare itself through a route.

This makes the route the root of page identity.

---

## Why Route Discovery Comes First

A route already contains the first layer of intent:

- route name
- URI pattern
- HTTP method
- controller or handler
- expected behavior
- middleware stack

The page or endpoint then adds the second layer of intent:

- expected request shape
- allowed fields
- field types and policies
- known structural markers
- model and view bindings where applicable

Together, those two layers define a trusted page pattern.

CodexSecure should enforce against that pattern instead of treating every request as raw, unstructured input.

---

## Route-Driven Schema Model

The application is responsible for generating a trusted schema file from its own route landscape.

### Generation Flow

1. Application routes are scanned.
2. `generatePageSchema()` builds or updates a schema record for each page or endpoint.
3. The generated schema file is loaded into CodexSecure.
4. CodexSecure enforces structural integrity using that trusted schema set.

This turns `pageSchema` into the bridge between the application and CodexSecure.

---

## What `generatePageSchema()` Should Capture

Each schema record should describe the declared identity of the route, not just the payload fields.

### Recommended Schema Fields

- `route_name`
- `uri`
- `methods`
- `controller`
- `action`
- `model`
- `view`
- `zone_hint`
- `intent`
- `expected_request`
- `allowed_fields`
- `required_fields`
- `field_rules`
- `structural_markers`
- `middleware`
- `version`
- `schema_fingerprint`

### Example Record

```json
{
  "route_name": "login.submit",
  "uri": "/login",
  "methods": ["POST"],
  "controller": "LoginController",
  "action": "authenticate",
  "model": "LoginModel",
  "view": "auth.login",
  "zone_hint": "Z1",
  "intent": "authenticate_user",
  "expected_request": {
    "content_type": "application/json",
    "allowed_fields": ["email", "password"],
    "required_fields": ["email", "password"],
    "field_rules": {
      "email": ["required", "email"],
      "password": ["required", "string", "min:8"]
    }
  },
  "structural_markers": {
    "page_type": "login",
    "auth_flow": true,
    "mutation_class": "credential_submission"
  },
  "middleware": ["guest", "throttle:login"],
  "version": "2026-04-21",
  "schema_fingerprint": "sha3-512:..."
}
```

### Example Review: Login Page Schema

The following example is close to the right shape, but it needs a few refinements to become a strong CodexSecure contract.

### Review Notes

- A single `login` route serving both `GET` and `POST` often represents two different intents: page render and credential submission. CodexSecure should usually store those as separate schema records or as separate route actions under one route family.
- `zone` should be treated as an initial trust envelope or zone hint, not the final authoritative outcome. Final zone alignment should still be derived from runtime pattern matching.
- `view.hash` is useful, but it should be explicitly defined as a normalized structural hash of the rendered page template or page manifest, not a raw file hash.
- `fingerprint` should distinguish route structure, page structure, and input contract. These are different drift surfaces and should not be collapsed into one checksum.
- `status.trusted` is useful, but CodexSecure also needs trust provenance: who generated it, what version generated it, and whether the schema is generated-only, human-reviewed, patched, merged, or locked as a production baseline.
- The schema should define or link to enforcement behavior. CodexSecure needs to know what action to take when structure breaks, not only what the correct structure looks like.
- `markers` is useful for documentation, but a machine-friendly `ui_contract` gives CodexSecure a cleaner bridge between page structure and enforcement logic.
- `field_order_sensitive` should remain explicit. It may be `false` for standard form processing, but it becomes important if the system later uses signed request contracts or strict input fingerprints.

### Reviewed Example

```json
{
  "schema_id": "auth.login.submit.v1",
  "schema_version": "2026-04-21",
  "source": {
    "generator": "generatePageSchema()",
    "application": "codexauth",
    "environment": "production",
    "generated_by": "route-scanner",
    "generated_at": "2026-04-21T14:30:00Z",
    "trust_mode": "human_reviewed_generation"
  },
  "route": {
    "name": "login.submit",
    "family": "login",
    "uri": "/login",
    "methods": ["POST"],
    "zone_hint": "Z0",
    "middleware": ["guest", "csrf", "throttle:login"],
    "host_pattern": "auth.example.com"
  },
  "controller": {
    "name": "LoginController",
    "action": "authenticate",
    "namespace": "App\\Http\\Controllers\\Auth"
  },
  "view": {
    "file": "auth.login",
    "type": "form",
    "render_mode": "server",
    "structure_hash": "sha3-512:a94f3c7e...",
    "markers": ["email_input", "password_input", "submit_login"]
  },
  "model": {
    "name": "User",
    "auth": true,
    "provider": "users"
  },
  "intent": {
    "type": "authentication",
    "label": "LoginExpected",
    "operation": "authenticate_user",
    "mutation_class": "credential_submission"
  },
  "ui_contract": {
    "inputs": ["email", "password"],
    "actions": ["submit_login"],
    "forbidden_inputs": ["adminType", "role", "is_admin"]
  },
  "request": {
    "content_types": ["application/json", "application/x-www-form-urlencoded"],
    "allow_query_string": false,
    "csrf_required": true,
    "session_expected": true,
    "allow_extra_fields": false,
    "max_fields": 2,
    "required_fields": ["email", "password"],
    "field_order_sensitive": false
  },
  "input_schema": {
    "email": {
      "type": "string",
      "format": "email",
      "required": true,
      "max_length": 255,
      "normalized": true
    },
    "password": {
      "type": "string",
      "required": true,
      "min_length": 8,
      "max_length": 128,
      "secret": true
    }
  },
  "constraints": {
    "deny_files": true,
    "deny_nested_objects": true,
    "deny_unexpected_arrays": true,
    "injection_signals": ["sql_meta_chars", "script_tags", "template_tokens"],
    "rate_limit_profile": "login"
  },
  "enforcement": {
    "on_structure_violation": "quarantine",
    "on_input_violation": "reject_request",
    "zone_lock": "Z0",
    "error_code": "LOGIN_SCHEMA_VIOLATION"
  },
  "fingerprint": {
    "route_hash": "sha3-512:f83a9b12...",
    "view_hash": "sha3-512:a94f3c7e...",
    "input_contract_hash": "sha3-512:bb29c1aa...",
    "composite_hash": "sha3-512:cc81d8ef..."
  },
  "response_expectation": {
    "success_statuses": [200, 302],
    "failure_statuses": [401, 422],
    "sets_session": true,
    "issues_auth_cookie": true
  },
  "status": {
    "trusted": true,
    "active": true,
    "review_state": "approved",
    "last_generated": "2026-04-21T14:30:00Z",
    "last_verified": "2026-04-21T14:35:00Z"
  }
}
```

### Useful Items To Keep In The Schema

- `schema_id`: Gives CodexSecure a stable identity even if route names drift.
- `schema_version`: Lets the service compare trusted and regenerated artifacts over time.
- `source`: Captures trust provenance so generated schemas are auditable.
- `source.trust_mode`: Distinguishes pure generation from human-reviewed generation, patched artifacts, merged artifacts, or locked production baselines.
- `family`: Groups related route records such as `login.form` and `login.submit`.
- `host_pattern`: Useful when the same URI exists across multiple subdomains.
- `middleware`: Important because middleware is part of route intent, not just implementation detail.
- `ui_contract`: Separates visible page actions and allowed inputs from lower-level request rules.
- `content_types`: Prevents requests from changing transport shape to bypass expected patterns.
- `csrf_required` and `session_expected`: Useful for distinguishing browser auth flows from machine-to-machine API flows.
- `deny_nested_objects` and `deny_unexpected_arrays`: Important for blocking structurally abnormal login payloads.
- `injection_signals`: Lets CodexSecure record pattern-level anomaly categories, not just pass or fail.
- `enforcement`: Defines what CodexSecure should do when the route contract is violated.
- `response_expectation`: Useful for detecting unexpected behavior after route execution.
- `review_state`: Separates generated artifacts from human-approved trusted artifacts.

### Recommended Modeling Rule

For security-sensitive flows, prefer separate schema records for:

- `GET /login` as page-render intent
- `POST /login` as credential-submission intent

That keeps view trust and mutation trust distinct, which makes drift detection and phase locking more accurate.

### Trust Modes

CodexSecure should track how a schema became trusted.

Recommended `source.trust_mode` values:

- `generated_only`
- `human_reviewed_generation`
- `patched`
- `merged`
- `locked_production_baseline`

This matters because enforcement confidence should depend on provenance. A locked production baseline should be treated differently from an artifact that was only auto-generated moments ago.

### Enforcement Behavior

The schema should define or link to the enforcement action CodexSecure must take when a pattern breaks.

Recommended enforcement fields:

- `on_structure_violation`
- `on_input_violation`
- `zone_lock`
- `error_code`
- `log_severity`
- `emit_stage7_record`

For high-trust authentication routes, a useful default is:

- structural violations quarantine the request into a low-trust envelope
- input violations reject the request immediately
- the request is retained as a Stage 7 drift record for later analysis

### Lifecycle

The next implementation step is not more schema polishing. It is defining the lifecycle that makes the schema operational:

1. `generate` — create the schema artifact from route discovery.
2. `store baseline` — persist the generated or approved schema as the current trusted baseline.
3. `verify` — check incoming requests against the loaded schema.
4. `update` — adjust the route's weighted phase-lock score based on valid and invalid request behavior.
5. `diff` — compare regenerated artifacts and runtime behavior against the trusted baseline.
6. `quarantine` — route structural drift into a safe enforcement path when trust breaks.

This is where a page schema stops being a good JSON file and starts becoming CodexSecure runtime behavior.

For the top-level artifact format that carries a full schema set, see [ _docs/CodexSecure/PAGE_SCHEMA_ARTIFACT_SPEC.md ](_docs/CodexSecure/PAGE_SCHEMA_ARTIFACT_SPEC.md).

### Phase Lock Update Step

After a request is verified against the schema, CodexSecure should update a weighted confidence value for that route pattern.

This weighted average is the beginning of phase lock.

Working definition:

- valid requests increase the score
- invalid requests decrease the score
- strong structural matches should carry more weight than weak matches
- structural violations should reduce the score more aggressively than minor input mistakes

Initial planning assumption:

- a route pattern at `90%` or higher can be treated as strongly phase-locked

That threshold can be detailed later, but the update step should already exist in the model.

The important part for now is the sequence:

1. schema exists
2. request is verified
3. weighted phase-lock value is updated
4. the updated value influences trust, zone confidence, and later quarantine decisions

This makes phase lock an accumulated trust signal, not a one-time pass/fail result.

### Z0 Orchestration Rule

All requests must land in `Z0` first.

Even when Stage 1 or MCP produces a strong weighted guess, that guess does not directly place the request into a realized zone.

The correct flow is:

1. request arrives
2. Stage 1 produces a weighted route or zone hypothesis
3. request enters `Z0`
4. `Z0` orchestrates verification, weighting, and later zone realization

This keeps unknown or suspicious data from being treated as trusted too early.

So Stage 1 is a guess engine, not an authority. `Z0` is the authority that decides whether a request stays quarantined, gains trust, or is later routed into a realized zone.

### Stage 1 Payload For Z0

The MCP or local preprocess response should be treated as a report for `Z0`.

That report should answer two questions:

1. what zone is the best current guess?
2. should `Z0` fast-route that guess or continue full middleware evaluation?

Recommended Stage 1 to Z0 fields:

- `proposed_zone_id`
- `proposed_zone_name`
- `confidence`
- `quarantine_candidate`
- `quarantine_reason`
- `z0_directive`
- `z0_reason`
- `z0_confidence_gate`

For the basic implementation, `z0_directive` should be one of:

- `redirect_guessed_zone`
- `run_full_middleware`

That keeps Stage 1 simple: it returns a weighted report, and `Z0` decides how to use it.

---

## Enforcement Model Inside CodexSecure

Once loaded, the schema set becomes a trusted route registry.

CodexSecure should use it in four stages:

### 1. Identity Resolution

Resolve the incoming request to a known route or route pattern.

### 2. Intent Verification

Load the schema record and verify that the request matches the route's declared purpose.

### 3. Structural Comparison

Compare the incoming request against:

- expected method
- expected content type
- expected field set
- expected field shapes
- expected middleware and execution context
- expected page or endpoint fingerprint

### 4. Zone Decision and Recording

If the request aligns with the schema, CodexSecure can assign or confirm the intended zone with higher confidence.

If it deviates, CodexSecure should record the mismatch as structural drift, intent drift, or injection risk.

The weighted phase-lock value should feed this stage. Higher phase-lock means the route is behaving consistently with its declared schema. Falling phase-lock means the route is drifting and may require tighter enforcement or quarantine.

The important boundary is that `Z0` performs this orchestration. A good Stage 1 guess should speed later orchestration, but it should not bypass `Z0`.

---

## Why This Improves Security

Traditional validation asks:

`Is this value allowed?`

Route-driven schema validation asks:

`Does this request belong to the route pattern it claims to be?`

That is stronger because most attacks introduce structure that does not belong:

- unexpected fields
- wrong methods
- mismatched content types
- injected expressions
- altered templates or page signatures
- payloads inconsistent with the route's declared intent

When the route and page schema are fingerprinted ahead of time, those mismatches become visible as pattern violations.

---

## Relationship to Phase Locking

This architecture is the missing upstream layer for phase locking.

Today, Stage 7 recording proves that CodexSecure can store zone-linked execution evidence.

The next step is to make that zone evidence originate from a trusted route schema instead of relying only on caller-supplied zone context.

The intended chain becomes:

1. Route discovery establishes trusted page identity.
2. `generatePageSchema()` exports the route-driven schema set.
3. CodexSecure loads the schema set before enforcement.
4. Incoming requests are compared against known intent patterns.
5. The final zone decision is derived from route intent plus runtime alignment.
6. Stage 7 records the aligned or drifted result as execution evidence.
7. A later MCP feedback loop can learn from confirmed route-to-zone outcomes.

This is how page trust, zone assignment, and phase locking become part of the same system.

---

## Drift Detection

The schema file should be regenerated whenever routes or page structures change.

CodexSecure can then compare:

- current generated schema
- last trusted schema
- runtime request behavior

That supports three important checks:

- **Application drift:** route/controller/view changed
- **Page drift:** template or structure changed
- **Runtime drift:** incoming request no longer matches the trusted route pattern

This creates a practical trust lifecycle instead of a one-time static definition.

---

## Implementation Direction

### Application Side

- Add a `generatePageSchema()` command or build step.
- Scan route definitions as the primary source of truth.
- Resolve controller, model, view, and request rule metadata.
- Emit a canonical schema artifact such as `storage/codexsecure/page-schema.json`.

### CodexSecure Side

- Add a schema loader for application-provided page schema artifacts.
- Build an in-memory trusted route registry from the artifact.
- Use the registry during route analysis, zone alignment, and Stage 7 recording.
- Add drift comparison between trusted schema and updated schema versions.

### Enforcement Goal

CodexSecure should not only validate requests. It should verify that each request harmonizes with the route-defined intent it claims to represent.

---

## Design Statement

All page trust begins with route discovery, because every valid page must declare itself through a route.
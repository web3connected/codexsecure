# CodexSecure Page Schema Artifact Specification

**Last Updated:** April 21, 2026  
**Purpose:** Define the canonical top-level `page-schema.json` artifact format that applications generate and CodexSecure loads.

---

## Purpose

The route-driven schema work defines what a single trusted page or endpoint record should look like.

This document defines the top-level artifact that contains the full trusted schema set for an application.

CodexSecure should load this artifact as a baseline registry of route identity, intent, trust state, and enforcement behavior.

---

## Artifact Role

`page-schema.json` is the handoff contract between the application and CodexSecure.

It supports five lifecycle steps:

1. `generate`
2. `store baseline`
3. `verify`
4. `update`
5. `diff`
6. `quarantine`

The artifact must therefore carry both route records and baseline metadata.

---

## Canonical File Location

Recommended application-side output locations:

- `storage/codexsecure/page-schema.json`
- `bootstrap/cache/codexsecure/page-schema.json`
- `var/lib/codexsecure/page-schema.json`

CodexSecure may later support remote retrieval, but the initial contract should assume a file-based artifact.

---

## Top-Level Structure

```json
{
  "artifact_version": "1.0.0",
  "schema_set_id": "codexauth.production.page-schema",
  "application": {
    "name": "codexauth",
    "environment": "production",
    "base_url": "https://auth.example.com",
    "framework": "laravel"
  },
  "generation": {
    "generator": "generatePageSchema()",
    "generator_version": "1.0.0",
    "generated_at": "2026-04-21T16:00:00Z",
    "trust_mode": "human_reviewed_generation",
    "source_commit": "abc123def456",
    "baseline_state": "locked_production_baseline"
  },
  "artifact_fingerprint": {
    "record_count": 24,
    "records_hash": "sha3-512:...",
    "composite_hash": "sha3-512:..."
  },
  "policies": {
    "default_enforcement_profile": "strict-browser-auth",
    "default_zone_on_unknown_route": "Z0",
    "default_on_schema_miss": "quarantine",
    "default_on_diff": "review_required"
  },
  "phase_lock": {
    "enabled": true,
    "algorithm": "weighted_average",
    "strong_lock_threshold": 0.90,
    "default_score": 0.50,
    "weights": {
      "strong_structure_match": 0.40,
      "strong_input_match": 0.30,
      "minor_input_violation": -0.15,
      "structure_violation": -0.40
    }
  },
  "records": [
    {
      "schema_id": "auth.login.form.v1",
      "schema_version": "2026-04-21",
      "source": {
        "generator": "generatePageSchema()",
        "application": "codexauth",
        "environment": "production",
        "generated_by": "route-scanner",
        "generated_at": "2026-04-21T16:00:00Z",
        "trust_mode": "human_reviewed_generation"
      },
      "route": {
        "name": "login.form",
        "family": "login",
        "uri": "/login",
        "methods": ["GET"],
        "zone_hint": "Z0"
      },
      "intent": {
        "type": "authentication",
        "operation": "render_login_form"
      },
      "status": {
        "trusted": true,
        "active": true,
        "review_state": "approved"
      }
    },
    {
      "schema_id": "auth.login.submit.v1",
      "schema_version": "2026-04-21",
      "source": {
        "generator": "generatePageSchema()",
        "application": "codexauth",
        "environment": "production",
        "generated_by": "route-scanner",
        "generated_at": "2026-04-21T16:00:00Z",
        "trust_mode": "human_reviewed_generation"
      },
      "route": {
        "name": "login.submit",
        "family": "login",
        "uri": "/login",
        "methods": ["POST"],
        "zone_hint": "Z0"
      },
      "intent": {
        "type": "authentication",
        "operation": "authenticate_user"
      },
      "status": {
        "trusted": true,
        "active": true,
        "review_state": "approved"
      }
    }
  ]
}
```

---

## Required Top-Level Fields

### `artifact_version`

Version of the top-level artifact contract, not the individual route records.

### `schema_set_id`

Stable identifier for the application schema set.

Recommended pattern:

`{application}.{environment}.page-schema`

### `application`

Describes the application that generated the artifact.

Recommended fields:

- `name`
- `environment`
- `base_url`
- `framework`
- `service_type`

### `generation`

Captures provenance and baseline status.

Recommended fields:

- `generator`
- `generator_version`
- `generated_at`
- `trust_mode`
- `source_commit`
- `baseline_state`

Recommended `baseline_state` values:

- `generated`
- `reviewed`
- `patched`
- `merged`
- `locked_production_baseline`

### `artifact_fingerprint`

Allows CodexSecure to detect whole-artifact drift.

Recommended fields:

- `record_count`
- `records_hash`
- `composite_hash`

### `policies`

Default behavior CodexSecure should apply when a route record is missing or a diff is detected.

Recommended fields:

- `default_enforcement_profile`
- `default_zone_on_unknown_route`
- `default_on_schema_miss`
- `default_on_diff`

### `phase_lock`

Defines the artifact-level defaults for the weighted phase-lock update step.

Recommended fields:

- `enabled`
- `algorithm`
- `strong_lock_threshold`
- `default_score`
- `weights`

The precise math can evolve later. For now, the important rule is simple:

- valid requests move the score upward
- invalid requests move the score downward
- structural violations should reduce trust more than minor field mistakes

Phase-lock does not bypass `Z0`. It gives `Z0` a stronger basis for later route realization.

### `records`

Array of individual route/page schema records.

Each record should follow the shape described in [ _docs/CodexSecure/ROUTE_DRIVEN_PAGE_SCHEMA_ARCHITECTURE.md ](_docs/CodexSecure/ROUTE_DRIVEN_PAGE_SCHEMA_ARCHITECTURE.md).

---

## Optional Top-Level Fields

Useful additions for stronger operational handling:

- `signatures`: digital signature or signer metadata for locked baselines
- `overrides`: human-approved patches applied after generation
- `deprecated_records`: route schemas kept only for comparison and audit
- `quarantine_profiles`: named fallback behaviors for structural violations
- `diff_history`: summary of previous baseline changes
- `notes`: human review or approval notes

---

## Trust and Baseline Semantics

CodexSecure should not treat every artifact the same.

Recommended interpretation:

- `generated`: loadable, but lower confidence
- `reviewed`: approved for active verification
- `patched`: approved with hand-edited changes
- `merged`: generated and patched content combined into one approved baseline
- `locked_production_baseline`: highest-confidence enforcement baseline

This lets CodexSecure decide whether a diff should trigger logging only, quarantine, or a hard enforcement path.

Phase-lock should interact with this baseline state. A locked production baseline with a falling phase-lock score should be treated as a stronger signal of runtime drift than a newly generated baseline with limited history.

---

## Diff Model

CodexSecure should compare three things:

1. current trusted artifact
2. newly generated artifact
3. runtime request behavior
4. current phase-lock score trajectory

Important diff classes:

- `record_added`
- `record_removed`
- `route_changed`
- `controller_changed`
- `view_changed`
- `input_contract_changed`
- `enforcement_changed`
- `trust_mode_changed`
- `baseline_state_changed`

This gives the system both design-time drift detection and runtime drift detection.

---

## Phase Lock Update Model

After `verify`, CodexSecure should update a weighted phase-lock value for the matched schema record.

This is not a replacement for validation. It is an accumulated confidence signal.

Working rule:

- repeated valid matches raise the score
- repeated invalid requests lower the score
- severe structural breaks lower the score faster than minor input mismatches

Initial planning threshold:

- `0.90` or higher means strongly phase-locked

The exact algorithm can be refined later. The immediate requirement is to represent the update step in the artifact contract so the runtime can persist and use it.

In the runtime flow, the Stage 1 payload returned by MCP or local preprocess should be interpreted as a `Z0` decision report, not as direct final routing.

---

## Loader Expectations Inside CodexSecure

The initial CodexSecure loader should:

1. parse the artifact
2. validate the top-level contract
3. validate each record
4. build an in-memory index by route name, URI pattern, and schema ID
5. store artifact fingerprint metadata for later diff checks
6. apply default policies when a matching record cannot be found

Recommended loader indexes:

- `schema_id`
- `route.name`
- `route.uri + route.methods`
- `route.family`

---

## Initial Implementation Rule

For the first implementation, keep the contract simple:

- one file per application
- one `records` array
- one baseline state
- file-based loading only

Do not start with remote sync, multi-file composition, or dynamic partial merges unless there is a concrete runtime need.

---

## Implementation Outcome

With this artifact in place, CodexSecure can move from isolated schema examples to a real trust registry.

That is the bridge from:

- route discovery
- schema generation
- trusted baseline storage
- runtime verification
- diff and quarantine enforcement

This is the point where `pageSchema` becomes a loadable security substrate rather than a documentation concept.
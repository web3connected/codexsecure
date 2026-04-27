# CodexSecure Zoned Endpoint Route Plan

## Purpose

Define the first real zoned API surface for `Docker/codexsecure_api` using only handlers the service already owns.

This plan avoids placeholder or misleading authority signals.
If an endpoint exists under a zone module, it must represent a real CodexSecure capability.

## Design Rules

1. Zone modules are authoritative.
2. No fake business routes such as `login` or `execute` unless CodexSecure truly owns them.
3. Zoned paths must not be introduced as simple mirrors of cross-cutting capability handlers.
4. Keep the route count intentionally small: 12 endpoints in 6 zones.
5. Prefer analysis, fingerprinting, pipeline, policy, and observability capabilities that already exist.

## Route Shape

Canonical pattern:

`/api/v1/zones/z{n}/{capability}/{action}`

Examples:

- `/api/v1/zones/z1/intake/detect`
- `/api/v1/zones/z5/pipeline/analyze`
- `/api/v1/zones/z12/policy/snapshot`

## Status

This document is now a route-shaping reference, not an implementation order.

See [HANDLER_ZONE_OWNERSHIP_AUDIT.md](/home/web3codex/projects/_docs/CodexSecure/HANDLER_ZONE_OWNERSHIP_AUDIT.md) before scaffolding any zoned modules.

Current conclusion:

- most existing handlers are cross-cutting and should remain capability-grouped
- zoned modules should be created only for truly zone-native endpoints
- pipeline and future Stage 7 token lifecycle are the strongest candidates for eventual zoned ownership

## Proposed 6 Zones

### Z1 Public Intake

Reason:
Public-safe discovery and intake operations. No auth-only claims.

Endpoints:

1. `POST /api/v1/zones/z1/intake/detect`
   Owner: `ZoneHandler.DetectZone`
   Purpose: detect likely zone for a request envelope.

2. `GET /api/v1/zones/z1/intake/health`
   Owner: `ZoneHandler.ZoneHealth`
   Purpose: lightweight zone health overview.

### Z2 Identity Signal

Reason:
Identity-class API analysis without implying login UI ownership.

Endpoints:

3. `POST /api/v1/zones/z2/identity/fingerprint-input`
   Owner: `FingerprintHandler.GenerateInputFingerprint`
   Purpose: fingerprint identity-related payload fields.

4. `POST /api/v1/zones/z2/identity/drift`
   Owner: `FingerprintHandler.CalculateDrift`
   Purpose: compare identity-related fingerprints and TIU drift.

### Z4 Protected Request Review

Reason:
Protected-request inspection and route validation for secure application flows.

Endpoints:

5. `POST /api/v1/zones/z4/protected/analyze-route`
   Owner: `AnalyzerHandler.AnalyzeSingleRoute`
   Purpose: classify a single protected route request.

6. `POST /api/v1/zones/z4/protected/check-zone`
   Owner: `AnalyzerHandler.CheckRouteInZone`
   Purpose: assert whether a route belongs in an expected secure zone.

### Z5 System Pipeline

Reason:
Core CodexSecure processing and canonical Stage 7 recording.

Endpoints:

7. `POST /api/v1/zones/z5/pipeline/analyze`
   Owner: `PipelineHandler.AnalyzeExecution`
   Purpose: run the 7-stage analysis path without persistence.

8. `POST /api/v1/zones/z5/pipeline/executions`
   Owner: `PipelineHandler.CreateExecution`
   Purpose: persist canonical Stage 7 execution records.

### Z9 Analytics

Reason:
Read/report-style endpoints for route and fingerprint analytics.

Endpoints:

9. `POST /api/v1/zones/z9/analytics/route-stats`
   Owner: `AnalyzerHandler.GetAnalysisStats`
   Purpose: aggregate route-zone analysis stats.

10. `POST /api/v1/zones/z9/analytics/fingerprint-batch`
    Owner: `FingerprintHandler.BatchAnalyze`
    Purpose: batch fingerprint analysis for metrics/reporting use.

### Z12 Admin Policy

Reason:
Administrative policy visibility and enforcement inspection.

Endpoints:

11. `GET /api/v1/zones/z12/admin/policy-snapshot`
    Owner: `PolicyHandler.GetSnapshot`
    Purpose: retrieve the complete current policy snapshot.

12. `POST /api/v1/zones/z12/admin/validate-method`
    Owner: `PolicyHandler.ValidateMethod`
    Purpose: validate zone/method enforcement rules.

## Why These 12

This set covers the real capabilities the Go service already exposes:

- intake and health
- identity-signal fingerprinting
- protected route validation
- full pipeline analysis and Stage 7 persistence
- analytics and batch reporting
- admin policy inspection

It avoids pretending that CodexSecure owns:

- browser login
- user dashboard routes
- direct payment checkout
- generic application CRUD flows

## Implementation Strategy

### Phase 1

Do not mass-alias current capability handlers into zoned route modules.

Goal:
Avoid duplicating route authority before zone-native handlers exist.

### Phase 2

Promote only truly zone-native handlers into zoned modules.

Goal:
Make zoned routes canonical only where ownership is real.

### Phase 3

Add zone-module-specific middleware presets after canonical zoned handlers exist.

Recommended presets:

- `z1`: request identity, zone detect, execution sync
- `z2`: request identity, zone detect, method, tiu, analysis, execution sync
- `z4`: request identity, zone detect, method, tiu, analysis, execution sync
- `z5`: request identity, zone detect, method, tiu, analysis, execution sync
- `z9`: request identity, zone detect, method, analysis, execution sync
- `z12`: request identity, zone detect, method, tiu, analysis, execution sync

## Route Module Layout

Recommended files:

- `internal/server/routes/zones/z1/routes.go`
- `internal/server/routes/zones/z2/routes.go`
- `internal/server/routes/zones/z4/routes.go`
- `internal/server/routes/zones/z5/routes.go`
- `internal/server/routes/zones/z9/routes.go`
- `internal/server/routes/zones/z12/routes.go`

Each module should only register the two endpoints assigned above.

## Notes

- Z2 here means identity-signal processing, not UI login ownership.
- Z5 remains the canonical home for the pipeline endpoints.
- Z12 admin policy endpoints should be treated as the strictest route group.
- No additional zone modules should be created until they map to real owned handlers.
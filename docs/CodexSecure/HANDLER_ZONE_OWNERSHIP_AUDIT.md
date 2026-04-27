# CodexSecure Handler Zone Ownership Audit

## Purpose

Classify the current `codexsecure_api` handler surface into:

- canonical zone-owned endpoints
- cross-cutting capability endpoints
- operational endpoints

This is the filter that should decide whether an endpoint belongs under a zone route module.

## Audit Rule

An endpoint is only zone-canonical if all of the following are true:

1. the endpoint's primary responsibility belongs to one zone authority
2. the route path does not misrepresent ownership by duplicating a cross-cutting tool API
3. the handler semantics remain correct when exposed as the canonical route for that zone

If those are not true, the endpoint should remain in a capability group such as `/policy`, `/analyzer`, `/fingerprint`, or `/pipeline`.

## Current Route Surface

Current API grouping in [api.go](/home/web3codex/projects/Docker/codexsecure_api/internal/server/routes/api.go):

- `/zones`
- `/policy`
- `/analyzer`
- `/fingerprint`
- `/pipeline`

Current zone modules:

- [z2/routes.go](/home/web3codex/projects/Docker/codexsecure_api/internal/server/routes/zones/z2/routes.go)
- [z5/routes.go](/home/web3codex/projects/Docker/codexsecure_api/internal/server/routes/zones/z5/routes.go)

Both are registration stubs today, not canonical functional modules.

## Classification

### ZoneHandler

Source: [zones.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/zones.go)

Endpoints:

- `ListZones`
- `GetZone`
- `DetectZone`
- `ZoneHealth`

Classification: cross-cutting platform capability

Reason:

- these describe the zone system itself, not one zone's owned behavior
- `DetectZone` is especially cross-cutting because it exists to determine zone alignment across the whole model
- `ZoneHealth` is fleet-wide, not Z1-owned or any other zone-owned capability

Verdict:

- keep under `/zones`
- do not move under `/zones/zN/...` as canonical endpoints

### PolicyHandler

Source: [policy.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/policy.go)

Endpoints:

- `GetSnapshot`
- `GetZonePolicy`
- `ListZonePolicies`
- `ValidateMethod`
- `GetZoneRequirements`

Classification: mostly cross-cutting admin/control-plane capability

Reason:

- policy is global and spans all zones
- `ValidateMethod` checks rule compatibility against any zone, so it is not owned by one zone module
- `GetSnapshot` is admin-facing and global, not a Z12 business object in the current implementation

Verdict:

- keep under `/policy`
- if a future Z12 admin module exists, it should own new admin-specific operations, not mirrored aliases of global policy endpoints

### AnalyzerHandler

Source: [analyzer.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/analyzer.go)

Endpoints:

- `AnalyzeRoutes`
- `AnalyzeSingleRoute`
- `DetectZone`
- `GetStrategies`
- `CheckRouteInZone`
- `GetAnalysisStats`

Classification: cross-cutting diagnostic and classification capability

Reason:

- analyzer logic exists to map requests into zones and compare routes against the full zone model
- these endpoints are tooling and inspection APIs across the whole system
- exposing them under one zone path would falsely imply local ownership

Verdict:

- keep under `/analyzer`
- do not zone-alias as canonical endpoints

### FingerprintHandler

Source: [fingerprint.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/fingerprint.go)

Endpoints:

- `Analyze`
- `GenerateRouteFingerprint`
- `GenerateInputFingerprint`
- `CalculateDrift`
- `FilterInput`
- `GetConfig`
- `QuickHash`
- `BatchAnalyze`

Classification: cross-cutting integrity and fingerprint capability

Reason:

- fingerprinting is used by multiple phases and multiple zones
- these endpoints are generic processing tools, not zone-native resources
- using a Z2 or Z9 path for them would be interpretive, not authoritative

Verdict:

- keep under `/fingerprint`
- only create zoned endpoints later if a future handler becomes explicitly zone-bound

### PipelineHandler

Sources:

- [pipeline.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/pipeline.go)
- [pipeline_analysis.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/pipeline_analysis.go)

Endpoints:

- `AnalyzeExecution`
- `CreateExecution`

Classification: canonical zone-adjacent system capability, partially eligible for zoned exposure

Reason:

- these handlers implement the 7-stage pipeline and Stage 7 recording, which is closer to the core execution authority than the diagnostic handlers
- they still operate across the zone model, but they represent an actual owned workflow of CodexSecure rather than a convenience tool
- if any current endpoints can honestly gain a canonical zoned home, this slice is the closest fit

Verdict:

- current canonical home can remain `/pipeline`
- a future Z5 system module could own native pipeline routes if the codebase formally decides that Z5 is the authoritative execution surface
- if that happens, the zoned route should replace the capability route, not merely mirror it

### HealthHandler and MetricsHandler

Sources:

- [health.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/health.go)
- [metrics.go](/home/web3codex/projects/Docker/codexsecure_api/internal/handlers/metrics.go)

Classification: operational surface

Reason:

- these are service health and observability concerns
- they are not part of zone ownership semantics

Verdict:

- keep outside zoned route authority

## Outcome Matrix

| Handler Surface | Current Nature | Zone-Canonical Today | Recommended Canonical Home |
|---|---|---:|---|
| ZoneHandler | global zone metadata | no | `/zones` |
| PolicyHandler | global control plane | no | `/policy` |
| AnalyzerHandler | diagnostic/classification | no | `/analyzer` |
| FingerprintHandler | shared integrity tooling | no | `/fingerprint` |
| PipelineHandler | core execution workflow | partial candidate | `/pipeline` now, possible future Z5 |
| Health/Metrics | operations | no | health/ops routes |

## Practical Conclusion

Today, the service is still capability-organized, not zone-organized.

That means the earlier idea of scaffolding six zone modules and mounting aliases to existing handlers is not the right default implementation path.

Why:

- it duplicates authority
- it overstates zone ownership
- it turns cross-cutting APIs into misleading zone resources

## Recommended Next Build Rule

1. Keep existing capability groups as canonical until a handler is truly zone-native.
2. Only create a zone route module when the module owns the business responsibility of that endpoint.
3. For the current codebase, the only realistic candidate for future zone-native promotion is the pipeline and Stage 7 execution slice.
4. If zone-token lifecycle becomes a first-class Go capability, it should be designed as a native system/Stage 7 surface, not copied as broad aliases into multiple zone folders.
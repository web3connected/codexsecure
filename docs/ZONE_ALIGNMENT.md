# Zone Registry Alignment

**Date:** 2026-10-09 (definition alignment added 2026-10-11; registry aligned with Web3Codex and aliases added 2026-10-25; Z0 renamed Boundary 2026-10-25, policy 3.1.0)
**Status:** Implemented in the Go source (`internal/policy/alignment.go`). The **local development** `codexsecure-api` container has served policy 3.1.0 since 2026-10-25, and Web3Codex submits `boundary` and aligns with it. Not deployed to production. See [Deployment status](#deployment-status) and [Web3Codex compatibility](#web3codex-compatibility).

CodexSecure and its client applications must agree on the zone registry: the identifier → key mapping for the Z0 Boundary and the twelve zones Z1–Z12. Alignment compares a client's registry with the policy registry the API is actually running. It does not create a second registry; the source of truth is `policy.DefaultZones()`.

## Three separate checks

These are different checks and must not be described as one another:

| Check | Owner | What it establishes |
|---|---|---|
| **Physical alignment** | Web3Codex | A route's zone comes from its zone directory (`routes/zones/z1`–`z12`), the identity dotfile in that directory (for example `.identity`), and the canonical registry. Resolved at route registration and kept in the route cache. |
| **Registry alignment** | Web3Codex SDK ↔ Go (this document) | Both systems agree on the zone identities (`POST /policy/alignment`) and, optionally, definitions (`POST /policy/alignment/definitions`). It compares registries; it never receives a route, method or communication. |
| **Request processing** | Seven-Fold Process | Progression through the seven stages. Stage 1 (Initialize) currently establishes physical and registry alignment. |

Registry alignment is **not** per-request authorization. An aligned registry says the two systems name the zones the same way; it does not say a given request belongs in, or may act in, its zone.

## Canonical registry

There are exactly twelve zones. Z0 is the **Boundary**: the entry boundary and the unassigned state for communication that has not been assigned to an operational zone or has not established a valid zone relationship. It is outside the zone structure, not a thirteenth zone. Go keeps thirteen technical identifiers (Z0–Z12) for compatibility, so registry responses list 13 entries (`total`, `zones_checked`). Those counts are identifiers, not zones.

Z0 is not in itself a quarantine or a malicious-traffic destination. Quarantine and rejection are separate security outcomes. Z0's rules (read-only, GET only, no auth) are unchanged by the rename.

Z1–Z12 are classes of internet traffic, not user roles or site departments.

| Identifier | Key | Name | Aliases (same identity) |
|---|---|---|---|
| Z0 | boundary | Boundary (not a zone) | lobby, negative |
| Z1 | public | Public | |
| Z2 | identity | Identity | |
| Z3 | operations | Operations | ops |
| Z4 | protected | Protected Data | |
| Z5 | processing | Processing | system |
| Z6 | logging | Logging | eventlogs |
| Z7 | communications | Communications | comms |
| Z8 | payments | Payments | — |
| Z9 | analytics | Analytics | |
| Z10 | streaming | Streaming | — |
| Z11 | synchronization | Synchronization | — |
| Z12 | administration | Administration | admin |

Fixtures in `testdata/zones/`:

| File | Content |
|---|---|
| `canonical_registry.json`, `canonical_definitions.json` | The agreed registry and definitions. They come from the Web3Codex SDK config with the Z0 Boundary entry. Used by the Go tests and `scripts/verify_zone_alignment.sh`. |
| `web3codex_registry.json`, `web3codex_definitions.json` | Web3Codex's config before 2026-10-25 (Z0 `lobby`, "Lobby / Gateway"). Kept to test the `lobby` alias. Web3Codex now matches `canonical_registry.json`. |

### Aliases and former names

An alias is an earlier key for the **same** zone and the same traffic class. It is recorded on the zone definition (`ZoneDefinition.Aliases`); there is no separate alias table.

- Z8, Z10 and Z11 have no aliases. Their former keys (`resourcesync`, `behavior`, `aimesh`) described different traffic (Resource Sync, Behavior Logic, AI/Agent Mesh), so they are not equivalent to Payments, Streaming or Synchronization. A registry still using them is a genuine mismatch.
- An alias matches only at its own zone ID. `"Z3": "admin"` is a mismatch, not an equivalent.
- Policy validation rejects an alias that equals another zone's key or alias, so a name always resolves to one zone.
- Lookups by key (`Registry.GetZoneByKey`, pipeline execution `zone_key`) accept aliases: `admin` and `administration` both resolve to Z12, and `boundary`, `lobby` and `negative` all resolve to Z0.
- Z0 keeps two aliases because existing clients use them: `lobby` is what the Web3Codex SDK submitted until 2026-10-25, and `negative` is the key the 2.0.0 API published (also used in `codex_admin`).

## Canonical signature — `sha256:zone-registry-v1`

1. Normalize each key: trim whitespace, lowercase.
2. Order zones by numeric zone ID ascending (Z0, Z1, …, Z10, Z11, Z12 — not string order).
3. Build one line per zone: `Z{id}:{key}`.
4. Join lines with `\n`. No trailing newline.
5. Signature = lowercase hex SHA-256 of the UTF-8 string.

For the canonical registry the signature is:

```
5eca793c1a786b864899595b2ae784a433e795efaa9a70c24cdcfba8ecbaaff5
```

Shell check: `printf 'Z0:boundary\nZ1:public\n…\nZ12:administration' | sha256sum`.

| Registry | Z0 key | Signature |
|---|---|---|
| Policy 2.0.0 (previous local build) | negative | `c767d194…cd9b` |
| Policy 3.0.0 (never deployed) | lobby | `dd2d6c35…960b`, Web3Codex's `local_signature` until 2026-10-25 |
| Policy 3.1.0 (this source, running locally) | boundary | `5eca793c…aff5`, also Web3Codex's current `local_signature` |

The signature is computed over keys exactly as named. Aliases are not substituted.

PHP equivalent (sort by integer zone ID, not by string key):

```php
$lines = [];
foreach ($registryByIntId as $id => $key) {   // ksort($registryByIntId, SORT_NUMERIC) first
    $lines[] = "Z{$id}:" . strtolower(trim($key));
}
$signature = hash('sha256', implode("\n", $lines));
```

This is the same form as `codex_admin`'s `ZoneAlignmentService::signature()` and the Web3Codex SDK's `ZoneAlignment::signature()` (`local_signature`).

## Endpoints

Both are in the existing `/api/v1/policy` group.

| Path | Direct (internal) | Through gateway |
|---|---|---|
| Alignment check | `POST http://codexsecure-api:8084/api/v1/policy/alignment` | `POST {CODEX_GATEWAY_URL}/secure/api/v1/policy/alignment` |
| Registry + signature | `GET http://codexsecure-api:8084/api/v1/policy/registry` | `GET {CODEX_GATEWAY_URL}/secure/api/v1/policy/registry` |

**Laravel should call the gateway URL.**

### Authentication

- **Gateway:** `X-API-Key: <tenant key>` is required (`401 AUTH_MISSING_KEY` / `AUTH_INVALID_KEY` otherwise). HMAC (`X-Signature`, `X-TIU`) and TIU-window checks also apply if the key record enables them.
- **API:** the `/api/v1/policy` group has no authentication of its own (same as `/policy/zones` and `/policy/snapshot`). Port 8084 is published on the host, so the internal path is reachable without a key from the host machine.

### `POST /api/v1/policy/alignment`

Request — the Laravel `codexsecure.zone_registry` config can be sent as is. Zone identifiers may be `"Z3"`, `"z3"` or `"3"`.

```json
{
  "zones": {
    "Z0": "boundary", "Z1": "public", "Z2": "identity", "Z3": "operations",
    "Z4": "protected", "Z5": "processing", "Z6": "logging", "Z7": "communications",
    "Z8": "payments", "Z9": "analytics", "Z10": "streaming", "Z11": "synchronization",
    "Z12": "administration"
  }
}
```

Response `200` — aligned:

```json
{
  "aligned": true,
  "status": "aligned",
  "policy_version": "3.1.0",
  "signature_algorithm": "sha256:zone-registry-v1",
  "signature": "5eca793c…aff5",
  "expected_signature": "5eca793c…aff5",
  "zones_checked": 13,
  "matched": ["Z0", "Z1", "Z2", "Z3", "Z4", "Z5", "Z6", "Z7", "Z8", "Z9", "Z10", "Z11", "Z12"],
  "differences": []
}
```

Response `200` — mismatch (a mismatch is a valid result, not an error). Example for a registry with Z8 submitted as its former key `resourcesync`, Z12 removed and Z13 added:

```json
{
  "aligned": false,
  "status": "mismatch",
  "policy_version": "3.1.0",
  "signature_algorithm": "sha256:zone-registry-v1",
  "signature": "5eca793c1a786b864899595b2ae784a433e795efaa9a70c24cdcfba8ecbaaff5",
  "expected_signature": "<signature of the registry as submitted>",
  "zones_checked": 14,
  "matched": ["Z0", "Z1", "Z2", "Z3", "Z4", "Z5", "Z6", "Z7", "Z9", "Z10", "Z11"],
  "differences": [
    { "zone_id": 8,  "zone_code": "Z8",  "type": "mismatched", "expected": "resourcesync", "actual": "payments" },
    { "zone_id": 12, "zone_code": "Z12", "type": "extra",      "expected": null,           "actual": "administration" },
    { "zone_id": 13, "zone_code": "Z13", "type": "missing",    "expected": "experimental", "actual": null }
  ]
}
```

Response `200` — submitted under an alias. A zone submitted as one of its aliases is listed in `matched`, and also in `equivalents`. Actual response from the 3.1.0 build to Web3Codex's former payload (Z0 `lobby`):

```json
{
  "aligned": true,
  "status": "aligned",
  "policy_version": "3.1.0",
  "signature_algorithm": "sha256:zone-registry-v1",
  "signature": "5eca793c1a786b864899595b2ae784a433e795efaa9a70c24cdcfba8ecbaaff5",
  "expected_signature": "dd2d6c35d024281381a814f44f3297d788553d310025a99f809b50b523b6960b",
  "zones_checked": 13,
  "matched": ["Z0", "Z1", "Z2", "Z3", "Z4", "Z5", "Z6", "Z7", "Z8", "Z9", "Z10", "Z11", "Z12"],
  "equivalents": [
    { "zone_id": 0, "zone_code": "Z0", "submitted": "lobby", "canonical": "boundary" }
  ],
  "differences": []
}
```

**Contract note:** `equivalents` is the only addition to this response. It is omitted when no alias was used, so every response for a registry that uses only canonical keys has the same 9 fields as before. When an alias is used, `signature` ≠ `expected_signature`. Each signature still describes its registry exactly as named. See [Web3Codex compatibility](#web3codex-compatibility) for how the current client treats this.

| `type` | Meaning |
|---|---|
| `mismatched` | Zone ID in both registries, keys differ |
| `missing` | Zone ID in the submitted registry, not in the API registry |
| `extra` | Zone ID in the API registry, not in the submitted registry |

`signature` describes the API registry and `expected_signature` the submitted one. `differences` is ordered by zone ID. `aligned` is true only when `differences` is empty.

Errors (convention `{ "error", "message" }`):

| Status | `error` | Cause |
|---|---|---|
| 400 | `invalid_request` | Body is not JSON or `zones` is absent |
| 422 | `invalid_registry` | Empty `zones`, bad identifier, empty key, or the same zone listed twice (`"Z3"` and `"3"`) |

### `GET /api/v1/policy/registry`

```json
{
  "policy_version": "3.1.0",
  "signature_algorithm": "sha256:zone-registry-v1",
  "signature": "5eca793c…aff5",
  "total": 13,
  "zones": [ { "zone_id": 0, "zone_code": "Z0", "key": "boundary" }, "…" ]
}
```

Use it to fetch and cache the API signature cheaply; use `POST /alignment` to get per-zone differences. `total` counts registry identifiers (the Boundary plus twelve zones).

## Definition alignment

Registry alignment only compares zone ID → key. Definition alignment compares what each zone *is*, against the API's own `policy.DefaultZones()` — the submitted definitions are never adopted.

### Compared fields (`compared_fields`)

| Field | Normalization |
|---|---|
| `key` | trimmed, lowercased; an alias of the zone's key compares as that key |
| `name`, `description`, `role` | trimmed; otherwise exact. `role` is set for the Z0 Boundary only; absent = `""` |
| `read_only` | boolean, **required** in submissions |
| `methods_allowed` | uppercased, de-duplicated, sorted (order does not matter) |
| `executive` | boolean; absent = `false` |
| `subzones` | absent, `null`, `[]` and `{}` are all `[]`; otherwise canonical JSON (object keys sorted) |

Not compared: `tier`, `auth_required`, `tiu_required`. They remain Go enforcement rules for the Sevenfold pipeline (Stage 5); the SDK model leaves authentication and synchronization to CodexAuth/CodexTime.

### Canonical definitions — `sha256:zone-definitions-v1`

1. For each zone, build the contract in this field order: `key, name, description, role, read_only, methods_allowed, executive, subzones`, normalized as above.
2. Encode it as compact JSON without escaping `/`, `&`, `<`, `>` or non-ASCII characters (PHP: `JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE`; Python: `separators=(",", ":"), ensure_ascii=False`).
3. One line per zone, `Z{id}:{json}`, ordered by numeric zone ID; join with `\n`, no trailing newline.
4. Signature = lowercase hex SHA-256.

Example line: `Z7:{"key":"communications","name":"Communications","description":"Messages, notifications, alerts, and interactive communications. Distinct from CodexSecure Communication Objects, which govern approved interfaces across zones.","role":"","read_only":false,"methods_allowed":["GET","OPTIONS","POST"],"executive":false,"subzones":[]}`

Canonical definitions signature (verified identical in Go, Python and PHP, 2026-10-25):

```
1316311da4960d6b633e81b35e47deb304e922cc1a6913916a1b8256b450589c
```

Fixture: `testdata/zones/canonical_definitions.json`. Web3Codex's current definitions (`web3codex_definitions.json`) differ only at Z0: `name`, `description` and `role` still describe "Lobby / Gateway". Its `lobby` key is an alias and compares as `boundary`.

### What a signature does and does not prove

Equal signatures mean the two definition sets are identical. A signature is not authentication (the gateway `X-API-Key` is), and it is not evidence that a route's behaviour belongs in its declared zone — that is the Sevenfold pipeline's intent analysis (Stage 4–6), a separate check.

### `POST /api/v1/policy/alignment/definitions`

Request — each zone in the SDK definition shape, including its registry `key`:

```json
{
  "zones": {
    "Z0": {
      "key": "boundary",
      "name": "Boundary",
      "description": "Entry boundary and unassigned state for communication that has not been assigned to an operational zone or has not established a valid zone relationship.",
      "role": "Outside the twelve operational zones; not an operational destination. Quarantine and rejection are separate security outcomes.",
      "rules": { "read_only": true, "methods_allowed": ["GET"] },
      "subzones": []
    },
    "…": "Z1–Z12"
  }
}
```

Response `200`:

```json
{
  "aligned": false,
  "status": "mismatch",
  "policy_version": "3.1.0",
  "signature_algorithm": "sha256:zone-definitions-v1",
  "signature": "<API definitions signature>",
  "expected_signature": "<submitted definitions signature>",
  "compared_fields": ["key", "name", "description", "role", "read_only", "methods_allowed", "executive", "subzones"],
  "zones_checked": 13,
  "matched": ["Z0", "…"],
  "differences": [
    {
      "zone_id": 9, "zone_code": "Z9", "type": "mismatched",
      "fields": [ { "field": "methods_allowed", "expected": ["GET", "OPTIONS", "POST"], "actual": ["GET", "OPTIONS"] } ]
    }
  ]
}
```

`type` is `mismatched` (with one `fields` entry per differing field), `missing` (submitted, not in the API) or `extra` (in the API, not submitted); the latter two have `fields: []`.

Errors: `400 invalid_request` (not JSON, `zones` absent, wrong types) and `422 invalid_definitions` (bad identifier, duplicate zone, empty key or name, `read_only` absent, no or invalid methods, invalid subzones JSON).

### `GET /api/v1/policy/definitions`

Returns `policy_version`, `signature_algorithm`, `signature`, `compared_fields`, `total` and `zones: [{zone_id, zone_code, definition}]` ordered by zone ID.

Definition alignment does not change the registry endpoints or their response shape.

## Gateway response envelope

The gateway wraps successful upstream responses:

```json
{ "status": "success", "service": "codexsecure", "tiu": null, "request_id": "…", "data": { "aligned": true, "…": "…" } }
```

Non-2xx upstream responses keep their status code, but the gateway replaces the body with `{"status":"error","service":"codexsecure","error":{"code":"UPSTREAM_ERROR",…}}`. Through the gateway, a 422's `message` is not visible to the client.

## Unavailable vs mismatch

A client must keep these apart:

| Outcome | How to recognise it | Meaning |
|---|---|---|
| **Aligned** | HTTP 200, `data.aligned === true` | Registries match |
| **Mismatch** | HTTP 200, `data.aligned === false` | API answered; registries differ — see `differences` |
| **Rejected request** | HTTP 400/422 | Client sent a malformed registry |
| **Unavailable** | Connection error, timeout, HTTP 401/403 (gateway auth), 404 (API without this endpoint), 5xx, or a 200 without a boolean `aligned` | Alignment unknown — never treat as aligned or as a mismatch |

`scripts/verify_zone_alignment.sh` follows this: exit 0 aligned, 1 mismatch, 2 unavailable/rejected.

```bash
scripts/verify_zone_alignment.sh                                    # internal :8084
ALIGNMENT_BASE_URL=http://localhost:8090/secure CODEX_API_KEY=… \
  scripts/verify_zone_alignment.sh                                  # through the gateway
```

## Web3Codex compatibility

Verified 2026-10-25 by sending Web3Codex's exact request body (its config, `ksort(SORT_NATURAL)`) to a 3.1.0 build in a separate container. The response was then checked with Web3Codex's own `ZoneAlignment::validResult()`.

| Go build | Web3Codex submits Z0 as | Go result | Web3Codex `validResult()` | Stage 1 effect in Web3Codex |
|---|---|---|---|---|
| 2.0.0 | `lobby` (before 2026-10-25) | mismatch, 9 differences | valid | Z1, Z2, Z4, Z9 continue; Z3, Z5–Z8, Z10–Z12 stop |
| 3.1.0 | `lobby` | aligned, Z0 listed in `equivalents` | **invalid** | every zoned route records `invalid_alignment_response` → `failed`; Receive skipped |
| 3.1.0 (local container) | `boundary` (Web3Codex since 2026-10-25) | aligned, 13 matched, signatures equal | valid | **verified:** Initialize continues to Receive for Z1, Z2 and Z12; aligned result cached |

`validResult()` was executed against real responses. On 2026-10-25 Web3Codex was switched to `boundary` and Initialize was run against the local 3.1.0 container through the gateway. Z1 (`/pages/home`), Z2 (`/login`) and Z12 (`/admin/blog/categories`) continued to Stage 2. The real responses are stored as Web3Codex test fixtures (`packages/web3codex/codexsecure/testdata/go-policy-alignment-3.1.0.json`). Details are in `docs/codexsecure/api/STAGE1_ZONE_ALIGNMENT.md` in Web3Codex. The `lobby` row still applies, so Web3Codex must not return to `lobby`.

Why Go's "aligned" answer to `lobby` is rejected: Web3Codex rebuilds Go's `signature` from the names it submitted for each matched zone, and for an aligned result it requires `signature` to equal its own `local_signature`. A Z0 matched under the alias `lobby` can never reproduce `5eca793c…`. The same check also rejects a *mismatch* response while Z0 is matched under the alias. So while Web3Codex submits `lobby`, no response from the 3.1.0 build validates. Go does not hide this by reporting the submitted registry's signature as its own.

Web3Codex changes (made in Web3Codex on 2026-10-25, not in this repository):

- `codexsecure.zone_registry['Z0']` → `boundary`. Its `local_signature` is `5eca793c…aff5`.
- `zone_definitions['Z0']` name, description and role → the Boundary definition above.
- **Cache:** Web3Codex keys cached alignment by gateway URL, API key and `local_signature`, and caches only validated aligned results. Changing Z0 changes the key, so stale entries are not reused. Nothing has been cached against the deployed 2.0.0 API, because it never returned aligned.
- **Order (production):** Web3Codex must submit `boundary` no later than the 3.1.0 production deployment. A 3.1.0 API with a `lobby` client fails Stage 1 for every zoned route.

## Deployment status

Local development, 2026-10-25: `codexsecure-api` was rebuilt from this source and restarted on its own (`docker compose build codexsecure-api` then `up -d --no-deps codexsecure-api` in the `codexweb3` project). Environment, `codex-net`, port `127.0.0.1:8084` and the database connection are unchanged, and no other container was recreated. It serves policy **3.1.0** (Z0 `boundary`, signature `5eca793c…aff5`, `total` 13). The previous 2.0.0 image is tagged `codexsecure-api:pre-boundary-20261025` for rollback.

At startup the rebuilt API logs that `security_zones` and `request_logs` do not exist in `codex_zones`. These are existing schema gaps, not caused by the rebuild. The startup log also still reports app `version 2.0.0`, which is the app version string and separate from the policy version.

**Production:** not deployed. Production alignment is not complete.

## Remaining work

- Production deployment of 3.1.0 (requires separate authorization), then re-run `scripts/verify_zone_alignment.sh` through the gateway.
- Laravel does not yet call `POST /secure/api/v1/policy/alignment/definitions`.
- Decide enforcement on mismatch/unavailable (strict, warn, off). The Laravel package currently reports only.
- No subzone catalogue exists on either side; both send/hold empty subzones.

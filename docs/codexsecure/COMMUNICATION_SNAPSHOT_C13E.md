# Approved Communications Snapshot (C13E)

**Date:** 2026-11-02
**Sides:** CodexSecure Go API, Codex Gateway, `codex_zones` migration (kept in Web3Codex, as all `codex_zones` migrations are)
**Status:** Implemented in source. The migration is **not applied** and no container has been rebuilt, so the endpoint is not live.

This is the contract Web3Codex synchronizes against. CodexHash provides integrity; CodexSecure determines authorization.

## 1. Endpoint

| Layer | Path |
| --- | --- |
| Gateway | `GET /secure/api/v1/sites/{site_key}/communications/snapshot` |
| Go API | `GET /api/v1/sites/{site_key}/communications/snapshot` |

Access is the same as the other communication reads:

- The Gateway authenticates the application and checks its site grant (`GATEWAY_APP_SITES`).
- The Gateway signs a read assertion bound to the site, method and path.
- Go verifies the assertion (`GatewayAssertion`) and the service key (`CommunicationAuth`).

There is no public route.

## 2. Response

`200` uses the existing `{data: …}` envelope:

```json
{
  "data": {
    "site_key": "web3codex-local",
    "snapshot_version": 9,
    "snapshot_generated_at": "2026-11-02T15:26:42Z",
    "expires_at": "2026-11-02T15:31:42Z",
    "snapshot_hash": "<128 lowercase hex>",
    "communications": {
      "universal": [ <communication> ],
      "pages": [ { "page_key": "about", "communications": [ <communication> ] } ],
      "admin": []
    }
  }
}
```

`<communication>`:

```json
{
  "communication_uuid": "aaaaaaaa-0000-4000-8000-000000000001",
  "scope": "universal",
  "page_key": null,
  "component_key": null,
  "type": "link",
  "destination": "/contact?a=1&b=2",
  "destination_match": "exact",
  "methods": ["GET"],
  "zone": "Z1",
  "enabled": true,
  "approved_revision": 2,
  "codex_hash": "<128 lowercase hex>"
}
```

- All three scope keys are always present, and each is an empty array when nothing applies.
- `admin` is always empty today because `codex_zones` allows only `universal` and `page` scope.
- Communications are ordered by `communication_uuid`, and `pages` by `page_key`, both in byte order.
- Timestamps are UTC whole seconds.

| Field | Meaning |
| --- | --- |
| `snapshot_generated_at` | When this version was published. Stable across reads of the same version |
| `expires_at` | This read's time plus the TTL (`COMMUNICATION_SNAPSHOT_TTL_SECONDS`, default 300). It moves forward on every read, because each read confirms the version is still current |

Failures use the existing `{error, code}` envelope:

| Status | Code | Meaning |
| --- | --- | --- |
| 401 / 403 | `SERVICE_KEY_INVALID`, `ADMIN_UNAUTHORIZED` | Not authorized for this site |
| 404 | `SITE_NOT_FOUND` | Site is not enrolled |
| 503 | `CODEXHASH_UNAVAILABLE` | The configuration changed and could not be hashed. An outdated version is never served in its place |
| 503 | `SNAPSHOT_UNAVAILABLE` | The configuration kept changing across three attempts; retry |

## 3. What is included

Discovery observations are never read. Inclusion starts from the stored approvals and applies the existing `Evaluate`, with the approved configuration standing in for an observation. The C13D destination-zone rule therefore applies without being copied or changed.

| Stored state | In the snapshot |
| --- | --- |
| Evaluates to `approved` | Yes, `enabled: true` |
| Disabled (decision or `enabled: false` in the approved document) | Yes, `enabled: false`. **It must not render** |
| Draft only, rejected, revoked, unapproved | No |
| Structure not valid (`suspended`) | No |
| Structure changed since approval (`reapproval_required`) | No |
| Approved zone differs from the destination's route zone, or destination has no assigned zone (`unauthorized`) | No |

The last row also covers every **cross-origin destination**, because `destinationZone` assigns no zone outside the site's origin. Each left-out approval logs a warning with its UUID when a version is published.

## 4. Versioning

State lives in `codex_zones`:

- `communication_snapshots`: one row per site with the version, hash and publication time.
- `communication_snapshot_items`: the canonical text and CodexHash of each included configuration.

On each read, Go builds the canonical text of every included configuration and compares it with the stored items.

- **Same:** the stored version and hash are returned with no CodexHash call.
- **Different:** the version increases by one.

A version therefore changes on approval, revocation, disable, replacement of an approved revision, a structural change, or any other change to the included content. Several changes between two reads produce one new version, so versions can skip intermediate states but never repeat or decrease.

Publication happens during the read, because hashing needs the CodexHash HTTP service and the database cannot call it:

1. Go reads the source and hashes outside any transaction.
2. Go calls `codex_communications_snapshot_publish`, a `SECURITY DEFINER` function. Under a per-site advisory lock, it requires the expected version and an identical source.
3. On `CX409` (another reader or a decision came first), Go reads again, up to three attempts.

The runtime role has `SELECT` on both tables and cannot write them directly. A trigger refuses any version that does not increase. The down migration refuses once a snapshot exists.

The first read of an enrolled site publishes version 1, even with no communications.

## 5. CodexHash canonicalization

**Operation:** `POST /api/hash/payload` with `algo: sha512`, `hash_size: 512`. It is stateless and fails closed, with no local fallback. The value is the lowercase hex SHA-512 of the canonical text's UTF-8 bytes.

**Text rules:** `codexsecure-protect-payload-v1` (`internal/protect/payload.go`):

- Object keys sorted by byte order at every depth; array order preserved.
- Compact JSON with no whitespace.
- Strings not escaping `/`, `&`, `<`, `>` or non-ASCII. U+2028 and U+2029 are escaped.
- Nulls written as `null`, never omitted.
- Integers written as plain integers.

In PHP this is `json_encode($value, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE)` over arrays built in the key order below.

### Communication hash (`codex_hash`)

Keys in exactly this order. `site_key` is the snapshot's top-level value, and the remaining values come from the communication:

```text
approved_revision, communication_uuid, component_key, destination, destination_match,
enabled, methods, page_key, representation, scope, site_key, type, zone
```

`representation` is the literal `codexsecure-approved-communication-v1`. `codex_hash` itself is not part of the input. Example:

```text
{"approved_revision":1,"communication_uuid":"aaaaaaaa-0000-4000-8000-000000000001","component_key":null,"destination":"/contact?a=1&b=2","destination_match":"exact","enabled":true,"methods":["GET"],"page_key":null,"representation":"codexsecure-approved-communication-v1","scope":"universal","site_key":"probe-site","type":"link","zone":"Z1"}
```

SHA-512: `5961b1d17aa309660ae34aa717b571d351e1ec7ba9287d3e555e847e07c9d19887d6a2c2813c77f16ab3d8b4b7d05f4368d67eab69bbddb28828b8d53388c992`

### Snapshot hash (`snapshot_hash`)

```text
{"communications":[{"codex_hash":…,"communication_uuid":…},…],"representation":"codexsecure-approved-snapshot-v1","site_key":…,"snapshot_version":…}
```

- `communications` holds **every** communication from all scopes, merged into one list and ordered by `communication_uuid` in byte order. Each entry has only those two keys.
- An empty site uses `"communications":[]`.
- `snapshot_version` is included, so the version cannot be relabeled.

**No timestamp is part of either hash.** `snapshot_generated_at` and `expires_at` rely on the authenticated Gateway response, not on a hash.

## 6. Web3Codex synchronization requirements

- Accept a snapshot atomically: replace all of the site's rows and its sync state together. **A communication absent from the snapshot is removed locally.**
- Verify all of the following before accepting, and reject the whole snapshot on any failure, keeping the previous one:
  - each hash matches `^[a-f0-9]{128}$`;
  - each `codex_hash` recomputes;
  - `snapshot_hash` recomputes;
  - `site_key` is the requested site.
- Compare versions against the accepted one:

| Incoming version | Action |
| --- | --- |
| Lower | Reject |
| Same, same hash | Refresh `expires_at` and `last_successful_sync_at` only |
| Same, different hash | Reject and treat as an integrity failure |
| Higher | Accept. Versions may skip numbers |

- Never render a communication with `enabled: false`.
- Flatten `pages[].communications` for storage; each row's `page_key` is already present.
- Treat scope and type as extensible strings, as the local schema does.
- On an error response, keep the accepted snapshot only until its `expires_at`.

Column mapping to `codex_communication_approvals` is one to one by field name.

Polling more often than the TTL is required to avoid expiry. The polling interval also sets how long a revocation can take to reach Web3Codex.

## 7. Files

| Project | File | Change |
| --- | --- | --- |
| Go API | `internal/communications/snapshot.go` | New: generation, evaluation, canonical forms, hashing, publication |
| Go API | `internal/communications/postgres.go` | Store fields for the hasher and TTL |
| Go API | `internal/handlers/communications.go` | `GET /snapshot` route and handler |
| Go API | `cmd/server/main.go` | Enables snapshots on the store |
| Gateway | `src/communications.ts` | Allows `/snapshot` as a read route |
| Web3Codex | `database/migrations/codex_zones/2026_11_02_000000_communication_approved_snapshots.php` and its `up`/`down` SQL | New tables, guard, source and publish functions, grants |

## 8. Before it can go live

1. Review and apply the migration with `php artisan codexzones:migrate migrate`, after a backup. Until then the endpoint returns `500 INTERNAL_ERROR`.
2. Rebuild and restart `codexsecure-api` and `codex-gateway`.

## 9. Known limitations

- **Existing approvals may drop out.** An approval whose stored zone does not match its destination's route zone is excluded by the C13D rule, as is every cross-origin destination. Check the published warnings after the first read.
- **Laravel still reads `zone` differently.** `PageCommunicationComparison::object()` compares the approved `zone` with the request's effective zone, while Go and this snapshot treat it as the destination's zone. Settle this before the frontend renders from the local snapshot.
- **A version is created on read.** Nothing advances the version until something reads the snapshot, so polling drives it. No reader can observe an outdated version as current.
- **First publication cost.** Each new or changed configuration takes one CodexHash call, made in sequence, inside the Gateway's 10-second upstream timeout. Later reads make no calls.
- **Authenticity.** Hashes detect changed data. Authenticity rests on the authenticated Gateway response; the snapshot is not signed.
- **Admin scope and Nav type** are not schema values yet. Nav authorization stays item-based, one `link` per item.

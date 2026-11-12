# Communication Channels — C2B → C3A Handoff: Protection and Activation Contracts

Date: 2026-10-28. Read-only audit; no source, migration, test or configuration was changed.
Audience: implementers of C3A (channel activation) in Web3Codex, and the matching Go work.
Predecessor: [COMMUNICATION_CHANNEL_MANAGER_IMPLEMENTATION.md](COMMUNICATION_CHANNEL_MANAGER_IMPLEMENTATION.md).

Every section separates **Existing** (verified in the source at the path given) from **Proposed** (not implemented).
Paths are relative to `Docker/CodexWeb3/` unless they start with `/mnt/c/laragon/www/web3codex` (Web3Codex).

---

## 0. Summary for C3A

| # | Finding | Consequence for C3A |
| --- | --- | --- |
| 1 | The CodexTime TIU reaches Go as a JSON number, is decoded into `float64`, and the CodexTime client **fails open** by default | Do not use `codextime.Client.CurrentTIU` for activation unchanged. A lossless, fail-closed TIU read is a Go prerequisite |
| 2 | `codexhash.Client.PayloadHash` fails closed. Its result is plain SHA-512 of the input, unauthenticated | Protection is *reproducible* (anyone can recompute it) but not *attributable* to CodexHash. The database can and should recompute it |
| 3 | The guard trigger rejects `pending → active` combined with a deadline change in one UPDATE (`CX409`) | Activation either uses two UPDATEs in one function, or the guard is amended (§4.4) |
| 4 | Nothing installs the `CommunicationPrincipal`, and the gateway forwards neither `X-Service-Key` nor any identity | Today **no caller can reach the channel routes**, through the gateway or directly. Gateway-to-Go identity transport is an unbuilt prerequisite |
| 5 | The gateway replaces every non-2xx upstream body with `UPSTREAM_ERROR` and wraps 2xx bodies | `CHANNEL_*` error codes and `data` do not survive the gateway today |
| 6 | `protection_tiu NUMERIC(20,6)` fits the legacy TIU, but not the CodexTime V1 TIU (decimal strings of 18–52 digits) | Decide the TIU scheme before C3A freezes the activation signature (§1.4) |

---

## 1. CodexTime TIU contract

### 1.1 Existing: server (`codextime_api`, production interim service, port 50002)

`GET /api/time/current-tiu` → `codextime_api/src/routes/time_routes.py:14` → `TimeService.get_current_tiu()`
(`codextime_api/src/services/time_service.py:86`). Unauthenticated. No signature.

```json
{"tiu": 214612574.695765, "timestamp": "2026-10-19T22:36:14.695801Z", "epoch": "2020-01-01 00:00:00", "cached": false}
```

| Field | Type on the wire | How it is produced |
| --- | --- | --- |
| `tiu` | **JSON number** (Python `float`) | `calculate_tiu()` (`time_service.py:64`): `floor(elapsed_seconds_since_2020-01-01 / tiu_interval) + round(now.microsecond/1e6, 6)`, then `round(…, 6)`. `tiu_interval = 1.0`, `tiu_precision = 6` (`src/config.py:37-39`). Uses `datetime.utcnow()` (host wall clock) |
| `timestamp` | string | `datetime.utcnow().isoformat() + "Z"`, taken **separately** from `tiu` |
| `epoch` | string | `"2020-01-01 00:00:00"` (not RFC 3339) |
| `cached` | bool | `true` when served from Redis (`TIU_CACHE_TTL=1`): `float(cached)` of `str(tiu)`. A cached `tiu` can be up to 1 s older than the freshly generated `timestamp` |

Precision: in practice the value is "seconds since 2020-01-01 UTC, with microseconds". It currently has 9 integer digits
and 6 decimals (15 significant digits).

### 1.2 Existing: CodexTime V1 (restoration track, not cut over)

Contract `codextime-tiu-api/1`, `GET /v1/tiu/snapshot` (X-Api-Key required). Source:
`docs/CodexTime/CODEXTIME_V1_L2_IMPLEMENTATION_REPORT.md` §2 and `docs/CodexTime/contracts/codextime-tiu-api-1.schema.json`.

- Every integer is a **decimal string**. Example: `"values": {"ns": "214612574695765961", "qm": "39807…439638"}` (52 digits).
- The scheme is `codextime-tiu-2020/1`, with an integer base of 1e-50 s, floor rounding, and units `s ms us ns ps fs as qm`.
- `coordinate.base` is a 59-digit integer string. Calibration state, uncertainty and `accuracy_verified: false` are reported.

This TIU fits neither `float64` (more than about 15.9 significant digits) nor `NUMERIC(20,6)`.

### 1.3 Existing: Go client path and every `float64` conversion

| Step | Location | Conversion |
| --- | --- | --- |
| 1 | `pkg/codextime/client.go:30` `TIUResponse.TIU float64` | Field type |
| 2 | `pkg/codextime/client.go:102` `json.Unmarshal(body, &result)` | **JSON number → `float64`** (nearest double) |
| 3 | `internal/handlers/protect.go:99-104` `IssueTIU` | `c.JSON(… "tiu": result.TIU …)`: **`float64` → JSON number** (Go shortest representation, so trailing zeros are dropped: `…574.5`, not `…574.500000`) |
| 4 | `internal/middleware/tiu.go:205-211` | `c.Set("codex_tiu", tiuResp.TIU)`: `float64` in the request context |
| 5 | Upstream, `time_service.py:84,96` | `round()` on a Python float, and `float(cached)`: float → float |

The client never treats the TIU as a decimal. The only lossless TIU path in Go is **PageMeta**:
`internal/handlers/pagemeta.go:48` (`json.Number`), validated by `tiuPattern` `^(0|[1-9][0-9]{0,13})(\.[0-9]{1,6})?$`
(`:21`, `:197-199`) and stored as text into `numeric(20,6)` (`internal/database/pagemeta.go:25,109-113`).
That TIU is **client-supplied and never checked against CodexTime**.

Numeric impact today: below 2^33 s (about 272 years after 2020), a double resolves better than 1e-6, so a legacy 6-decimal
TIU survives `float64` once it is re-formatted to 6 decimals. The risks are format instability (step 3) and the V1 scheme,
which cannot survive at all.

### 1.4 Proposed: lossless representation for channel protection

1. **The TIU travels as a decimal string end to end.** Go decodes it with `json.Number` or `json.RawMessage`, validates it
   against a scheme-specific regex, and passes it to PostgreSQL as text cast to `numeric`. There is never a
   `float64`/`float` step, in Go, PHP or SQL.
2. **Name the scheme next to the value.** A bare number cannot tell a legacy TIU (seconds.µs since 2020) from a V1 integer
   in some unit. Proposed C3A columns, replacing the reserved `protection_tiu NUMERIC(20,6)` before any row uses it:

   | Column | Type | Legacy interim value | V1 value |
   | --- | --- | --- | --- |
   | `protection_tiu` | `NUMERIC` (unconstrained precision), `CHECK (protection_tiu > 0)` | `214612574.695765` | `214612574695765961` |
   | `protection_tiu_scheme` | `VARCHAR(64)`, `CHECK` against a closed list | `codextime-legacy-seconds-2020/µs` | `codextime-tiu-2020/1` |
   | `protection_tiu_unit` | `VARCHAR(4)` | `s` | `ns` (or the chosen unit) |

   The all-or-none protection constraint and the immutability guard must include the new columns.
3. If C3A must stay on the legacy service only, `NUMERIC(20,6)` is sufficient, provided the value is carried as a string.
   Record that the decision is legacy-only, because a V1 cut-over would then need a migration.

**Decision required (CodexTime owner):** which service and scheme channel activation uses. Recommendation: V1
`ns` decimal string once V1 is cut over; legacy is acceptable only as an explicitly labelled interim scheme.

---

## 2. CodexHash contract

### 2.1 Existing: Go method

`pkg/codexhash/client.go:202`

```go
func (c *Client) PayloadHash(ctx context.Context, data string, algo string, hashSize int) (PayloadHashResult, error)

type PayloadHashResult struct {      // :185
    PayloadHash string `json:"payload_hash"`
    Algo        string `json:"algo"`
    HashSize    int    `json:"hash_size"`
}
var ErrUnavailable = errors.New("codexhash unavailable") // :182
```

- Request: `POST {CODEXHASH_API_URL}/api/hash/payload` with `{"data": <string>, "algo": <string>, "hash_size": <int>}`.
  Base URL default `http://codexhash-api:8001`; timeout `CODEXHASH_TIMEOUT_SEC` (default 3 s). No authentication header.
- Server: `codexhash_api/src/routes/hash_tracking.py:371`. `hash_size` must be 256 or 512 (else 422).
  `resolve_payload_algo` (`:25`) upgrades `sha256`+512 to `sha512`. Only `sha256`/`sha512` are allowed.
  The result is `hashlib.sha512(data.encode()).hexdigest()` (`:41-43`), i.e. SHA-512 of the UTF-8 bytes, lowercase hex.
  Stateless: no event, no database write.
- Inputs used today (Protect and PageMeta): `algo = "sha512"`, `hashSize = 512` (`internal/handlers/protect.go:20-22`).
  Output: 128 lowercase hex characters.

### 2.2 Existing: errors and fallback

| Method | On disabled / transport / non-2xx / decode failure | Validates the response? |
| --- | --- | --- |
| `PayloadHash` | Returns `PayloadHashResult{}` and an error wrapping `ErrUnavailable`. **Never** a local hash, regardless of `CODEXHASH_FAIL_OPEN` | **No.** `algo`, `hash_size` and hex format are not checked. Missing fields decode as `""`/`0`; unknown fields are ignored |
| `Hash` / `QuickHash` (`:88`, `:135`) | With `CODEXHASH_FAIL_OPEN=true` (the default, see `docker-compose.yml:38`): **local SHA-256**, `Source: "fallback"`; disabled → `Source: "disabled"` | No |

### 2.3 Distinguishing a genuine service result

- **`PayloadHash`:** a nil error means the bytes came from an HTTP 2xx response at `CODEXHASH_API_URL`. A failure is always
  an error, and no fallback value exists. The **caller** must still require `Algo == "sha512"`, `HashSize == 512` and
  `^[0-9a-f]{128}$`, as `protect.go:67` and `pagemeta.go:99` already do (`pagemeta.go` does not check `HashSize`).
- **`Hash`:** check `Source == "api"`. Note that the client sets `Source = "api"` itself (`:129`) after any decodable 2xx,
  overwriting whatever the server sent. The fallback is 64 hex (SHA-256), so it can never satisfy the 128-hex channel constraint.
- **Cryptographically, nothing distinguishes them.** SHA-512 is public and unkeyed, and the endpoint is unauthenticated over
  plain HTTP on the Docker network. A CodexHash payload hash proves *integrity of a known input*, not *who computed it*.
  Verification therefore means recomputing it. **Never call `Hash`/`QuickHash` for channel protection.**

---

## 3. Channel activation payload (proposed; not implemented)

### 3.1 Content

Built **only from the database row** returned by `codex_channels_get`, never from client input:

```json
{"channel":"3f0e1c2a-8b4d-4e6f-9a1b-2c3d4e5f6a7b","opened_at":"2026-10-27T14:00:00.123456Z","representation":"codexsecure-channel-binding-v1","site":"web3codex-local","zone":"Z1"}
```

| Key | Source column | Format |
| --- | --- | --- |
| `representation` | constant | `"codexsecure-channel-binding-v1"`, the version of this layout |
| `channel` | `channel_key` | lowercase, hyphenated 36-character UUID |
| `site` | `sites.site_key` | exact stored text (≤128 characters) |
| `zone` | `initial_zone` | `Z1`–`Z12` |
| `opened_at` | `opened_at` (`timestamptz(6)`) | UTC, **exactly 6 fractional digits**, `Z` suffix: Go layout `2006-01-02T15:04:05.000000Z07:00` with the time in UTC; SQL `to_char(opened_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')` |

`codexhash` and `tiu` are excluded, which is consistent with the reserved fields of `protect.Representation`.
Opening context beyond these fields (requesting application, client address) is deliberately excluded: it is not stored
on the channel, and nobody could re-derive it later.

### 3.2 Canonicalization rules

Use the **existing** `protect.CanonicalPayload` (`internal/protect/payload.go:40`, representation
`codexsecure-protect-payload-v1`). No new rules are needed:

1. One JSON object; keys sorted by byte order; compact (no whitespace); duplicate keys rejected.
2. Strings re-encoded without escaping `/ & < >` or non-ASCII (U+2028/U+2029 escaped), i.e. PHP
   `JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE`.
3. All values are strings, so there are no number-formatting rules (this is why the TIU stays out of the hash).
4. Hash input: the UTF-8 bytes of the canonical text, sent as `data` to `PayloadHash(ctx, canonical, "sha512", 512)`.

Additional rules C3A must fix:

- **Timestamps must not come from any JSON serializer.** The database's `jsonb` emits `+00:00`, and Go's `time.Time` emits
  RFC 3339 with trailing zeros trimmed (see the C2B report, §8.3). Format `opened_at` explicitly as above.
- **`sites.site_key` is mutable** (`2026_10_20_235826_communications.up.sql:15` has only a touch trigger, `:213`).
  Renaming a site would break re-verification of every protected channel. Either make `site_key` immutable once it has
  channels, or drop `site` from the binding (the channel UUID already implies the site). **Decision required.**

---

## 4. Activation sequence

### 4.1 Existing pieces

- `channels.Manager.Get` (`internal/channels/manager.go`) gives the validated row.
- `protect.CanonicalPayload` and `codexhash.Client.PayloadHash` give a genuine, fail-closed SHA-512 (§2).
- `codextime.Client.CurrentTIU` is **not suitable as is** (float, fail-open; §1.3, §6).
- Schema: the guard allows `pending → active` (`…communication_channels.up.sql`, `codex_channels_guard`). The
  `active_protected_check` and `protection_complete_check` constraints require all three protection columns. **No function
  activates, and `codex_zones_runtime` has no UPDATE privilege.**

### 4.2 Proposed Go flow (`Manager.Activate(ctx, site, channel)`)

1. `Get` → it must be `found` and `pending`; otherwise return the mapped outcome. This is a fast path only: step 5 re-decides.
2. Build the binding from the row (§3) and canonicalize it.
3. `PayloadHash(ctx, canonical, "sha512", 512)`. Require a nil error, `Algo == "sha512"`, `HashSize == 512` and 128 hex.
   On any failure, stop: the channel stays `pending`.
4. Obtain the TIU through a **new fail-closed, lossless** CodexTime read (proposed `CurrentTIUExact(ctx) (TIU{Value, Scheme, Unit string}, error)`).
   It must decode the value as `json.Number`, never return a fallback, and reject a missing, zero or negative value.
   Whether a `cached: true` legacy value is acceptable is a decision.
5. A single call to `codex_channels_activate` (below), in its own short transaction.
6. Validate the returned channel exactly as C2B validates the other functions, plus `protection.codexhash` equal to step 3.

Steps 3–4 make network calls **outside** any database transaction. Step 5 must therefore re-check everything under the row
lock, because the channel may be closed or lapse in between.

### 4.3 Proposed database function (C3A, Web3Codex)

```sql
codex_channels_activate(
    p_site      text,     -- site_key from the authorized route
    p_channel   uuid,     -- channel identity
    p_binding   text,     -- canonical binding text from step 2
    p_codexhash text,     -- 128 lowercase hex from CodexHash
    p_tiu       text,     -- decimal string; cast to numeric inside, never float
    p_tiu_scheme text     -- only if §1.4 option 2 is adopted
) RETURNS jsonb  -- {result, channel}; SECURITY DEFINER; EXECUTE granted to codex_zones_runtime only
```

| Result | When | Row change |
| --- | --- | --- |
| `activated` | Was `pending`, not lapsed, all checks pass | `status='active'`, protection set, `protected_at = now()`, deadline reset (§4.4) |
| `already_active` | Already `active` (a concurrent activation won) | None |
| `closed` / `expired` | Final state, or lapsed (lazily expired by `codex_channels_locate`) | Lazy expiry only |
| `not_found` | Unknown channel, or another site's channel | None |

Raised: `CX422` for malformed inputs (hash format, TIU format/scheme, or a binding that is not a JSON object).
Proposed: `CX409` `Channel binding mismatch` when the binding does not describe this row or the hash does not match it.

Required checks inside the function, in order:

1. `codex_channels_locate(p_site, p_channel)`. This locks the row, joins on `site_key` (so another site's channel is
   `not_found`), and **expires a lapsed channel first**. Without it, the guard would allow activating a lapsed pending
   channel, because the guard does not check `expires_at` for `pending → active`.
2. Status gate: only `pending` proceeds.
3. Binding check: `p_binding::jsonb` must have exactly the keys of §3.1, and each value must equal the locked row
   (`channel_key`, `site_key`, `initial_zone`, `opened_at` formatted as in §3.1, and the representation constant).
4. Hash check: `encode(sha512(convert_to(p_binding, 'UTF8')), 'hex') = p_codexhash` (built-in since PostgreSQL 11; no
   pgcrypto). The database thereby verifies the protection itself instead of trusting Go or CodexHash, which §2.3
   makes necessary.
5. TIU: `p_tiu ~ '^[1-9][0-9]*(\.[0-9]+)?$'` (scheme-specific), then cast to `numeric`.
6. Update and return `codex_channels_view`.

### 4.4 Deadline reset (requirements open decision 3)

The guard allows `last_activity_at`/`expires_at` to change only when `OLD.status = 'active' AND NEW.status = 'active'`.
A single `UPDATE … SET status='active', last_activity_at=now(), …` therefore raises `CX409 Channel activity is only renewed`.
There are two options:

- **(a) Two statements in the function, with no guard change:** first `UPDATE … SET status='active', protection…,
  protected_at=now()`, then `UPDATE … SET last_activity_at=now(), expires_at=now()+codex_channels_inactivity_timeout()
  WHERE last_activity_at < now()`. The second passes the guard because the row is now `active`, not lapsed, and moves
  forward to `now()`. **Recommended**, because it reuses the renewal rule unchanged.
- **(b) Amend the guard** to also allow the activity change on `pending → active` when protection is set in the same
  statement.

Either way the new deadline is `now() + 30 minutes` in database time, and `protected_at = last_activity_at = now()`.
If decision 3 is "no reset", omit the second statement; the deadline then remains `opened_at + 30 min`.

### 4.5 Activation after expiry, closure or site mismatch

| Threat | Prevented by |
| --- | --- |
| Lapsed during the CodexHash/CodexTime calls | `codex_channels_locate` lazy expiry under the row lock → `expired` |
| Closed concurrently | Row lock; status gate → `closed` |
| Another site's channel | `locate` joins `sites.site_key = p_site` → `not_found`; Go route authorization (§5) |
| Binding or hash from another channel | Binding and hash checks against the locked row (§4.3, 3–4) |
| Double activation | Row lock; second call → `already_active`; protection is immutable (guard) |
| Re-activation of `closed`/`expired` | Guard: final states never change |

---

## 5. Gateway authorization

### 5.1 Existing: what Go requires (`internal/middleware/channel_auth.go`)

1. `X-Service-Key` equal (constant-time) to the Go process's `SERVICE_API_KEY`. If that is unset or empty, every request gets
   `401 SERVICE_KEY_INVALID`.
2. A `CommunicationPrincipal` (`internal/middleware/communication_auth.go`) stored in the Gin context under
   `codexsecure.communication.principal` **by Go middleware** via `SetCommunicationPrincipal`, with:
   - `GatewayAuthenticated: true`
   - `ChannelSites map[string]bool` where `ChannelSites[<site path segment>] == true`, compared exactly. `ReadSites`,
     `AdminSites` and `DiscoverySites` do **not** authorize channels.
3. Not required: `X-CodexSecure-Actor`, `Idempotency-Key` (both ignored on channel routes).

No HTTP header can create the principal. **No production code calls `SetCommunicationPrincipal`**: the only callers are tests.

### 5.2 Existing: what the gateway does (`codex_gateway/src`)

- It authenticates `X-API-Key` (`middleware/auth.ts`) as either an application key (`w3c_app_…` → `{type:'application', appId}`)
  or a tenant key (`{type:'tenant', tenantId, keyId}`). It has **no application-to-site mapping**.
- It proxies `/secure/*` to `codexsecure-api:8084`, stripping `/secure` (`services.ts:24-28`). It forwards **only**
  `content-type`, `x-request-id`, `x-forwarded-for` and `x-gateway` (`index.ts:73-79`): no `X-Service-Key`, and no identity.
  The identity stays in `request.authContext` and is never forwarded.
- It re-serializes the body (`JSON.stringify(request.body)`), so duplicate keys are collapsed before Go can reject them.
- It replaces every non-2xx upstream response with `wrapError(…, 'UPSTREAM_ERROR')` (`index.ts:96-101`), and wraps 2xx bodies
  as `{status:'success', service, data: <Go body>, …}` (`response.ts:5-16`).

**Result:** today the channel routes are unreachable for every caller (401 through the gateway; 403 when called directly
with the key, because no principal is installed).

### 5.3 Proposed: what must be built and supplied

| Party | Must supply / build |
| --- | --- |
| Web3Codex (application) | Its gateway application key (`X-API-Key: w3c_app_…`). Nothing else is trusted from it. Path `{site}` = its own `site_key`; body `{"zone": "Zn"}` |
| Gateway | (1) Configuration mapping `appId → {site_key…}` for channel access. (2) Inject `X-Service-Key` from its own secret store; never pass a client-supplied one. (3) Forward the verified identity and site grants as an **integrity-protected assertion** (mechanism undecided: an HMAC-signed header with a key separate from `SERVICE_API_KEY`, or mTLS), and strip any client-supplied copy. (4) Pass Go's 4xx bodies through unchanged for `/secure/api/v1/sites/*/channels*`, or the SDK cannot see `CHANNEL_*` codes and `data` |
| Go | Middleware that verifies the assertion and calls `SetCommunicationPrincipal` with `GatewayAuthenticated: true` and `ChannelSites` from the verified grants, registered before `ChannelAuth` |

### 5.4 Trusted vs client-controlled values

| Value | Status |
| --- | --- |
| `SERVICE_API_KEY` (Go environment), principal and grants (Go context) | Trusted, once 5.3 exists |
| Channel UUID, all times, deadline, status | Trusted: generated and decided by the database |
| CodexHash result | Trusted for integrity only after the database recomputes it (§4.3) |
| CodexTime TIU | Trusted as far as the unauthenticated internal network is (§6.3) |
| `{site}` path segment | Client-controlled; authorized against `ChannelSites`, and scoped by the database |
| `{channel}` path segment | Client-controlled; canonical-UUID validated; found only within `{site}` |
| `zone` body field | Client-controlled; validated by the database (`Z1`–`Z12`). It is a declaration, not verified against the zone the request actually came from |
| `X-CodexSecure-Actor`, `Idempotency-Key`, any other header | Ignored on channel routes |

---

## 6. Security findings

### 6.1 Fail-open paths (existing)

| # | Path | Effect |
| --- | --- | --- |
| F1 | `codextime.Client.CurrentTIU` with `CODEXTIME_FAIL_OPEN` default `true` (`client.go:49,126-131`; `docker-compose.yml:43`) | Errors become `{TIU: 0, Source: "unavailable"}, nil` |
| F2 | `CurrentTIU` with `CODEXTIME_ENABLED=false` (`:76-78`) | `{Source: "disabled"}, nil`, with no error **even when fail-open is off** |
| F3 | `CurrentTIU` sets `Source = "api"` after any decodable 2xx (`:105`) | A 2xx body without `tiu`, or with `"tiu": 0`, yields `Source: "api"`, TIU 0. A string `tiu` (e.g. V1-style) is a decode error → F1. Callers must check `TIU > 0` (`protect.go:91` does) |
| F4 | `TIUMiddleware` (`internal/middleware/tiu.go:204-211`) | It logs and continues on error, and stores `TIU 0 / "unavailable"` in context under fail-open. Downstream code must not treat `codex_tiu` as genuine |
| F5 | `codexhash.Client.Hash`/`QuickHash` with `CODEXHASH_FAIL_OPEN` default `true` | Local SHA-256 is substituted for a CodexHash result (`Source: "fallback"/"disabled"`) |
| F6 | Legacy `ServiceKeyMiddleware` (`service_key.go:24-28`) | With `SERVICE_API_KEY` unset, it is a no-op. Not used by channel routes (`ChannelAuth` denies instead) |
| F7 | CodexTime server `_send_event_hash` (`time_service.py:57-62`) | CodexHash event recording failure is logged and ignored |

Channel routes themselves have **no** fail-open path: unset key → 401; missing principal → 403; contract mismatch → 500.

### 6.2 Precision-loss risks

| # | Risk |
| --- | --- |
| P1 | JSON number → `float64` in `pkg/codextime/client.go:102`; re-encoded by Go's shortest formatting (`protect.go:100`). Lossless for 6-decimal legacy values today, but not format-stable |
| P2 | Python `round()` on floats and `float(str(tiu))` cache round trip in CodexTime (`time_service.py:82-84,96`) |
| P3 | V1 TIU strings (18–52+ digits) cannot pass through `float64` at all, and cannot fit `NUMERIC(20,6)` |
| P4 | Any PHP/SDK `json_decode` of the TIU produces a PHP float unless it uses `JSON_BIGINT_AS_STRING` or decimal strings (SDK not audited here) |
| P5 | Timestamp text differs between `jsonb` (`+00:00`, 6 digits) and Go (`Z`, trimmed). This is harmless for display, but fatal if either is hashed: hence §3.2 |

### 6.3 Unresolved trust-boundary questions

1. **Gateway → Go identity transport** (§5.3): mechanism, key management, replay protection, and the `appId → site` grant source.
2. **Go → CodexHash/CodexTime:** plain, unauthenticated HTTP on the Docker network. Anything able to answer for
   `codexhash-api`/`codextime-api` (DNS, network position) can supply values. The database recomputation (§4.3) covers the
   hash. **Nothing covers the TIU**: no TIU value is signed (the HMAC `tiu-token`, `time_service.py:160`, signs only
   `v/iat/exp` Unix seconds, not a TIU). Decide whether the TIU must be attested (e.g. a signed V1 snapshot) or is accepted
   on network trust.
3. **What the TIU is bound to:** it is stored next to the hash but not covered by it (§3.2). If the TIU must be
   tamper-evident, the binding needs a second, explicitly versioned representation that includes it. Do not reuse v1.
4. **Clock consistency:** `protected_at` and the deadline use PostgreSQL `now()`, while the legacy TIU uses the CodexTime host's
   `datetime.utcnow()`. The two are not cross-checked, and no tolerance is defined.
5. **`zone` is a client declaration** (§5.4). Whether activation must confirm it against the Seven-Fold zone of the
   activating request is undecided.
6. **`site_key` mutability** (§3.2).
7. **Gateway error rewriting** (§5.2) hides lifecycle results from applications. Without a fix, the SDK cannot distinguish
   `CHANNEL_CLOSED` from an outage.

---

## 7. Decisions needed before or during C3A

| # | Decision | Recommendation |
| --- | --- | --- |
| D1 | TIU service and scheme for protection, and the column form (§1.4) | Decimal string; `NUMERIC` with scheme and unit columns; V1 `ns` once V1 is cut over |
| D2 | Activation resets the deadline (open decision 3) | Yes, via option (a) in §4.4 |
| D3 | `site` in the binding vs `site_key` immutability (§3.2) | Make `site_key` immutable once channels exist |
| D4 | Database-side hash recomputation (§4.3, step 4) | Yes |
| D5 | Acceptance of a cached legacy TIU | No: require `cached: false`, or move to V1 |
| D6 | TIU attestation (§6.3, item 2) | Required before production; acceptable to defer for local C3A |
| D7 | Gateway identity transport and error pass-through (§5.3) | A prerequisite for any client use; separate gateway phase |

Go-side follow-ups after C3A (not part of this handoff): `CurrentTIUExact`, `Manager.Activate`, the principal-installing
middleware, and response validation for `activated`/`already_active`.

# CodexSecure — Feature Build Specification v0.1

> October 20, 2026 review: this remains a feature-design document. Its historical table-existence and module-status statements are not current database evidence. The live `codex_zones` schema currently has PageMeta storage only; harmonizer, confidence/approval and communication authorization targets must not be reported as completed. See `docs/CodexSecure/CODEXSECURE_STABILITY_REVIEW.md` in Web3Codex.

**Date:** 2026-05-08
**Status:** Design phase — not yet built
**Companion docs:** `ARCHITECTURE.md` (pipeline overview), `SETUP_LOG.md` (build history)
**Related spec:** `_docs/web3Infra/Web3Codex_Core_Spec_v0.2.md` (Sections 19–25)

---

## Purpose

This document defines the **feature layer** of CodexSecure — the 10 modules that turn the 7-stage pipeline into a self-building, pattern-aware, attack-surface mapping system.

The existing pipeline classifies and routes. These modules make it **learn**, **detect**, and **enforce** at the structural level before any app code runs.

---

## What Separates This From App-Level Security

Traditional security works like this:
1. Request arrives
2. App validates input
3. App checks auth
4. App proceeds or rejects

CodexSecure works like this:
1. Request arrives
2. **Infrastructure checks structural shape** — are the right field types present, at the right counts, with the right names?
3. **Infrastructure compares shape to pre-built design hash** — does this match what deploy-time scanning expected?
4. **Infrastructure confirms zone and transition validity** — is this the right zone for this route?
5. App gets clean, shape-verified data
6. App validates business logic (email format, password complexity, DB lookup)

The infra layer is the outer gate. It runs first. It does not replace the app layer — it reduces the attack surface the app layer ever has to see.

---

## Module Index

| # | Module | Core responsibility |
|---|---|---|
| 1 | [Zone Harmonizer](#1-zone-harmonizer) | Developer declares what each route is meant to move |
| 2 | [Route Vocabulary](#2-route-vocabulary) | Canonical route names → auto-configure entire infra stack |
| 3 | [Semantic Field Mapper](#3-semantic-field-mapper) | Field type + name → semantic category (identity, secret, token, etc.) |
| 4 | [Pattern Hash Engine](#4-pattern-hash-engine) | designHash vs dataHash — structural shape verification |
| 5 | [Confidence Array Builder](#5-confidence-array-builder) | Pre-built attack-surface map for every route, computed at deploy time |
| 6 | [Request Phase Validator](#6-request-phase-validator) | Per-request: lookup → hash → compare → on-phase or off-phase decision |
| 7 | [Zone Observation Stream](#7-zone-observation-stream) | Every request becomes a chained, tamper-evident observation |
| 8 | [Cross-Zone Correlation](#8-cross-zone-correlation) | TIU-linked observations across zone boundaries — anomaly detection |
| 9 | [Middleware / API Gateway](#9-middleware--api-gateway) | Zone chain notation, middleware stacking, admin gate |
| 10 | [Admin Security Dashboard](#10-admin-security-dashboard) | Surfaces confidence scores, off-phase rates, attack drift, zone stream |

---

## 1. Zone Harmonizer

### What It Is

The developer's single point of truth for what data a route is expected to move. One declaration at design time configures the entire infra stack for that route.

### The Problem It Solves

Without harmonization, every part of the infra stack has to independently guess what a route does. The middleware guesses. The pattern engine guesses. The classifier guesses. Guesses diverge.

The harmonizer removes the guessing. The developer declares once. Everything derives from that.

### Config Format

```ts
// harmonize.config.ts
export default {
  routes: [
    {
      path: '/login',
      purpose: 'authenticate',
      // Optional — inferred from purpose if not declared:
      // zone: 'Z2',
      // methods: ['GET', 'POST'],
      // moves: ['identity', 'secret', 'token?'],
    },
    {
      path: '/checkout',
      purpose: 'checkout',
      moves: ['payment_method', 'amount', 'billing_address'],
    },
    {
      path: '/superloginpage',
      purpose: 'authenticate',  // inherits everything from canonical 'authenticate'
    },
  ]
}
```

### Two-Layer Architecture

**Layer 1 — Global Zone Schema (Web3Codex, immutable):**
- The Z0 Boundary and Z1–Z12 zone definitions are fixed (twelve zones; Z0 is the Boundary, not a zone)
- Zone rules cannot be weakened by company config
- Zone chain rules are canonical (Z12.Z2 always means: admin context, identity section)

**Layer 2 — Company Harmonization Config (developer-owned):**
- Routes are declared in `harmonize.config.ts`
- Developers can assign purpose, override moves, add custom zone chains
- Cannot override zone security policy — only route declarations

```
Web3Codex Global Schema    Company Config
─────────────────────────  ──────────────────────────────
Z2: identity zone          /login → purpose: authenticate
  methods: GET, POST       /super-login → purpose: authenticate
  tier: business           /checkout → purpose: checkout
  auth_required: false       moves: [payment_method, amount]
  (cannot be changed)      (fully developer-controlled)
```

### Blockchain Anchoring (planned v0.2)

The harmonization config can be committed to a blockchain ledger at deploy time. The hash of the config becomes part of the confidence array. Any config drift triggers an integrity alert.

```
deploy → harmonize.config.ts → CodexHarmonicHash → anchor hash
                                                  → stored on-chain
                                                  → compared on startup
```

### API Contract

```
GET  /api/v1/harmonize/config          → current config snapshot
POST /api/v1/harmonize/validate        → validate a config before deploy
GET  /api/v1/harmonize/drift           → detect config changes since last anchor
```

---

## 2. Route Vocabulary

### What It Is

A global lookup table of recognized route names → purpose → auto-configured infra stack. The developer who uses `/login` pays zero cost. CodexSecure already knows what `/login` is and how to protect it.

### The Core Insight

Route names are already harmonized in the real world. Every developer independently arrives at `/login`, `/register`, `/dashboard`, `/checkout`. The convergence is real. CodexSecure makes that implicit convention explicit and enforceable.

### Canonical Route Table

| Route pattern | Purpose | Auto-zone | Auto-middleware | Auto-methods | Default data moves | Target zone |
|---|---|---|---|---|---|---|
| `/login`, `/signin`, `/auth` | `authenticate` | Z2 | `guest` | GET, POST | identity + secret + ?token | Z4 |
| `/admin/login`, `/admin/signin` | `authenticate` | Z12.Z2 | `guest` | GET, POST | identity + secret + ?token | Z12 |
| `/register`, `/signup`, `/create-account` | `register` | Z2 | `guest` | GET, POST | identity + secret + ?email | Z4 |
| `/forgot-password`, `/reset-password` | `reset_password` | Z2 | `guest` | GET, POST | identity + token | Z2 |
| `/logout`, `/signout` | `deauthenticate` | Z4 | `auth` | POST | — | Z1 |
| `/dashboard`, `/home`, `/overview` | `dashboard` | Z4 | `auth` | GET | ?filter + ?resource_id | — |
| `/profile`, `/account`, `/me` | `view_profile` | Z4 | `auth` | GET | ?resource_id | — |
| `/settings`, `/preferences` | `edit_settings` | Z4 | `auth` | GET, POST, PATCH | ?content | — |
| `/admin`, `/admin/dashboard` | `admin_panel` | Z12 | `auth`, `admin` | GET | ?filter | — |
| `/admin/users`, `/admin/accounts` | `manage_users` | Z12 | `auth`, `admin` | GET, POST, DELETE | resource_id + ?content | — |
| `/checkout`, `/payment` | `checkout` | Z8 | `auth`, `verified` | GET, POST | payment_method + amount + billing_address | Z4 |
| `/cart`, `/basket` | `cart` | Z4 | `auth` | GET, POST, DELETE | resource_id + ?amount | — |
| `/pricing`, `/plans` | `pricing` | Z1 | `web` | GET | — | — |
| `/api/*` | `api_endpoint` | Z3 | `api` | GET, POST, DELETE | — (per endpoint) | — |
| `/webhook/*`, `/hooks/*` | `webhook` | Z3 | `api` | POST | — (token-validated) | — |
| `/health`, `/ping`, `/status` | `health_check` | Z3 | `api` | GET | — | — |

### Unknown Route Handling

Any route not in the canonical table generates a CLI warning during `codex-harmonize audit`:

```
codex-harmonize audit:

  /login           ✅  recognized → authenticate → Z2
  /dashboard       ✅  recognized → dashboard → Z4
  /superloginpage  ⚠️  unknown name
                       → declare purpose in harmonize.config.ts to inherit infra config

  14 routes auto-configured. 1 requires purpose declaration.
```

Once declared in `harmonize.config.ts`, the unknown route is treated identically to the canonical name it maps to. Its observations also contribute to the same global pool.

### Global Observation Pool

All apps using the canonical `authenticate` purpose contribute observations to the same global pool. A brand new app's `/login` route inherits confidence from millions of prior observations — it starts at `canonical` confidence, not `unverified`.

### API Contract

```
GET  /api/v1/vocabulary                → full canonical route table
GET  /api/v1/vocabulary/:purpose       → details for a specific purpose
POST /api/v1/vocabulary/resolve        → resolve a route path → purpose + config
GET  /api/v1/vocabulary/audit          → list unrecognized routes in project
```

---

## 3. Semantic Field Mapper

### What It Is

A mapping layer that converts raw HTML field types and names into semantic categories — `identity`, `secret`, `token`, `payment_method`, `amount`, etc. The pattern engine works on these categories, not on field values.

### Why This Matters

Traditional security checks field **values** (is this a valid email?). That's app-layer work. CodexSecure checks field **structure** (is this the expected collection of field types and names for this route?).

This is the point where CodexSecure becomes infra-level security instead of app validation.

### Category Registry

| Category | Field types | Common field names | Zone affinity |
|---|---|---|---|
| `identity` | `text`, `email`, `tel` | username, email, phone, handle, login | Z2, Z4 |
| `secret` | `password` | password, secret, pin, passphrase, pass | Z2, Z12 |
| `token` | `hidden` | _token, csrf_token, _csrf, authenticity_token, session_id, nonce | Z2, Z3, Z4 |
| `email_addr` | `email`, `text` | email, email_address, user_email | Z1, Z2 |
| `payment_method` | `text`, `tel`, `hidden` | card_number, cc_num, pan, payment_method | Z8 |
| `amount` | `number`, `text`, `hidden` | amount, price, total, charge, cost | Z8 |
| `billing_address` | `text`, `textarea` | address, billing_address, street, city, zip, postcode | Z8 |
| `content` | `text`, `textarea` | title, body, message, content, description, comment | Z1, Z4 |
| `search` | `text`, `search` | q, query, search, filter, keyword | Z1, Z3 |
| `resource_id` | `hidden`, `text` | id, user_id, post_id, item_id, ref | Z3, Z4 |
| `filter` | `text`, `select`, `hidden` | sort, order, page, limit, per_page, tab | Z1, Z3, Z4 |
| `file` | `file` | file, attachment, upload, document, image | Z3, Z4 |
| `mfa_code` | `text`, `number` | otp, mfa_code, totp, verification_code, auth_code | Z2, Z12 |
| `noise` | `hidden` | _method, __method, _utf8, _locale | any |

### Mapping Process

The mapper runs at two levels:

**Level 1 — Type profile:**
What field types appear in this form, at what frequency?
```
text:     1.0   (required — always present)
password: 1.0   (required — always present)
hidden:   0.62  (weighted — present in 62% of submissions)
```

**Level 2 — Name distribution per type:**
For each type slot, what names fill it and at what frequency?
```
text:     {username:0.42, email:0.38, phone:0.09, handle:0.06, other:0.05}
password: {password:0.71, secret:0.14, pin:0.08, passphrase:0.04, other:0.03}
hidden:   {_token:0.34, csrf_token:0.28, _csrf:0.11, auth_token:0.08, session_id:0.07, other:0.12}
```

### Weight Semantics

| Weight | Meaning | Violation action |
|---|---|---|
| `1.0` | Required — field type must be present | Absent → off-phase |
| `0.2–0.8` | Weighted — expected at given frequency | Count exceeds tolerance → off-phase |
| `0.0` / absent | Unexpected — not part of this form's profile | Any presence → off-phase |

### Self-Building — Zero Developer Setup

The mapper is seeded at deploy time by scanning rendered page HTML:

```
Deploy time:
  1. Renderer fetches page content (SSR or static)
  2. Extracts all <input>, <select>, <textarea> elements
  3. Reads type and name attributes
  4. Builds type profile + name distribution
  5. Queries global pool for this purpose type → inherits baseline distributions
  6. Merges local scan with global baseline → final field profile

As requests flow in:
  7. Each on-phase submission updates name distribution weights
  8. Distributions self-correct to reflect actual app usage
  9. Global pool receives anonymized weight updates
```

### API Contract

```
POST /api/v1/semantic/classify          → classify a set of field names → categories
GET  /api/v1/semantic/categories        → full category registry
GET  /api/v1/semantic/category/:name    → detail for one category
POST /api/v1/semantic/scan              → scan rendered HTML → return field profile
POST /api/v1/semantic/compare           → compare two field profiles → drift score
```

---

## 4. Pattern Hash Engine

### What It Is

The system that converts a semantic field profile into a deterministic hash, and compares that hash against the pre-built design hash to determine if a request is on-phase or off-phase.

### Two Hashes, Two Moments

**designHash** — computed at deploy time from the expected field profile.
**dataHash** — computed per-request from the incoming data's actual field structure.

If `dataHash === designHash` → on-phase → request proceeds to app.
If `dataHash !== designHash` → off-phase → blocked, observation recorded, zone stream updated.

### The designHash

Computed once at deploy time from the field profile built by the Semantic Field Mapper:

```ts
designHash = CodexHarmonicHash.create({
  type_profile: {
    text:     1.0,
    password: 1.0,
    hidden:   0.62,
  },
  name_distribution: {
    text:     { username: 0.42, email: 0.38, phone: 0.09, handle: 0.06 },
    password: { password: 0.71, secret: 0.14, pin: 0.08 },
    hidden:   { _token: 0.34, csrf_token: 0.28, _csrf: 0.11 },
  }
})
// → "a3f8b2c1d4e5..."  (stored in confidence array at route entry)
```

### The dataHash (per-request)

Computed from the incoming request body. The mapper runs on actual form fields:

```
POST /login body:
  username=alice&password=hunter2&_token=abc123

Field extraction:
  text:     [username]      → maps to category: identity
  password: [password]      → maps to category: secret
  hidden:   [_token]        → maps to category: token

type_profile:   {text: 1, password: 1, hidden: 1}
name_profile:   {text: {username: 1.0}, password: {password: 1.0}, hidden: {_token: 1.0}}

dataHash = CodexHarmonicHash.create(above)
```

### Phase Decision

```
dataHash === designHash?
  ✅ Match (or within tolerance band) → on-phase
  ❌ No match → off-phase
```

**Tolerance band** — slight name distribution drift (new name appearing in the long tail) is within tolerance. Structural violations (wrong type count, unexpected field type) are hard off-phase.

### Attack Examples

```
ATTACK — prototype pollution injection:
  POST body: {username, password, _token, __proto__}
  hidden names: [_token, __proto__]
  → __proto__ not in name distribution for hidden
  → dataHash diverges at name distribution level
  → off-phase ❌ — blocked before app sees it

ATTACK — hidden field flooding:
  POST body: {username, password, _token, csrf, inject, x}
  hidden count: 3 (expected weight 0.62 → 1 field expected)
  → type count exceeds tolerance
  → off-phase ❌

ATTACK — wrong field types:
  POST body where "password" is type text (not type password)
  → type profile: {text: 2, hidden: 1}  — no password type at all
  → designHash has password:1.0 (required)
  → off-phase ❌

LEGITIMATE — no CSRF token:
  POST body: {username, password}
  hidden count: 0 (expected weight 0.62 → weighted, optional)
  → within tolerance
  → on-phase ✅
```

### CodexHarmonicHash

A deterministic hash function designed for semantic field profiles. Properties:
- Deterministic — same input always produces same hash
- Ordered — field order in profile is normalized before hashing
- Tolerance-aware — produces a hash + a tolerance distance metric
- Collision resistant — different profiles produce different hashes

Built on top of the existing `fingerprint/hash` endpoint. The harmonic extension adds the two-level profile structure and tolerance distance.

### API Contract

```
POST /api/v1/pattern/design-hash        → compute designHash from field profile
POST /api/v1/pattern/data-hash          → compute dataHash from incoming request body
POST /api/v1/pattern/compare            → compare two hashes → match/off-phase + distance
POST /api/v1/pattern/validate           → full on-phase/off-phase decision for a request
GET  /api/v1/pattern/route/:path        → get designHash for a specific route
```

---

## 5. Confidence Array Builder

### What It Is

A pre-computed, per-route data structure that maps every route in the app to its full attack-surface profile. Built once at deploy/startup time. Consulted as a single O(1) lookup on every request.

This is one of CodexSecure's primary selling points: **the attack surface map is pre-built before any user touches the system.**

### The RouteConfidenceEntry

```go
type RouteConfidenceEntry struct {
    Path            string           // "/login"
    Purpose         string           // "authenticate"
    Zone            string           // "Z2"
    Chain           string           // "Z2" or "Z12.Z2"
    DesignHash      string           // hash of expected field profile + name distribution
    ConfidenceScore float64          // 0.0–1.0
    ObservationCount int64           // from global pool
    ConfidenceBand  ConfidenceBand   // unverified|low|medium|high|canonical
    FieldProfile    FieldProfile     // type weights + name distributions
    AllowedMethods  []string         // ["GET", "POST"]
    DataPattern     []string         // ["identity", "secret", "token?"]
}
```

### Confidence Bands

| Band | Score | Observations | Meaning |
|---|---|---|---|
| `unverified` | < 0.3 | < 1,000 | New pattern, not enough data |
| `low` | 0.3–0.6 | 1,000–10,000 | Pattern exists, spread not established |
| `medium` | 0.6–0.8 | 10,000–50,000 | Spread baseline emerging |
| `high` | 0.8–0.95 | 50,000–500,000 | Spread established, anomalies detectable |
| `canonical` | > 0.95 | > 500,000 | Global reference — new apps inherit immediately |

### Inheritance — No Cold Start Problem

A new app's `/login` starts at `canonical` confidence — NOT `unverified` — because it inherits the global `authenticate` purpose pool.

```
New app deploys /login:
  → vocabulary lookup: /login → purpose: authenticate
  → global pool: authenticate has 2.4M observations, confidence: 0.97
  → RouteConfidenceEntry created with inherited score: 0.97, band: canonical
  → app's /login is immediately protected at canonical confidence
  → zero warm-up period
```

### Build Process (deploy time)

```
App starts / deploys:
  1. codex-harmonize scans all declared routes
  2. For each route:
     a. vocabulary lookup → purpose assigned (or declared in harmonize.config.ts)
     b. page content scanned → field types + names extracted from rendered HTML
     c. type profile computed
     d. name distribution seeded from global pool for this purpose
     e. designHash computed from merged profile
     f. global pool queried → observationCount, confidenceScore inherited
     g. RouteConfidenceEntry constructed
  3. All entries written to process-scoped memory (map[string]RouteConfidenceEntry)
  4. Array is ready — app begins serving requests

Build cost: O(routes) — runs once at startup.
Request cost: O(1) — single map lookup.
```

### Per-Request Flow

```
POST /login arrives
  → map lookup: "/login" → RouteConfidenceEntry  (O(1))
  → extract: designHash, fieldProfile, allowedMethods
  → method check: POST ∈ ["GET", "POST"] → ✅
  → compute: dataHash from incoming POST body
  → compare: dataHash === designHash?
      ✅ on-phase → pass to Zone Phase Validator → app
      ❌ off-phase → 400, observation recorded to zone stream, entry flagged
```

### API Contract

```
GET  /api/v1/confidence                     → full confidence array for this project
GET  /api/v1/confidence/:path               → entry for a specific route
POST /api/v1/confidence/rebuild             → rebuild confidence array (post-deploy)
GET  /api/v1/confidence/bands               → summary of routes by confidence band
GET  /api/v1/confidence/unverified          → routes below canonical confidence
```

---

## 6. Request Phase Validator

### What It Is

The per-request enforcement layer. Runs after the confidence array lookup and pattern hash comparison. Produces the final on-phase / off-phase verdict for every request.

This is where all prior modules converge into a single binary decision: **proceed** or **block**.

### Phase Validator Pipeline

```
Request arrives at route
         │
         ▼
[1] Confidence array lookup (O(1))
    → RouteConfidenceEntry for this path
    → If path unknown → off-phase (unrecognized route)
         │
         ▼
[2] Method check
    → Incoming method ∈ entry.AllowedMethods?
    → No → off-phase (method not allowed)
         │
         ▼
[3] Zone chain validation (left → right)
    → If chain = "Z12.Z2": validate Z12 context first, then Z2 rules
    → Context check fails → off-phase
         │
         ▼
[4] Data hash comparison (POST/PUT/PATCH only)
    → Extract field types + names from request body
    → Compute dataHash
    → Compare to entry.DesignHash
    → Diverges beyond tolerance → off-phase
         │
         ▼
[5] Phase decision
    ✅ on-phase → add X-Codex-Phase: on-phase header → forward to app
    ❌ off-phase → record observation → update zone stream → 400 response
```

### On-Phase vs Off-Phase

**On-phase:** Request structural shape matches the pre-built design hash within tolerance. The request is what we expected it to be. Proceed.

**Off-phase:** Request structural shape deviates from the design hash. Something unexpected is in the form. Block before the app sees the data.

Off-phase does NOT necessarily mean malicious. It can mean:
- A new field was added to a form (legitimate, but triggers a config update)
- A browser extension injected a field
- A genuine attack

The observation stream distinguishes these over time via confidence score and spread distribution.

### Response Headers Set

| Header | On-phase value | Off-phase value |
|---|---|---|
| `X-Codex-Phase` | `on-phase` | `off-phase` |
| `X-Codex-Zone` | `Z2` | `Z2` (zone still resolved) |
| `X-Codex-Zone-Key` | `identity` | `identity` |
| `X-Codex-Confidence` | `0.97` | `0.97` |
| `X-Codex-DesignHash` | `a3f8b2...` | `a3f8b2...` |
| `X-Codex-DataHash` | `a3f8b2...` | `9c2d4f...` (diverged) |
| `X-Codex-Request-ID` | UUID | UUID |

### Strict vs Warn Mode

Configurable per-route or globally:

| Mode | On off-phase |
|---|---|
| `strict` | Block request — 400 response |
| `warn` | Pass request — but flag headers and record observation |
| `audit` | Pass request — record observation only, no headers |

Recommended: `strict` in production, `warn` during initial deploy to calibrate.

### API Contract

```
POST /api/v1/phase/validate             → validate a request → on-phase/off-phase
GET  /api/v1/phase/stats                → on-phase/off-phase rates by route
GET  /api/v1/phase/off-phase            → recent off-phase events
POST /api/v1/phase/mode                 → set strict/warn/audit mode per route
```

---

## 7. Zone Observation Stream

### What It Is

A real-time, chained event log where every request crossing a zone becomes a tamper-evident observation. Each observation is chained to the previous one via a hash — making the stream an append-only ledger of all zone traffic.

### The ZoneObservation

```go
type ZoneObservation struct {
    ID              string    // UUID
    TIU             string    // CodexTime timestamp — links observations cross-zone
    Zone            string    // "Z2"
    Path            string    // "/login"
    Method          string    // "POST"
    Phase           string    // "on-phase" | "off-phase"
    DataHash        string    // hash of actual incoming structure
    DesignHash      string    // hash of expected structure
    ConfidenceScore float64   // score at time of observation
    Timestamp       time.Time
    PreviousHash    string    // hash of the previous observation (chain link)
    ObservationHash string    // hash of this observation (includes PreviousHash)
    FieldProfile    FieldProfileSnapshot  // snapshot of type/name profile at time of request
    AnomalyFlags    []string  // ["extra_hidden_field", "unknown_name_in_distribution"]
}
```

### The Chain

Each observation includes `PreviousHash` — the hash of the immediately prior observation in the stream. This makes the stream tamper-evident:

```
Observation N-1:  hash = "aaa..."
Observation N:    prev_hash = "aaa...", own_hash = "bbb..."
Observation N+1:  prev_hash = "bbb...", own_hash = "ccc..."

If any observation is modified or deleted:
  all subsequent prev_hash values become invalid
  tampering is immediately detectable
```

### Real-Time Metrics (per route, per zone)

| Metric | What it measures |
|---|---|
| `on_phase_rate` | % of requests that match design hash |
| `off_phase_rate` | % of requests that deviate |
| `shape_distribution` | Distribution of field profiles seen in real traffic |
| `name_distribution_drift` | How far actual name distributions have moved from baseline |
| `confidence_trend` | Confidence score over time — rising or falling? |
| `off_phase_velocity` | Rate of off-phase events per minute — spikes signal attacks |
| `unique_field_names` | New field names appearing that aren't in the distribution |

### 90/10 Spread — The Baseline

In real traffic, field name distributions follow a natural pattern: a small number of names account for 90% of submissions, and a long tail of uncommon-but-legitimate names accounts for the remaining 10%.

```
Login form — username field (type text):
  username     42%  ─────────────────────────── }
  email        38%  ──────────────────────────  } 90%
  phone         9%  ──────────                  }
  handle        6%  ─────                        }
  user          3%  ───                          } 10% tail
  login         1%  ─                            }
  other         1%  ─                            }
```

A sudden spike in `other` — names not in the distribution — is an anomaly signal. The observation stream is what makes that spike visible.

### Stream Storage

```
v0.1: In-memory ring buffer per zone (existing route_observations table in PostgreSQL)
v0.2: Append-only PostgreSQL with prev_hash chain column
v0.3: Optional: blockchain ledger for critical zones (Z12, Z5)
```

### API Contract

```
GET  /api/v1/stream/:zone               → live stream for a zone (SSE)
GET  /api/v1/stream/:zone/history       → paginated observation history
GET  /api/v1/stream/:zone/metrics       → real-time metrics for a zone
GET  /api/v1/stream/route/:path         → stream for a specific route across zones
POST /api/v1/stream/subscribe           → subscribe to off-phase events (webhook)
```

---

## 8. Cross-Zone Correlation

### What It Is

A layer that links observations across zone boundaries using the TIU (Time Integrity Unit) — CodexTime's per-request timestamp token. A single user action may produce observations in multiple zones (Z2 login → Z4 dashboard → Z6 data read). Correlation detects when a session behaves unexpectedly across those zones.

### The TIU as Correlation Key

Every request carries a TIU in `X-Codex-TIU`. The TIU is:
- Issued by CodexTime at the start of a session
- Bound to the session — same user, same session, same TIU
- Immutable for the lifetime of the session

When a zone observation records a TIU, that TIU can be used to find all other observations from the same session across any zone.

```
TIU: "tiu_2026-05-08T12:00:00.000Z_session_abc"

Observations:
  Z2 /login POST  on-phase   12:00:01
  Z4 /dashboard GET on-phase  12:00:03
  Z6 /api/data GET on-phase   12:00:04
  Z12 /admin/keys GET  ← ❌ unexpected — Z12 from a Z4 session without Z12.Z2 crossing
```

A TIU appearing in Z12 without a valid Z12.Z2 zone chain transition → anomaly flag.

### Correlation Patterns

| Pattern | Signal | Action |
|---|---|---|
| Z2 → Z4 → Z6 | Expected flow for authenticated data read | Normal |
| Z2 → Z4 → Z12 (no chain) | Session jumped to Z12 without admin zone entry | Alert — possible privilege escalation |
| Z2 on-phase → Z4 off-phase | Login succeeded but dashboard request has wrong shape | Investigate — session integrity issue |
| High off-phase rate from single TIU | One session generating many off-phase events | Rate limit / block that session |
| Same TIU in Z2 and Z3 simultaneously | One session appearing in two incompatible zones | Session collision — possible token theft |
| Off-phase events spreading across Z1 → Z12 | Broad-surface probe — scanning many zones | Block source IP |

### Transition Graph

The expected zone transitions for a session are encoded in the zone chain notation:

```
Valid transitions for "authenticate" purpose:
  Z2 (guest, GET)  →  on form load
  Z2 (guest, POST) →  on submit
  Z4 (auth)        →  on success (transition: Z2 → Z4)

  Z12.Z2 → Z12     on success for admin login (transition: Z2 → Z12)
```

Any TIU that follows a transition not in the expected graph → anomaly.

### Zone Correlation Index

At the end of each session (or on a rolling window), the correlation engine builds a ZoneCorrelationIndex for that TIU:

```go
type ZoneCorrelationIndex struct {
    TIU            string
    Zones          []string          // ["Z2", "Z4", "Z6"] — zones visited
    Transitions    []ZoneTransition  // actual transitions observed
    ExpectedGraph  []ZoneTransition  // expected from zone chain notation
    Anomalies      []CorrelationAnomaly
    RiskScore      float64           // 0.0–1.0
}
```

### API Contract

```
GET  /api/v1/correlation/tiu/:tiu       → full correlation index for a TIU
GET  /api/v1/correlation/anomalies      → recent cross-zone anomalies
GET  /api/v1/correlation/risk/:tiu      → risk score for a session
POST /api/v1/correlation/alert          → configure alert thresholds
GET  /api/v1/correlation/transitions    → observed vs expected transition rates
```

---

## 9. Middleware / API Gateway

### What It Is

The request-interception layer that ties all modules together. Every request passes through this before reaching any app code. The middleware is also the implementation of zone chain notation — where `Z12.Z2` becomes real enforcement.

### Zone Chain Notation — Implementation

Dot notation: left = outermost context, right = current section. Same as reading a URL path.

```
Z2         →  identity zone, no outer context
Z12.Z2     →  identity zone inside admin context
Z4.Z2      →  identity zone inside protected area (step-up auth)
Z12.Z4.Z2  →  identity zone inside protected admin area (MFA re-auth)
```

**Resolution from URL:**
```
/login          →  Z2
/admin/login    →  Z12.Z2
/admin/settings →  Z12.Z4
```

**Validation order (left → right):**
1. Outer zone context check first
2. Each inner zone's own rules
3. Crossing check at POST success (elevated from outer zone)

**Middleware stacking:**
```
Z12.Z2 page served (GET):
  guest middleware runs  (user must NOT be authenticated)

Z12.Z2 form submitted (POST success):
  auth + admin middleware runs  (user MUST be authenticated with admin role)
```

These do not conflict — they run at different stages.

### createZoneMiddleware

The single developer-facing entry point for zone enforcement:

```ts
import { createZoneMiddleware } from '@web3codex/codex-core/middleware'

export const middleware = createZoneMiddleware({
  strategy: 'both',        // try subdomain first, then path
  pathZoneMap: {           // override for specific paths
    '/admin/login': 'Z12.Z2',
    '/checkout': 'Z8',
  },
  mode: 'strict',          // strict | warn | audit
  onOffPhase: (event) => { // optional hook
    logger.warn('off-phase', event)
  }
})
```

### Headers Set on Every Request

| Header | Value | Source |
|---|---|---|
| `X-Codex-Zone` | `Z2` | Zone classification |
| `X-Codex-Zone-Key` | `identity` | Zone vocabulary name |
| `X-Codex-Zone-Chain` | `Z12.Z2` | Full chain if applicable |
| `X-Codex-Phase` | `on-phase` | Phase validator result |
| `X-Codex-Confidence` | `0.97` | Route confidence score |
| `X-Codex-TIU` | `tiu_...` | CodexTime session timestamp |
| `X-Codex-Request-ID` | UUID | Unique request ID |

### Admin Gate

The gateway can be placed in admin-gate mode — all traffic blocked until a trusted token is presented. Useful for staging environments and dark launches.

```
CODEXNET_ADMIN_TOKEN=<token>

All requests must carry:
  X-Codex-Admin-Token: <token>
```

This is already implemented in `codexsecure-networks` (the Go gateway). For the API itself, the `ServiceKeyMiddleware` handles service-to-service auth.

### Integration Points

| Layer | How middleware connects |
|---|---|
| Next.js (web3connected, codexauth) | `createZoneMiddleware` in `middleware.ts` — edge middleware |
| Go codexsecure_api | Chain middleware in `internal/middleware/chain.go` |
| codexsecure-networks | Gateway on `:3000` — zone token classifier + reverse proxy |

### API Contract

```
GET  /api/v1/middleware/config          → current middleware config
POST /api/v1/middleware/simulate        → simulate middleware for a request (dry run)
GET  /api/v1/middleware/chains          → list all active zone chains
POST /api/v1/middleware/mode            → set mode (strict/warn/audit) globally
```

---

## 10. Admin Security Dashboard

### What It Is

The operator-facing view into the running CodexSecure system. Surfaces everything the backend knows: confidence scores, phase rates, anomaly events, zone stream, cross-zone correlation.

This is not a developer tool — it is a **security operations view**. The person reading this dashboard is watching the system's immune response to real traffic.

### Dashboard Panels

#### Panel 1 — Route Health Overview

One row per route. At-a-glance attack surface state:

| Route | Purpose | Zone | Confidence | On-phase % | Off-phase % | Last off-phase | Status |
|---|---|---|---|---|---|---|---|
| `/login` | authenticate | Z2 | 0.97 ● canonical | 99.2% | 0.8% | 2m ago | Healthy |
| `/checkout` | checkout | Z8 | 0.82 ● high | 97.1% | 2.9% | 30s ago | Watch |
| `/admin/login` | authenticate | Z12.Z2 | 0.97 ● canonical | 99.8% | 0.2% | 8h ago | Healthy |
| `/superloginpage` | authenticate | Z2 | 0.61 ● medium | 94.0% | 6.0% | 10s ago | Alert |

#### Panel 2 — Zone Stream (live)

Real-time scrolling feed of zone observations:

```
12:24:01  Z2 /login POST    on-phase   ● confidence 0.97   TIU: tiu_12:23:59...
12:24:01  Z4 /dashboard GET on-phase   ● confidence 0.91   TIU: tiu_12:23:59...
12:24:02  Z2 /login POST    off-phase  ▲ extra hidden field  TIU: tiu_12:24:01...
12:24:02  Z2 /login POST    off-phase  ▲ __proto__ in hidden TIU: tiu_12:24:01...
12:24:03  Z2 /login POST    on-phase   ● confidence 0.97   TIU: tiu_12:24:02...
```

Color coding:
- ● green — on-phase
- ▲ amber — off-phase (single event, not yet a pattern)
- ■ red — off-phase burst (3+ events in 30s from same TIU or IP)

#### Panel 3 — Confidence Trend (per route)

Line chart per route — confidence score over the past 24h/7d/30d.

A steadily rising confidence curve is healthy. A sudden drop signals:
- A form change (legitimate) — expectation: recovers within hours as new observations calibrate
- An attack campaign (hostile) — expectation: stays low or gets worse, off-phase rate climbs

#### Panel 4 — Field Distribution Drift

For each route, the name distribution per field type. Two columns: baseline (design) vs current (last 1000 observations).

```
Route: /login   Field type: hidden

                    BASELINE    CURRENT     DRIFT
  _token            34%         32%         -2%  ✓ normal
  csrf_token        28%         26%         -2%  ✓ normal
  _csrf             11%         10%         -1%  ✓ normal
  auth_token         8%          7%         -1%  ✓ normal
  session_id         7%          6%         -1%  ✓ normal
  session_token      5%          5%          0%  ✓ normal
  __proto__          0%          2%         +2%  ▲ ANOMALY — not in baseline
  inject             0%          2%         +2%  ▲ ANOMALY — not in baseline
  other              7%         10%         +3%  ▲ elevated tail
```

The `__proto__` and `inject` entries appearing in current traffic while absent from baseline = active attack.

#### Panel 5 — Cross-Zone Anomalies

Table of sessions (by TIU) that have triggered cross-zone anomalies:

| TIU | Zones visited | Expected | Anomaly | Risk score | Action |
|---|---|---|---|---|---|
| tiu_abc... | Z2 → Z4 → Z12 | Z2 → Z4 | Jumped to Z12 without admin chain | 0.91 | Block |
| tiu_def... | Z2 → Z3 | Z2 → Z4 | Z3 entry without Z4 transition | 0.61 | Watch |
| tiu_ghi... | Z2(×12) | Z2 | 12 login attempts, same session | 0.77 | Rate limit |

#### Panel 6 — Off-Phase Event Inspector

Click any off-phase observation to see full detail:

```
Event: off-phase
Route: /login
Time:  2026-05-08 12:24:02

Design hash:   a3f8b2c1d4...
Data hash:     9c2d4f7a8b...  (diverged)

Expected field profile:
  text:     {username:0.42, email:0.38, ...}  weight: 1.0 (required)
  password: {password:0.71, ...}              weight: 1.0 (required)
  hidden:   {_token:0.34, ...}                weight: 0.62 (weighted)

Actual fields received:
  username   (type text)    ✅ in distribution
  password   (type password) ✅ in distribution
  _token     (type hidden)  ✅ in distribution
  __proto__  (type hidden)  ❌ NOT in distribution ← violation point

Anomaly flag: extra_name_in_distribution
Recommendation: prototype pollution injection attempt
```

### Alert Configuration

Operators configure alert thresholds:

```yaml
alerts:
  off_phase_rate_threshold: 5%        # alert if off-phase > 5% of requests
  confidence_drop_threshold: 0.1      # alert if confidence drops 0.1 in 1h
  off_phase_burst_window: 30s         # alert if 5+ off-phase in 30s
  cross_zone_anomaly_risk: 0.7        # alert on sessions with risk > 0.7
  new_field_name_alert: true          # alert on any name not in baseline distribution
```

### API Contract

```
GET  /api/v1/dashboard/overview         → all routes health summary
GET  /api/v1/dashboard/stream           → live observation stream (SSE)
GET  /api/v1/dashboard/anomalies        → recent anomaly events
GET  /api/v1/dashboard/confidence/:path → confidence trend for a route
GET  /api/v1/dashboard/distribution/:path → field distribution drift
GET  /api/v1/dashboard/event/:id        → full detail for one observation
POST /api/v1/dashboard/alerts           → configure alert thresholds
```

---

## Implementation Order

These modules are interdependent. Build order:

| Phase | Modules | Prerequisite |
|---|---|---|
| Phase 1 | Semantic Field Mapper (Module 3) | Nothing — pure classification logic |
| Phase 2 | Pattern Hash Engine (Module 4) | Module 3 — needs field profiles |
| Phase 3 | Zone Observation Stream (Module 7) | Module 4 — needs phase decisions |
| Phase 4 | Route Vocabulary (Module 2) | Nothing — lookup table |
| Phase 5 | Zone Harmonizer (Module 1) | Module 2 — vocabulary needed for purpose resolution |
| Phase 6 | Confidence Array Builder (Module 5) | Modules 1, 2, 3, 4 — needs everything |
| Phase 7 | Request Phase Validator (Module 6) | Module 5 — needs confidence array |
| Phase 8 | Cross-Zone Correlation (Module 8) | Module 7 — needs observation stream |
| Phase 9 | Middleware / API Gateway (Module 9) | Module 6, 7 — needs validator + stream |
| Phase 10 | Admin Security Dashboard (Module 10) | All modules — reads everything |

---

## What's Already Built (Current State)

| Module | Status | Notes |
|---|---|---|
| Semantic Field Mapper | ⚡ Partial | `fingerprint` package handles field hashing. Category registry not yet defined. |
| Pattern Hash Engine | ⚡ Partial | `fingerprint/hash` works. Two-level profile (type + name distribution) not yet implemented. |
| Zone Observation Stream | ⚡ Partial | `route_observations` table exists. `prev_hash` chaining not yet added. |
| Route Vocabulary | ❌ Not started | Canonical route table exists in spec only. |
| Zone Harmonizer | ❌ Not started | harmonize.config.ts format defined in spec. |
| Confidence Array | ❌ Not started | Designed. Not implemented. |
| Request Phase Validator | ⚡ Partial | `coherence_middleware` does phase-style checks. Not hash-based yet. |
| Cross-Zone Correlation | ❌ Not started | TIU exists in middleware. Correlation engine not built. |
| Middleware / API Gateway | ⚡ Partial | `chain.go` + `zone.go`. Zone chain notation not yet implemented. |
| Admin Dashboard | ❌ Not started | Endpoint stubs exist. UI not built. |

---

## Open Questions

| # | Question | Decision needed |
|---|---|---|
| 1 | Does `designHash` compare by exact match or by tolerance distance? | Exact match risks false positives from legitimate form changes. Tolerance band is friendlier but harder to tune. |
| 2 | When a form changes (developer adds a field), how does the operator update the designHash? | `codex-harmonize rebuild` command? Automatic detection on next deploy? |
| 3 | How does Module 8 (Cross-Zone Correlation) handle anonymous users (no auth, no TIU)? | Correlate by request fingerprint (IP + User-Agent hash) instead of TIU? |
| 4 | Where does the global confidence pool live? | Centralized Web3Codex service? Or per-instance with optional sync? |
| 5 | MCP Stage 1 — is this permanently disabled or planned? | Stage 1 currently falls back to Z0 defaults. Decision needed on whether to wire MCP or replace Stage 1 with pattern-based classification. |

---

*This document defines the feature target. The `ARCHITECTURE.md` defines the pipeline it runs inside. The `_docs/web3Infra/Web3Codex_Core_Spec_v0.2.md` defines the SDK and zone model this feature set extends.*

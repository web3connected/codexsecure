// ---------------------------------------------------------------------------
// CodexSecure Docs page data — all typed arrays, no JSX
// Content verified against:
//   - NextJs/codexsecure/src/lib/zones/
//   - _docs/Servers/INTEGRATION_VERIFICATION_REPORT.md
//   - codexsecure-api (Go/Gin) port 8084
// ---------------------------------------------------------------------------

export interface DocNavItem {
  id: string;
  label: string;
  iconName: string;
}

export const docsNavItems: DocNavItem[] = [
  { id: 'overview',       label: 'Overview',         iconName: 'Book'    },
  { id: 'zone-framework', label: 'Zone Framework',   iconName: 'Layers'  },
  { id: 'route-analysis', label: 'Route Analysis',   iconName: 'GitFork' },
  { id: 'tiu-enforcement',label: 'TIU Enforcement',  iconName: 'Clock'   },
  { id: 'api-reference',  label: 'API Reference',    iconName: 'Code'    },
  { id: 'examples',       label: 'Code Examples',    iconName: 'Play'    },
  { id: 'phase-locking',  label: 'Phase Locking',    iconName: 'Lock'    },
];

// ---------------------------------------------------------------------------
// Overview
// ---------------------------------------------------------------------------

export const overviewDescription =
  'CodexSecure is a zone-based security middleware layer for distributed systems. ' +
  'Every inbound request passes through the Z0 orchestrator, which fingerprints the path, assigns a zone (Z1–Z12), ' +
  'and enforces the zone\'s policy — all in a single middleware pass. ' +
  'No manual route tagging, no scattered auth checks, no ad-hoc middleware chains.';

export const overviewChecklistTitle = 'Core Capabilities';
export const overviewChecklistItems = [
  'Twelve-zone security classification (Z0 Orchestrator → Z1–Z12 enforcement)',
  'Automatic route fingerprinting — keyword-strategy detection with confidence scoring',
  'TIU token enforcement — offline HMAC-SHA256 validation, no CodexTime runtime dependency',
  'Zone learning stabilisation — outlier detection with configurable tolerance windows',
  'Drift detection — SHA-256 route fingerprints catch pattern changes across deploys',
  'Recursive zone model — each zone can define its own Z0–Z12 sub-hierarchy',
  'Go/Gin backend (codexsecure-api) on port 8084 with full REST API',
];

// ---------------------------------------------------------------------------
// Zone Framework — concept cards + code
// ---------------------------------------------------------------------------

export const zoneFrameworkCodeBlock = `// Every request is classified by the Z0 Orchestrator
// CodexRouteAnalyzer.detectZoneFromPath(path)

// Route → Zone resolution example:
// /login          → Z2 (identity)   — keyword: "login"
// /dashboard      → Z4 (protected)  — keyword: "dashboard"
// /admin/users    → Z12 (admin)     — keyword: "admin"
// /payments/sync  → Z8 (payments)   — keyword: "payments"

// Zone policy is enforced automatically:
const result = CodexRouteAnalyzer.detectZoneFromPath('/dashboard/settings');
// { zone_id: 4, zone_name: 'protected', confidence: 0.2, matched_keywords: ['settings'] }`;

export const zoneFrameworkBulletItems = [
  { label: 'Z0', value: 'Orchestrator — classification engine, entry point for all requests' },
  { label: 'Z1–Z3', value: 'Open tier — public, identity, ops (no auth required)' },
  { label: 'Z4–Z6', value: 'Protected tier — auth + TIU + session required' },
  { label: 'Z7–Z9', value: 'Secure tier — comms, payments, analytics' },
  { label: 'Z10–Z12', value: 'Critical tier — behavior, realtime, admin' },
  { label: 'Recursive', value: 'Each zone can contain its own Z0–Z12 sub-hierarchy' },
];

// ---------------------------------------------------------------------------
// Route Analysis — concept cards + code
// ---------------------------------------------------------------------------

export const routeAnalysisCodeBlock = `// CodexRouteAnalyzer keyword strategies (from CodexRouteAnalyzer.ts)

const ZONE_STRATEGIES = {
  public:    ['public', 'home', 'welcome', 'about', 'contact', 'landing'],
  identity:  ['auth', 'login', 'logout', 'register', 'verify', 'password'],
  protected: ['dashboard', 'profile', 'account', 'settings', 'user', 'my'],
  payments:  ['payments', 'billing', 'invoice', 'checkout', 'transaction'],
  admin:     ['admin', 'manage', 'control', 'superuser', 'root', 'master'],
  // ... all 13 strategies, one per zone
};

// Confidence score = matched_keywords.length / total_keywords_for_zone
// Highest confidence wins — no manual tagging required`;

export const routeAnalysisBulletItems = [
  { label: 'Detection',   value: 'URL path segments matched against keyword strategies per zone' },
  { label: 'Confidence',  value: 'Ratio of matched keywords — highest confidence wins' },
  { label: 'Fingerprint', value: 'SHA-256 digest of route pattern — drift detected across deploys' },
  { label: 'Fallback',    value: 'Unresolvable paths promoted to Z0 (orchestrator rejects)' },
  { label: 'Methods',     value: 'Each zone restricts allowed HTTP verbs independently' },
];

// ---------------------------------------------------------------------------
// TIU Enforcement — concept cards + code
// ---------------------------------------------------------------------------

export const tiuEnforcementCodeBlock = `# CodexTime mints the token — CodexSecure validates it offline

# 1. Mint a TIU token from CodexTime
curl http://localhost:50002/api/time/tiu-token
# → { token: "eyJ...", expires_at: 1714242000, ttl_seconds: 30 }

# 2. Send token with your request to a protected zone
curl http://localhost:8084/api/v1/status \\
  -H "Authorization: Bearer eyJ..." \\
  -H "X-TIU-Token: eyJ..."

# 3. CodexSecure validates offline (no CodexTime network call):
#    - HMAC-SHA256 signature check
#    - Expiry check (exp > now)
#    - Optional drift window check
#    Environment: TIU_HMAC_SECRET must match the minting secret`;

export const tiuEnforcementBulletItems = [
  { label: 'Model',       value: 'Offline validation — CodexTime not required at request time' },
  { label: 'Algorithm',   value: 'HMAC-SHA256 over stable query-string payload' },
  { label: 'Payload',     value: 'v=1 | iat=<unix> | exp=<unix> | [device_uid=<uid>]' },
  { label: 'TTL',         value: 'Default 30s, configurable up to 300s' },
  { label: 'Config',      value: 'TIU_HMAC_SECRET env var — must match CodexTime minting secret' },
  { label: 'Device Bind', value: 'Optional X-Device-UID header binds token to a specific device' },
];

// ---------------------------------------------------------------------------
// API Reference
// ---------------------------------------------------------------------------

export interface ApiEndpoint {
  method: string;
  path: string;
  description: string;
  notes?: string;
}

export const apiEndpoints: ApiEndpoint[] = [
  {
    method: 'GET',
    path: '/api/v1/status',
    description: 'Service health and configuration status. Returns zone registry state, TIU config, and current service version.',
    notes: 'No auth required — Zone 1 (public)',
  },
  {
    method: 'POST',
    path: '/api/v1/classify',
    description: 'Classify a given path into a zone. Returns zone_id, zone_name, confidence score, and matched keywords.',
    notes: 'Useful for debugging zone assignments and testing your route structure',
  },
  {
    method: 'POST',
    path: '/api/v1/analyze',
    description: 'Full request analysis — fingerprint, zone classification, policy check, and TIU token validation in one call.',
    notes: 'Primary middleware integration point',
  },
  {
    method: 'GET',
    path: '/api/v1/zones',
    description: 'Returns the complete zone registry — all 13 zones with names, IDs, and enforced policies.',
    notes: 'No auth required — Zone 1 (public)',
  },
];

// ---------------------------------------------------------------------------
// Code Examples
// ---------------------------------------------------------------------------

export interface CodeExample {
  language: string;
  label: string;
  code: string;
}

// ---------------------------------------------------------------------------
// Phase Locking
// ---------------------------------------------------------------------------

// Phase Lock = intent-derived pattern baseline per route.
// Programmer naming conventions (LoginController, login.submit, LoginModel, /login)
// all declare the intent of the route. CodexSecure reads that intent and locks
// the expected patterns: which fields appear, what data formats they carry,
// and what keys are allowed. Any deviation is out-of-phase and flagged.

// How a phase is built from intent signals
export const phaseLockingIntentSources = [
  {
    id: 'Controller',
    example: 'LoginController',
    color: '#2DF4A1',
    description: 'Class name declares the action domain. "Login" signals an identity/auth intent.',
  },
  {
    id: 'Model',
    example: 'LoginModel',
    color: '#00E4FF',
    description: 'Model binding confirms the data shape expected — credential fields, not profile or payment fields.',
  },
  {
    id: 'Route Name',
    example: 'login.submit',
    color: '#7C3AED',
    description: 'Dot-notation name groups intent family (login.*). All members of the family share the same locked phase.',
  },
  {
    id: 'URI',
    example: '/login',
    color: '#F59E0B',
    description: 'Path segment matches the intent keyword. The route analyzer and the intent source agree.',
  },
  {
    id: 'View',
    example: 'auth.login',
    color: '#EF4444',
    description: 'View file name confirms the render intent. A structural hash of the view template is stored as part of the locked phase.',
  },
  {
    id: 'Form Fields',
    example: 'email, password',
    color: '#A78BFA',
    description: 'The declared input keys are locked. Only these two keys are valid. Any additional or renamed key is out-of-phase.',
  },
];

export const phaseLockingCodeBlock = `// ── Phase Lock — intent-derived pattern baseline ────────────────────────────
//
// Programmers already declare intent through naming conventions.
// CodexSecure reads those signals and locks the expected pattern.
//
// Example: Login route
//
//   Controller : LoginController         → intent = "login" (auth domain)
//   Action     : authenticate            → mutation class = credential_submission
//   Model      : LoginModel              → expected data shape = credentials
//   Route name : login.submit            → family = login.*
//   URI        : POST /login             → keyword match: "login"
//   View       : auth.login              → structure_hash stored as phase anchor
//
// Locked phase pattern for login.submit:
// {
//   "intent":          "authenticate_user",
//   "allowed_keys":    ["email", "password"],      // ONLY these keys
//   "required_keys":   ["email", "password"],
//   "field_patterns":  {
//     "email":    "EMAIL_FORMAT",                  // pattern hash, not raw data
//     "password": "PASSWORD_FORMAT"
//   },
//   "max_fields":      2,
//   "allow_extra":     false
// }

// ── In-phase request (passes) ────────────────────────────────────────────────
POST /login
{ "email": "user@example.com", "password": "••••••••" }
// ✓ 2 fields, correct keys, email matches EMAIL_FORMAT, password matches PASSWORD_FORMAT
// → in-phase → score += weight

// ── Out-of-phase: extra field added (flagged) ────────────────────────────────
POST /login
{ "email": "user@example.com", "password": "••••••••", "admin": true }
// ✗ unexpected key "admin" — not in locked pattern
// → non-phasing data → Z0 quarantine → Stage 7 record → score untouched

// ── Out-of-phase: wrong data format (flagged) ────────────────────────────────
POST /login
{ "email": "not-an-email", "password": "••••••••" }
// ✗ email field does not match EMAIL_FORMAT pattern hash
// → non-phasing data → Z0 quarantine → Stage 7 record → score untouched

// ── Out-of-phase: renamed/injected keys (flagged) ────────────────────────────
POST /login
{ "user_email": "user@example.com", "pass": "••••••••", "role": "superadmin" }
// ✗ none of these keys are in the locked pattern
// → non-phasing data → quarantine → review_state: pending`;

export const phaseLockingBulletItems = [
  { label: 'Intent Sources',    value: 'Controller · Model · Route name · URI · View · Form fields — all read together' },
  { label: 'Locked Pattern',    value: 'Allowed keys, required keys, field data-format patterns, max field count — stored as baseline' },
  { label: 'In-Phase',          value: 'Request matches locked pattern exactly — all keys present, all formats correct, no extras' },
  { label: 'Out-of-Phase',      value: 'Extra field, renamed key, wrong data format, or unexpected key — non-phasing, quarantined' },
  { label: 'Score Update',      value: 'Only in-phase (phasing) requests update the confidence score; out-of-phase requests do not' },
  { label: 'Dephasing',         value: 'Sustained out-of-phase traffic on a route — quarantined and logged, score protected' },
  { label: 'Pattern Review',    value: 'Quarantined patterns logged as review_state: pending — classify, dismiss, or escalate' },
];

export const phaseLockingConceptCards = [
  {
    title: 'Intent = The Phase',
    description:
      'A "phase" is the declared identity of a route, read from naming conventions. ' +
      'LoginController + LoginModel + login.submit + /login + auth.login all say the same thing: ' +
      'this is a credential-submission route. CodexSecure locks that intent into a pattern baseline ' +
      '— the expected keys, data formats, and field count — before the first request ever arrives.',
  },
  {
    title: 'Pattern Matching Confirms Lock',
    description:
      'Programmers already do pattern matching: email fields carry email data, password fields carry ' +
      'password data. CodexSecure formalises that. It hashes the pattern structure (EMAIL_FORMAT, ' +
      'PASSWORD_FORMAT) and stores those hashes as part of the locked phase. ' +
      'A request that sends an email field containing an IP address is immediately out-of-phase.',
  },
  {
    title: 'Extra or Unexpected Fields = Out-of-Phase',
    description:
      'The login route has exactly two allowed keys: email and password. ' +
      'If a third field appears — admin, role, token, anything — the request does not match ' +
      'the locked pattern. It is classified as non-phasing data, routed to Z0, quarantined, ' +
      'and recorded for review. The phase-lock confidence score is not touched.',
  },
  {
    title: 'Dephasing Cannot Degrade Lock',
    description:
      'A flood of requests with injected fields, renamed keys, or malformed data all produce ' +
      'non-phasing records. None of them interact with the accumulated confidence score. ' +
      'The route stays locked at its current confidence while every out-of-phase pattern ' +
      'is logged with review_state: pending for classification, dismissal, or escalation.',
  },
];

// ---------------------------------------------------------------------------
// Code Examples
// ---------------------------------------------------------------------------

export const codeExamples: CodeExample[] = [
  {
    language: 'bash',
    label: 'cURL',
    code: `# Check service status
curl -sS http://localhost:8084/api/v1/status

# Classify a path
curl -sS -X POST http://localhost:8084/api/v1/classify \\
  -H 'Content-Type: application/json' \\
  -d '{"path": "/admin/users"}'
# → { "zone_id": 12, "zone_name": "admin", "confidence": 0.33, "matched_keywords": ["admin"] }

# Full request analysis with TIU token
curl -sS -X POST http://localhost:8084/api/v1/analyze \\
  -H 'Content-Type: application/json' \\
  -H 'X-TIU-Token: <your-tiu-token>' \\
  -d '{"path": "/dashboard", "method": "GET"}'`,
  },
  {
    language: 'javascript',
    label: 'JavaScript',
    code: `// Classify a route
const res = await fetch('http://localhost:8084/api/v1/classify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ path: '/payments/checkout' }),
});
const { zone_id, zone_name, confidence } = await res.json();
console.log(\`Zone: \${zone_name} (Z\${zone_id}) — confidence: \${confidence}\`);
// Zone: payments (Z8) — confidence: 0.14

// Full analysis with TIU token
const tiuToken = await fetchTiuToken(); // from CodexTime

const analysis = await fetch('http://localhost:8084/api/v1/analyze', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-TIU-Token': tiuToken,
  },
  body: JSON.stringify({ path: '/dashboard', method: 'GET' }),
});
const result = await analysis.json();
console.log(result.valid, result.zone_name);`,
  },
  {
    language: 'python',
    label: 'Python',
    code: `import requests

BASE_URL = "http://localhost:8084/api/v1"

# Classify a path
resp = requests.post(f"{BASE_URL}/classify", json={"path": "/admin/settings"})
data = resp.json()
print(f"Zone: {data['zone_name']} (Z{data['zone_id']}) — confidence: {data['confidence']}")

# Verify a TIU token offline (no CodexSecure round-trip needed)
import base64, hashlib, hmac, time

def verify_tiu_token(token: str, secret: str) -> bool:
    payload_b64, sig_b64 = token.rsplit(".", 1)
    payload = base64.urlsafe_b64decode(payload_b64 + "==")
    expected = hmac.new(secret.encode(), payload, hashlib.sha256).digest()
    sig = base64.urlsafe_b64decode(sig_b64 + "==")
    if not hmac.compare_digest(expected, sig):
        return False
    params = dict(p.split("=", 1) for p in payload.decode().split("&"))
    return int(params["exp"]) > int(time.time())`,
  },
];

import type { NumberedStep } from '@/components/codex/shared/panels/NumberedStepsPanel';
import type { QuickStartLanguage, InstallCommand } from '@/components/codex/shared/panels/CodeBlockQuickStart';
import type { CtaButton } from '@/components/codex/shared/panels/CenteredCtaPanel';

// ─── Hero ────────────────────────────────────────────────────────────────────
export const gettingStartedHero = {
  badge: 'Developer Guide',
  eyebrow: 'Developer Guide',
  title: 'Get Started with',
  titleHighlight: 'CodexSecure',
  subtitle:
    'Zone-based request security for distributed systems. Integrate, classify, and enforce in minutes — no manual route tagging required.',
  backgroundImageSrc: '/assets/images/blue-digital-grid-background-network-connection-structure-blue-background-3d-rendering_698341-237.avif',
} as const;

// ─── Feature cards (icon assembled in page) ──────────────────────────────────
export interface GsFeatureCard {
  iconName: 'Shield' | 'Zap' | 'Lock';
  title: string;
  tagline: string;
  description: string;
  stat: string;
  statLabel: string;
}

export const gettingStartedFeatureCards: GsFeatureCard[] = [
  {
    iconName: 'Shield',
    title: 'Zone-First Security',
    tagline: 'Z0 → Z12 classification',
    description:
      'Every inbound request passes through the Z0 orchestrator. CodexSecure fingerprints the path, assigns a zone (Z1–Z12), and enforces that zone\'s policy — no manual middleware chains.',
    stat: '12',
    statLabel: 'security zones',
  },
  {
    iconName: 'Zap',
    title: 'No Runtime Dependency',
    tagline: 'Offline TIU validation',
    description:
      'TIU tokens are validated offline with HMAC-SHA256. CodexSecure does not call CodexTime at request time — the shared secret is enough to confirm freshness and authenticity.',
    stat: '< 1ms',
    statLabel: 'offline token validation',
  },
  {
    iconName: 'Lock',
    title: 'Phase Locking',
    tagline: 'Intent-derived baseline',
    description:
      'CodexSecure reads naming conventions — Controller, Model, Route name, URI, View, Form fields — and locks the expected request pattern. Any deviation is flagged as out-of-phase.',
    stat: '6',
    statLabel: 'intent signal sources',
  },
];

// ─── Numbered steps ───────────────────────────────────────────────────────────
export const gettingStartedSteps: NumberedStep[] = [
  {
    step: 1,
    title: 'Start the CodexSecure API',
    description: 'Run the Go API on port 8084. All classification, analysis, and zone policy is served from here.',
    detail: 'cd Docker/CodexWeb3/codexsecure_api && go run cmd/main.go  |  Or via Docker: docker compose up codexsecure-api',
  },
  {
    step: 2,
    title: 'Set Environment Variables',
    description: 'Configure the SDK in your application service with the CodexSecure base URL and your zone assignment.',
    detail: 'CODEXSECURE_SDK_ENABLED=true  |  CODEXSECURE_SDK_BASE_URL=http://codexsecure-api:8084  |  CODEXSECURE_SDK_DEFAULT_ZONE=5  |  CODEXSECURE_SDK_FAIL_OPEN=true  |  TIU_HMAC_SECRET=<shared-secret>',
  },
  {
    step: 3,
    title: 'Classify Your First Route',
    description: 'POST a path to /api/v1/classify and see which zone CodexSecure assigns, with confidence score and matched keywords.',
    detail: 'curl -X POST http://localhost:8084/api/v1/classify -d \'{"path":"/dashboard/settings"}\' — returns zone_id, zone_name, confidence, matched_keywords',
  },
  {
    step: 4,
    title: 'Add a TIU Token',
    description: 'Mint a short-lived TIU token from CodexTime and send it with protected-zone requests.',
    detail: 'X-TIU-Token: <token>  — CodexSecure validates HMAC-SHA256 + expiry offline. Default TTL is 30 seconds.',
  },
  {
    step: 5,
    title: 'Enable Full Request Analysis',
    description: 'Switch from /classify to /analyze for the full pipeline: fingerprint → zone → policy → TIU validation in one call.',
    detail: 'POST /api/v1/analyze with path, method, and X-TIU-Token header — use this as your primary middleware integration point.',
  },
];

// ─── Code examples ────────────────────────────────────────────────────────────
export const gettingStartedLanguages: QuickStartLanguage[] = [
  { id: 'bash',       name: 'cURL',       icon: '$_' },
  { id: 'javascript', name: 'JavaScript', icon: 'JS' },
  { id: 'python',     name: 'Python',     icon: 'PY' },
];

export const gettingStartedCodeExamples: Record<string, string> = {
  bash: `# 1. Check service status
curl -sS http://localhost:8084/api/v1/status

# 2. Classify a route
curl -sS -X POST http://localhost:8084/api/v1/classify \\
  -H 'Content-Type: application/json' \\
  -d '{"path": "/dashboard/settings"}'
# → { "zone_id": 4, "zone_name": "protected", "confidence": 0.2, "matched_keywords": ["settings"] }

# 3. Mint a TIU token from CodexTime (port 8007)
TOKEN=$(curl -sS http://localhost:8007/api/time/tiu-token | jq -r '.token')

# 4. Full request analysis with TIU token
curl -sS -X POST http://localhost:8084/api/v1/analyze \\
  -H 'Content-Type: application/json' \\
  -H "X-TIU-Token: $TOKEN" \\
  -d '{"path": "/dashboard", "method": "GET"}'
# → { "valid": true, "zone_name": "protected", "zone_id": 4, "tiu_valid": true }`,

  javascript: `// Configure once (e.g. in a middleware setup file)
const CODEXSECURE_URL = process.env.CODEXSECURE_SDK_BASE_URL ?? 'http://localhost:8084';

// Classify a route
async function classifyRoute(path: string) {
  const res = await fetch(\`\${CODEXSECURE_URL}/api/v1/classify\`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path }),
  });
  return res.json();
  // { zone_id: 4, zone_name: 'protected', confidence: 0.2, matched_keywords: ['settings'] }
}

// Full request analysis with a TIU token
async function analyzeRequest(path: string, method: string, tiuToken: string) {
  const res = await fetch(\`\${CODEXSECURE_URL}/api/v1/analyze\`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-TIU-Token': tiuToken,
    },
    body: JSON.stringify({ path, method }),
  });
  const result = await res.json();
  if (!result.valid) throw new Error(\`Zone enforcement failed: \${result.reason}\`);
  return result;
}`,

  python: `import os
import requests

BASE_URL = os.getenv("CODEXSECURE_SDK_BASE_URL", "http://localhost:8084")

# Classify a route
def classify_route(path: str) -> dict:
    resp = requests.post(f"{BASE_URL}/api/v1/classify", json={"path": path})
    resp.raise_for_status()
    return resp.json()
    # {"zone_id": 4, "zone_name": "protected", "confidence": 0.2}

# Full analysis with a TIU token
def analyze_request(path: str, method: str, tiu_token: str) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/analyze",
        json={"path": path, "method": method},
        headers={"X-TIU-Token": tiu_token},
    )
    resp.raise_for_status()
    result = resp.json()
    if not result.get("valid"):
        raise PermissionError(f"Zone enforcement failed: {result.get('reason')}")
    return result

# Example
zone = classify_route("/admin/users")
print(f"Zone: {zone['zone_name']} (Z{zone['zone_id']}) — confidence: {zone['confidence']}")
# Zone: admin (Z12) — confidence: 0.33`,
};

export const gettingStartedInstallCommands: InstallCommand[] = [
  { manager: 'env',    command: 'CODEXSECURE_SDK_BASE_URL=http://codexsecure-api:8084' },
  { manager: 'docker', command: 'docker compose up codexsecure-api' },
  { manager: 'go',     command: 'go run cmd/main.go   # port 8084' },
];

// ─── CTA ─────────────────────────────────────────────────────────────────────
export const gettingStartedCtaButtons: CtaButton[] = [
  { label: 'Read the Documentation', href: '/docs',  primary: true },
  { label: 'Explore Zones',          href: '/zones' },
];

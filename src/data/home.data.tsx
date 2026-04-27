import React from 'react';
import type { PanelCard } from '@/components/codex/shared/panels/ThreePanelCardDesign';
import type { NumberedStep } from '@/components/codex/shared/panels/NumberedStepsPanel';
import type { CtaButton } from '@/components/codex/shared/panels/CenteredCtaPanel';
import type { StatCard } from '@/components/codex/shared/panels/SplitTextStatsPanel';

// All stats and zone definitions are verified from:
//   Docker/CodexWeb3/codexsecure_api/internal/policy/defaults.go
//   Docker/CodexWeb3/codexsecure_api/internal/server/routes/api.go

// ─── Core Concepts ────────────────────────────────────────────────────────────

export const coreConcepts: [PanelCard, PanelCard, PanelCard] = [
  {
    icon: (
      <svg width="28" height="28" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
      </svg>
    ),
    title: 'Zone-Based Security',
    tagline: '12 zones · 3 tiers · Auto-classified',
    description:
      'Every route in your application maps to one of 12 security zones — from open public (Z1) to root system oversight (Z12). Zones carry enforced auth, TIU sync, and method policies. No manual configuration required.',
    stat: '12',
    statLabel: 'security zones (Z1–Z12)',
  },
  {
    icon: (
      <svg width="28" height="28" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" />
      </svg>
    ),
    title: 'Route Analysis',
    tagline: 'Per-request · Fingerprint · Drift detection',
    description:
      'The CodexSecure analyzer classifies every inbound route, generates a fingerprint, detects input drift, and validates against the zone schema — all in a single middleware pass before your handler runs.',
    stat: 'Go',
    statLabel: 'high-performance backend (Gin framework)',
  },
  {
    icon: (
      <svg width="28" height="28" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 13.5l10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z" />
      </svg>
    ),
    title: 'Phase Lock',
    tagline: 'TIU-synchronized · Schema-validated · Z2 + Z5',
    description:
      'High-trust zones (Z2 Identity, Z5 System Execution) require TIU sync from CodexTime before execution proceeds. Phase Lock middleware validates schema and blocks desynchronized requests at the edge.',
    stat: 'TIU',
    statLabel: 'CodexTime sync required for critical zones',
  },
];

// ─── How It Works ─────────────────────────────────────────────────────────────

export const howItWorksSteps: NumberedStep[] = [
  {
    step: 1,
    title: 'Request Enters the Zone Middleware',
    description:
      'Every inbound request passes through the CodexSecure middleware chain. The chain attaches a request ID, checks the HTTP method policy for the target zone, and begins fingerprint collection.',
    detail: 'Z1 (Public) passes freely. Z2–Z12 trigger progressively stricter checks — auth, TIU sync, schema validation, and executive trust for Z11–Z12.',
  },
  {
    step: 2,
    title: 'Route Analyzer Classifies the Request',
    description:
      'The analyzer fingerprints the route pattern, evaluates it against registered strategies, and assigns a zone code. Multiple strategies run in parallel — the highest-confidence match wins.',
    detail: 'You can also call POST /analyzer/route directly to classify routes in your CI pipeline before deploying — catch misconfigured paths before they go live.',
  },
  {
    step: 3,
    title: 'Policy Enforcement',
    description:
      'With the zone determined, the policy engine loads the zone\'s requirements: auth required (T/F), TIU sync required (T/F), allowed HTTP methods, and executive trust flag. Non-compliant requests are rejected immediately.',
    detail: 'Policy snapshots are versioned and served from GET /policy/snapshot. Your clients can cache the policy and validate locally without round-trips.',
  },
  {
    step: 4,
    title: 'Request Proceeds or Is Blocked',
    description:
      'Compliant requests reach your handler untouched. Blocked requests receive a structured error response with the zone code, policy violation reason, and request ID for tracing.',
    detail: 'Phase Lock zones (Z2, Z5) additionally record execution sync events so desynchronized bursts are detectable in the event log.',
  },
];

export const howItWorksFormula = {
  title: 'Zone Tier Model',
  lines: [
    'Intake Layer   — Z0              (classification + quarantine)',
    'Tier 1 Public  — Z1              (open access)',
    'Tier 2 Business — Z2 … Z10      (auth + TIU gates)',
    'Tier 3 Critical — Z11, Z12      (executive trust required)',
  ],
};

// ─── By the Numbers ───────────────────────────────────────────────────────────

export const byTheNumbersEyebrow = 'Platform Stats';
export const byTheNumbersTitle = 'Built on Verified Policy';
export const byTheNumbersParagraphs = [
  'CodexSecure zone policies are not configuration files you maintain — they are compiled into the policy engine and versioned. Every zone carries a machine-readable snapshot consumable by your clients at GET /policy/snapshot.',
  'The analyzer can classify routes before they hit production. Pipe your route manifest through POST /analyzer/routes in CI and catch zone mismatches before deployment.',
  'Phase Lock middleware on Z2 and Z5 ensures that identity and system-execution flows are temporally synchronized with CodexTime — request bursts outside the TIU window are blocked at the edge.',
];
export const byTheNumbersHighlight = 'policy engine';

export const byTheNumbersStats: StatCard[] = [
  { value: '12',    label: 'security zones enforced by the policy engine (Z1–Z12)', color: 'text-secure-primary' },
  { value: '3',     label: 'zone tiers — Public / Business / Critical', color: 'text-secure-secondary' },
  { value: 'Z2+Z5', label: 'zones with Phase Lock — Identity & System Execution (TIU sync required)', color: 'text-secure-accent' },
  { value: 'Go',    label: 'backend runtime — Gin framework, compiled binary, no interpreter overhead', color: 'text-secure-primary' },
];

// ─── CTA ──────────────────────────────────────────────────────────────────────

export const ctaEyebrow = 'Get Started';
export const ctaTitle = 'Ready to Secure Your Routes?';
export const ctaSubtitle =
  'Add intelligent zone enforcement to your infrastructure in minutes. Drop in the middleware, call the policy API, and let CodexSecure handle the rest.';

export const ctaButtons: CtaButton[] = [
  { label: 'Get API Access',  href: '/getting-started', primary: true },
  { label: 'View Docs',       href: '/docs' },
  { label: 'Contact Sales',   href: 'https://web3connected.com/contact' },
];

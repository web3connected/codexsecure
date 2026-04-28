import type { OverlayGridCard } from '@/components/codex/shared/panels/BgOverlayCardGridPanel';
import type { TimelineItem } from '@/components/codex/shared/panels/VerticalTimelinePanel';
import type { StatCard } from '@/components/codex/shared/panels/SplitTextStatsPanel';
import type { FactCard } from '@/components/codex/shared/panels/SplitTextFactsPanel';
import type { CtaButton } from '@/components/codex/shared/panels/CenteredCtaPanel';

export const aboutMissionParagraphs: string[] = [
  'CodexSecure is a zone-based middleware security platform built for distributed systems and blockchain applications. Every inbound request is analyzed, classified, and routed through a structured security zone — automatically, without manual tagging.',
  'The zone framework spans Z1 through Z12, from fully open public routes to maximum-enforcement private ledger zones. Zone classification happens in a single middleware pass — no separate API calls, no configuration files per route.',
  'Phase locking ensures that once a request is assigned a zone, its security posture cannot be downgraded mid-session. The TIU token minted at zone entry expires with the request. There is no replay. There is no drift.',
];

export const aboutMissionHighlight = 'CodexSecure';

export const aboutStatCards: StatCard[] = [
  { value: 'Z1–Z12', label: 'Twelve structured zones from public to private ledger', color: 'text-secure-primary' },
  { value: '1 pass', label: 'Single middleware pass — fingerprint, classify, enforce', color: 'text-secure-secondary' },
  { value: '0 drift', label: 'Phase-locked zones cannot be downgraded mid-session', color: 'text-white' },
];

export const aboutPrinciples: OverlayGridCard[] = [
  {
    symbol: 'Z',
    tag: 'zone framework · Z1–Z12',
    name: 'The Zone Model',
    description: 'Security is not binary. A public blog endpoint and a private ledger write are not the same risk class — treating them the same is where breaches happen. CodexSecure assigns each route a zone classification that defines its enforcement policy, token requirements, and audit trail depth.',
    stat: '12 zones',
    statLabel: 'From Z1 (public) to Z12 (ledger)',
  },
  {
    symbol: '⊕',
    tag: 'phase locking · TIU token',
    name: 'Phase Locking',
    description: 'A phase-locked request cannot have its security classification changed after zone entry. The TIU token minted at entry encodes the zone, the timestamp, and the device fingerprint. Attempting to replay or elevate the token fails validation — every time.',
    stat: '0 replays',
    statLabel: 'Tokens expire with the request',
  },
  {
    symbol: '∇',
    tag: 'route analysis · drift detection',
    name: 'Route Fingerprinting',
    description: 'The CodexSecure analyzer builds a fingerprint for every route on first access — method, payload schema, origin signature, and zone class. When a request deviates from the fingerprint, drift is detected and the request is either flagged or blocked, depending on the zone policy.',
    stat: '1 pass',
    statLabel: 'Analyze, classify, enforce',
  },
  {
    symbol: '⧖',
    tag: 'device trust · Z5–Z12',
    name: 'Device Trust Layer',
    description: 'For zones Z5 and above, CodexSecure requires a device fingerprint to be bound to the session. If the fingerprint changes mid-session — new device, new IP, new browser — the session is terminated. Trust is not assumed. It is continuously verified.',
    stat: 'Z5+',
    statLabel: 'Device trust required',
  },
];

export const aboutNamingParagraphs: string[] = [
  'Security systems fail most often at the boundary — where one zone ends and another begins. The handoff is where assumptions are made, where tokens are trusted without verification, where "secure by default" drifts into "secure most of the time".',
  'The zone model eliminates the ambiguity at that boundary. Every route has a defined zone. Every zone has a defined policy. The middleware does not guess — it enforces.',
  'We named the enforcement mechanism CodexSecure because a codex is structured, sequential, and tamper-evident. Security should be the same: not a list of rules applied loosely, but a bound structure where every layer depends on the one before it.',
];

export const aboutNamingFacts: FactCard[] = [
  { label: 'Zone assignment', value: 'Automatic via middleware — no per-route config', mono: false },
  { label: 'Enforcement model', value: 'declarative zone policy → enforced on every request', mono: true },
  { label: 'Drift detection', value: 'Fingerprint delta → flag or block per zone policy', mono: false },
];

export const aboutTimeline: TimelineItem[] = [
  {
    era: 'Problem',
    label: 'Route-level security is manual and inconsistent',
    detail: 'Most security middleware requires per-route annotation. Teams forget routes, misconfigure policies, or apply blanket rules that are either too permissive or too restrictive.',
  },
  {
    era: 'Insight',
    label: 'Security zones are structural, not decorative',
    detail: 'A route\'s security class is determined by what it does — not by what a developer remembers to tag. A ledger write is always Z12. A public health check is always Z1. The middleware should know this.',
  },
  {
    era: 'Design',
    label: 'The zone framework: Z1–Z12',
    detail: 'We built a classification system that maps route behavior, payload sensitivity, authentication requirements, and device trust into 12 distinct zones — each with a defined enforcement policy.',
  },
  {
    era: 'Now',
    label: 'CodexSecure',
    detail: 'A single middleware drop-in that fingerprints every route, classifies it into the correct zone, enforces the zone policy, and mints a phase-locked TIU token — in one pass.',
  },
];

export const aboutCtaButtons: CtaButton[] = [
  { label: 'View the Zone Framework', href: '/zones', primary: true },
  { label: 'Read the Docs', href: '/docs', external: false },
];

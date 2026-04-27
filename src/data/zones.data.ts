export interface ZoneEntry {
  id: number;
  badge: string;
  name: string;
  tier: string;
  tierColor: string;
  tagline: string;
  description: string;
  methods: string[];
  requirements: { label: string; met: boolean }[];
  useCases: string[];
  defaultOpen?: boolean;
}

export const zonesPageHero = {
  eyebrow: 'Zone Reference',
  title: 'The Twelve-Zone Framework',
  subtitle:
    'Every route in your system belongs to exactly one zone. CodexSecure classifies inbound requests automatically and enforces the policy for that zone — no manual tagging required.',
};

export const zoneConceptPanel = {
  eyebrow: 'How Zones Work',
  title: 'Zones Within Zones',
  subtitle: 'The framework is recursive by design.',
  body: [
    'Each zone is not just a classification label — it is a complete security boundary with its own internal structure. A Z4 Protected service can define its own Z1 through Z12 hierarchy internally, applying the same twelve-zone model to its own routes, sub-services, and data planes.',
    'This recursive pattern means security scales with your architecture. Microservices inherit the outer zone context, then classify their own internal traffic. A payment service living in Z8 can internally route its logging to its own Z6, its admin console to its own Z12, and its public status page to its own Z1 — all governed independently.',
    'You are never forced to flatten a complex system into a single classification layer. CodexSecure gives you the same framework at every depth.',
  ],
  pillars: [
    { icon: 'layers', label: 'Recursive', description: 'Every zone can contain its own Z0–Z12 structure.' },
    { icon: 'shield', label: 'Inherited Context', description: 'Inner zones inherit the trust level of their outer zone.' },
    { icon: 'git-branch', label: 'Independently Enforced', description: 'Each level enforces its own policy without coupling.' },
    { icon: 'zap', label: 'Automatic Classification', description: 'CodexSecure resolves depth automatically — no manual nesting.' },
  ],
};

export const zoneTiers = [
  { label: 'Classification', range: 'Z0',   color: '#EF4444' },
  { label: 'Open',      range: 'Z1 – Z3',   color: '#2DF4A1' },
  { label: 'Protected', range: 'Z4 – Z6',   color: '#00E4FF' },
  { label: 'Secure',    range: 'Z7 – Z9',   color: '#A78BFA' },
  { label: 'Critical',  range: 'Z10 – Z12', color: '#F59E0B' },
];

export const zones: ZoneEntry[] = [
  {
    id: 0,
    badge: 'Z0',
    name: 'Orchestrator',
    tier: 'Classification',
    tierColor: '#EF4444',
    tagline: 'The classification engine — every request enters here first.',
    description:
      'Zone 0 is the orchestration layer. Before any request reaches application logic, it passes through Z0 where the zone classifier fingerprints the path, inspects headers, and assigns the correct zone (Z1–Z12). It is the entry point of the entire CodexSecure pipeline. Requests that cannot be classified exit here with a rejection — all others are promoted to their assigned zone.',
    methods: ['GET'],
    requirements: [
      { label: 'Auth Required', met: false },
      { label: 'TIU Required', met: false },
      { label: 'Session Required', met: false },
    ],
    useCases: ['All inbound requests (pre-classification)', 'Route fingerprinting', 'Zone assignment engine', 'Unresolvable path rejection', 'Classification failure boundary'],
  },
  {
    id: 1,
    badge: 'Z1',
    name: 'Public',
    tier: 'Open',
    tierColor: '#2DF4A1',
    tagline: 'Fully open — no authentication or device trust required.',
    description:
      'Zone 1 covers routes that any visitor can access without credentials. This is the correct zone for marketing pages, public documentation, health-check endpoints, and status pages. No session, no device fingerprint, no TIU sync needed.',
    methods: ['GET', 'HEAD', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: false },
      { label: 'TIU Required', met: false },
      { label: 'Session Required', met: false },
    ],
    useCases: ['/', '/about', '/docs (public)', '/health', '/status', '/pricing'],
    defaultOpen: true,
  },
  {
    id: 2,
    badge: 'Z2',
    name: 'Identity',
    tier: 'Open',
    tierColor: '#2DF4A1',
    tagline: 'Auth entry — login, registration, and token verification.',
    description:
      'Zone 2 handles identity entry flows. Users are not yet authenticated but the system must validate time integrity (TIU) to prevent replay attacks during credential exchange. Registration, login, and password-reset endpoints live here.',
    methods: ['GET', 'POST', 'HEAD', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: false },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: false },
    ],
    useCases: ['/login', '/register', '/verify-email', '/password-reset', '/oauth/callback'],
  },
  {
    id: 3,
    badge: 'Z3',
    name: 'Ops',
    tier: 'Open',
    tierColor: '#2DF4A1',
    tagline: 'Operational endpoints — background workers and infrastructure.',
    description:
      'Zone 3 is reserved for operational and infrastructure routes that need to be reachable by orchestrators, monitoring agents, and CI pipelines without requiring user credentials. Treat these as internal service-entry points, not user-facing.',
    methods: ['GET', 'POST', 'HEAD', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: false },
      { label: 'TIU Required', met: false },
      { label: 'Session Required', met: false },
    ],
    useCases: ['/health', '/ready', '/metrics', '/ping', 'Cron webhook receivers', 'Load-balancer probes'],
  },
  {
    id: 4,
    badge: 'Z4',
    name: 'Protected',
    tier: 'Protected',
    tierColor: '#00E4FF',
    tagline: 'Standard auth zone — the majority of application routes.',
    description:
      'Zone 4 is the workhorse. Authenticated users interacting with the main application — dashboards, profiles, settings, projects — all land here. Full auth, device trust, and a valid session are required. All standard HTTP methods are allowed.',
    methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/dashboard', '/profile', '/settings', '/projects/*', '/account/*'],
  },
  {
    id: 5,
    badge: 'Z5',
    name: 'System',
    tier: 'Protected',
    tierColor: '#00E4FF',
    tagline: 'Internal system operations and service-to-service calls.',
    description:
      'Zone 5 covers internal service operations — inter-service communication, background job processing, and infrastructure management routes. Requests must be fully authenticated with device trust and a valid session even when originating from another service.',
    methods: ['GET', 'POST', 'PUT', 'DELETE'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/api/internal/*', 'Service-mesh endpoints', 'Background job triggers', 'System configuration reads'],
  },
  {
    id: 6,
    badge: 'Z6',
    name: 'Event Logs',
    tier: 'Protected',
    tierColor: '#00E4FF',
    tagline: 'Audit trail and event streaming — immutable by design.',
    description:
      'Zone 6 is dedicated to audit and event-log routes. Writes are append-only; reads require elevated context. Every request is validated against auth, TIU sync, and session before the event pipeline is touched. This zone underpins compliance and forensic traceability.',
    methods: ['GET', 'POST'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/events', '/audit-log', '/activity', 'Webhook event receivers', 'Compliance log endpoints'],
  },
  {
    id: 7,
    badge: 'Z7',
    name: 'Comms',
    tier: 'Secure',
    tierColor: '#A78BFA',
    tagline: 'Communications — messaging, notifications, and email dispatch.',
    description:
      'Zone 7 handles all communication flows: in-app messaging, push notifications, and outbound email dispatch. Full auth stack required. All standard methods are permitted to accommodate both read and write operations across communication channels.',
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/messages', '/notifications', '/email/send', '/webhooks/dispatch', '/inbox'],
  },
  {
    id: 8,
    badge: 'Z8',
    name: 'Payments',
    tier: 'Secure',
    tierColor: '#A78BFA',
    tagline: 'Financial transactions — payment processing and invoicing.',
    description:
      'Zone 8 is the payments boundary. Any route that initiates, reads, or modifies financial records belongs here. Method access is intentionally restricted — DELETE is not permitted at this level. Every request is validated across the full auth stack before the payment engine is engaged.',
    methods: ['GET', 'POST', 'PUT'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/checkout', '/invoices', '/payments/*', '/subscriptions', '/billing/history'],
  },
  {
    id: 9,
    badge: 'Z9',
    name: 'Analytics',
    tier: 'Secure',
    tierColor: '#A78BFA',
    tagline: 'Telemetry collection and reporting — data-plane access.',
    description:
      'Zone 9 covers analytics ingestion and reporting. Telemetry writes and report reads both require the full auth stack. The method set is intentionally narrow — GET and POST only — to prevent destructive operations against the analytics data plane.',
    methods: ['GET', 'POST'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/analytics/*', '/reports/*', '/telemetry', '/stats', '/insights'],
  },
  {
    id: 10,
    badge: 'Z10',
    name: 'Behavior',
    tier: 'Critical',
    tierColor: '#F59E0B',
    tagline: 'Behavioral signals — pattern detection and anomaly input.',
    description:
      'Zone 10 handles behavioral intelligence routes: ML feature ingestion, anomaly signals, and pattern-match inputs. These endpoints feed downstream threat models and risk scoring. Access is tightly scoped — GET and POST only — with the full auth stack enforced.',
    methods: ['GET', 'POST'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/behavior-tracking', '/anomaly-signals', '/pattern-match', '/risk-score/input', '/ml-features'],
  },
  {
    id: 11,
    badge: 'Z11',
    name: 'Realtime',
    tier: 'Critical',
    tierColor: '#F59E0B',
    tagline: 'Live data streams — WebSockets and real-time feeds.',
    description:
      'Zone 11 governs real-time connections: WebSocket upgrades, server-sent event streams, and live data feeds. All standard HTTP methods are allowed to support the full lifecycle of a real-time connection. Every connection must be authenticated with device trust and a valid session before the stream is opened.',
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/ws', '/stream/*', '/live/*', 'WebSocket upgrade endpoints', 'Server-sent events'],
  },
  {
    id: 12,
    badge: 'Z12',
    name: 'Admin',
    tier: 'Critical',
    tierColor: '#F59E0B',
    tagline: 'Root system oversight — highest privilege, broadest access.',
    description:
      'Zone 12 is the top of the trust hierarchy. Admin-level operations — user management, system configuration, access revocation, and emergency overrides — all live here. Every HTTP method is permitted. All three auth checks are mandatory and cannot be bypassed.',
    methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'],
    requirements: [
      { label: 'Auth Required', met: true },
      { label: 'TIU Required', met: true },
      { label: 'Session Required', met: true },
    ],
    useCases: ['/admin/*', '/sudo/*', '/system/config', '/users/revoke', '/emergency-override'],
  },
];

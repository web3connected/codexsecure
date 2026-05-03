/**
 * Z1 Route List — CodexSecure Dev Tooling
 *
 * The /routes page — shows all registered routes across all zones.
 * Zone: Z1_PUBLIC (web, GET, no auth)
 *
 * This is a developer portal — the route list is intentionally public.
 * Zone contract applied automatically by Routes.loadCodexZonedRoutes().
 */

export const routes = [
  {
    path: '/routes',
    name: 'route-list',
    zone: 'Z1_PUBLIC',
    target_zone: 'Z12',
    pattern: {
      inside: ['zone', 'path', 'middleware', 'methods', 'zone_override', 'pattern'],
      threshold: 0.0,
    },
    meta: { title: 'Route List · CodexSecure' },
  },
];

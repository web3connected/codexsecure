/**
 * Z1 Public Routes — CodexSecure
 *
 * Every route in this file is Z1_PUBLIC.
 * Zone contract applied automatically by Routes.loadCodexZonedRoutes():
 *   middleware: ['web']
 *   methods:    GET, OPTIONS
 *   auth:       not required
 */

export const routes = [
  { path: '/',                name: 'home',            meta: { title: 'CodexSecure - Zone-Based Security Platform' } },
  { path: '/about',           name: 'about',           meta: { title: 'About CodexSecure' } },
  { path: '/docs',            name: 'docs',            meta: { title: 'CodexSecure Documentation' } },
  { path: '/zones',           name: 'zones',           meta: { title: 'Zone Reference' } },
  { path: '/getting-started', name: 'getting-started', meta: { title: 'Getting Started' } },
  { path: '/security',        name: 'security',        meta: { title: 'Security' } },
  { path: '/blog',            name: 'blog',            meta: { title: 'Blog' } },
];

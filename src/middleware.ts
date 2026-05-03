/**
 * CodexSecure - Next.js Edge Middleware with Zone Routing
 *
 * Detects zones from URL path and sets X-Codex-Zone headers.
 * codexsecure-networks reads X-Codex-Zone: Z{id} to route to the
 * correct isolated port segment.
 *
 * @see src/lib/zones/edge-middleware.ts
 */

// Import directly from edge-specific file — NOT the barrel index.ts
// (index.ts re-exports CodexEthosLattice which uses Node.js crypto, banned in edge runtime)
import { createZoneMiddleware } from '@/lib/zones/edge-middleware';

export default createZoneMiddleware({
  addHeaders: true,

  // Secure portal has docs and zone management — protect settings paths
  protectedPaths: [
    '/account',
    '/settings',
    '/api-keys',
    '/webhooks',
  ],

  // Admin paths for zone management
  adminPaths: [
    '/admin',
    '/zones/manage',
  ],

  loginUrl: '/login',

  skipPaths: [
    '/_next',
    '/favicon.ico',
    '/public',
    '/api/health',
  ],

  debug: process.env.NODE_ENV === 'development',
});

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico|public/).*)',],
};

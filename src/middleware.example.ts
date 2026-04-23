/**
 * Next.js Edge Middleware with Zone Routing
 *
 * This middleware detects zones from the URL path and applies
 * zone-based access control.
 *
 * Features:
 * - Zone detection based on URL keywords
 * - Zone headers (X-Codex-Zone, X-Codex-Zone-Name)
 * - Protected route redirection
 * - Admin route protection
 *
 * @see https://nextjs.org/docs/app/building-your-application/routing/middleware
 */

import { createZoneMiddleware, zoneMiddlewareConfig } from '@/lib/zones';

/**
 * Configure the zone middleware
 *
 * Available options:
 * - addHeaders: Add X-Codex-Zone headers to responses (default: true)
 * - protectedPaths: Paths that require authentication
 * - adminPaths: Paths that require admin role
 * - loginUrl: URL to redirect unauthenticated users
 * - skipPaths: Paths to skip zone checking
 * - debug: Add debug headers
 */
export default createZoneMiddleware({
  // Add zone headers to all responses
  addHeaders: true,

  // Paths that require authentication
  protectedPaths: [
    '/dashboard',
    '/profile',
    '/settings',
    '/account',
  ],

  // Paths that require admin role
  adminPaths: [
    '/admin',
  ],

  // Redirect URL for unauthenticated users
  loginUrl: '/login',

  // Paths to skip zone checking (static assets, etc.)
  skipPaths: [
    '/_next',
    '/favicon.ico',
    '/public',
    '/api/health', // Allow health checks without zone
  ],

  // Enable debug mode in development
  debug: process.env.NODE_ENV === 'development',
});

/**
 * Middleware matcher configuration
 *
 * Matches all paths except:
 * - _next/static (static files)
 * - _next/image (image optimization files)
 * - favicon.ico (favicon)
 * - public folder
 */
export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    '/((?!_next/static|_next/image|favicon.ico|public/).*)',
  ],
};

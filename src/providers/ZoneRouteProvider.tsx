'use client';

/**
 * ZoneRouteProvider - React Context for Zone Routing
 *
 * Provides zone information to React components via context.
 * This is the Next.js equivalent of the Laravel ZoneRouteServiceProvider.
 *
 * @example
 * // In app/layout.tsx
 * import { ZoneRouteProvider } from '@/providers/ZoneRouteProvider';
 *
 * export default function RootLayout({ children }) {
 *   return (
 *     <html>
 *       <body>
 *         <ZoneRouteProvider>{children}</ZoneRouteProvider>
 *       </body>
 *     </html>
 *   );
 * }
 *
 * // In a component
 * import { useZone } from '@/providers/ZoneRouteProvider';
 *
 * export function MyComponent() {
 *   const { zoneId, zoneName, isAdmin, requiresAuth } = useZone();
 *   return <div>Current Zone: {zoneName} (Z{zoneId})</div>;
 * }
 *
 * @version 1.0.0
 */

import React, { createContext, useContext, useEffect, useState, useMemo, ReactNode } from 'react';
import { usePathname } from 'next/navigation';

// Zone configuration types
export interface ZoneInfo {
  zoneId: number;
  zoneName: string;
  isAdmin: boolean;
  isProtected: boolean;
  isPublic: boolean;
  requiresAuth: boolean;
  requiresTiu: boolean;
  allowedMethods: string[];
}

export interface ZoneContextValue extends ZoneInfo {
  /** Check if current zone allows a specific method */
  isMethodAllowed: (method: string) => boolean;
  /** Check if user can access a specific zone */
  canAccessZone: (zoneId: number) => boolean;
  /** Loading state */
  isLoading: boolean;
  /** Raw zone detection result */
  detection: ZoneDetectionResult | null;
}

interface ZoneDetectionResult {
  zone_id: number;
  zone_name: string;
  confidence: number;
  matched_keywords: string[];
}

// Zone strategies for client-side detection
const ZONE_STRATEGIES: Record<string, string[]> = {
  negative: ['blocked', 'quarantine', 'banned', 'restricted'],
  public: ['public', 'home', 'welcome', 'about', 'contact', 'landing'],
  identity: ['auth', 'login', 'logout', 'register', 'identity', 'verify', 'password', 'forgot', 'reset', 'session'],
  ops: ['health', 'status', 'ping', 'api', 'webhook', 'callback', 'test'],
  protected: ['dashboard', 'profile', 'account', 'settings', 'user', 'my'],
  system: ['internal', 'system', 'service', 'worker', 'job', 'queue', 'cron'],
  eventlogs: ['logs', 'events', 'audit', 'history', 'activity'],
  comms: ['messages', 'notifications', 'chat', 'email', 'sms', 'push', 'inbox'],
  sync: ['sync', 'import', 'export', 'transfer', 'migrate', 'backup'],
  analytics: ['analytics', 'metrics', 'stats', 'reports', 'insights', 'charts'],
  behavior: ['behavior', 'tracking', 'usage', 'engagement', 'patterns'],
  ai: ['ai', 'ml', 'model', 'predict', 'inference', 'train', 'agent', 'copilot'],
  admin: ['admin', 'manage', 'control', 'superuser', 'root', 'master'],
};

const ZONE_NAME_TO_ID: Record<string, number> = {
  negative: 0, public: 1, identity: 2, ops: 3, protected: 4,
  system: 5, eventlogs: 6, comms: 7, sync: 8, analytics: 9,
  behavior: 10, ai: 11, admin: 12,
};

const ZONE_RULES: Record<number, { auth: boolean; tiu: boolean; methods: string[] }> = {
  0: { auth: false, tiu: false, methods: ['GET'] },
  1: { auth: false, tiu: false, methods: ['GET', 'POST', 'OPTIONS'] },
  2: { auth: false, tiu: false, methods: ['GET', 'POST', 'PUT', 'OPTIONS'] },
  3: { auth: false, tiu: false, methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'] },
  4: { auth: true, tiu: false, methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'] },
  5: { auth: true, tiu: true, methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'] },
  6: { auth: true, tiu: true, methods: ['GET', 'POST', 'OPTIONS'] },
  7: { auth: true, tiu: false, methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'] },
  8: { auth: true, tiu: true, methods: ['GET', 'POST', 'PUT', 'PATCH', 'OPTIONS'] },
  9: { auth: true, tiu: false, methods: ['GET', 'POST', 'OPTIONS'] },
  10: { auth: true, tiu: true, methods: ['GET', 'POST', 'OPTIONS'] },
  11: { auth: true, tiu: true, methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'] },
  12: { auth: true, tiu: true, methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'] },
};

// Default context value
const defaultZoneContext: ZoneContextValue = {
  zoneId: -1,
  zoneName: 'unassigned',
  isAdmin: false,
  isProtected: false,
  isPublic: false,
  requiresAuth: false,
  requiresTiu: false,
  allowedMethods: [],
  isMethodAllowed: () => true,
  canAccessZone: () => false,
  isLoading: true,
  detection: null,
};

// Create context
const ZoneContext = createContext<ZoneContextValue>(defaultZoneContext);

// Provider props
export interface ZoneRouteProviderProps {
  children: ReactNode;
  /** Initial zone (for SSR) */
  initialZone?: { zoneId: number; zoneName: string };
  /** User's accessible zones (for authorization) */
  userZones?: number[];
}

/**
 * Detect zone from pathname
 */
function detectZone(pathname: string): ZoneDetectionResult {
  const segments = pathname.toLowerCase().split('/').filter(Boolean);
  
  let bestMatch: ZoneDetectionResult = {
    zone_id: -1,
    zone_name: 'unassigned',
    confidence: 0,
    matched_keywords: [],
  };

  for (const [zoneName, keywords] of Object.entries(ZONE_STRATEGIES)) {
    const matched: string[] = [];
    
    for (const keyword of keywords) {
      if (segments.some(seg => seg.includes(keyword))) {
        matched.push(keyword);
      }
    }

    const confidence = matched.length / keywords.length;
    
    if (confidence > bestMatch.confidence) {
      bestMatch = {
        zone_id: ZONE_NAME_TO_ID[zoneName] ?? -1,
        zone_name: zoneName,
        confidence,
        matched_keywords: matched,
      };
    }
  }

  return bestMatch;
}

/**
 * ZoneRouteProvider Component
 */
export function ZoneRouteProvider({
  children,
  initialZone,
  userZones = [],
}: ZoneRouteProviderProps) {
  const pathname = usePathname();
  const [isLoading, setIsLoading] = useState(!initialZone);
  const [detection, setDetection] = useState<ZoneDetectionResult | null>(
    initialZone
      ? { zone_id: initialZone.zoneId, zone_name: initialZone.zoneName, confidence: 1, matched_keywords: [] }
      : null
  );

  // Detect zone when pathname changes
  useEffect(() => {
    if (pathname) {
      const result = detectZone(pathname);
      setDetection(result);
      setIsLoading(false);
    }
  }, [pathname]);

  // Build context value
  const contextValue = useMemo<ZoneContextValue>(() => {
    const zoneId = detection?.zone_id ?? -1;
    const zoneName = detection?.zone_name ?? 'unassigned';
    const rules = ZONE_RULES[zoneId] ?? { auth: false, tiu: false, methods: [] };

    return {
      zoneId,
      zoneName,
      isAdmin: zoneId === 12,
      isProtected: zoneId >= 4,
      isPublic: zoneId === 1,
      requiresAuth: rules.auth,
      requiresTiu: rules.tiu,
      allowedMethods: rules.methods,
      isMethodAllowed: (method: string) => rules.methods.includes(method.toUpperCase()),
      canAccessZone: (targetZone: number) => {
        // Public zones (0-3) are always accessible
        if (targetZone <= 3) return true;
        // Check if user has access to protected zones
        return userZones.includes(targetZone);
      },
      isLoading,
      detection,
    };
  }, [detection, isLoading, userZones]);

  return (
    <ZoneContext.Provider value={contextValue}>
      {children}
    </ZoneContext.Provider>
  );
}

/**
 * Hook to access zone context
 */
export function useZone(): ZoneContextValue {
  const context = useContext(ZoneContext);
  if (!context) {
    throw new Error('useZone must be used within a ZoneRouteProvider');
  }
  return context;
}

/**
 * Hook to check if current route is in a specific zone
 */
export function useIsInZone(zoneId: number): boolean {
  const { zoneId: currentZone } = useZone();
  return currentZone === zoneId;
}

/**
 * Hook to check if current route requires authentication
 */
export function useRequiresAuth(): boolean {
  const { requiresAuth } = useZone();
  return requiresAuth;
}

/**
 * Hook to get zone display info
 */
export function useZoneDisplay(): { badge: string; color: string } {
  const { zoneId, zoneName } = useZone();
  
  const colors: Record<number, string> = {
    0: 'bg-red-500',
    1: 'bg-green-500',
    2: 'bg-blue-500',
    3: 'bg-yellow-500',
    4: 'bg-purple-500',
    5: 'bg-gray-500',
    6: 'bg-orange-500',
    7: 'bg-pink-500',
    8: 'bg-cyan-500',
    9: 'bg-indigo-500',
    10: 'bg-teal-500',
    11: 'bg-violet-500',
    12: 'bg-red-600',
  };

  return {
    badge: `Z${zoneId}: ${zoneName}`,
    color: colors[zoneId] ?? 'bg-gray-400',
  };
}

export default ZoneRouteProvider;

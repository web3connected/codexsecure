/**
 * Shared Types for Codex Ecosystem
 * Used across all sites: web3connected, codexhash, codexsecure
 */

// Site identifiers
export const CODEX_SITES = {
  WEB3CONNECTED: 'web3connected',
  CODEXHASH: 'codexhash',
  CODEXSECURE: 'codexsecure',
} as const;

export type CodexSite = typeof CODEX_SITES[keyof typeof CODEX_SITES];

// User interface matching shared database schema
export interface User {
  id: string;
  email: string;
  name: string;
  reg_sites: CodexSite[];
  created_at: string;
  updated_at: string;
  last_login_at?: string;
  last_login_site?: CodexSite;
  profile_data?: Record<string, any>;
  preferences?: Record<string, any>;
}

// Extended user with site context
export interface UserWithSiteContext extends User {
  isRegisteredOnCurrentSite: boolean;
  currentSite: CodexSite;
}

// Auth API responses
export interface LoginResponse {
  user: UserWithSiteContext;
  token: string;
  message?: string;
}

export interface RegisterResponse {
  user: UserWithSiteContext;
  token: string;
  message?: string;
}

// Helper function to get current site from environment
export function getCurrentSite(): CodexSite {
  const siteName = process.env.NEXT_PUBLIC_SITE_NAME;
  
  if (!siteName || !Object.values(CODEX_SITES).includes(siteName as CodexSite)) {
    console.warn(`Invalid or missing NEXT_PUBLIC_SITE_NAME: ${siteName}, defaulting to codexsecure`);
    return CODEX_SITES.CODEXSECURE;
  }
  
  return siteName as CodexSite;
}

// Helper to check if user is registered on a specific site
export function isUserRegisteredOnSite(user: User, site: CodexSite): boolean {
  return user.reg_sites.includes(site);
}

// Helper to get site display name
export function getSiteDisplayName(site: CodexSite): string {
  const names: Record<CodexSite, string> = {
    web3connected: 'Web3 Connected',
    codexhash: 'CodexHash',
    codexsecure: 'CodexSecure',
  };
  return names[site] || site;
}

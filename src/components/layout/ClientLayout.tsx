'use client';

import React from 'react';
import { ThemeProvider } from "@/contexts/ThemeContext";
import { LoadingProvider } from "@/contexts/LoadingContext";
import { AuthProvider } from "@/contexts/AuthContext";
import CodexSecureHeader from "@/components/layout/CodexSecureHeader";
import CodexSecureFooter from "@/components/layout/CodexSecureFooter";

/**
 * ClientLayout - Client-side wrapper for all providers and layout components
 * This separates client-side logic from the server-side RootLayout
 * Fixes hydration and loading issues by properly managing client/server boundaries
 * 
 * Provider Order (inside to outside):
 * 1. AuthProvider - Authentication state
 * 2. LoadingProvider - Global loading states
 * 3. ThemeProvider - Theme management
 */
export function ClientLayout({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      <LoadingProvider>
        <AuthProvider>
          <CodexSecureHeader />
          <main>
            {children}
          </main>
          <CodexSecureFooter />
        </AuthProvider>
      </LoadingProvider>
    </ThemeProvider>
  );
}

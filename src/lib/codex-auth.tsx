'use client';

/**
 * Local auth stubs — replaces @web3codex/react-codex-auth
 * Replace with real auth integration when ready.
 */

import React, { createContext, useContext, useState } from 'react';

interface AuthUser {
  id: string;
  email: string;
  name: string;
}

interface AuthContextValue {
  isAuthenticated: boolean;
  user: AuthUser | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue>({
  isAuthenticated: false,
  user: null,
  loading: false,
  login: async () => {},
  logout: () => {},
});

export function useCodexAuth() {
  return useContext(AuthContext);
}

// ---- AuthContainer --------------------------------------------------------

interface AuthContainerProps {
  title?: string;
  subtitle?: string;
  logo?: React.ReactNode;
  showCodexBranding?: boolean;
  children: React.ReactNode;
}

export function AuthContainer({ title, subtitle, logo, children }: AuthContainerProps) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950 px-4">
      <div className="w-full max-w-md bg-slate-900 rounded-xl border border-slate-700 p-8 shadow-xl">
        {logo && <div className="flex justify-center mb-6">{logo}</div>}
        {title && <h1 className="text-2xl font-bold text-white text-center mb-2">{title}</h1>}
        {subtitle && <p className="text-slate-400 text-center mb-8 text-sm">{subtitle}</p>}
        {children}
      </div>
    </div>
  );
}

// ---- LoginForm ------------------------------------------------------------

interface LoginFormProps {
  onSuccess?: () => void;
  onError?: (error: string) => void;
  showRegisterLink?: boolean;
  registerUrl?: string;
  showForgotPassword?: boolean;
  forgotPasswordUrl?: string;
}

export function LoginForm({ onSuccess, onError, showRegisterLink, registerUrl, showForgotPassword, forgotPasswordUrl }: LoginFormProps) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) throw new Error('Invalid credentials');
      onSuccess?.();
    } catch (err) {
      onError?.((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Email</label>
        <input
          type="email"
          value={email}
          onChange={e => setEmail(e.target.value)}
          required
          className="w-full px-4 py-2 bg-slate-800 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-slate-400"
          placeholder="you@example.com"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Password</label>
        <input
          type="password"
          value={password}
          onChange={e => setPassword(e.target.value)}
          required
          className="w-full px-4 py-2 bg-slate-800 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-slate-400"
          placeholder="••••••••"
        />
      </div>
      {showForgotPassword && forgotPasswordUrl && (
        <div className="text-right">
          <a href={forgotPasswordUrl} className="text-sm text-slate-400 hover:text-white transition-colors">Forgot password?</a>
        </div>
      )}
      <button
        type="submit"
        disabled={submitting}
        className="w-full py-2.5 bg-slate-700 hover:bg-slate-600 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
      >
        {submitting ? 'Signing in…' : 'Sign In'}
      </button>
      {showRegisterLink && registerUrl && (
        <p className="text-center text-sm text-slate-400">
          No account? <a href={registerUrl} className="text-white hover:underline">Register</a>
        </p>
      )}
    </form>
  );
}

// ---- PasswordResetForm ----------------------------------------------------

interface PasswordResetFormProps {
  onSubmit?: (email: string) => Promise<{ success: boolean; message: string }>;
  onSuccess?: () => void;
  onError?: (error: string) => void;
  showBackToLogin?: boolean;
  loginUrl?: string;
}

export function PasswordResetForm({ onSubmit, onSuccess, onError, showBackToLogin, loginUrl }: PasswordResetFormProps) {
  const [email, setEmail] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      if (onSubmit) {
        const result = await onSubmit(email);
        if (result.success) {
          onSuccess?.();
        } else {
          onError?.(result.message);
        }
      } else {
        const res = await fetch('/api/auth/reset-password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email }),
        });
        if (!res.ok) throw new Error('Reset failed');
        onSuccess?.();
      }
    } catch (err) {
      onError?.((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-1">Email</label>
        <input
          type="email"
          value={email}
          onChange={e => setEmail(e.target.value)}
          required
          className="w-full px-4 py-2 bg-slate-800 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-slate-400"
          placeholder="you@example.com"
        />
      </div>
      <button
        type="submit"
        disabled={submitting}
        className="w-full py-2.5 bg-slate-700 hover:bg-slate-600 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
      >
        {submitting ? 'Sending…' : 'Send Reset Link'}
      </button>
      {showBackToLogin && loginUrl && (
        <p className="text-center text-sm text-slate-400">
          <a href={loginUrl} className="text-slate-300 hover:text-white underline">Back to Login</a>
        </p>
      )}
    </form>
  );
}

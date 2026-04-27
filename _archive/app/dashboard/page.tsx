'use client';

import React from 'react';
import { useCodexAuth } from '@/lib/codex-auth';
import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

export default function DashboardPage() {
  const { isAuthenticated, user, logout, loading } = useCodexAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      router.push('/login');
    }
  }, [isAuthenticated, loading, router]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p>Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto">
        <div className="bg-white rounded-lg shadow-lg p-8">
          <div className="flex justify-between items-center mb-8">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">
                CodexHash Dashboard
              </h1>
              <p className="text-gray-600">
                Welcome back, {user?.name || user?.email}!
              </p>
            </div>
            <button
              onClick={logout}
              className="bg-red-600 text-white px-6 py-2 rounded-lg hover:bg-red-700 transition-colors"
            >
              Sign Out
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-blue-50 p-6 rounded-lg border border-blue-200">
              <h3 className="text-lg font-semibold text-blue-900 mb-2">
                🔐 Quantum-Resistant
              </h3>
              <p className="text-blue-700">
                Your data is secured with cutting-edge quantum-resistant cryptography
              </p>
            </div>

            <div className="bg-green-50 p-6 rounded-lg border border-green-200">
              <h3 className="text-lg font-semibold text-green-900 mb-2">
                ⚡ Real-Time Hashing
              </h3>
              <p className="text-green-700">
                Generate secure hashes with temporal integrity units (TIU)
              </p>
            </div>

            <div className="bg-purple-50 p-6 rounded-lg border border-purple-200">
              <h3 className="text-lg font-semibold text-purple-900 mb-2">
                🌐 Global Sync
              </h3>
              <p className="text-purple-700">
                Synchronized with CodexTime for universal validation
              </p>
            </div>
          </div>

          <div className="mt-8 bg-gray-50 p-6 rounded-lg">
            <h2 className="text-xl font-semibold text-gray-900 mb-4">
              User Information
            </h2>
            <div className="space-y-2 text-gray-700">
              <p><strong>Email:</strong> {user?.email || 'N/A'}</p>
              <p><strong>Name:</strong> {user?.name || 'N/A'}</p>
              <p><strong>Authentication:</strong> Active</p>
              <p><strong>Security Level:</strong> Quantum-Resistant (95%)</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

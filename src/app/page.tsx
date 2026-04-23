"use client";

import { useAuth } from "@/hooks";

export default function Home() {
  const { user, isAuthenticated, currentSite } = useAuth();

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
      <div className="container mx-auto px-4 py-16">
        {/* Hero Section */}
        <div className="text-center mb-16">
          <h1 className="text-5xl font-bold text-white mb-6">
            Welcome to CodexSecure
          </h1>
          <p className="text-xl text-slate-300 max-w-2xl mx-auto">
            Zone-based security, threat detection, and privacy framework for
            enterprise blockchain applications.
          </p>
          {isAuthenticated && (
            <p className="mt-4 text-green-400">Welcome back, {user?.name}!</p>
          )}
        </div>

        {/* Simple Content Grid */}
        <div className="grid md:grid-cols-3 gap-8 max-w-6xl mx-auto">
          <div className="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-lg p-6">
            <div className="text-4xl mb-4">🛡️</div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Security Zones
            </h3>
            <p className="text-slate-300">
              Implement granular security controls across your infrastructure.
            </p>
          </div>

          <div className="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-lg p-6">
            <div className="text-4xl mb-4">🔐</div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Threat Detection
            </h3>
            <p className="text-slate-300">
              Real-time monitoring and automated threat response systems.
            </p>
          </div>

          <div className="bg-slate-800/50 backdrop-blur-sm border border-slate-700 rounded-lg p-6">
            <div className="text-4xl mb-4">🔒</div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Privacy Framework
            </h3>
            <p className="text-slate-300">
              Built-in compliance and data protection mechanisms.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

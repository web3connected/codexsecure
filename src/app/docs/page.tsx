'use client';

import React, { useState } from 'react';
import { Book, Code, Clock, Play, Lock } from 'lucide-react';
import { DocsSidebarLayout } from '@/components/codex/shared/panels/DocsSidebarLayout';
import { FeatureOverviewPanel } from '@/components/codex/shared/panels/FeatureOverviewPanel';
import { ConceptCardsWithCodePanel } from '@/components/codex/shared/panels/ConceptCardsWithCodePanel';
import { ApiEndpointTablePanel } from '@/components/codex/shared/panels/ApiEndpointTablePanel';
import { TabbedCodeViewer } from '@/components/codex/shared/panels/TabbedCodeViewer';
import type { DocNavItem } from '@/components/codex/shared/panels/DocsSidebarLayout';
import {
  docsNavItems,
  phaseLockingIntentSources,
  phaseLockingCodeBlock,
  phaseLockingBulletItems,
  phaseLockingConceptCards,
  overviewDescription,
  overviewChecklistTitle,
  overviewChecklistItems,
  zoneFrameworkCodeBlock,
  zoneFrameworkBulletItems,
  routeAnalysisCodeBlock,
  routeAnalysisBulletItems,
  tiuEnforcementCodeBlock,
  tiuEnforcementBulletItems,
  apiEndpoints,
  codeExamples,
} from '@/data/docs.data';

// ─── Icon map ─────────────────────────────────────────────────────────────────

const LayersIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
  </svg>
);
const GitForkIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <circle cx="6" cy="6" r="2" /><circle cx="18" cy="6" r="2" /><circle cx="12" cy="18" r="2" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 8v2a4 4 0 004 4h4a4 4 0 004-4V8M6 8v2c0 2.21 1.79 4 4 4" />
  </svg>
);
const ShieldIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
  </svg>
);

const iconMap: Record<string, React.ReactNode> = {
  Book:    <Book    className="w-4 h-4" />,
  Layers:  <LayersIcon />,
  GitFork: <GitForkIcon />,
  Clock:   <Clock   className="w-4 h-4" />,
  Code:    <Code    className="w-4 h-4" />,
  Play:    <Play    className="w-4 h-4" />,
  Lock:    <Lock    className="w-4 h-4" />,
};

const navItems: DocNavItem[] = docsNavItems.map((item) => ({
  id:    item.id,
  label: item.label,
  icon:  iconMap[item.iconName] ?? null,
}));

// ─── Shared dot bullet ────────────────────────────────────────────────────────
const Dot = () => <span className="w-2 h-2 rounded-full bg-secure-secondary inline-block" />;

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function DocsPage() {
  const [activeSection, setActiveSection] = useState('overview');

  return (
    <>
      {/* Hero banner */}
      <div className="bg-secure-bg border-b border-slate-800">
        <div className="container mx-auto px-6 py-12 max-w-7xl">
          <span className="inline-block text-xs font-semibold tracking-widest uppercase text-secure-secondary bg-secure-secondary/10 border border-secure-secondary/20 rounded-full px-3 py-1 mb-4">
            Documentation
          </span>
          <h1 className="text-4xl lg:text-5xl font-bold text-white mb-3">
            CodexSecure{' '}
            <span className="bg-gradient-to-r from-secure-primary to-secure-accent bg-clip-text text-transparent">
              Technical Docs
            </span>
          </h1>
          <p className="text-lg text-slate-400 max-w-2xl">
            Zone-based security middleware, automatic route classification, TIU token enforcement, and the full API reference — everything you need to integrate CodexSecure.
          </p>
        </div>
      </div>

      <DocsSidebarLayout
        sidebarTitle="Documentation"
        navItems={navItems}
        activeSection={activeSection}
        onSectionChange={setActiveSection}
      >
        {/* ── Overview ─────────────────────────────────────────────────────── */}
        {activeSection === 'overview' && (
          <FeatureOverviewPanel
            title="CodexSecure Overview"
            description={overviewDescription}
            statCards={[
              {
                icon: <ShieldIcon />,
                title: 'Automatic Zone Classification',
                description:
                  'Every request is classified into one of 13 zones (Z0–Z12) via the route analyzer — no manual tagging, no scattered middleware.',
              },
              {
                icon: <LayersIcon />,
                title: 'Recursive Zone Model',
                description:
                  'Each zone can define its own Z0–Z12 sub-hierarchy. Security scales with your architecture, from a single service to a full mesh.',
              },
            ]}
            checklistTitle={overviewChecklistTitle}
            checklistItems={overviewChecklistItems}
          />
        )}

        {/* ── Zone Framework ───────────────────────────────────────────────── */}
        {activeSection === 'zone-framework' && (
          <ConceptCardsWithCodePanel
            title="Zone Framework"
            subtitle="From Z0 (Orchestrator) to Z12 (Admin) — a fixed hierarchy that maps to your system's real trust levels."
            conceptCards={[
              {
                icon: <ShieldIcon />,
                title: 'Z0 — The Orchestrator',
                description:
                  'Every request enters through Z0. The route analyzer fingerprints the path and promotes the request to its assigned zone. Unclassifiable requests are rejected here.',
              },
              {
                icon: <LayersIcon />,
                title: 'Tiered Enforcement',
                description:
                  'Open (Z1–Z3): no auth. Protected (Z4–Z6): full auth + TIU + session. Secure (Z7–Z9): restricted methods. Critical (Z10–Z12): full stack, broadest or narrowest access.',
              },
            ]}
            codeBlockContent={zoneFrameworkCodeBlock}
            bulletTitle="Zone Tiers"
            bulletItems={zoneFrameworkBulletItems.map((item) => ({
              icon: <Dot />,
              label: item.label,
              value: item.value,
            }))}
          />
        )}

        {/* ── Route Analysis ───────────────────────────────────────────────── */}
        {activeSection === 'route-analysis' && (
          <ConceptCardsWithCodePanel
            title="Route Analysis"
            subtitle="CodexRouteAnalyzer fingerprints every inbound path using keyword-strategy matching — no route decoration required."
            conceptCards={[
              {
                icon: <GitForkIcon />,
                title: 'Keyword Strategy Detection',
                description:
                  'Each zone has a keyword list. The analyzer scores every zone against the path segments and assigns the zone with the highest confidence. All 13 strategies run in parallel.',
              },
              {
                icon: <ShieldIcon />,
                title: 'Drift Detection',
                description:
                  'A SHA-256 fingerprint of the route pattern is stored. If the pattern changes across deploys — schema drift, renamed segments — the system detects and flags the change.',
              },
            ]}
            codeBlockContent={routeAnalysisCodeBlock}
            bulletTitle="Analysis Properties"
            bulletItems={routeAnalysisBulletItems.map((item) => ({
              icon: <Dot />,
              label: item.label,
              value: item.value,
            }))}
          />
        )}

        {/* ── TIU Enforcement ──────────────────────────────────────────────── */}
        {activeSection === 'tiu-enforcement' && (
          <ConceptCardsWithCodePanel
            title="TIU Enforcement"
            subtitle="CodexTime mints the token. CodexSecure validates it offline — no runtime dependency on CodexTime at request time."
            conceptCards={[
              {
                icon: <ShieldIcon />,
                title: 'Offline Validation',
                description:
                  'When TIU_HMAC_SECRET is configured, CodexSecure validates the HMAC-SHA256 signature and expiry locally. No outbound call to CodexTime is made during request processing.',
              },
              {
                icon: <LayersIcon />,
                title: 'Device Binding',
                description:
                  'Tokens can be bound to a device UID via the X-Device-UID header. CodexSecure rejects tokens that were minted for a different device fingerprint.',
              },
            ]}
            codeBlockContent={tiuEnforcementCodeBlock}
            bulletTitle="Token Properties"
            bulletItems={tiuEnforcementBulletItems.map((item) => ({
              icon: <Dot />,
              label: item.label,
              value: item.value,
            }))}
          />
        )}

        {/* ── API Reference ────────────────────────────────────────────────── */}
        {activeSection === 'api-reference' && (
          <ApiEndpointTablePanel
            title="API Reference"
            subtitle="codexsecure-api · Go/Gin · port 8084"
            baseUrl="http://localhost:8084"
            endpoints={apiEndpoints.map((ep) => ({
              method:      ep.method,
              path:        ep.path,
              description: ep.description,
              notes:       ep.notes,
            }))}
          />
        )}

        {/* ── Code Examples ────────────────────────────────────────────────── */}
        {activeSection === 'examples' && (
          <TabbedCodeViewer
            title="Code Examples"
            subtitle="Working examples across cURL, JavaScript, and Python."
            examples={codeExamples.map((ex) => ({
              id:       ex.language,
              title:    ex.label,
              language: ex.language,
              code:     ex.code,
            }))}
          />
        )}

        {/* ── Phase Locking ────────────────────────────────────────────────── */}
        {activeSection === 'phase-locking' && (
          <div className="space-y-10">
            {/* Header */}
            <div>
              <span className="inline-block text-xs font-semibold tracking-widest uppercase text-secure-secondary bg-secure-secondary/10 border border-secure-secondary/20 rounded-full px-3 py-1 mb-3">
                Security Model
              </span>
              <h2 className="text-3xl font-bold text-white mb-2">Phase Locking</h2>
              <p className="text-slate-400 max-w-2xl">
                A <span className="text-secure-secondary font-medium">phase</span> is the declared
                identity of a route — read from the naming conventions programmers already use.{' '}
                <code className="text-xs text-slate-300">LoginController</code>,{' '}
                <code className="text-xs text-slate-300">login.submit</code>,{' '}
                <code className="text-xs text-slate-300">LoginModel</code>, and{' '}
                <code className="text-xs text-slate-300">/login</code> all say the same thing.
                CodexSecure locks that intent into a pattern baseline — the allowed fields, their
                data formats, and the exact field count. Any deviation is out-of-phase and flagged.
              </p>
            </div>

            {/* Intent sources grid */}
            <div>
              <h3 className="text-lg font-semibold text-white mb-4">How Intent Is Read</h3>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                {phaseLockingIntentSources.map((source) => (
                  <div
                    key={source.id}
                    className="p-4 rounded-xl border bg-slate-900/60 backdrop-blur-sm"
                    style={{ borderColor: `${source.color}33` }}
                  >
                    <div
                      className="inline-block text-xs font-bold font-mono px-2 py-0.5 rounded mb-2 border"
                      style={{ borderColor: source.color, color: source.color, backgroundColor: `${source.color}15` }}
                    >
                      {source.id}
                    </div>
                    <div
                      className="text-sm font-mono font-semibold mb-1"
                      style={{ color: source.color }}
                    >
                      {source.example}
                    </div>
                    <p className="text-xs text-slate-400 leading-relaxed">{source.description}</p>
                  </div>
                ))}
              </div>
            </div>

            {/* Concept cards */}
            <div>
              <h3 className="text-lg font-semibold text-white mb-4">How Phase Lock Works</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {phaseLockingConceptCards.map((card) => (
                  <div
                    key={card.title}
                    className="p-5 rounded-xl border border-slate-700 bg-slate-900/50 backdrop-blur-sm"
                  >
                    <div className="flex items-center gap-2 mb-2">
                      <Lock className="w-4 h-4 text-secure-primary flex-shrink-0" />
                      <h4 className="font-semibold text-white text-sm">{card.title}</h4>
                    </div>
                    <p className="text-slate-400 text-sm leading-relaxed">{card.description}</p>
                  </div>
                ))}
              </div>
            </div>

            {/* Code block */}
            <div>
              <h3 className="text-lg font-semibold text-white mb-4">Locked Pattern &amp; Out-of-Phase Examples</h3>
              <div className="rounded-xl overflow-hidden border border-slate-700">
                <div className="flex items-center gap-2 px-4 py-2 bg-slate-800/80 border-b border-slate-700">
                  <div className="w-3 h-3 rounded-full bg-red-500/60" />
                  <div className="w-3 h-3 rounded-full bg-yellow-500/60" />
                  <div className="w-3 h-3 rounded-full bg-green-500/60" />
                  <span className="ml-2 text-xs text-slate-500 font-mono">zone_router.go — phase lock</span>
                </div>
                <pre className="bg-slate-900 p-5 overflow-x-auto">
                  <code className="text-sm font-mono text-slate-300 whitespace-pre">{phaseLockingCodeBlock}</code>
                </pre>
              </div>
            </div>

            {/* Properties table */}
            <div>
              <h3 className="text-lg font-semibold text-white mb-4">Phase Lock Properties</h3>
              <div className="rounded-xl border border-slate-700 overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-slate-800/60 border-b border-slate-700">
                      <th className="text-left px-4 py-3 font-semibold text-slate-300 w-1/3">Property</th>
                      <th className="text-left px-4 py-3 font-semibold text-slate-300">Detail</th>
                    </tr>
                  </thead>
                  <tbody>
                    {phaseLockingBulletItems.map((item, idx) => (
                      <tr
                        key={item.label}
                        className={`border-b border-slate-800 ${
                          idx % 2 === 0 ? 'bg-slate-900/40' : 'bg-slate-900/20'
                        }`}
                      >
                        <td className="px-4 py-3 font-mono text-secure-secondary font-medium">{item.label}</td>
                        <td className="px-4 py-3 text-slate-400">{item.value}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </DocsSidebarLayout>
    </>
  );
}

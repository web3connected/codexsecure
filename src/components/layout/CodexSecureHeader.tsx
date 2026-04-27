'use client';

import React, { useState, useRef, useEffect } from 'react';
import GlobalHeader from '@/components/codex/shared/Header/GlobalHeader';
import ApplicationLogo from '@/components/codex/shared/widgets/ApplicationLogo';
import DateGreeterWidget from '@/components/codex/shared/widgets/DateGreeterWidget';
import HeaderActions from '@/components/codex/shared/widgets/HeaderActions';

const navLinks = [
  { label: 'Home',            href: '/' },
  { label: 'Security Zones',  href: '/zones' },
  { label: 'Docs',            href: '/docs' },
  { label: 'Getting Started', href: '/getting-started' },
];

const sdkItems = [
  { label: 'Go Client',          href: 'https://github.com/web3connected/codexsecure', external: true },
  { label: 'JavaScript / TypeScript', href: 'https://github.com/web3connected/codexsecure', external: true },
];

const CodexSecureNav: React.FC = () => {
  const [sdkOpen, setSdkOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setSdkOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  return (
    <nav className="flex items-center gap-6">
      {navLinks.map((link) => (
        <a
          key={link.href}
          href={link.href}
          className="text-sm text-white/70 hover:text-white transition-colors"
        >
          {link.label}
        </a>
      ))}

      {/* SDK dropdown */}
      <div className="relative" ref={dropdownRef}>
        <button
          onClick={() => setSdkOpen((o) => !o)}
          className="flex items-center gap-1 text-sm text-white/70 hover:text-white transition-colors"
        >
          SDK
          <svg
            className={`w-3 h-3 transition-transform ${sdkOpen ? 'rotate-180' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2.5}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {sdkOpen && (
          <div className="absolute top-full right-0 mt-2 w-56 bg-secure-surface border border-secure-border rounded-lg shadow-xl z-50 py-1">
            {sdkItems.map((item) => (
              <a
                key={item.href}
                href={item.href}
                target={item.external ? '_blank' : undefined}
                rel={item.external ? 'noopener noreferrer' : undefined}
                className="block px-4 py-2 text-sm text-white/70 hover:text-white hover:bg-white/5 transition-colors"
              >
                {item.label}
              </a>
            ))}
          </div>
        )}
      </div>
    </nav>
  );
};

export default function CodexSecureHeader() {
  return (
    <GlobalHeader
      topbarWidgets={[
        {
          slot: 'widget_01',
          name: 'DateGreeter',
          component: DateGreeterWidget,
        },
        {
          slot: 'widget_03',
          name: 'HeaderActions',
          component: HeaderActions,
          props: {
            githubUrl:     'https://github.com/web3connected/codexsecure',
            getStartedUrl: '/getting-started',
          },
        },
      ]}
      logoWidget={{
        name: 'ApplicationLogo',
        component: ApplicationLogo,
        props: { logo: 'CodexSecure', showIcon: true },
      }}
      navWidget={{
        name: 'CodexSecureNav',
        component: CodexSecureNav,
      }}
    />
  );
}

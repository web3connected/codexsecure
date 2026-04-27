'use client';

import React from 'react';
import Link from 'next/link';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faGithub, faTwitter, faDiscord } from '@fortawesome/free-brands-svg-icons';

const footerLinks = {
  product: [
    { label: 'Security Zones',  href: '/zones' },
    { label: 'Getting Started', href: '/getting-started' },
    { label: 'API Docs',        href: '/docs' },
    { label: 'About',           href: '/about' },
  ],
  sdks: [
    { label: 'Go Client',               href: 'https://github.com/web3connected/codexsecure', external: true },
    { label: 'JavaScript / TypeScript', href: 'https://github.com/web3connected/codexsecure', external: true },
  ],
  company: [
    { label: 'About', href: '/about' },
    { label: 'Blog',  href: 'https://web3connected.com/blog', external: true },
  ],
  legal: [
    { label: 'Terms of Service', href: '/terms' },
    { label: 'Privacy Policy',   href: '/privacy' },
  ],
};

const socialLinks = {
  github:  'https://github.com/web3connected',
  twitter: 'https://twitter.com/web3codex',
  discord: 'https://discord.gg/web3codex',
};

export default function CodexSecureFooter() {
  return (
    <footer className="bg-secure-bg border-t border-secure-border">
      <div className="container mx-auto px-6 py-16">
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-8 lg:gap-10">

          {/* Brand Column */}
          <div className="col-span-2 md:col-span-3 lg:col-span-2 mb-8 lg:mb-0">
            <Link href="/" className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-secure-primary to-secure-accent flex items-center justify-center">
                <span className="text-white font-brand font-bold text-lg">S</span>
              </div>
              <span className="font-brand text-2xl font-semibold text-secure-primary">
                CodexSecure
              </span>
            </Link>
            <p className="text-foreground/60 text-sm mb-6 max-w-xs">
              Zone-based security and intelligent route analysis for distributed systems and blockchain applications.
            </p>
            <div className="flex items-center gap-4">
              <a href={socialLinks.github} target="_blank" rel="noopener noreferrer"
                className="text-foreground/40 hover:text-secure-primary transition-colors">
                <FontAwesomeIcon icon={faGithub} className="w-5 h-5" />
              </a>
              <a href={socialLinks.twitter} target="_blank" rel="noopener noreferrer"
                className="text-foreground/40 hover:text-secure-primary transition-colors">
                <FontAwesomeIcon icon={faTwitter} className="w-5 h-5" />
              </a>
              <a href={socialLinks.discord} target="_blank" rel="noopener noreferrer"
                className="text-foreground/40 hover:text-secure-primary transition-colors">
                <FontAwesomeIcon icon={faDiscord} className="w-5 h-5" />
              </a>
            </div>
          </div>

          {/* Product Column */}
          <div>
            <h4 className="text-foreground/90 font-semibold text-sm uppercase tracking-wider mb-4">Product</h4>
            <ul className="space-y-3">
              {footerLinks.product.map((link) => (
                <li key={link.href}>
                  <Link href={link.href} className="text-foreground/50 hover:text-secure-primary text-sm transition-colors">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* SDKs Column */}
          <div>
            <h4 className="text-foreground/90 font-semibold text-sm uppercase tracking-wider mb-4">SDKs</h4>
            <ul className="space-y-3">
              {footerLinks.sdks.map((link) => (
                <li key={link.label}>
                  <a
                    href={link.href}
                    target={link.external ? '_blank' : undefined}
                    rel={link.external ? 'noopener noreferrer' : undefined}
                    className="text-foreground/50 hover:text-secure-primary text-sm transition-colors"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* Company Column */}
          <div>
            <h4 className="text-foreground/90 font-semibold text-sm uppercase tracking-wider mb-4">Company</h4>
            <ul className="space-y-3">
              {footerLinks.company.map((link) => (
                <li key={link.label}>
                  <a
                    href={link.href}
                    target={'external' in link && link.external ? '_blank' : undefined}
                    rel={'external' in link && link.external ? 'noopener noreferrer' : undefined}
                    className="text-foreground/50 hover:text-secure-primary text-sm transition-colors"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          {/* Legal Column */}
          <div>
            <h4 className="text-foreground/90 font-semibold text-sm uppercase tracking-wider mb-4">Legal</h4>
            <ul className="space-y-3">
              {footerLinks.legal.map((link) => (
                <li key={link.href}>
                  <Link href={link.href} className="text-foreground/50 hover:text-secure-primary text-sm transition-colors">
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

        </div>

        {/* Bottom bar */}
        <div className="border-t border-secure-border mt-12 pt-8 flex flex-col md:flex-row items-center justify-between gap-4">
          <p className="text-foreground/40 text-sm">
            © {new Date().getFullYear()} CodexSecure. All rights reserved.
          </p>
          <p className="text-foreground/30 text-xs">
            Built on the Codex Zone Framework · Z1–Z12
          </p>
        </div>
      </div>
    </footer>
  );
}

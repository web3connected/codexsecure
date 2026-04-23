# CodexSecure

Developer portal for **CodexSecure** — the zone-based security enforcement layer of the Codex ecosystem. Live at [codexsecure.io](https://codexsecure.io).

CodexSecure provides route protection, device trust validation, and zone enforcement (Z1–Z12) across all Codex services. This portal serves as the developer-facing documentation, API reference, and integration hub.

## Features

- **Zone Enforcement** — Z1–Z12 zone model for granular route and resource protection
- **Device Trust** — Device fingerprinting and trust scoring integrated into auth flow
- **Route Protection** — Middleware-based route guards with zone-aware redirects
- **Auth Integration** — Session management, JWT validation, and login/register flows
- **Developer Portal** — API reference, integration guides, and zone configuration docs
- **Next.js 15 + App Router** — TypeScript, Tailwind CSS, server and client components

## Tech Stack

- **Framework**: Next.js 15 (App Router)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Auth**: CodexAuth (shared Codex auth system)
- **Port**: 3003 (dev) — proxied by Forge nginx in production
- **Domain**: [codexsecure.io](https://codexsecure.io)

## Project Structure

```
src/
├── app/                  # Next.js App Router pages + API routes
├── components/
│   ├── codex/            # Published SDK widgets (GlobalHeader, Navigation, etc.)
│   ├── layout/           # CodexSecureHeader, CodexSecureFooter, ClientLayout, PageLayout
│   ├── common/           # TitleBar and shared UI
│   └── ...               # Feature components
├── lib/
│   ├── codex-auth.tsx    # Auth stub (useCodexAuth, AuthContainer, LoginForm, etc.)
│   └── zones/            # Zone enforcement logic (CodexZoneGuard, CodexZoneRegistry, etc.)
├── providers/            # AuthProvider, ZoneRouteProvider
├── contexts/             # AuthContext, LoadingContext
├── hooks/                # Shared hooks
└── types/                # Shared TypeScript types
```

## Local Development

```bash
npm install
npm run dev       # Starts on http://localhost:3003
npm run build     # Production build
npm start         # Start production server (port 3003)
```

## Deployment

Deployed via Laravel Forge to `codexsecure.io`. GitHub Actions workflow (`.github/workflows/deploy.yml`) triggers on push to `main`. Forge runs `next start` on port 3003 with nginx proxying inbound traffic.

## Related Codex Sites

| Site | Port | Repo |
|------|------|------|
| [codexhash.io](https://codexhash.io) | 3001 | `web3connected/codexhash` |
| [codextime.io](https://codextime.io) | 3002 | `web3connected/codex-time-server` |
| [codexsecure.io](https://codexsecure.io) | 3003 | `web3connected/codexsecure` |


[![Next.js](https://img.shields.io/badge/Next.js-15.4.6-black)](https://nextjs.org/)

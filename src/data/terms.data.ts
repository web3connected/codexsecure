import type { DocSection, DocNavLink } from '@/components/codex/shared/panels/DocumentSectionsPanel';

export const termsSections: DocSection[] = [
  {
    title: '1. Acceptance of Terms',
    body: 'By accessing or using CodexSecure services, APIs, or SDKs (collectively, "the Service"), you agree to be bound by these Terms of Service. If you do not agree, do not use the Service.',
  },
  {
    title: '2. Description of Service',
    body: 'CodexSecure provides zone-based network security infrastructure including zone enforcement middleware (Z1–Z12), device trust layers, route protection, and developer SDKs. The Service is provided for lawful application security purposes only.',
  },
  {
    title: '3. API Usage',
    body: 'You may use the CodexSecure API for legitimate application security and zone enforcement development. Automated abuse, intentional flooding, probing zone boundaries without authorisation, or attempts to circumvent zone restrictions are prohibited. We reserve the right to suspend access for violations without notice.',
  },
  {
    title: '4. Zone Configuration Responsibilities',
    body: 'You are responsible for correctly configuring zone rules, route mappings, and security policies. Misconfigured zones that create security vulnerabilities in your own systems are your responsibility. We recommend thorough testing in development zones (Z1) before promoting configurations to production zones.',
  },
  {
    title: '5. Developer Responsibilities',
    body: 'You are responsible for securing your API keys and credentials. Do not expose keys in public repositories or client-side code. You are solely responsible for any traffic routed through your zones and for ensuring your use complies with applicable laws.',
  },
  {
    title: '6. Intellectual Property',
    body: 'CodexSecure, the Z1–Z12 zone system, and associated brand assets are owned by Web3Codex. SDKs distributed under open-source licenses retain those licenses. Nothing in these terms grants you rights to the CodexSecure brand or proprietary zone enforcement algorithms beyond what is explicitly licensed.',
  },
  {
    title: '7. Disclaimer of Warranties',
    body: 'The Service is provided "as is" without warranties of any kind. We do not warrant uninterrupted availability, error-free operation, or fitness for a particular security purpose. Use of the Service is at your own risk.',
  },
  {
    title: '8. Limitation of Liability',
    body: 'To the maximum extent permitted by law, Web3Codex shall not be liable for any indirect, incidental, special, or consequential damages arising from your use of the Service, including security incidents resulting from misconfigured zone policies or your own key management failures.',
  },
  {
    title: '9. Changes to Terms',
    body: 'We may update these terms at any time. Continued use of the Service after changes constitutes acceptance. Material changes will be communicated via the developer changelog or email if you are a registered user.',
  },
  {
    title: '10. Governing Law',
    body: 'These terms are governed by the laws of the jurisdiction in which Web3Codex operates. Any disputes shall be resolved in the competent courts of that jurisdiction.',
  },
  {
    title: '11. Contact',
    body: 'For questions about these terms, contact us at legal@web3connected.com.',
  },
];

export const termsNavLinks: DocNavLink[] = [
  { label: 'Privacy Policy', href: '/privacy' },
  { label: '← Back to Home', href: '/' },
];

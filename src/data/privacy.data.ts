import type { DocSection, DocNavLink } from '@/components/codex/shared/panels/DocumentSectionsPanel';

export const privacySections: DocSection[] = [
  {
    title: '1. Overview',
    body: 'This Privacy Policy explains how Web3Codex ("we", "our") collects, uses, and protects information when you use CodexSecure APIs, SDKs, and the developer portal. We are committed to handling your data responsibly and transparently.',
  },
  {
    title: '2. Information We Collect',
    body: 'We collect information you provide directly (such as email for developer accounts), technical usage data (API request metadata, IP addresses, zone identifiers, error logs), and aggregate analytics to improve the Service. We do not collect the content of data flows routed through your zone configurations.',
  },
  {
    title: '3. How We Use Information',
    body: 'Collected information is used to operate and improve the Service, enforce rate limits, detect and prevent abuse, send important service notifications, and respond to support requests. We do not sell or rent your personal information to third parties.',
  },
  {
    title: '4. Zone Configuration Data',
    body: 'Zone definitions, route mappings, and security policies you configure are stored to provide the Service. This configuration data is never shared with third parties. You retain full ownership of your zone configurations and may export or delete them at any time.',
  },
  {
    title: '5. API Keys & Credentials',
    body: 'API keys are hashed and stored securely. We do not have access to your plaintext keys after creation. You are responsible for key rotation and access control. Compromised keys should be revoked immediately through your developer dashboard.',
  },
  {
    title: '6. Data Retention',
    body: 'Account data is retained for the lifetime of your developer account plus 90 days after deletion. Request logs and zone activity are retained for 30 days for operational purposes, then purged. You may request deletion of your account data at any time.',
  },
  {
    title: '7. Third-Party Services',
    body: 'We use third-party services for infrastructure (hosting, monitoring). These providers are bound by data processing agreements and are not permitted to use your data for their own purposes. A full list of sub-processors is available on request.',
  },
  {
    title: '8. Cookies',
    body: 'The developer portal uses session cookies for authentication and preference storage only. We do not use tracking cookies or third-party advertising cookies.',
  },
  {
    title: '9. Your Rights',
    body: 'Depending on your jurisdiction, you may have rights to access, correct, or delete personal data we hold about you. To exercise these rights, contact privacy@web3connected.com. We will respond within 30 days.',
  },
  {
    title: '10. Changes to This Policy',
    body: 'We may update this policy to reflect changes in our practices or legal requirements. Material changes will be communicated via the developer changelog or email. Continued use of the Service after changes constitutes acceptance.',
  },
  {
    title: '11. Contact',
    body: 'For privacy-related questions or requests, contact privacy@web3connected.com.',
  },
];

export const privacyNavLinks: DocNavLink[] = [
  { label: 'Terms of Service', href: '/terms' },
  { label: '← Back to Home', href: '/' },
];

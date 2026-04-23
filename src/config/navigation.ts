/**
 * CodexHash Navigation Configuration
 */

export interface NavItem {
  label: string;
  href: string;
  icon?: string;
  children?: NavItem[];
}

export const CODEXHASH_NAV_ITEMS: NavItem[] = [
  {
    label: 'Home',
    href: '/',
  },
  {
    label: 'Features',
    href: '/features',
  },
  {
    label: 'Documentation',
    href: '/docs',
    children: [
      {
        label: 'Getting Started',
        href: '/docs/getting-started',
      },
      {
        label: 'API Reference',
        href: '/docs/api',
      },
      {
        label: 'SDK Documentation',
        href: '/docs/sdk',
      },
      {
        label: 'Guides',
        href: '/docs/guides',
      },
    ],
  },
  {
    label: 'Pricing',
    href: '/pricing',
  },
  {
    label: 'Blog',
    href: '/blog',
  },
  {
    label: 'Support',
    href: '/support',
    children: [
      {
        label: 'FAQ',
        href: '/support/faq',
      },
      {
        label: 'Community',
        href: '/support/community',
      },
      {
        label: 'Contact',
        href: '/support/contact',
      },
    ],
  },
];

export const CODEXHASH_FOOTER_LINKS = {
  left: [
    { label: 'Features', href: '/features' },
    { label: 'Pricing', href: '/pricing' },
    { label: 'Documentation', href: '/docs' },
    { label: 'API Reference', href: '/docs/api' },
  ],
  right: [
    { label: 'About', href: '/about' },
    { label: 'Blog', href: '/blog' },
    { label: 'Support', href: '/support' },
    { label: 'Privacy Policy', href: '/privacy' },
    { label: 'Terms of Service', href: '/terms' },
  ],
};

export const CODEXHASH_SOCIAL_LINKS = {
  twitter: 'https://twitter.com/codexhash',
  discord: 'https://discord.gg/codexhash',
  // Add more as needed
};

import { ProductSlide } from '@/components/codex/shared/Sliders/types';

/**
 * HeroSlide — Main intro for CodexSecure
 * Layout: overlay — background image with centered overlaid content
 */
const HeroSlide: ProductSlide = {
  id: 'codexsecure-hero',
  layout: 'overlay',
  tag: 'Zone-Based Security',
  tagColor: '#7C3AED',
  title: 'Security Has a Structure',
  description:
    'CodexSecure enforces intelligent security zones across your infrastructure. Every route is analyzed, classified, and protected — automatically.',
  ctaText: 'Get Started',
  ctaLink: '/getting-started',
  image: '/assets/images/secure_slide_one.webp',
  imageAlt: 'Security processing visualization — chip with digital data streams',
  backgroundColor: '#0A0F1E',
};

export default HeroSlide;

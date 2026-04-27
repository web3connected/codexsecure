import { ProductSlide } from '@/components/codex/shared/Sliders/types';

/**
 * ZoneSlide — Explains the 12-zone security structure
 * Layout: split-right — content left, image right
 */
const ZoneSlide: ProductSlide = {
  id: 'codexsecure-zones',
  layout: 'split-right',
  tag: 'Zone Framework',
  tagColor: '#2DF4A1',
  title: 'Twelve Zones. One Consistent Framework.',
  description:
    'From open public access (Z1) to root system oversight (Z12), every route in your application has a home. CodexSecure classifies and enforces policies automatically — no manual tagging required.',
  ctaText: 'View Zone Reference',
  ctaLink: '/docs',
  image: '/assets/images/codexzones.png',
  imageAlt: 'CodexZones — the Z1–Z12 zone framework visualised',
  backgroundColor: '#0F1729',
  stats: [
    { label: 'Zones',  value: 'Z1–Z12' },
    { label: 'Tiers',  value: '3' },
    { label: 'Policy', value: 'auto' },
  ],
};

export default ZoneSlide;

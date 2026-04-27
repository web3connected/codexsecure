import { ProductSlide } from '@/components/codex/shared/Sliders/types';

/**
 * AnalyzerSlide — Route analyzer and fingerprint detection
 * Layout: grid — content header + 3-column stats cards
 */
const AnalyzerSlide: ProductSlide = {
  id: 'codexsecure-analyzer',
  layout: 'grid',
  tag: 'Route Analysis',
  tagColor: '#00E4FF',
  title: 'Every Request. Analyzed.',
  description:
    'The CodexSecure analyzer fingerprints every inbound route, detects drift, validates schema, and assigns zone classification — all in a single middleware pass.',
  ctaText: 'Read the Docs',
  ctaLink: '/docs',
  image: '/assets/images/istockphoto-1367477512-612x612.jpg',
  imageAlt: 'Route analysis — data streams flowing through the analyzer',
  backgroundColor: '#0A0F1E',
  stats: [
    { label: 'Analysis',     value: 'per-route' },
    { label: 'Detection',    value: 'drift + drift' },
    { label: 'Phase Lock',   value: 'Z2 + Z5' },
  ],
};

export default AnalyzerSlide;

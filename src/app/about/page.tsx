import { HeroBgImagePanel } from '@/components/codex/shared/panels/HeroBgImagePanel';
import { SplitTextStatsPanel } from '@/components/codex/shared/panels/SplitTextStatsPanel';
import { BgOverlayCardGridPanel } from '@/components/codex/shared/panels/BgOverlayCardGridPanel';
import { SplitTextFactsPanel } from '@/components/codex/shared/panels/SplitTextFactsPanel';
import { VerticalTimelinePanel } from '@/components/codex/shared/panels/VerticalTimelinePanel';
import { CenteredCtaPanel } from '@/components/codex/shared/panels/CenteredCtaPanel';
import {
  aboutMissionParagraphs,
  aboutMissionHighlight,
  aboutStatCards,
  aboutPrinciples,
  aboutNamingParagraphs,
  aboutNamingFacts,
  aboutTimeline,
  aboutCtaButtons,
} from '@/data/about.data';

export const metadata = {
  title: 'About | CodexSecure',
  description: 'CodexSecure enforces intelligent zone-based security across your infrastructure. Every route is analyzed, classified, and protected — automatically.',
};

export default function AboutPage() {
  return (
    <main className="min-h-screen">
      <HeroBgImagePanel
        badge="About"
        eyebrow="About the platform"
        title="Security that"
        titleHighlight="has a structure."
        titleHighlightClassName="text-transparent bg-clip-text bg-gradient-to-r from-secure-secondary to-secure-accent"
        subtitle="Most security systems tell you what to block. CodexSecure tells you what zone every request belongs in — and enforces the policy automatically."
        backgroundImageSrc="/assets/images/secure_slide_one.webp"
      />
      <SplitTextStatsPanel
        eyebrow="What we built"
        title="Zone-based security middleware"
        paragraphs={aboutMissionParagraphs}
        highlight={aboutMissionHighlight}
        statCards={aboutStatCards}
      />
      <BgOverlayCardGridPanel
        eyebrow="The pillars"
        title="Four principles. One enforcement layer."
        subtitle="Each one addresses a specific failure mode in traditional route security."
        cards={aboutPrinciples}
        backgroundImageSrc="/assets/images/istockphoto-1367477512-612x612.jpg"
      />
      <SplitTextFactsPanel
        eyebrow="The design"
        title="Why zones"
        paragraphs={aboutNamingParagraphs}
        factCards={aboutNamingFacts}
      />
      <VerticalTimelinePanel
        eyebrow="The thinking"
        title="From manual annotations to automatic zone enforcement"
        items={aboutTimeline}
      />
      <CenteredCtaPanel
        eyebrow="Ready to build?"
        title="Start protecting your routes with CodexSecure."
        subtitle="Explore the zone framework and integrate in minutes — no manual route tagging required."
        buttons={aboutCtaButtons}
      />
    </main>
  );
}

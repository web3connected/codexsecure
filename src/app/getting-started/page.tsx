'use client';

import React from 'react';
import { Shield, Zap, Lock } from 'lucide-react';
import { HeroBgImagePanel } from '@/components/codex/shared/panels/HeroBgImagePanel';
import { ThreePanelCardDesign } from '@/components/codex/shared/panels/ThreePanelCardDesign';
import { NumberedStepsPanel } from '@/components/codex/shared/panels/NumberedStepsPanel';
import { CodeBlockQuickStart } from '@/components/codex/shared/panels/CodeBlockQuickStart';
import { CenteredCtaPanel } from '@/components/codex/shared/panels/CenteredCtaPanel';
import type { PanelCard } from '@/components/codex/shared/panels/ThreePanelCardDesign';
import {
  gettingStartedHero,
  gettingStartedFeatureCards,
  gettingStartedSteps,
  gettingStartedLanguages,
  gettingStartedCodeExamples,
  gettingStartedInstallCommands,
  gettingStartedCtaButtons,
} from '@/data/getting-started.data';

// Map icon name strings → React nodes (keeps data file JSX-free)
const iconMap: Record<string, React.ReactNode> = {
  Shield: <Shield className="w-6 h-6" />,
  Zap:    <Zap    className="w-6 h-6" />,
  Lock:   <Lock   className="w-6 h-6" />,
};

const featureCards = gettingStartedFeatureCards.map((card) => ({
  icon:        iconMap[card.iconName],
  title:       card.title,
  tagline:     card.tagline,
  description: card.description,
  stat:        card.stat,
  statLabel:   card.statLabel,
})) as [PanelCard, PanelCard, PanelCard];

export default function GettingStartedPage() {
  return (
    <>
      <HeroBgImagePanel
        badge={gettingStartedHero.badge}
        eyebrow={gettingStartedHero.eyebrow}
        title={gettingStartedHero.title}
        titleHighlight={gettingStartedHero.titleHighlight}
        titleHighlightClassName="text-transparent bg-clip-text bg-gradient-to-r from-secure-secondary to-secure-accent"
        subtitle={gettingStartedHero.subtitle}
        backgroundImageSrc={gettingStartedHero.backgroundImageSrc}
      />

      <ThreePanelCardDesign cards={featureCards} />

      <section className="py-20 lg:py-28 bg-secure-bg">
        <div className="container mx-auto px-4">
          <div className="max-w-3xl mx-auto">
            <NumberedStepsPanel
              title="Integration &amp; Setup"
              subtitle="Five steps from zero to a zone-enforced, TIU-verified request in production."
              steps={gettingStartedSteps}
            />
          </div>
        </div>
      </section>

      <CodeBlockQuickStart
        title="Quick Start"
        subtitle="Classify a route, mint a TIU token, and run a full analysis — no SDK required."
        languages={gettingStartedLanguages}
        codeExamples={gettingStartedCodeExamples}
        installCommands={gettingStartedInstallCommands}
      />

      <CenteredCtaPanel
        eyebrow="What&apos;s next?"
        title="Ready to go deeper?"
        subtitle="Explore the zone framework, phase locking model, and full API reference in the documentation."
        buttons={gettingStartedCtaButtons}
      />
    </>
  );
}

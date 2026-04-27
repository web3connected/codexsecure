import MainSliderComponent from '@/components/codex/shared/Sliders/MainSliderComponent';
import { ThreePanelCardDesign } from '@/components/codex/shared/panels/ThreePanelCardDesign';
import { NumberedStepsPanel } from '@/components/codex/shared/panels/NumberedStepsPanel';
import { CenteredCtaPanel } from '@/components/codex/shared/panels/CenteredCtaPanel';
import { SplitTextStatsPanel } from '@/components/codex/shared/panels/SplitTextStatsPanel';
import HeroSlide from '@/data/slides/HeroSlide';
import ZoneSlide from '@/data/slides/ZoneSlide';
import AnalyzerSlide from '@/data/slides/AnalyzerSlide';
import {
  coreConcepts,
  howItWorksSteps,
  howItWorksFormula,
  byTheNumbersEyebrow,
  byTheNumbersTitle,
  byTheNumbersParagraphs,
  byTheNumbersHighlight,
  byTheNumbersStats,
  ctaEyebrow,
  ctaTitle,
  ctaSubtitle,
  ctaButtons,
} from '@/data/home.data';

const slides = [HeroSlide, ZoneSlide, AnalyzerSlide];

export default function HomePage() {
  return (
    <>
      {/* Header = 40px topbar + 100px nav = 140px total */}
      <MainSliderComponent slides={slides} height="calc(100vh - 140px)" />

      {/* Core Concepts — 3 feature cards */}
      <ThreePanelCardDesign cards={coreConcepts} />

      {/* How It Works — 4 numbered steps + zone tier model */}
      <NumberedStepsPanel
        title="How It Works"
        subtitle="From inbound request to enforced zone policy — in a single middleware pass."
        steps={howItWorksSteps}
        formulaTitle={howItWorksFormula.title}
        formulaLines={howItWorksFormula.lines}
        backgroundImage="/assets/images/quantum-computing.jpg"
      />

      {/* By the Numbers */}
      <SplitTextStatsPanel
        eyebrow={byTheNumbersEyebrow}
        title={byTheNumbersTitle}
        paragraphs={byTheNumbersParagraphs}
        highlight={byTheNumbersHighlight}
        statCards={byTheNumbersStats}
      />

      {/* CTA */}
      <CenteredCtaPanel
        eyebrow={ctaEyebrow}
        title={ctaTitle}
        subtitle={ctaSubtitle}
        buttons={ctaButtons}
      />
    </>
  );
}

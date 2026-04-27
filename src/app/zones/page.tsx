import { ZoneConceptPanel } from '@/components/codex/shared/panels/ZoneConceptPanel';
import { ZoneDualAccordionPanel } from '@/components/codex/shared/panels/ZoneDualAccordionPanel';
import { zones, zoneConceptPanel } from '@/data/zones.data';

export const metadata = {
  title: 'Zone Reference — CodexSecure',
  description:
    'Explore the CodexSecure twelve-zone framework. Every route belongs to exactly one zone. Learn what each zone enforces, what methods it allows, and where it applies.',
};

export default function ZonesPage() {
  return (
    <>
      {/* Top — split accordion reference */}
      <ZoneDualAccordionPanel
        eyebrow="Zone Reference"
        title="All Twelve Zones"
        subtitle="Every route in your system belongs to exactly one zone. Expand any zone to see its policy, allowed methods, and example routes."
        zones={zones}
      />

      {/* Bottom — iterative nature explainer */}
      <ZoneConceptPanel
        eyebrow={zoneConceptPanel.eyebrow}
        title={zoneConceptPanel.title}
        subtitle={zoneConceptPanel.subtitle}
        body={zoneConceptPanel.body}
        pillars={zoneConceptPanel.pillars}
      />
    </>
  );
}

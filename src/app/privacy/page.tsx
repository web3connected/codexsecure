import { DocumentSectionsPanel } from '@/components/codex/shared/panels/DocumentSectionsPanel';
import { privacySections, privacyNavLinks } from '@/data/privacy.data';

export const metadata = {
  title: 'Privacy Policy | CodexSecure',
  description: 'How CodexSecure collects, uses, and protects your data.',
};

export default function PrivacyPage() {
  return (
    <main className="min-h-screen">
      <DocumentSectionsPanel
        eyebrow="Legal"
        title="Privacy Policy"
        lastUpdated="May 2026"
        sections={privacySections}
        navLinks={privacyNavLinks}
      />
    </main>
  );
}

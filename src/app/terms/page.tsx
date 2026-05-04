import { DocumentSectionsPanel } from '@/components/codex/shared/panels/DocumentSectionsPanel';
import { termsSections, termsNavLinks } from '@/data/terms.data';

export const metadata = {
  title: 'Terms of Service | CodexSecure',
  description: 'Terms of service for using the CodexSecure API, SDKs, and zone enforcement tools.',
};

export default function TermsPage() {
  return (
    <main className="min-h-screen">
      <DocumentSectionsPanel
        eyebrow="Legal"
        title="Terms of Service"
        lastUpdated="May 2026"
        sections={termsSections}
        navLinks={termsNavLinks}
      />
    </main>
  );
}

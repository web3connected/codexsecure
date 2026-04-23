/**
 * Example: Using GlobalAssets Components in CodexHash
 * 
 * This file demonstrates how to import and use components from the
 * @web3codex/global-assets package.
 */

// ===== LANDING PAGE EXAMPLE =====
import {
  CodexHashLandingNavigationHeader,
  CodexHashLandingContactFooter,
  SecureDataProtectionHero,
  CodexHashFeaturesOverviewPanel,
  QuantumResistantTechnologyExperience,
  EnterpriseReadySolutionCallToAction
} from '@web3codex/global-assets';

export function LandingPageExample() {
  return (
    <div className="min-h-screen bg-gray-900">
      <CodexHashLandingNavigationHeader />
      <SecureDataProtectionHero />
      <CodexHashFeaturesOverviewPanel />
      <QuantumResistantTechnologyExperience />
      <EnterpriseReadySolutionCallToAction />
      <CodexHashLandingContactFooter />
    </div>
  );
}

// ===== DOCUMENTATION PAGE EXAMPLE =====
import {
  CodexHashTechnicalDocumentationPage,
  StandardPageLayout
} from '@web3codex/global-assets';

export function DocsPageExample() {
  return (
    <StandardPageLayout includeHeader includeFooter>
      <CodexHashTechnicalDocumentationPage />
    </StandardPageLayout>
  );
}

// ===== VERIFICATION WORKFLOW EXAMPLE =====
import {
  CodexHashVerificationWorkflow,
  CenteredContentLayout
} from '@web3codex/global-assets';

export function VerificationPageExample() {
  return (
    <CenteredContentLayout
      title="Hash Verification"
      subtitle="Verify your quantum-resistant hashes"
      breadcrumbs={[
        { label: 'Home', href: '/' },
        { label: 'Tools', href: '/tools' },
        { label: 'Verification' }
      ]}
    >
      <CodexHashVerificationWorkflow />
    </CenteredContentLayout>
  );
}

// ===== PRICING PAGE EXAMPLE =====
import {
  CodexHashDynamicPricingCalculator,
  FullWidthLayout
} from '@web3codex/global-assets';

export function PricingPageExample() {
  return (
    <FullWidthLayout>
      <CodexHashDynamicPricingCalculator />
    </FullWidthLayout>
  );
}

// ===== EDUCATIONAL PANELS EXAMPLE =====
import {
  QuantumLibraryGettingStartedHero,
  HarmonicCorePrinciplesExplainer,
  UniversalLawFoundationsEducation,
  FibonacciHarmonicTechnologyExplainer,
  LiveHashTestingWorkbench
} from '@web3codex/global-assets';

export function EducationalPageExample() {
  return (
    <div className="bg-gray-900">
      <QuantumLibraryGettingStartedHero />
      <HarmonicCorePrinciplesExplainer />
      <UniversalLawFoundationsEducation />
      <FibonacciHarmonicTechnologyExplainer />
      <LiveHashTestingWorkbench />
    </div>
  );
}

// ===== COMPARISON PAGE EXAMPLE =====
import {
  TraditionalVsCodexHashComparison,
  QuantumVsTraditionalComparisonEducation,
  IndustryStandardsPerformanceBenchmarks
} from '@web3codex/global-assets';

export function ComparisonPageExample() {
  return (
    <div className="bg-gray-900">
      <TraditionalVsCodexHashComparison />
      <QuantumVsTraditionalComparisonEducation />
      <IndustryStandardsPerformanceBenchmarks />
    </div>
  );
}

// ===== INTERACTIVE DEMO EXAMPLE =====
import {
  CodexHashInteractiveLandingPage,
  InteractiveTechnologyExplorer,
  LiveCodexHashAPIDemo,
  QuantumHarmonicsInteractiveVisualizer
} from '@web3codex/global-assets';

export function InteractiveDemoExample() {
  return (
    <>
      <CodexHashInteractiveLandingPage />
      {/* Or individual interactive panels: */}
      <InteractiveTechnologyExplorer />
      <LiveCodexHashAPIDemo />
      <QuantumHarmonicsInteractiveVisualizer />
    </>
  );
}

// ===== USE CASE PAGE EXAMPLE =====
import {
  RealWorldEnterpriseUseCases,
  BlockchainIntegrationShowcase
} from '@web3codex/global-assets';

export function UseCasesPageExample() {
  return (
    <div className="bg-gray-900">
      <RealWorldEnterpriseUseCases />
      <BlockchainIntegrationShowcase />
    </div>
  );
}

// ===== CUSTOM LAYOUT EXAMPLE =====
import {
  ApplicationLogo,
  DropdownNavigationMenu,
  SearchAndCallToAction,
  PageTitleWithBreadcrumbs,
  EnterpriseServicesFooter
} from '@web3codex/global-assets';

export function CustomLayoutExample() {
  const menuItems = [
    { title: 'Docs', href: '/docs' },
    { title: 'API', href: '/api' },
    { 
      title: 'Services', 
      href: '/services',
      subMenu: [
        { label: 'Hash Generation', href: '/services/generate' },
        { label: 'Verification', href: '/services/verify' }
      ]
    }
  ];

  return (
    <div className="min-h-screen bg-gray-900">
      <header className="border-b border-gray-800">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <ApplicationLogo logo="CodexHash" />
          <DropdownNavigationMenu menuItems={menuItems} />
          <SearchAndCallToAction />
        </div>
      </header>

      <PageTitleWithBreadcrumbs
        title="My Custom Page"
        description="Using GlobalAssets components"
        breadcrumbs={[
          { label: 'Home', href: '/' },
          { label: 'Custom', href: '/custom' }
        ]}
      />

      <main>
        {/* Your content here */}
      </main>

      <EnterpriseServicesFooter />
    </div>
  );
}

// ===== AGGREGATED SHOWCASE EXAMPLE =====
import {
  CodexHashAllPanelsAggregator,
  CodexHashSinglePageLayout
} from '@web3codex/global-assets';

export function ShowcasePageExample() {
  return (
    <>
      {/* Single page with all sections */}
      <CodexHashSinglePageLayout />
      
      {/* Or all panels in one aggregator */}
      <CodexHashAllPanelsAggregator />
    </>
  );
}

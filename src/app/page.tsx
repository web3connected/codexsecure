import MainHeader from '@/components/common/MainHeader'
import Hero from '@/components/codexhash/Hero'
import HashDemo from '@/components/codexhash/HashDemo'
import HarmonicSpacingExplainer from '@/components/codexhash/HarmonicSpacingExplainer'

export default function Home() {
  return (
    <div className="min-h-screen bg-slate-900">
      <MainHeader />
      
      <Hero />
      
      {/* Interactive Demo Section */}
      <section className="py-16 bg-slate-900">
        <div className="container mx-auto px-6">
          <div className="text-center mb-12">
            <h2 className="text-3xl lg:text-4xl font-bold text-white mb-4">
              Experience CodexHash
            </h2>
            <p className="text-lg text-slate-300 max-w-2xl mx-auto">
              Try our quantum-resistant hashing algorithm with real-time performance metrics
            </p>
          </div>
          
          <HashDemo />
          
          {/* Harmonic Spacing Explainer */}
          <div className="mt-12">
            <HarmonicSpacingExplainer />
          </div>
        </div>
      </section>
    </div>
  )
}

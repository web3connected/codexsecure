'use client'

import React from 'react'
import LandingHeader from '../layout/LandingHeader'
import SecureDataHero from '../panels/codexhash/SecureDataHero'
import CodexHashFeatures from '../panels/codexhash/CodexHashFeatures'
import ExperienceQuantumResistant from '../panels/codexhash/ExperienceQuantumResistant'
import TechnologyStack from '../panels/codexhash/TechnologyStack'
import LandingFooter from '../common/LandingFooter'

const CodexHashSinglePage: React.FC = () => {

  return (
    <div className="bg-gray-900 text-white">
      {/* Header */}
      <LandingHeader />

      {/* Hero Section */}
      <SecureDataHero />

      {/* Features Section */}
      <CodexHashFeatures />

      {/* Demo Section */}
      <ExperienceQuantumResistant />

      {/* Technology Section */}
      <TechnologyStack />

      {/* Footer */}
      <LandingFooter />
    </div>
  )
}

export default CodexHashSinglePage
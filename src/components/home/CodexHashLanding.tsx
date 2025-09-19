'use client'

import React, { useState } from 'react'
import { Shield, Menu, X, Activity, Cpu, Target, Waves } from 'lucide-react'

// Import our modular panel components
import IntroductionPanel from '@/components/panels/IntroductionPanel'
import TechnologyPanel from '@/components/panels/TechnologyPanel'
import ComparisonsPanel from '@/components/panels/ComparisonsPanel'
import HarmonicsPanel from '@/components/panels/HarmonicsPanel'
import TestingPanel from '@/components/panels/TestingPanel'
import SliderHeaderNav from '../layout/SliderHeaderNav'

type Panel = 'intro' | 'technology' | 'comparisons' | 'harmonics' | 'testing'

const CodexHashLanding: React.FC = () => {
  const [activePanel, setActivePanel] = useState<Panel>('intro')
  const [isMenuOpen, setIsMenuOpen] = useState(false)

  const panels = [
    { id: 'intro', label: 'Introduction', icon: Shield },
    { id: 'technology', label: 'Technology', icon: Cpu },
    { id: 'comparisons', label: 'Comparisons', icon: Target },
    { id: 'harmonics', label: 'Harmonic Tech', icon: Waves },
    { id: 'testing', label: 'Live Testing', icon: Activity }
  ]

  const renderPanel = () => {
    switch (activePanel) {
      case 'intro':
        return <IntroductionPanel />
      case 'technology':
        return <TechnologyPanel />
      case 'comparisons':
        return <ComparisonsPanel />
      case 'harmonics':
        return <HarmonicsPanel />
      case 'testing':
        return <TestingPanel />
      default:
        return <IntroductionPanel />
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50 dark:from-slate-900 dark:to-blue-900">
      {/* Sliding Navigation Menu */}
      <SliderHeaderNav
        panels={panels}
        activePanel={activePanel}
        setActivePanel={(panelId: string) => setActivePanel(panelId as Panel)}
        isMenuOpen={isMenuOpen}
        setIsMenuOpen={setIsMenuOpen}
      />

      {/* Header Spacer */}
      <div className="h-20"></div>

      {/* Main Content */}
      <main className="min-h-screen">
        <div className="container mx-auto max-w-6xl px-6 py-8">
          {renderPanel()}
        </div>
      </main>
    </div>
  )
}

export default CodexHashLanding
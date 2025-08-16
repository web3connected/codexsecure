'use client'

import React, { useState } from 'react'

const HarmonicSpacingExplainer: React.FC = () => {
  const [showDetails, setShowDetails] = useState(false)

  return (
    <div className="bg-gradient-to-br from-slate-800 to-slate-700 p-6 rounded-xl border border-slate-600">
      <div className="flex items-center gap-3 mb-4">
        <div className="text-2xl">🎵</div>
        <h3 className="text-xl font-bold text-white">
          What is Harmonic Collision Spacing?
        </h3>
        <button
          onClick={() => setShowDetails(!showDetails)}
          className="ml-auto text-blue-400 hover:text-blue-300 text-sm font-medium"
        >
          {showDetails ? 'Hide Details' : 'Learn More'}
        </button>
      </div>
      
      <p className="text-slate-300 mb-4">
        Unlike traditional hash functions that scatter outputs randomly, CodexHash arranges them in 
        <span className="font-semibold text-blue-400"> mathematical harmonic patterns</span> - 
        like musical notes in a chord.
      </p>

      {showDetails && (
        <div className="space-y-4 mt-6 pt-4 border-t border-slate-600">
          <div className="grid md:grid-cols-2 gap-4">
            <div className="bg-red-900/30 p-4 rounded-lg border border-red-700">
              <h4 className="font-semibold text-red-400 mb-2">❌ Random (Traditional)</h4>
              <div className="h-8 bg-slate-800 rounded border border-red-600 relative mb-2">
                <div className="absolute top-1 left-2 w-1.5 h-6 bg-red-400"></div>
                <div className="absolute top-1 left-8 w-1.5 h-6 bg-red-400"></div>
                <div className="absolute top-1 left-20 w-1.5 h-6 bg-red-400"></div>
                <div className="absolute top-1 left-32 w-1.5 h-6 bg-red-400"></div>
              </div>
              <p className="text-sm text-red-300">Quantum computers can exploit random patterns</p>
            </div>
            
            <div className="bg-green-900/30 p-4 rounded-lg border border-green-700">
              <h4 className="font-semibold text-green-400 mb-2">✅ Harmonic (CodexHash)</h4>
              <div className="h-8 bg-slate-800 rounded border border-green-600 relative mb-2">
                <div className="absolute top-1 left-4 w-1.5 h-6 bg-green-400"></div>
                <div className="absolute top-1 left-12 w-1.5 h-6 bg-green-400"></div>
                <div className="absolute top-1 left-20 w-1.5 h-6 bg-green-400"></div>
                <div className="absolute top-1 left-28 w-1.5 h-6 bg-green-400"></div>
                {/* Forbidden zones */}
                <div className="absolute top-1 left-8 w-2 h-6 bg-gray-600 opacity-70"></div>
                <div className="absolute top-1 left-24 w-2 h-6 bg-gray-600 opacity-70"></div>
              </div>
              <p className="text-sm text-green-300">Structured patterns resist quantum attacks</p>
            </div>
          </div>
          
          <div className="bg-blue-900/30 p-4 rounded-lg border border-blue-700">
            <h4 className="font-semibold text-blue-400 mb-2">🔬 The Science</h4>
            <ul className="text-sm text-blue-200 space-y-1">
              <li>• <strong>Mathematical Harmony:</strong> Like musical intervals (1:2:3:4), hash outputs follow precise mathematical relationships</li>
              <li>• <strong>Forbidden Zones:</strong> Certain hash values become mathematically impossible, preventing collisions</li>
              <li>• <strong>Quantum Resistance:</strong> Even quantum computers can&apos;t break harmonic mathematical constraints</li>
            </ul>
          </div>
          
          <div className="text-center">
            <button className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg text-sm font-medium transition-colors">
              Try Interactive Demo →
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default HarmonicSpacingExplainer

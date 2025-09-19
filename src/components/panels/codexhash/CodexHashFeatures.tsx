import { Activity, Shield, Waves } from 'lucide-react'
import React from 'react'

const CodexHashFeatures = () => {
  return (
    <section id="features" className="py-20 bg-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">Revolutionary Features</h2>
            <p className="text-xl text-gray-400">Advanced cryptographic capabilities for the quantum age</p>
          </div>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            <div className="bg-gray-900 p-8 rounded-xl border border-gray-700 hover:border-purple-500 transition">
              <div className="w-12 h-12 bg-purple-500 rounded-lg flex items-center justify-center mb-6">
                <Waves className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-bold mb-4">HarmonicHash Algorithm</h3>
              <p className="text-gray-400 mb-4">
                Physics-based hashing using speed of light (299,792,458 m/s), Planck frequency (1.855×10⁴³ Hz), and golden ratio (φ = 1.618).
              </p>
              <ul className="text-sm text-gray-500 space-y-1">
                <li>• Universal physics constants</li>
                <li>• Harmonic frequency modulation</li>
                <li>• Sacred geometry integration</li>
              </ul>
            </div>
            
            <div className="bg-gray-900 p-8 rounded-xl border border-gray-700 hover:border-blue-500 transition">
              <div className="w-12 h-12 bg-blue-500 rounded-lg flex items-center justify-center mb-6">
                <Shield className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-bold mb-4">Quantum Resistance</h3>
              <p className="text-gray-400 mb-4">
                Real-time security assessment (30-99% quantum resistance) with adaptive protection against quantum computing attacks.
              </p>
              <ul className="text-sm text-gray-500 space-y-1">
                <li>• Multi-layered security analysis</li>
                <li>• Dynamic resistance scaling</li>
                <li>• Future-proof architecture</li>
              </ul>
            </div>
            
            <div className="bg-gray-900 p-8 rounded-xl border border-gray-700 hover:border-green-500 transition">
              <div className="w-12 h-12 bg-green-500 rounded-lg flex items-center justify-center mb-6">
                <Activity className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-bold mb-4">TIU Temporal Dynamics</h3>
              <p className="text-gray-400 mb-4">
                Time Distortion Units (TIU) add temporal complexity, creating time-dependent security patterns.
              </p>
              <ul className="text-sm text-gray-500 space-y-1">
                <li>• Time-aware hash generation</li>
                <li>• Entropy distortion modeling</li>
                <li>• Temporal security binding</li>
              </ul>
            </div>
          </div>
        </div>
      </section>
  )
}

export default CodexHashFeatures
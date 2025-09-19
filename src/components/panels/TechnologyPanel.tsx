import React from 'react'
import { Cpu, CheckCircle } from 'lucide-react'
import Image from 'next/image'

const TechnologyPanel: React.FC = () => {
  return (
    <div className="min-h-full flex flex-col justify-center space-y-8 relative overflow-hidden">
      {/* Enhanced Tech Background with Data Stream */}
      <div className="absolute inset-0 bg-gradient-to-br from-indigo-900/20 via-blue-900/30 to-purple-900/20"></div>
      <div className="absolute inset-0 bg-[conic-gradient(from_90deg_at_50%_50%,_var(--tw-gradient-stops))] from-transparent via-indigo-500/5 to-transparent animate-spin" style={{animationDuration: '20s'}}></div>
      
      {/* Blue Data Stream Background */}
      <div className="absolute inset-0 opacity-10">
        <Image
          src="/assets/images/blue-data-stream-abstract-background-future-technology-innovation-concept-389117198.webp"
          alt="Data Stream Background"
          fill
          className="object-cover"
        />
      </div>
      
      {/* Circuit Pattern Background */}
      <div className="absolute inset-0 opacity-10">
        <div className="absolute top-10 left-10 w-20 h-1 bg-blue-500"></div>
        <div className="absolute top-10 left-32 w-1 h-20 bg-blue-500"></div>
        <div className="absolute top-32 left-32 w-20 h-1 bg-purple-500"></div>
        <div className="absolute top-20 right-20 w-16 h-1 bg-indigo-500"></div>
        <div className="absolute bottom-20 left-1/4 w-24 h-1 bg-cyan-500"></div>
        <div className="absolute bottom-40 right-1/3 w-1 h-16 bg-blue-500"></div>
      </div>

      <div className="text-center mb-12 relative z-10">
        <div className="relative group mb-6">
          <div className="absolute inset-0 bg-gradient-to-r from-indigo-600 to-purple-600 rounded-full blur-xl opacity-30 group-hover:opacity-50 transition-opacity duration-300"></div>
          <Cpu className="w-16 h-16 text-indigo-600 mx-auto relative z-10 group-hover:scale-110 transition-transform duration-300 drop-shadow-2xl animate-pulse" />
        </div>
        <h2 className="text-4xl font-bold mb-4 text-slate-800 dark:text-slate-200 hover:scale-105 transition-transform duration-300">
          🔬 Web3 Quantum Technology Stack
        </h2>
        <p className="text-xl text-slate-600 dark:text-slate-400 hover:text-slate-500 dark:hover:text-slate-300 transition-colors duration-300">
          The first blockchain-native quantum-resistant cryptographic primitive
        </p>
      </div>

      <div className="grid md:grid-cols-2 gap-8 relative z-10">
        <div className="group bg-white/90 dark:bg-slate-800/90 backdrop-blur-lg rounded-2xl p-8 shadow-xl border border-indigo-200 dark:border-indigo-800 hover:shadow-2xl hover:shadow-indigo-500/20 transition-all duration-300 hover:scale-105 hover:border-indigo-400 dark:hover:border-indigo-600">
          <div className="absolute inset-0 bg-gradient-to-br from-indigo-500/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          <h3 className="text-2xl font-bold mb-4 text-slate-800 dark:text-slate-200 group-hover:text-indigo-700 dark:group-hover:text-indigo-300 transition-colors duration-300">💎 Smart Contract Integration</h3>
          <p className="text-slate-600 dark:text-slate-400 mb-4 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">
            CodexHash is designed as a Web3-native protocol with direct smart contract integration, 
            enabling seamless deployment across all EVM-compatible chains.
          </p>
          <ul className="space-y-2 text-slate-600 dark:text-slate-400">
            <li className="flex items-center group-hover:scale-105 transition-transform duration-300">
              <CheckCircle className="w-4 h-4 text-green-500 mr-2 group-hover:text-green-400" />
              <span className="group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">Solidity library integration</span>
            </li>
            <li className="flex items-center group-hover:scale-105 transition-transform duration-300" style={{transitionDelay: '0.1s'}}>
              <CheckCircle className="w-4 h-4 text-green-500 mr-2 group-hover:text-green-400" />
              <span className="group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">EVM bytecode optimization</span>
            </li>
            <li className="flex items-center group-hover:scale-105 transition-transform duration-300" style={{transitionDelay: '0.2s'}}>
              <CheckCircle className="w-4 h-4 text-green-500 mr-2 group-hover:text-green-400" />
              <span className="group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">Multi-chain deployment</span>
            </li>
          </ul>
        </div>

        <div className="group bg-white/90 dark:bg-slate-800/90 backdrop-blur-lg rounded-2xl p-8 shadow-xl border border-purple-200 dark:border-purple-800 hover:shadow-2xl hover:shadow-purple-500/20 transition-all duration-300 hover:scale-105 hover:border-purple-400 dark:hover:border-purple-600">
          <div className="absolute inset-0 bg-gradient-to-br from-purple-500/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          <h3 className="text-2xl font-bold mb-4 text-slate-800 dark:text-slate-200 group-hover:text-purple-700 dark:group-hover:text-purple-300 transition-colors duration-300">🧮 Quantum Mathematics</h3>
          <p className="text-slate-600 dark:text-slate-400 mb-4 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">
            Our quantum-resistant algorithms leverage harmonic frequency mathematics and 
            physics-based constants to create unbreakable cryptographic primitives.
          </p>
          <ul className="space-y-2 text-slate-600 dark:text-slate-400">
            <li className="flex items-center group-hover:scale-105 transition-transform duration-300">
              <CheckCircle className="w-4 h-4 text-green-500 mr-2 group-hover:text-green-400" />
              <span className="group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">Golden ratio (φ) quantum harmonics</span>
            </li>
            <li className="flex items-center group-hover:scale-105 transition-transform duration-300" style={{transitionDelay: '0.1s'}}>
              <CheckCircle className="w-4 h-4 text-green-500 mr-2 group-hover:text-green-400" />
              <span className="group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">Time Invariant Units (TIU)</span>
            </li>
            <li className="flex items-center group-hover:scale-105 transition-transform duration-300" style={{transitionDelay: '0.2s'}}>
              <CheckCircle className="w-4 h-4 text-green-500 mr-2 group-hover:text-green-400" />
              <span className="group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">Planck constant integration</span>
            </li>
          </ul>
        </div>
      </div>
    </div>
  )
}

export default TechnologyPanel
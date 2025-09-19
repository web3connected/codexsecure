import React from 'react'
import { Shield, Lock, Zap, Globe, Sparkles } from 'lucide-react'
import Image from 'next/image'

const IntroductionPanel: React.FC = () => {
  return (
    <div className="min-h-full flex flex-col justify-center space-y-8 relative overflow-hidden">
      {/* Enhanced Background with Image */}
      <div className="absolute inset-0 bg-gradient-to-br from-blue-900/20 via-purple-900/30 to-emerald-900/20"></div>
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-blue-500/10 via-transparent to-transparent"></div>
      
      {/* Quantum Computing Background */}
      <div className="absolute inset-0 opacity-5">
        <Image
          src="/assets/images/quantum-computing.jpg"
          alt="Quantum Computing Background"
          fill
          className="object-cover"
          priority
        />
      </div>
      
      {/* Floating Particles */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="animate-pulse absolute top-20 left-1/4 w-2 h-2 bg-blue-400 rounded-full opacity-60"></div>
        <div className="animate-pulse absolute top-40 right-1/3 w-1 h-1 bg-purple-400 rounded-full opacity-40" style={{animationDelay: '1s'}}></div>
        <div className="animate-pulse absolute bottom-32 left-1/3 w-3 h-3 bg-emerald-400 rounded-full opacity-50" style={{animationDelay: '2s'}}></div>
        <div className="animate-pulse absolute top-60 right-1/4 w-1.5 h-1.5 bg-yellow-400 rounded-full opacity-30" style={{animationDelay: '3s'}}></div>
      </div>

      <div className="text-center relative z-10">
        <div className="flex justify-center mb-8">
          <div className="relative group">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-600 to-purple-600 rounded-full blur-xl opacity-30 group-hover:opacity-50 transition-opacity duration-300"></div>
            {/* Replace with CodexHash Icon */}
            <div className="relative z-10 group-hover:scale-110 transition-transform duration-300">
              <Image
                src="/assets/icons/codexIcon.png"
                alt="CodexHash Icon"
                width={80}
                height={80}
                className="drop-shadow-2xl"
              />
            </div>
            <Sparkles className="w-8 h-8 text-yellow-500 absolute -top-2 -right-2 animate-spin slow-spin group-hover:text-yellow-400 transition-colors duration-300" />
          </div>
        </div>
        
        <h1 className="text-6xl font-bold mb-6 bg-gradient-to-r from-blue-600 via-purple-600 to-emerald-600 bg-clip-text text-transparent hover:scale-105 transition-transform duration-300 cursor-default">
          CodexHash
        </h1>
        
        <p className="text-2xl text-slate-600 dark:text-slate-300 mb-4 font-medium hover:text-slate-500 dark:hover:text-slate-200 transition-colors duration-300">
          🌌 The Web3 Native Quantum-Resistant Hashing Protocol
        </p>
        
        <p className="text-lg text-slate-500 dark:text-slate-400 max-w-3xl mx-auto mb-12 leading-relaxed hover:text-slate-400 dark:hover:text-slate-300 transition-colors duration-300">
          Welcome to the future of decentralized cryptography. CodexHash is Web3&apos;s first physics-based quantum-resistant 
          hashing protocol, built specifically for the decentralized ecosystem. Our harmonic frequency algorithms and 
          Time Invariant Units (TIU) create unbreakable digital signatures that secure the metaverse.
        </p>
      </div>

      <div className="grid md:grid-cols-3 gap-8 relative z-10">
        <div className="group bg-white/80 dark:bg-slate-800/80 backdrop-blur-lg rounded-2xl p-6 shadow-xl border border-blue-200 dark:border-blue-800 hover:shadow-2xl hover:shadow-blue-500/20 transition-all duration-300 hover:scale-105 hover:border-blue-400 dark:hover:border-blue-600">
          <div className="absolute inset-0 bg-gradient-to-br from-blue-500/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          <Lock className="w-12 h-12 text-blue-600 mx-auto mb-4 group-hover:text-blue-500 group-hover:scale-110 transition-all duration-300 drop-shadow-lg" />
          <h3 className="text-xl font-bold mb-3 text-slate-800 dark:text-slate-200 group-hover:text-blue-700 dark:group-hover:text-blue-300 transition-colors duration-300">🛡️ DeFi-Grade Security</h3>
          <p className="text-slate-600 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">
            Built for smart contracts, DeFi protocols, and decentralized applications with quantum-resistant protection.
          </p>
        </div>

        <div className="group bg-white/80 dark:bg-slate-800/80 backdrop-blur-lg rounded-2xl p-6 shadow-xl border border-purple-200 dark:border-purple-800 hover:shadow-2xl hover:shadow-purple-500/20 transition-all duration-300 hover:scale-105 hover:border-purple-400 dark:hover:border-purple-600" style={{animationDelay: '0.1s'}}>
          <div className="absolute inset-0 bg-gradient-to-br from-purple-500/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          <Zap className="w-12 h-12 text-purple-600 mx-auto mb-4 group-hover:text-purple-500 group-hover:scale-110 transition-all duration-300 drop-shadow-lg" />
          <h3 className="text-xl font-bold mb-3 text-slate-800 dark:text-slate-200 group-hover:text-purple-700 dark:group-hover:text-purple-300 transition-colors duration-300">⚡ Gas Optimized</h3>
          <p className="text-slate-600 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">
            Minimal gas consumption with lightning-fast execution designed for Layer 1 and Layer 2 networks.
          </p>
        </div>

        <div className="group bg-white/80 dark:bg-slate-800/80 backdrop-blur-lg rounded-2xl p-6 shadow-xl border border-emerald-200 dark:border-emerald-800 hover:shadow-2xl hover:shadow-emerald-500/20 transition-all duration-300 hover:scale-105 hover:border-emerald-400 dark:hover:border-emerald-600" style={{animationDelay: '0.2s'}}>
          <div className="absolute inset-0 bg-gradient-to-br from-emerald-500/5 to-transparent rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          <Globe className="w-12 h-12 text-emerald-600 mx-auto mb-4 group-hover:text-emerald-500 group-hover:scale-110 transition-all duration-300 drop-shadow-lg" />
          <h3 className="text-xl font-bold mb-3 text-slate-800 dark:text-slate-200 group-hover:text-emerald-700 dark:group-hover:text-emerald-300 transition-colors duration-300">🌐 Cross-Chain Ready</h3>
          <p className="text-slate-600 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">
            Native integration with Ethereum, Polygon, BSC, and all major blockchain networks.
          </p>
        </div>
      </div>
    </div>
  )
}

export default IntroductionPanel
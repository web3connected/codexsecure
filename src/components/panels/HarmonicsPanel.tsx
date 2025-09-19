import React from 'react'
import { Waves } from 'lucide-react'
import Image from 'next/image'

const HarmonicsPanel: React.FC = () => {
  return (
    <div className="min-h-full flex flex-col justify-center space-y-8 relative overflow-hidden">
      {/* Enhanced Wave Animation Background with Quantum Time */}
      <div className="absolute inset-0 bg-gradient-to-br from-cyan-900/20 via-blue-900/30 to-teal-900/20"></div>
      
      {/* Quantum Time Background */}
      <div className="absolute inset-0 opacity-8">
        <Image
          src="/assets/images/quantum-time.jpg"
          alt="Quantum Time Background"
          fill
          className="object-cover"
        />
      </div>
      
      {/* Animated Wave Elements */}
      <div className="absolute inset-0 overflow-hidden">
        <svg className="absolute inset-0 w-full h-full" viewBox="0 0 1200 800" preserveAspectRatio="xMidYMid slice">
          <defs>
            <linearGradient id="waveGradient1" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="rgba(6, 182, 212, 0.1)" />
              <stop offset="50%" stopColor="rgba(59, 130, 246, 0.2)" />
              <stop offset="100%" stopColor="rgba(6, 182, 212, 0.1)" />
            </linearGradient>
            <linearGradient id="waveGradient2" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="rgba(14, 165, 233, 0.15)" />
              <stop offset="50%" stopColor="rgba(34, 197, 94, 0.1)" />
              <stop offset="100%" stopColor="rgba(14, 165, 233, 0.15)" />
            </linearGradient>
          </defs>
          
          <path d="M0,400 Q300,200 600,400 T1200,400 L1200,800 L0,800 Z" fill="url(#waveGradient1)" className="animate-pulse" style={{animationDuration: '4s'}} />
          <path d="M0,450 Q300,250 600,450 T1200,450 L1200,800 L0,800 Z" fill="url(#waveGradient2)" className="animate-pulse" style={{animationDuration: '6s', animationDelay: '1s'}} />
        </svg>
        
        {/* Floating Physics Symbols */}
        <div className="absolute top-20 left-1/4 text-cyan-400 text-2xl animate-bounce opacity-30" style={{animationDelay: '0s'}}>φ</div>
        <div className="absolute top-40 right-1/3 text-blue-400 text-xl animate-bounce opacity-40" style={{animationDelay: '1s'}}>ℏ</div>
        <div className="absolute bottom-32 left-1/3 text-teal-400 text-3xl animate-bounce opacity-20" style={{animationDelay: '2s'}}>π</div>
        <div className="absolute top-60 right-1/4 text-cyan-300 text-lg animate-bounce opacity-50" style={{animationDelay: '3s'}}>∞</div>
      </div>

      <div className="text-center mb-12 relative z-10">
        <div className="relative group mb-6">
          <div className="absolute inset-0 bg-gradient-to-r from-cyan-600 to-blue-600 rounded-full blur-xl opacity-30 group-hover:opacity-50 transition-opacity duration-300"></div>
          <Waves className="w-16 h-16 text-cyan-600 mx-auto relative z-10 group-hover:scale-110 transition-transform duration-300 drop-shadow-2xl" />
          <div className="absolute inset-0 animate-ping rounded-full bg-cyan-400 opacity-20"></div>
        </div>
        <h2 className="text-4xl font-bold mb-4 text-slate-800 dark:text-slate-200 hover:scale-105 transition-transform duration-300">
          🌊 Harmonic Frequency Technology
        </h2>
        <p className="text-xl text-slate-600 dark:text-slate-400 hover:text-slate-500 dark:hover:text-slate-300 transition-colors duration-300">
          The physics behind Web3&apos;s most advanced cryptographic primitive
        </p>
      </div>

      <div className="grid md:grid-cols-1 gap-8 relative z-10">
        <div className="group bg-gradient-to-br from-cyan-50 to-blue-100 dark:from-cyan-900/20 dark:to-blue-900/20 rounded-2xl p-8 border border-cyan-200 dark:border-cyan-800 hover:shadow-2xl hover:shadow-cyan-500/20 transition-all duration-300 hover:scale-[1.02] backdrop-blur-lg">
          <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/5 to-blue-500/5 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
          <h3 className="text-2xl font-bold mb-6 text-slate-800 dark:text-slate-200 group-hover:text-cyan-700 dark:group-hover:text-cyan-300 transition-colors duration-300">🎵 Quantum Harmonic Oscillators</h3>
          <p className="text-slate-600 dark:text-slate-400 mb-6 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors duration-300">
            CodexHash harnesses the fundamental frequencies of quantum harmonic oscillators to generate 
            cryptographic entropy that is inherently unpredictable and quantum-resistant.
          </p>
          
          <div className="grid md:grid-cols-3 gap-6 mb-6">
            <div className="group bg-white/90 dark:bg-slate-800/90 rounded-xl p-4 text-center hover:scale-110 transition-all duration-300 hover:shadow-lg hover:shadow-yellow-500/20 border border-yellow-200 dark:border-yellow-800">
              <div className="text-3xl mb-2 text-yellow-600 group-hover:scale-125 transition-transform duration-300">φ</div>
              <div className="font-semibold text-sm group-hover:text-yellow-600 transition-colors duration-300">Golden Ratio</div>
              <div className="text-xs text-slate-500 group-hover:text-slate-600 transition-colors duration-300">1.618033988...</div>
              <div className="w-full h-1 bg-yellow-200 dark:bg-yellow-800 rounded-full mt-2 overflow-hidden">
                <div className="h-full bg-yellow-500 rounded-full group-hover:animate-pulse" style={{width: '61.8%'}}></div>
              </div>
            </div>
            <div className="group bg-white/90 dark:bg-slate-800/90 rounded-xl p-4 text-center hover:scale-110 transition-all duration-300 hover:shadow-lg hover:shadow-purple-500/20 border border-purple-200 dark:border-purple-800">
              <div className="text-3xl mb-2 text-purple-600 group-hover:scale-125 transition-transform duration-300">ℏ</div>
              <div className="font-semibold text-sm group-hover:text-purple-600 transition-colors duration-300">Planck Constant</div>
              <div className="text-xs text-slate-500 group-hover:text-slate-600 transition-colors duration-300">6.626 × 10⁻³⁴</div>
              <div className="w-full h-1 bg-purple-200 dark:bg-purple-800 rounded-full mt-2 overflow-hidden">
                <div className="h-full bg-purple-500 rounded-full group-hover:animate-pulse" style={{width: '45%'}}></div>
              </div>
            </div>
            <div className="group bg-white/90 dark:bg-slate-800/90 rounded-xl p-4 text-center hover:scale-110 transition-all duration-300 hover:shadow-lg hover:shadow-green-500/20 border border-green-200 dark:border-green-800">
              <div className="text-3xl mb-2 text-green-600 group-hover:scale-125 transition-transform duration-300">π</div>
              <div className="font-semibold text-sm group-hover:text-green-600 transition-colors duration-300">Pi Constant</div>
              <div className="text-xs text-slate-500 group-hover:text-slate-600 transition-colors duration-300">3.141592653...</div>
              <div className="w-full h-1 bg-green-200 dark:bg-green-800 rounded-full mt-2 overflow-hidden">
                <div className="h-full bg-green-500 rounded-full group-hover:animate-pulse" style={{width: '75%'}}></div>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <h4 className="text-lg font-bold text-slate-700 dark:text-slate-300">🔬 How It Works:</h4>
            <ul className="space-y-3 text-slate-600 dark:text-slate-400">
              <li className="flex items-start">
                <span className="text-cyan-500 mr-3 mt-1">1.</span>
                <div>
                  <strong>Frequency Mapping:</strong> Input data is mapped to harmonic frequencies using golden ratio scaling
                </div>
              </li>
              <li className="flex items-start">
                <span className="text-cyan-500 mr-3 mt-1">2.</span>
                <div>
                  <strong>Time Distortion:</strong> TIU (Time Invariant Units) apply relativistic time dilation to hash computation
                </div>
              </li>
              <li className="flex items-start">
                <span className="text-cyan-500 mr-3 mt-1">3.</span>
                <div>
                  <strong>Quantum Resonance:</strong> Harmonic oscillators create quantum entangled hash signatures
                </div>
              </li>
              <li className="flex items-start">
                <span className="text-cyan-500 mr-3 mt-1">4.</span>
                <div>
                  <strong>Crystallization:</strong> Final hash emerges from quantum decoherence at measurement
                </div>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}

export default HarmonicsPanel
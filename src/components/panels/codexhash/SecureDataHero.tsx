import { Button } from '@/components/widgets/ui/button'
import React from 'react'
import Image from 'next/image'
import Link from 'next/link'

const SecureDataHero = () => {
  return (
     <section id="security" className="relative min-h-screen flex items-center justify-center overflow-hidden">
        {/* Background with quantum grid */}
        <div className="absolute inset-0 bg-gradient-to-br from-gray-900 via-blue-900/20 to-purple-900/20"></div>
        <div className="absolute inset-0" style={{
          backgroundImage: `linear-gradient(rgba(139, 92, 246, 0.1) 1px, transparent 1px), linear-gradient(90deg, rgba(139, 92, 246, 0.1) 1px, transparent 1px)`,
          backgroundSize: '50px 50px'
        }}></div>
        
      {/* Background Image */}
      <div className="absolute inset-0">
        <Image
          src="/assets/quantum-computing.jpg"
          alt="Quantum Computing Background"
          fill
          className="object-cover opacity-20"
          priority
        />
        {/* Overlay for better text contrast */}
        <div className="absolute inset-0 bg-gradient-to-br from-gray-900/80 via-purple-900/60 to-blue-900/70"></div>
      </div>        {/* Floating Particles */}
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute top-20 left-10 w-4 h-4 bg-purple-500 rounded-full animate-bounce opacity-60"></div>
          <div className="absolute top-40 right-20 w-6 h-6 bg-blue-500 rounded-full animate-bounce opacity-40" style={{animationDelay: '-2s'}}></div>
          <div className="absolute bottom-40 left-20 w-3 h-3 bg-pink-500 rounded-full animate-bounce opacity-50" style={{animationDelay: '-4s'}}></div>
          <div className="absolute bottom-20 right-40 w-5 h-5 bg-green-500 rounded-full animate-bounce opacity-30" style={{animationDelay: '-1s'}}></div>
        </div>
        
        <div className="relative z-10 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 text-center pt-32">
          {/* Status Badge */}
          <div className="mb-8">
            <span className="inline-flex items-center gap-2 px-4 py-2 bg-purple-500/20 text-purple-400 rounded-full text-sm font-medium">
              <span className="w-2 h-2 bg-purple-500 rounded-full animate-pulse"></span>
              QUANTUM-RESISTANT TECHNOLOGY
            </span>
          </div>
          
          {/* Main Headline */}
          <h1 className="text-5xl lg:text-7xl font-bold mb-8 leading-tight">
            Secure Your Data<br />
            Against <span className="bg-gradient-to-r from-purple-500 via-pink-500 to-blue-500 bg-clip-text text-transparent">Quantum</span> Threats
          </h1>
          
          {/* Subtitle */}
          <p className="text-xl lg:text-2xl text-gray-300 mb-12 max-w-4xl mx-auto">
            Revolutionary <strong>HarmonicHash</strong> algorithm combines physics-based constants with harmonic frequencies to create cryptographically secure hashes that withstand quantum computing attacks.
          </p>
          
          {/* Key Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-12">
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="text-3xl font-bold text-purple-400 mb-2">99%</div>
              <div className="text-sm text-gray-400">Quantum Resistance</div>
            </div>
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="text-3xl font-bold text-blue-400 mb-2">540ms</div>
              <div className="text-sm text-gray-400">Average Response</div>
            </div>
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="text-3xl font-bold text-pink-400 mb-2">512+-bit</div>
              <div className="text-sm text-gray-400">Hash Output</div>
            </div>
            <div className="bg-gray-800/50 backdrop-blur-sm rounded-xl p-6 border border-gray-700">
              <div className="text-3xl font-bold text-green-400 mb-2">4+</div>
              <div className="text-sm text-gray-400">Security Phases</div>
            </div>
          </div>
          
          {/* CTA Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <Button className="bg-gradient-to-r from-purple-500 to-blue-500 px-8 py-4 rounded-lg text-lg font-semibold hover:opacity-90 transition">
              Get Early Access
            </Button>
            <Link href="/technical-details">
              <Button variant="outline" className="border-2 border-gray-600 px-8 py-4 rounded-lg text-lg font-semibold hover:border-purple-500 transition">
                View Documentation
              </Button>
            </Link>
          </div>
        </div>
      </section>
  )
}

export default SecureDataHero
'use client'

import React from 'react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'
import { ArrowRight, Shield, Zap, Globe } from 'lucide-react'

const Hero: React.FC = () => {
  return (
    <section className="relative bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 text-white overflow-hidden">
      {/* Quantum Computing Background Image */}
      <div 
        className="absolute inset-0 bg-cover bg-center bg-no-repeat opacity-30"
        style={{
          backgroundImage: "url('/assets/images/quantum-computing.jpg')"
        }}
      ></div>
      
      {/* Overlay for better text readability */}
      <div className="absolute inset-0 bg-gradient-to-br from-slate-900/80 via-blue-900/70 to-slate-900/80"></div>
      
      <div className="relative container mx-auto px-6 py-24 lg:py-32">
        <div className="max-w-4xl mx-auto text-center">
          {/* Badge */}
          <div className="inline-flex items-center gap-2 bg-blue-500/20 border border-blue-500/30 rounded-full px-4 py-2 mb-8">
            <Shield className="w-4 h-4 text-blue-400" />
            <span className="text-sm font-medium text-blue-100">
              Quantum-Resistant • Enterprise Ready • Open Source
            </span>
          </div>

          {/* Headline */}
          <h1 className="text-5xl lg:text-7xl font-bold mb-6 bg-gradient-to-r from-white via-blue-100 to-white bg-clip-text text-transparent">
            Next-Generation
            <br />
            <span className="bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text">
              Hashing Technology
            </span>
          </h1>

          {/* Subheadline */}
          <p className="text-xl lg:text-2xl text-slate-300 mb-10 max-w-3xl mx-auto leading-relaxed">
            CodexHash delivers quantum-resistant cryptographic hashing with 
            <span className="text-blue-400 font-semibold"> harmonic collision spacing</span> and 
            <span className="text-purple-400 font-semibold"> enterprise-grade performance</span>.
            Secure your data for the post-quantum era.
          </p>

          {/* CTAs */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center items-center mb-16">
            <Button 
              asChild
              size="lg" 
              className="bg-blue-600 hover:bg-blue-700 text-white px-8 py-4 text-lg font-semibold shadow-lg hover:shadow-xl transition-all duration-200"
            >
              <Link href="/docs/getting-started">
                Get Started
                <ArrowRight className="w-5 h-5 ml-2" />
              </Link>
            </Button>
            
            <Button 
              asChild
              variant="outline" 
              size="lg"
              className="border-white/30 text-white hover:bg-white/10 px-8 py-4 text-lg font-semibold"
            >
              <Link href="/tools/generator">
                Try Playground
              </Link>
            </Button>
          </div>

          {/* Key Features */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-4xl mx-auto">
            <div className="text-center p-6 bg-white/5 backdrop-blur-sm rounded-lg border border-white/10">
              <div className="w-12 h-12 bg-blue-500/20 rounded-lg flex items-center justify-center mx-auto mb-4">
                <Zap className="w-6 h-6 text-blue-400" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Lightning Fast</h3>
              <p className="text-slate-400 text-sm">
                Sub-millisecond hash generation with optimized algorithms for modern hardware
              </p>
            </div>

            <div className="text-center p-6 bg-white/5 backdrop-blur-sm rounded-lg border border-white/10">
              <div className="w-12 h-12 bg-purple-500/20 rounded-lg flex items-center justify-center mx-auto mb-4">
                <Shield className="w-6 h-6 text-purple-400" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Quantum Secure</h3>
              <p className="text-slate-400 text-sm">
                Built with post-quantum cryptography principles to resist future threats
              </p>
            </div>

            <div className="text-center p-6 bg-white/5 backdrop-blur-sm rounded-lg border border-white/10">
              <div className="w-12 h-12 bg-green-500/20 rounded-lg flex items-center justify-center mx-auto mb-4">
                <Globe className="w-6 h-6 text-green-400" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Enterprise Scale</h3>
              <p className="text-slate-400 text-sm">
                Proven reliability with 99.99% uptime and global CDN distribution
              </p>
            </div>
          </div>

          {/* Social Proof */}
          <div className="mt-16 pt-8 border-t border-white/10">
            <p className="text-slate-400 text-sm mb-4">Trusted by developers worldwide</p>
            <div className="flex justify-center items-center gap-8 opacity-60">
              <div className="text-sm font-mono">10K+ Developers</div>
              <div className="w-1 h-1 bg-slate-400 rounded-full"></div>
              <div className="text-sm font-mono">1M+ Hashes/Day</div>
              <div className="w-1 h-1 bg-slate-400 rounded-full"></div>
              <div className="text-sm font-mono">99.99% Uptime</div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

export default Hero

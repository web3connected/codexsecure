'use client'

import { ArrowRight, Globe, Shield, Zap } from 'lucide-react'
import React from 'react'
import OpenSourceBadge from '@/components/widgets/ui/OpenSourceBadge'
import ApplicationLogo from '../../common/ApplicationLogo'
import { Button } from '../../widgets/ui/button'
import Link from 'next/link'

const CodexHashHero: React.FC = () => {
  return (
    <section className="relative bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 overflow-hidden min-h-screen">
      {/* Background image */}
      <div
        className="absolute inset-0 bg-cover bg-center bg-no-repeat"
        style={{
          backgroundImage: "url('/assets/images/quantum-computing.jpg')",
        }}
      />
      {/* Enhanced dark overlay with multiple layers */}
      <div className="absolute inset-0 bg-gradient-to-br from-slate-900/90 via-blue-900/85 to-slate-900/90"></div>
      <div className="absolute inset-0 bg-black/20"></div>

      {/* Content */}
      <div className="relative container mx-auto px-6 py-24 lg:py-32 text-center">
        <OpenSourceBadge />

        {/* Headline */}
        <h1 className="text-6xl lg:text-8xl xl:text-9xl font-bold mb-8">
          <ApplicationLogo logo="CodexHash" showIcon={false} size="4xl" />
          <span className="bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent block mt-2">
            Quantum Library
          </span>
        </h1>

        {/* Subheadline */}
        <p className="text-xl lg:text-2xl text-slate-300 mb-10 max-w-3xl mx-auto leading-relaxed">
          Bring your applications into the quantum realm. Future-proof your systems with
          next-generation cryptographic solutions designed for the quantum era.
        </p>

        {/* CTAs */}
        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center mb-16">
          <Link href="/docs/getting-started">
            <Button
              size="lg"
              className="bg-blue-600 hover:bg-blue-700 text-white px-8 py-4 text-lg font-semibold shadow-lg hover:shadow-xl transition-all duration-200"
            >
              Get Started
              <ArrowRight className="w-5 h-5 ml-2" />
            </Button>
          </Link>

          <Link href="/tools/generator">
            <Button
              variant="outline"
              size="lg"
              className="border-white/30 text-white hover:bg-white/10 px-8 py-4 text-lg font-semibold"
            >
              View Documentation
            </Button>
          </Link>
        </div>

        {/* Key Features */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-6xl mx-auto">
          <div className="group text-center p-8 bg-white/10 backdrop-blur-md rounded-2xl border border-white/20 shadow-2xl hover:shadow-3xl hover:bg-white/15 transition-all duration-300 hover:scale-105">
            <div className="w-16 h-16 bg-gradient-to-br from-blue-500 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-6 group-hover:scale-110 transition-transform duration-300 shadow-lg">
              <Zap className="w-8 h-8 text-white" />
            </div>
            <h3 className="text-xl font-bold mb-3 text-white">Easy to Use</h3>
            <p className="text-slate-300 leading-relaxed">Simple API design with comprehensive documentation and intuitive implementation</p>
          </div>

          <div className="group text-center p-8 bg-white/10 backdrop-blur-md rounded-2xl border border-white/20 shadow-2xl hover:shadow-3xl hover:bg-white/15 transition-all duration-300 hover:scale-105">
            <div className="w-16 h-16 bg-gradient-to-br from-purple-500 to-purple-600 rounded-2xl flex items-center justify-center mx-auto mb-6 group-hover:scale-110 transition-transform duration-300 shadow-lg">
              <Shield className="w-8 h-8 text-white" />
            </div>
            <h3 className="text-xl font-bold mb-3 text-white">Secure</h3>
            <p className="text-slate-300 leading-relaxed">
              Built with security best practices and modern cryptographic standards for quantum resistance
            </p>
          </div>

          <div className="group text-center p-8 bg-white/10 backdrop-blur-md rounded-2xl border border-white/20 shadow-2xl hover:shadow-3xl hover:bg-white/15 transition-all duration-300 hover:scale-105">
            <div className="w-16 h-16 bg-gradient-to-br from-green-500 to-green-600 rounded-2xl flex items-center justify-center mx-auto mb-6 group-hover:scale-110 transition-transform duration-300 shadow-lg">
              <Globe className="w-8 h-8 text-white" />
            </div>
            <h3 className="text-xl font-bold mb-3 text-white">Versatile</h3>
            <p className="text-slate-300 leading-relaxed">
              Supports multiple hashing algorithms and encoding formats for all your needs
            </p>
          </div>
        </div>
      </div>
    </section>
  )
}

export default CodexHashHero

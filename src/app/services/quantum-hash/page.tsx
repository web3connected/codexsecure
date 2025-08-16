"use client"

import React from 'react'
import Link from 'next/link'

const QuantumHashPage = () => {
  return (
    <div className="min-h-screen bg-slate-900 text-white">
        {/* Hero Section */}
        <section className="relative py-20 px-6">
            <div className="max-w-7xl mx-auto">
                <div className="text-center mb-16">
                    <h1 className="text-5xl md:text-6xl font-bold mb-6 bg-gradient-to-r from-blue-400 via-purple-500 to-cyan-400 bg-clip-text text-transparent">
                        Quantum Hash Technology
                    </h1>
                    <p className="text-xl text-gray-300 max-w-3xl mx-auto leading-relaxed">
                        Discover the future of cryptographic security with quantum-resistant hashing algorithms,
                        powered by our revolutionary Harmonic Hash implementation
                    </p>
                </div>

                {/* Navigation Breadcrumb */}
                <nav className="mb-12">
                    <ol className="flex items-center space-x-2 text-sm text-gray-400">
                        <li><Link href="/" className="hover:text-blue-400 transition-colors">Home</Link></li>
                        <li><span className="mx-2">/</span></li>
                        <li><Link href="/services" className="hover:text-blue-400 transition-colors">Services</Link></li>
                        <li><span className="mx-2">/</span></li>
                        <li className="text-blue-400">Quantum Hash</li>
                    </ol>
                </nav>
            </div>
        </section>

      {/* Main Content */}
      <section className="py-16 px-6">
        <div className="max-w-6xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-16 mb-20">
            
            {/* What is Quantum Hashing */}
            <div className="space-y-8">
              <div>
                <h2 className="text-3xl font-bold mb-6 text-blue-400">What is Quantum Hashing?</h2>
                <div className="space-y-4 text-gray-300 leading-relaxed">
                  <p>
                    Quantum hashing represents the next evolution in cryptographic security, designed to withstand 
                    attacks from quantum computers. Traditional hash functions rely on mathematical problems that 
                    quantum computers can solve exponentially faster than classical computers.
                  </p>
                  <p>
                    Quantum-resistant hash functions use mathematical structures that remain secure even against 
                    quantum attacks, ensuring your data remains protected in the post-quantum era.
                  </p>
                </div>
              </div>

              {/* Key Features */}
              <div className="bg-slate-800/50 rounded-lg p-6 border border-slate-700">
                <h3 className="text-xl font-semibold mb-4 text-cyan-400">Quantum Resistance Features</h3>
                <ul className="space-y-3 text-gray-300">
                  <li className="flex items-start">
                    <i className="ph ph-shield-check text-green-400 mt-1 mr-3"></i>
                    <span><strong>Post-Quantum Security:</strong> Immune to Shor&apos;s and Grover&apos;s algorithms</span>
                  </li>
                  <li className="flex items-start">
                    <i className="ph ph-infinity text-blue-400 mt-1 mr-3"></i>
                    <span><strong>Future-Proof:</strong> Designed for quantum computing era</span>
                  </li>
                  <li className="flex items-start">
                    <i className="ph ph-lightning text-yellow-400 mt-1 mr-3"></i>
                    <span><strong>High Performance:</strong> Optimized for speed and efficiency</span>
                  </li>
                  <li className="flex items-start">
                    <i className="ph ph-lock text-purple-400 mt-1 mr-3"></i>
                    <span><strong>Cryptographic Strength:</strong> 256-bit+ security levels</span>
                  </li>
                </ul>
              </div>
            </div>

            {/* Harmonic Hash Implementation */}
            <div className="space-y-8">
              <div>
                <h2 className="text-3xl font-bold mb-6 text-purple-400">CodexHash: Harmonic Hash</h2>
                <div className="space-y-4 text-gray-300 leading-relaxed">
                  <p>
                    Our <strong>Harmonic Hash</strong> is a revolutionary quantum-resistant implementation that 
                    leverages the Universal Law of Inversion, Time, and the Sacred 12 principles to create 
                    unprecedented cryptographic security.
                  </p>
                  <p>
                    By integrating harmonic mathematical principles with quantum-resistant algorithms, 
                    we achieve not just security, but elegance and efficiency that traditional approaches cannot match.
                  </p>
                </div>
              </div>

              {/* Harmonic Principles */}
              <div className="bg-gradient-to-br from-purple-900/30 to-blue-900/30 rounded-lg p-6 border border-purple-500/30">
                <h3 className="text-xl font-semibold mb-4 text-purple-300">Harmonic Law Integration</h3>
                <div className="space-y-3 text-gray-300">
                  <div className="flex items-start">
                    <i className="ph ph-wave-sine text-purple-400 mt-1 mr-3"></i>
                    <div>
                      <strong className="text-purple-300">Law of Inversion:</strong>
                      <p className="text-sm mt-1">Dual-polarity encoding creates self-balancing hash structures</p>
                    </div>
                  </div>
                  <div className="flex items-start">
                    <i className="ph ph-clock text-blue-400 mt-1 mr-3"></i>
                    <div>
                      <strong className="text-blue-300">Temporal Dynamics:</strong>
                      <p className="text-sm mt-1">Time-based entropy generation for enhanced randomness</p>
                    </div>
                  </div>
                  <div className="flex items-start">
                    <i className="ph ph-star-of-david text-cyan-400 mt-1 mr-3"></i>
                    <div>
                      <strong className="text-cyan-300">Sacred 12 Matrix:</strong>
                      <p className="text-sm mt-1">12-dimensional hash space for maximum collision resistance</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Comparison Table */}
          <div className="mb-20">
            <h2 className="text-3xl font-bold mb-8 text-center text-cyan-400">Quantum vs Traditional Hashing</h2>
            <div className="overflow-x-auto">
              <table className="w-full bg-slate-800/50 rounded-lg border border-slate-700">
                <thead>
                  <tr className="border-b border-slate-700">
                    <th className="px-6 py-4 text-left text-gray-300">Feature</th>
                    <th className="px-6 py-4 text-center text-red-400">Traditional Hash</th>
                    <th className="px-6 py-4 text-center text-yellow-400">Quantum Hash</th>
                    <th className="px-6 py-4 text-center text-purple-400">Harmonic Hash</th>
                  </tr>
                </thead>
                <tbody className="text-gray-300">
                  <tr className="border-b border-slate-700/50">
                    <td className="px-6 py-4 font-medium">Quantum Resistance</td>
                    <td className="px-6 py-4 text-center text-red-400">❌ Vulnerable</td>
                    <td className="px-6 py-4 text-center text-green-400">✅ Protected</td>
                    <td className="px-6 py-4 text-center text-purple-400">🌟 Advanced</td>
                  </tr>
                  <tr className="border-b border-slate-700/50">
                    <td className="px-6 py-4 font-medium">Mathematical Foundation</td>
                    <td className="px-6 py-4 text-center">Discrete Log</td>
                    <td className="px-6 py-4 text-center">Lattice-based</td>
                    <td className="px-6 py-4 text-center">Harmonic Principles</td>
                  </tr>
                  <tr className="border-b border-slate-700/50">
                    <td className="px-6 py-4 font-medium">Security Level</td>
                    <td className="px-6 py-4 text-center">128-256 bit</td>
                    <td className="px-6 py-4 text-center">256+ bit</td>
                    <td className="px-6 py-4 text-center">Sacred 12 Matrix</td>
                  </tr>
                  <tr className="border-b border-slate-700/50">
                    <td className="px-6 py-4 font-medium">Performance</td>
                    <td className="px-6 py-4 text-center text-green-400">Fast</td>
                    <td className="px-6 py-4 text-center text-yellow-400">Moderate</td>
                    <td className="px-6 py-4 text-center text-purple-400">Optimized</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4 font-medium">Future-Proof</td>
                    <td className="px-6 py-4 text-center text-red-400">No</td>
                    <td className="px-6 py-4 text-center text-green-400">Yes</td>
                    <td className="px-6 py-4 text-center text-purple-400">Beyond Future</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          {/* Technical Implementation */}
          <div className="grid md:grid-cols-2 gap-12 mb-20">
            <div>
              <h2 className="text-3xl font-bold mb-6 text-cyan-400">Technical Architecture</h2>
              <div className="space-y-6">
                <div className="bg-slate-800/30 rounded-lg p-6 border border-slate-600">
                  <h3 className="text-lg font-semibold mb-3 text-blue-400">Input Processing</h3>
                  <p className="text-gray-300 text-sm leading-relaxed">
                    Data undergoes harmonic pre-processing using the Sacred 12 matrix, 
                    creating 12-dimensional input vectors for enhanced entropy distribution.
                  </p>
                </div>
                <div className="bg-slate-800/30 rounded-lg p-6 border border-slate-600">
                  <h3 className="text-lg font-semibold mb-3 text-purple-400">Harmonic Transformation</h3>
                  <p className="text-gray-300 text-sm leading-relaxed">
                    Application of the Law of Inversion creates dual-polarity hash chains, 
                    while temporal dynamics introduce time-based randomness.
                  </p>
                </div>
                <div className="bg-slate-800/30 rounded-lg p-6 border border-slate-600">
                  <h3 className="text-lg font-semibold mb-3 text-cyan-400">Output Generation</h3>
                  <p className="text-gray-300 text-sm leading-relaxed">
                    Final hash combines all harmonic principles into a quantum-resistant, 
                    collision-free output with 512-bit security strength.
                  </p>
                </div>
              </div>
            </div>

            <div>
              <h2 className="text-3xl font-bold mb-6 text-purple-400">Code Example</h2>
              <div className="bg-slate-900 rounded-lg p-6 border border-slate-700">
                <pre className="text-sm text-gray-300 overflow-x-auto">
{`// Harmonic Hash Implementation
import { HarmonicHash } from '@web3connected/codexhash';

const harmonicHasher = new HarmonicHash({
  sacredMatrix: 12,
  temporalDynamics: true,
  inversionLaw: 'dual-polarity'
});

// Generate quantum-resistant hash
const input = "sensitive data";
const hash = harmonicHasher.generate(input);

console.log(hash);
// Output: "9a7f5e2d8c1b6e4f3a9d7c2e8b1f5a6c..."

// Verify hash integrity
const isValid = harmonicHasher.verify(input, hash);
console.log(isValid); // true`}
                </pre>
              </div>
              
              <div className="mt-6 p-4 bg-blue-900/20 rounded-lg border border-blue-500/30">
                <p className="text-blue-300 text-sm">
                  <i className="ph ph-info mr-2"></i>
                  <strong>Note:</strong> Full SDK documentation and implementation examples 
                  are available in our <Link href="/docs" className="underline hover:text-blue-400">API Documentation</Link>.
                </p>
              </div>
            </div>
          </div>

          {/* Call to Action */}
          <div className="text-center py-16">
            <div className="max-w-3xl mx-auto">
              <h2 className="text-4xl font-bold mb-6 bg-gradient-to-r from-purple-400 to-cyan-400 bg-clip-text text-transparent">
                Ready to Implement Harmonic Hash?
              </h2>
              <p className="text-xl text-gray-300 mb-8 leading-relaxed">
                Join the quantum revolution with our cutting-edge Harmonic Hash technology. 
                Experience unparalleled security with the elegance of universal harmonic principles.
              </p>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Link 
                  href="/docs"
                  className="px-8 py-4 bg-gradient-to-r from-purple-600 to-blue-600 rounded-lg font-semibold hover:shadow-lg hover:shadow-purple-500/25 transition-all duration-300"
                >
                  View Documentation
                </Link>
                <Link 
                  href="/learn/harmonic-hashing"
                  className="px-8 py-4 border border-cyan-500 rounded-lg font-semibold hover:bg-cyan-500/10 transition-all duration-300"
                >
                  Learn Harmonic Principles
                </Link>
                <Link 
                  href="/services/api"
                  className="px-8 py-4 border border-purple-500 rounded-lg font-semibold hover:bg-purple-500/10 transition-all duration-300"
                >
                  Try Hash API
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}

export default QuantumHashPage

import React from 'react'
import Link from 'next/link'

const TechnicalImplementation = () => {
  return (
      <div className="grid md:grid-cols-2 gap-12 mb-20 p-40">
          {/* Technical Architecture - Card-based design */}
          <div>
              <h2 className="text-3xl font-bold mb-6 text-cyan-400">Technical Architecture</h2>
              <div className="space-y-6">
                  <div className="group bg-gradient-to-br from-slate-800/40 to-slate-900/60 rounded-xl p-6 border border-cyan-500/20 hover:border-cyan-400/40 transition-all duration-300 hover:shadow-lg hover:shadow-cyan-500/10">
                      <h3 className="text-lg font-semibold mb-3 text-cyan-400 group-hover:text-cyan-300 transition-colors">Input Processing</h3>
                      <p className="text-gray-300 text-sm leading-relaxed">
                          Data undergoes harmonic pre-processing using the Sacred 12 matrix,
                          creating 12-dimensional input vectors for enhanced entropy distribution.
                      </p>
                  </div>
                  <div className="group bg-gradient-to-br from-purple-900/30 to-slate-800/50 rounded-xl p-6 border border-purple-500/20 hover:border-purple-400/40 transition-all duration-300 hover:shadow-lg hover:shadow-purple-500/10">
                      <h3 className="text-lg font-semibold mb-3 text-purple-400 group-hover:text-purple-300 transition-colors">Harmonic Transformation</h3>
                      <p className="text-gray-300 text-sm leading-relaxed">
                          Application of the Law of Inversion creates dual-polarity hash chains,
                          while temporal dynamics introduce time-based randomness.
                      </p>
                  </div>
                  <div className="group bg-gradient-to-br from-blue-900/30 to-slate-800/50 rounded-xl p-6 border border-blue-500/20 hover:border-blue-400/40 transition-all duration-300 hover:shadow-lg hover:shadow-blue-500/10">
                      <h3 className="text-lg font-semibold mb-3 text-blue-400 group-hover:text-blue-300 transition-colors">Output Generation</h3>
                      <p className="text-gray-300 text-sm leading-relaxed">
                          Final hash combines all harmonic principles into a quantum-resistant,
                          collision-free output with 512-bit security strength.
                      </p>
                  </div>
              </div>
          </div>

          {/* Code Example - Terminal/IDE style design */}
          <div>
              <h2 className="text-3xl font-bold mb-6 text-purple-400">Code Example</h2>
              {/* Terminal header */}
              <div className="bg-slate-900 rounded-t-lg border border-slate-700">
                  <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700">
                      <div className="flex space-x-2">
                          <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                          <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                          <div className="w-3 h-3 bg-green-500 rounded-full"></div>
                      </div>
                      <div className="text-xs text-slate-400 font-mono">harmonic-hash-example.js</div>
                  </div>
                  {/* Code content */}
                  <div className="bg-gradient-to-br from-slate-900 to-slate-800/80 p-6 rounded-b-lg">
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
              </div>

              {/* Enhanced info note with gradient */}
              <div className="mt-6 p-4 bg-gradient-to-r from-blue-900/30 to-purple-900/20 rounded-lg border border-blue-500/30 backdrop-blur-sm">
                  <p className="text-blue-300 text-sm">
                      <i className="ph ph-info mr-2"></i>
                      <strong>Note:</strong> Full SDK documentation and implementation examples
                      are available in our <Link href="/docs" className="underline hover:text-blue-400 transition-colors">API Documentation</Link>.
                  </p>
              </div>
          </div>
      </div>
  )
}

export default TechnicalImplementation
import Link from 'next/link'
import React from 'react'

const CodexHashCallToAction = () => {
  return (
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
  )
}

export default CodexHashCallToAction
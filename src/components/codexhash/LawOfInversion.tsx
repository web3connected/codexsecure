import React from 'react'

const LawOfInversion = () => {
  return (
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
  )
}

export default LawOfInversion
import React from 'react'

const WhatIsQuantumHashing = () => {
  return (
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
                      <span><strong>Cryptographic Strength:</strong> 256-bit to 1024-bit security levels</span>
                  </li>
              </ul>
          </div>
      </div>
  );
}

export default WhatIsQuantumHashing
import React from 'react'

const TechnologyStack = () => {
  return (
     <section id="technology" className="py-20 bg-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">Technology Stack</h2>
            <p className="text-xl text-gray-400">Built with cutting-edge technologies for maximum performance</p>
          </div>
          
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="bg-gray-900 rounded-xl p-8 border border-gray-700">
                <h3 className="text-xl font-bold mb-4">Frontend</h3>
                <ul className="space-y-2 text-gray-400">
                  <li>Next.js 15.4.6</li>
                  <li>TypeScript</li>
                  <li>Tailwind CSS</li>
                  <li>React Components</li>
                </ul>
              </div>
            </div>
            
            <div className="text-center">
              <div className="bg-gray-900 rounded-xl p-8 border border-gray-700">
                <h3 className="text-xl font-bold mb-4">Backend</h3>
                <ul className="space-y-2 text-gray-400">
                  <li>Python FastAPI</li>
                  <li>HarmonicHash Engine</li>
                  <li>RESTful APIs</li>
                  <li>Real-time Processing</li>
                </ul>
              </div>
            </div>
            
            <div className="text-center">
              <div className="bg-gray-900 rounded-xl p-8 border border-gray-700">
                <h3 className="text-xl font-bold mb-4">Security</h3>
                <ul className="space-y-2 text-gray-400">
                  <li>SHA-3/BLAKE2b</li>
                  <li>CRYSTALS-Dilithium</li>
                  <li>SPHINCS+ Signatures</li>
                  <li>Quantum Resistance</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </section>
  )
}

export default TechnologyStack
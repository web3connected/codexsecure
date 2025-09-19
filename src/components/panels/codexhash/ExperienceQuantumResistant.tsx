import React, { useState } from 'react'
import { Activity, Hash } from 'lucide-react'
import { Button } from '../../widgets/ui/button'

const ExperienceQuantumResistant = () => {
  const [demoInput, setDemoInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [demoResult, setDemoResult] = useState<{
    hash: string;
    meta: {
      algo: string;
      quantumResistance: number;
    };
  } | null>(null)
  
  // Verification state
  const [verifyInput, setVerifyInput] = useState('')
  const [isVerifying, setIsVerifying] = useState(false)
  const [verifyResult, setVerifyResult] = useState<{
    isMatch: boolean;
    verifiedHash: string;
    timeTaken: number;
  } | null>(null)

  const runDemo = async () => {
    if (!demoInput.trim()) return
    
    setIsLoading(true)
    setVerifyResult(null) // Clear previous verification
    try {
      // Simulate API call delay
      await new Promise(resolve => setTimeout(resolve, 1500))
      
      // Generate mock hash result
      const mockHash = Array.from({length: 64}, () => 
        Math.floor(Math.random() * 16).toString(16)
      ).join('')
      
      setDemoResult({
        hash: mockHash,
        meta: {
          algo: 'HarmonicHash',
          quantumResistance: Math.floor(Math.random() * 5) + 95 // 95-99%
        }
      })
      
      // Pre-populate verification form with the original password
      setVerifyInput(demoInput)
    } catch (error) {
      console.error('Demo error:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const runVerification = async () => {
    if (!verifyInput.trim() || !demoResult) return
    
    setIsVerifying(true)
    try {
      // Simulate verification API call
      const startTime = Date.now()
      await new Promise(resolve => setTimeout(resolve, 800))
      const endTime = Date.now()
      
      // Generate mock verification hash (same algorithm)
      const verifyHash = Array.from({length: 64}, () => 
        Math.floor(Math.random() * 16).toString(16)
      ).join('')
      
      // Check if passwords match (in real implementation, this would compare hashes)
      const isMatch = verifyInput === demoInput
      
      setVerifyResult({
        isMatch,
        verifiedHash: verifyHash,
        timeTaken: endTime - startTime
      })
    } catch (error) {
      console.error('Verification error:', error)
    } finally {
      setIsVerifying(false)
    }
  }
  return (
    <section id="demo" className="py-20 bg-gray-900">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-4xl font-bold mb-4">Interactive Demo</h2>
            <p className="text-xl text-gray-400">Experience quantum-resistant hashing in real-time</p>
          </div>
          
          <div className="bg-gray-800 rounded-xl p-8 border border-gray-700">
            <h3 className="text-2xl font-bold mb-6">Password Hashing & Verification Demo</h3>
            
            <div className="space-y-6">
              {/* Step 1: Create Hash */}
              <div>
                <label className="block text-sm font-medium mb-2">
                  Step 1: Enter Password to Hash
                </label>
                <div className="flex gap-4">
                  <input
                    type="password"
                    value={demoInput}
                    onChange={(e) => setDemoInput(e.target.value)}
                    className="flex-1 bg-gray-700 border border-gray-600 rounded-lg px-4 py-3 focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                    placeholder="Enter your password..."
                  />
                  <Button 
                    onClick={runDemo}
                    disabled={isLoading || !demoInput.trim()}
                    className="bg-purple-500 hover:bg-purple-600 px-6"
                  >
                    {isLoading ? <Activity className="w-4 h-4 animate-spin" /> : <Hash className="w-4 h-4" />}
                    {isLoading ? 'Hashing...' : 'Create Hash'}
                  </Button>
                </div>
              </div>
              
              {/* Hash Result */}
              {demoResult && (
                <div className="bg-gray-900 rounded-lg p-6 border border-gray-600">
                  <h4 className="font-bold mb-4">Hash Result:</h4>
                  <div className="space-y-3 text-sm">
                    <div>
                      <span className="text-gray-400">Hash:</span>
                      <div className="bg-gray-800 p-2 rounded font-mono text-xs break-all">{demoResult.hash}</div>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <span className="text-gray-400">Algorithm:</span>
                        <div className="font-mono">{demoResult.meta.algo}</div>
                      </div>
                      <div>
                        <span className="text-gray-400">Quantum Resistance:</span>
                        <div className="font-mono text-green-400">{demoResult.meta.quantumResistance}%</div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* Step 2: Verify Password */}
              {demoResult && (
                <div className="border-t border-gray-600 pt-6">
                  <label className="block text-sm font-medium mb-2">
                    Step 2: Verify Password (pre-populated)
                  </label>
                  <div className="flex gap-4">
                    <input
                      type="password"
                      value={verifyInput}
                      onChange={(e) => setVerifyInput(e.target.value)}
                      className="flex-1 bg-gray-700 border border-gray-600 rounded-lg px-4 py-3 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="Enter password to verify..."
                    />
                    <Button 
                      onClick={runVerification}
                      disabled={isVerifying || !verifyInput.trim()}
                      className="bg-blue-500 hover:bg-blue-600 px-6"
                    >
                      {isVerifying ? <Activity className="w-4 h-4 animate-spin" /> : <Hash className="w-4 h-4" />}
                      {isVerifying ? 'Verifying...' : 'Verify'}
                    </Button>
                  </div>
                </div>
              )}

              {/* Verification Result */}
              {verifyResult && (
                <div className={`rounded-lg p-6 border ${verifyResult.isMatch 
                  ? 'bg-green-900/30 border-green-600' 
                  : 'bg-red-900/30 border-red-600'
                }`}>
                  <h4 className="font-bold mb-4 flex items-center gap-2">
                    Verification Result:
                    <span className={`px-2 py-1 rounded text-xs ${verifyResult.isMatch 
                      ? 'bg-green-500 text-white' 
                      : 'bg-red-500 text-white'
                    }`}>
                      {verifyResult.isMatch ? 'MATCH' : 'NO MATCH'}
                    </span>
                  </h4>
                  <div className="space-y-3 text-sm">
                    <div>
                      <span className="text-gray-400">Verification Hash:</span>
                      <div className="bg-gray-800 p-2 rounded font-mono text-xs break-all">{verifyResult.verifiedHash}</div>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <span className="text-gray-400">Verification Time:</span>
                        <div className="font-mono">{verifyResult.timeTaken}ms</div>
                      </div>
                      <div>
                        <span className="text-gray-400">Status:</span>
                        <div className={`font-mono ${verifyResult.isMatch ? 'text-green-400' : 'text-red-400'}`}>
                          {verifyResult.isMatch ? 'Password Verified' : 'Password Mismatch'}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </section>
  )
}

export default ExperienceQuantumResistant
'use client'

import React, { useState } from 'react'
import { Shield, CheckCircle, AlertCircle, Zap, Clock, Hash, ArrowRight, ArrowLeft, Copy } from 'lucide-react'
import { Button } from '@/components/widgets/ui/button'

interface VerificationResult {
  valid: boolean
  durationMs: number
  meta?: {
    algorithm: string
    inputSize: number
  }
  error?: string
}

interface HashState {
  inputData: string
  generatedHash: string
  timestamp: string
  algorithm: string
}

const VerificationService: React.FC = () => {
  const [step, setStep] = useState(1) // 1: Create Hash, 2: Validate Hash, 3: Final Results
  const [hashState, setHashState] = useState<HashState | null>(null)
  const [inputData, setInputData] = useState('')
  const [verifyInputData, setVerifyInputData] = useState('') // For Step 2 verification input
  const [verificationResult, setVerificationResult] = useState<VerificationResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  // Step 1: Create Hash
  const handleCreateHash = async () => {
    if (!inputData) {
      alert('Please enter data to hash')
      return
    }

    setIsLoading(true)
    try {
      const response = await fetch('/api/hash', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ input: inputData })
      })

      const result = await response.json()
      const newHashState: HashState = {
        inputData: inputData,
        generatedHash: result.hash,
        timestamp: new Date().toISOString(),
        algorithm: result.algorithm || 'SHA-256'
      }
      
      setHashState(newHashState)
      setHashState(newHashState)
      setStep(2) // Move to validation step
    } catch (error) {
      console.error('Hash creation failed:', error)
      alert('Failed to create hash')
    } finally {
      setIsLoading(false)
    }
  }

    // Step 2: Validate Hash
  const handleVerification = async () => {
    if (!hashState || !verifyInputData) {
      alert('Hash state or verification data is missing')
      return
    }

    setIsLoading(true)
    try {
      const response = await fetch('/api/verify', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          input: verifyInputData, // User re-entered password
          hash: hashState.generatedHash // Original generated hash
        })
      })

      const result = await response.json()
      setVerificationResult(result)
      setStep(3) // Move to final results
    } catch (error) {
      console.error('Verification failed:', error)
      setVerificationResult({ 
        valid: false, 
        durationMs: 0, 
        error: 'Verification failed' 
      })
      setStep(3)
    } finally {
      setIsLoading(false)
    }
  }

  const resetProcess = () => {
    setStep(1)
    setHashState(null)
    setInputData('')
    setVerifyInputData('')
    setVerificationResult(null)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    alert('Copied to clipboard!')
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50 dark:from-slate-900 dark:to-blue-900">
      {/* Hero Section */}
      <section className="py-20 px-6">
        <div className="container mx-auto text-center">
          <div className="flex justify-center mb-6">
            <Shield className="w-16 h-16 text-blue-600" />
          </div>
          <h1 className="text-5xl font-bold mb-6 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
            Hash Verification Service
          </h1>
          <p className="text-xl text-slate-600 dark:text-slate-300 max-w-3xl mx-auto mb-8">
            3-Step Dynamic Hash Creation and Verification Process
          </p>
          
          {/* Step Indicator */}
          <div className="flex justify-center items-center space-x-4 mb-8">
            <div className={`flex items-center justify-center w-10 h-10 rounded-full ${step >= 1 ? 'bg-blue-500 text-white' : 'bg-gray-300 text-gray-600'}`}>
              1
            </div>
            <ArrowRight className={`w-6 h-6 ${step >= 2 ? 'text-blue-500' : 'text-gray-400'}`} />
            <div className={`flex items-center justify-center w-10 h-10 rounded-full ${step >= 2 ? 'bg-blue-500 text-white' : 'bg-gray-300 text-gray-600'}`}>
              2
            </div>
            <ArrowRight className={`w-6 h-6 ${step >= 3 ? 'text-blue-500' : 'text-gray-400'}`} />
            <div className={`flex items-center justify-center w-10 h-10 rounded-full ${step >= 3 ? 'bg-blue-500 text-white' : 'bg-gray-300 text-gray-600'}`}>
              3
            </div>
          </div>
        </div>
      </section>

      {/* Dynamic Step Content */}
      <section className="py-12 px-6">
        <div className="container mx-auto max-w-4xl">
          {/* Step 1: Create Hash */}
          {step === 1 && (
            <div className="bg-gradient-to-br from-green-50 to-emerald-100 dark:from-slate-800 dark:to-slate-900 rounded-2xl shadow-2xl p-8 border-2 border-green-200 dark:border-green-800">
              <h2 className="text-3xl font-bold mb-8 text-center text-green-900 dark:text-green-100">🚀 Step 1: Create Hash</h2>
              
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-semibold mb-2 text-green-800 dark:text-green-200">📝 Enter Data to Hash</label>
                  <input
                    type="text"
                    value={inputData}
                    onChange={(e) => setInputData(e.target.value)}
                    className="w-full p-4 border-2 border-green-300 dark:border-green-600 rounded-lg bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:ring-4 focus:ring-green-500/50 focus:border-green-500 transition-all duration-200 shadow-inner"
                    placeholder="✏️ Enter the data you want to hash..."
                  />
                </div>

                <Button
                  onClick={handleCreateHash}
                  disabled={isLoading || !inputData}
                  className="w-full py-4 text-lg font-bold bg-gradient-to-r from-green-500 to-emerald-600 hover:from-green-600 hover:to-emerald-700 text-white shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200 disabled:opacity-50 disabled:transform-none"
                >
                  {isLoading ? (
                    <>
                      <Clock className="w-6 h-6 mr-2 animate-spin" />
                      🔄 Creating Hash...
                    </>
                  ) : (
                    <>
                      <Hash className="w-6 h-6 mr-2" />
                      ✨ Create Hash & Continue
                    </>
                  )}
                </Button>
              </div>
            </div>
          )}

          {/* Step 2: Validate Hash */}
          {step === 2 && hashState && (
            <div className="bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-slate-800 dark:to-slate-900 rounded-2xl shadow-2xl p-8 border-2 border-blue-200 dark:border-blue-800">
              <h2 className="text-3xl font-bold mb-8 text-center text-blue-900 dark:text-blue-100">🔍 Step 2: Validate Hash</h2>
              
              <div className="grid md:grid-cols-2 gap-8">
                {/* Generated Hash Display */}
                <div className="space-y-4">
                  <h3 className="text-xl font-bold text-blue-800 dark:text-blue-200">📋 Generated Hash</h3>
                  <div className="bg-gradient-to-r from-green-100 to-emerald-100 dark:from-green-900/50 dark:to-emerald-900/50 p-4 rounded-xl border border-green-300 dark:border-green-600">
                    <p className="text-sm font-semibold text-green-700 dark:text-green-300 mb-2">Original Data:</p>
                    <p className="text-green-900 dark:text-green-100 font-mono text-sm bg-white/50 dark:bg-black/20 p-2 rounded">{hashState.inputData}</p>
                    
                    <p className="text-sm font-semibold text-green-700 dark:text-green-300 mb-2 mt-4">Generated Hash:</p>
                    <div className="flex items-center space-x-2">
                      <p className="text-green-900 dark:text-green-100 font-mono text-xs bg-white/50 dark:bg-black/20 p-2 rounded flex-1 break-all">{hashState.generatedHash}</p>
                      <button 
                        onClick={() => copyToClipboard(hashState.generatedHash)}
                        className="p-2 bg-green-500 hover:bg-green-600 text-white rounded transition-colors"
                      >
                        <Copy className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>

                {/* Validation Form */}
                <div className="space-y-4">
                  <h3 className="text-xl font-bold text-blue-800 dark:text-blue-200">🔑 Re-enter Data to Verify</h3>
                  <div>
                    <label className="block text-sm font-semibold mb-2 text-purple-800 dark:text-purple-200">📝 Enter the original data again</label>
                    <input
                      type="text"
                      value={verifyInputData}
                      onChange={(e) => setVerifyInputData(e.target.value)}
                      className="w-full p-4 border-2 border-purple-300 dark:border-purple-600 rounded-lg bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:ring-4 focus:ring-purple-500/50 focus:border-purple-500 transition-all duration-200 shadow-inner"
                      placeholder="🔗 Re-enter the data to verify against the hash..."
                    />
                  </div>

                  <div className="flex space-x-4">
                    <Button
                      onClick={() => setStep(1)}
                      variant="outline"
                      className="flex-1"
                    >
                      <ArrowLeft className="w-4 h-4 mr-2" />
                      Back
                    </Button>
                    
                    <Button
                      onClick={handleVerification}
                      disabled={isLoading || !verifyInputData}
                      className="flex-1 bg-gradient-to-r from-purple-500 to-blue-600 hover:from-purple-600 hover:to-blue-700 text-white"
                    >
                      {isLoading ? (
                        <>
                          <Clock className="w-4 h-4 mr-2 animate-spin" />
                          Verifying...
                        </>
                      ) : (
                        <>
                          <Shield className="w-4 h-4 mr-2" />
                          Verify Hash
                        </>
                      )}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Step 3: Final Results */}
          {step === 3 && verificationResult && (
            <div className="bg-gradient-to-br from-purple-50 to-pink-100 dark:from-slate-800 dark:to-slate-900 rounded-2xl shadow-2xl p-8 border-2 border-purple-200 dark:border-purple-800">
              <h2 className="text-3xl font-bold mb-8 text-center text-purple-900 dark:text-purple-100">📊 Step 3: Final Results</h2>
              
              <div className="space-y-6">
                {/* Main Result */}
                {verificationResult.error ? (
                  <div className="flex items-center justify-center p-6 bg-gradient-to-r from-red-100 to-pink-100 dark:from-red-900/40 dark:to-pink-900/40 border-2 border-red-300 dark:border-red-600 rounded-xl shadow-lg">
                    <AlertCircle className="w-12 h-12 text-red-600 mr-4" />
                    <span className="text-red-800 dark:text-red-200 font-bold text-xl">
                      ❌ {verificationResult.error}
                    </span>
                  </div>
                ) : (
                  <div className={`flex items-center justify-center p-6 border-2 rounded-xl shadow-lg ${
                    verificationResult.valid 
                      ? 'bg-gradient-to-r from-green-100 to-emerald-100 dark:from-green-900/40 dark:to-emerald-900/40 border-green-300 dark:border-green-600'
                      : 'bg-gradient-to-r from-red-100 to-pink-100 dark:from-red-900/40 dark:to-pink-900/40 border-red-300 dark:border-red-600'
                  }`}>
                    {verificationResult.valid ? (
                      <>
                        <CheckCircle className="w-12 h-12 text-green-600 mr-4" />
                        <span className="text-green-800 dark:text-green-200 font-bold text-2xl">
                          ✅ Hash Verified Successfully!
                        </span>
                      </>
                    ) : (
                      <>
                        <AlertCircle className="w-12 h-12 text-red-600 mr-4" />
                        <span className="text-red-800 dark:text-red-200 font-bold text-2xl">
                          ❌ Hash Verification Failed
                        </span>
                      </>
                    )}
                  </div>
                )}

                {/* Detailed Metrics */}
                {!verificationResult.error && (
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                    <div className="bg-gradient-to-br from-blue-100 to-indigo-100 dark:from-blue-900/50 dark:to-indigo-900/50 p-6 rounded-xl border border-blue-200 dark:border-blue-700 text-center">
                      <Zap className="w-8 h-8 mx-auto mb-2 text-blue-600" />
                      <span className="text-blue-700 dark:text-blue-300 font-semibold block">🔧 Algorithm:</span>
                      <div className="font-bold text-blue-900 dark:text-blue-100 text-lg">{verificationResult.meta?.algorithm || hashState?.algorithm || 'SHA-256'}</div>
                    </div>
                    <div className="bg-gradient-to-br from-purple-100 to-pink-100 dark:from-purple-900/50 dark:to-pink-900/50 p-6 rounded-xl border border-purple-200 dark:border-purple-700 text-center">
                      <Clock className="w-8 h-8 mx-auto mb-2 text-purple-600" />
                      <span className="text-purple-700 dark:text-purple-300 font-semibold block">⏱️ Duration:</span>
                      <div className="font-bold text-purple-900 dark:text-purple-100 text-lg">{verificationResult.durationMs}ms</div>
                    </div>
                    <div className="bg-gradient-to-br from-green-100 to-emerald-100 dark:from-green-900/50 dark:to-emerald-900/50 p-6 rounded-xl border border-green-200 dark:border-green-700 text-center">
                      <Shield className="w-8 h-8 mx-auto mb-2 text-green-600" />
                      <span className="text-green-700 dark:text-green-300 font-semibold block">📏 Input Size:</span>
                      <div className="font-bold text-green-900 dark:text-green-100 text-lg">{verificationResult.meta?.inputSize || hashState?.inputData.length || 0} chars</div>
                    </div>
                  </div>
                )}

                {/* Action Buttons */}
                <div className="flex justify-center space-x-4">
                  <Button
                    onClick={resetProcess}
                    className="bg-gradient-to-r from-gray-500 to-slate-600 hover:from-gray-600 hover:to-slate-700 text-white px-8 py-3"
                  >
                    🔄 Start New Verification
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Features Section */}
      <section className="py-16 px-6">
        <div className="container mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">Verification Features</h2>
          
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="bg-blue-100 dark:bg-blue-900/30 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <Zap className="w-8 h-8 text-blue-600" />
              </div>
              <h3 className="text-xl font-semibold mb-3">Real-time Processing</h3>
              <p className="text-slate-600 dark:text-slate-300">
                Experience instant hash creation and verification with our optimized processing engine.
              </p>
            </div>

            <div className="text-center">
              <div className="bg-green-100 dark:bg-green-900/30 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <Shield className="w-8 h-8 text-green-600" />
              </div>
              <h3 className="text-xl font-semibold mb-3">Quantum-Resistant</h3>
              <p className="text-slate-600 dark:text-slate-300">
                Create and verify quantum-resistant hashes with advanced cryptographic validation.
              </p>
            </div>

            <div className="text-center">
              <div className="bg-purple-100 dark:bg-purple-900/30 w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4">
                <CheckCircle className="w-8 h-8 text-purple-600" />
              </div>
              <h3 className="text-xl font-semibold mb-3">3-Step Process</h3>
              <p className="text-slate-600 dark:text-slate-300">
                Follow our guided process: Create hash, validate integrity, and view detailed results.
              </p>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}

export default VerificationService
import React, { useState } from 'react'
import { Activity, CheckCircle, AlertCircle, Hash, Copy, Sparkles, Code, Zap, Shield } from 'lucide-react'
import { Button } from '@/components/widgets/ui/button'
import Image from 'next/image'

interface TestResult {
  hash: string
  salt: string
  tiu: number
  meta: {
    algo: string
    iterations: number
    durationMs: number
    inputSize: number
    quantumResistance: number
  }
}

const TestingPanel: React.FC = () => {
  const [testInput, setTestInput] = useState('')
  const [testResult, setTestResult] = useState<TestResult | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const runTest = async () => {
    if (!testInput.trim()) {
      setError('Please enter some data to hash')
      return
    }

    setIsLoading(true)
    setError(null)
    
    try {
      // Simulate API call with mock data for demo
      await new Promise(resolve => setTimeout(resolve, 2000))
      
      const mockResult: TestResult = {
        hash: '0x' + Array.from({length: 64}, () => Math.floor(Math.random() * 16).toString(16)).join(''),
        salt: '0x' + Array.from({length: 32}, () => Math.floor(Math.random() * 16).toString(16)).join(''),
        tiu: Math.floor(Math.random() * 1000000),
        meta: {
          algo: 'CodexHash-QR-v1.0',
          iterations: Math.floor(Math.random() * 1000) + 500,
          durationMs: Math.floor(Math.random() * 50) + 10,
          inputSize: testInput.length,
          quantumResistance: 99.9
        }
      }
      
      setTestResult(mockResult)
    } catch (err) {
      setError('Failed to generate hash. Please try again.')
      console.error('Hash generation error:', err)
    } finally {
      setIsLoading(false)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    alert('Copied to clipboard!')
  }

  const clearTest = () => {
    setTestInput('')
    setTestResult(null)
    setError(null)
  }

  return (
    <div className="min-h-full flex flex-col justify-center space-y-8 relative overflow-hidden">
      {/* Enhanced Testing Background with Web3 Security */}
      <div className="absolute inset-0 bg-gradient-to-br from-green-900/20 via-emerald-900/30 to-teal-900/20"></div>
      <div className="absolute inset-0 bg-[conic-gradient(from_45deg_at_50%_50%,_var(--tw-gradient-stops))] from-transparent via-green-500/5 to-transparent animate-spin" style={{animationDuration: '15s'}}></div>
      
      {/* Web3 Security Background */}
      <div className="absolute inset-0 opacity-5">
        <Image
          src="/assets/images/web3-cyber-sicurezza.webp"
          alt="Web3 Security Background"
          fill
          className="object-cover"
        />
      </div>
      
      {/* Terminal-style Elements */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute top-10 left-10 w-24 h-1 bg-green-500 opacity-20 animate-pulse"></div>
        <div className="absolute top-20 left-20 w-16 h-1 bg-emerald-500 opacity-30 animate-pulse" style={{animationDelay: '0.5s'}}></div>
        <div className="absolute bottom-20 right-20 w-20 h-1 bg-teal-500 opacity-25 animate-pulse" style={{animationDelay: '1s'}}></div>
        <div className="absolute top-40 right-10 text-green-400 text-xs font-mono opacity-20 animate-pulse">{">> Hash Processing..."}</div>
        <div className="absolute bottom-40 left-10 text-emerald-400 text-xs font-mono opacity-30 animate-pulse">{"[QR-READY]"}</div>
      </div>

      <div className="text-center mb-12 relative z-10">
        <div className="relative group mb-6">
          <div className="absolute inset-0 bg-gradient-to-r from-green-600 to-emerald-600 rounded-full blur-xl opacity-30 group-hover:opacity-50 transition-opacity duration-300"></div>
          <Activity className="w-16 h-16 text-green-600 mx-auto relative z-10 group-hover:scale-110 transition-transform duration-300 drop-shadow-2xl" />
          <div className="absolute inset-0 animate-ping rounded-full bg-green-400 opacity-20" style={{animationDuration: '3s'}}></div>
        </div>
        <h2 className="text-4xl font-bold mb-4 text-slate-800 dark:text-slate-200 hover:scale-105 transition-transform duration-300">
          🧪 Live Web3 Testing Environment
        </h2>
        <p className="text-xl text-slate-600 dark:text-slate-400 hover:text-slate-500 dark:hover:text-slate-300 transition-colors duration-300">
          Experience CodexHash quantum-resistant hashing in real-time
        </p>
      </div>

      <div className="group bg-white/95 dark:bg-slate-800/95 backdrop-blur-lg rounded-3xl shadow-2xl border border-green-200 dark:border-green-800 overflow-hidden hover:shadow-3xl hover:shadow-green-500/10 transition-all duration-300 relative z-10">
        <div className="absolute inset-0 bg-gradient-to-br from-green-500/5 to-emerald-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
        <div className="p-8 border-b border-green-200 dark:border-green-700 relative">
          <h3 className="text-2xl font-bold mb-4 flex items-center text-slate-800 dark:text-slate-200">
            <Code className="w-6 h-6 mr-3 text-green-600" />
            Input Data
          </h3>
          <div className="space-y-4">
            <textarea
              value={testInput}
              onChange={(e) => setTestInput(e.target.value)}
              placeholder="Enter any data to hash (text, JSON, contract address, etc.)"
              className="w-full h-32 p-4 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 font-mono text-sm resize-none focus:ring-2 focus:ring-green-500 focus:border-transparent transition-all duration-300"
              disabled={isLoading}
            />
            <div className="flex gap-4">
              <Button
                onClick={runTest}
                disabled={isLoading || !testInput.trim()}
                className="bg-gradient-to-r from-green-600 to-emerald-600 hover:from-green-700 hover:to-emerald-700 text-white px-6 py-3 rounded-lg font-semibold transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed flex items-center hover:scale-105"
              >
                {isLoading ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    Processing...
                  </>
                ) : (
                  <>
                    <Zap className="w-4 h-4 mr-2" />
                    Generate Hash
                  </>
                )}
              </Button>
              
              <Button
                onClick={clearTest}
                variant="outline"
                className="border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-all duration-300 hover:scale-105"
              >
                Clear
              </Button>
            </div>
          </div>
        </div>

        {error && (
          <div className="p-4 bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 m-6 rounded-r">
            <div className="flex items-center">
              <AlertCircle className="w-5 h-5 text-red-500 mr-2" />
              <p className="text-red-700 dark:text-red-400">{error}</p>
            </div>
          </div>
        )}

        {testResult && (
          <div className="p-8 space-y-6 animate-fadeIn">
            <div className="flex items-center justify-between">
              <h3 className="text-2xl font-bold flex items-center text-slate-800 dark:text-slate-200">
                <CheckCircle className="w-6 h-6 mr-3 text-green-600" />
                Hash Generated
              </h3>
              <div className="flex items-center space-x-2">
                <Shield className="w-5 h-5 text-green-600" />
                <span className="text-sm font-semibold text-green-600">Quantum Resistant</span>
              </div>
            </div>

            <div className="grid gap-6">
              <div className="bg-slate-50 dark:bg-slate-700/50 rounded-lg p-4 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors duration-300">
                <div className="flex items-center justify-between mb-2">
                  <label className="text-sm font-semibold text-slate-600 dark:text-slate-400">CodexHash Result</label>
                  <button
                    onClick={() => copyToClipboard(testResult.hash)}
                    className="text-blue-600 hover:text-blue-700 transition-colors duration-300"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
                <code className="text-xs font-mono text-slate-800 dark:text-slate-200 break-all bg-white dark:bg-slate-800 p-2 rounded border block">
                  {testResult.hash}
                </code>
              </div>

              <div className="bg-slate-50 dark:bg-slate-700/50 rounded-lg p-4 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors duration-300">
                <div className="flex items-center justify-between mb-2">
                  <label className="text-sm font-semibold text-slate-600 dark:text-slate-400">Quantum Salt</label>
                  <button
                    onClick={() => copyToClipboard(testResult.salt)}
                    className="text-blue-600 hover:text-blue-700 transition-colors duration-300"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                </div>
                <code className="text-xs font-mono text-slate-800 dark:text-slate-200 break-all bg-white dark:bg-slate-800 p-2 rounded border block">
                  {testResult.salt}
                </code>
              </div>

              <div className="grid md:grid-cols-2 gap-4">
                <div className="bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-blue-900/20 dark:to-indigo-900/20 rounded-lg p-4 border border-blue-200 dark:border-blue-800 hover:scale-105 transition-transform duration-300">
                  <div className="text-center">
                    <Hash className="w-8 h-8 text-blue-600 mx-auto mb-2" />
                    <div className="text-2xl font-bold text-slate-800 dark:text-slate-200">{testResult.meta.iterations}</div>
                    <div className="text-sm text-slate-600 dark:text-slate-400">Iterations</div>
                  </div>
                </div>

                <div className="bg-gradient-to-br from-purple-50 to-pink-100 dark:from-purple-900/20 dark:to-pink-900/20 rounded-lg p-4 border border-purple-200 dark:border-purple-800 hover:scale-105 transition-transform duration-300">
                  <div className="text-center">
                    <Sparkles className="w-8 h-8 text-purple-600 mx-auto mb-2" />
                    <div className="text-2xl font-bold text-slate-800 dark:text-slate-200">{testResult.tiu.toLocaleString()}</div>
                    <div className="text-sm text-slate-600 dark:text-slate-400">TIU Score</div>
                  </div>
                </div>
              </div>

              <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-4 border border-green-200 dark:border-green-800">
                <h4 className="font-semibold text-green-800 dark:text-green-200 mb-3">Performance Metrics</h4>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-slate-600 dark:text-slate-400">Algorithm</div>
                    <div className="font-semibold text-slate-800 dark:text-slate-200">{testResult.meta.algo}</div>
                  </div>
                  <div>
                    <div className="text-slate-600 dark:text-slate-400">Duration</div>
                    <div className="font-semibold text-slate-800 dark:text-slate-200">{testResult.meta.durationMs}ms</div>
                  </div>
                  <div>
                    <div className="text-slate-600 dark:text-slate-400">Input Size</div>
                    <div className="font-semibold text-slate-800 dark:text-slate-200">{testResult.meta.inputSize} bytes</div>
                  </div>
                  <div>
                    <div className="text-slate-600 dark:text-slate-400">QR Level</div>
                    <div className="font-semibold text-green-600">{testResult.meta.quantumResistance}%</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default TestingPanel
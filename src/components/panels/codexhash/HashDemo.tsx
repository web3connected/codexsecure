'use client'

import React, { useState } from 'react'
import { Button } from '@/components/widgets/ui/button'
import { Hash, Clock, Shield, Zap } from 'lucide-react'

interface HashResult {
  hash: string
  duration: number
  algorithm: string
  inputSize: number
}

const HashDemo: React.FC = () => {
  const [input, setInput] = useState('Hello, CodexHash!')
  const [result, setResult] = useState<HashResult | null>(null)
  const [isHashing, setIsHashing] = useState(false)

  // Mock hashing function - replace with actual API call
  const generateHash = async (text: string): Promise<HashResult> => {
    const start = Date.now()
    
    // Simulate network delay
    await new Promise(resolve => setTimeout(resolve, 100 + Math.random() * 200))
    
    // Mock hash generation (in reality, this would call your API)
    const mockHash = `codex_${btoa(text).replace(/[^a-zA-Z0-9]/g, '').toLowerCase().slice(0, 32)}_${Date.now().toString(36)}`
    
    return {
      hash: mockHash,
      duration: Date.now() - start,
      algorithm: 'CodexHash-v1.0',
      inputSize: new TextEncoder().encode(text).length
    }
  }

  const handleHash = async () => {
    if (!input.trim()) return
    
    setIsHashing(true)
    try {
      const result = await generateHash(input)
      setResult(result)
    } catch (error) {
      console.error('Hashing failed:', error)
    } finally {
      setIsHashing(false)
    }
  }

  const handleVerify = async () => {
    if (!input.trim() || !result) return
    
    setIsHashing(true)
    try {
      const newResult = await generateHash(input)
      const isValid = newResult.hash === result.hash
      
      // Visual feedback for verification
      const resultElement = document.getElementById('hash-result')
      if (resultElement) {
        resultElement.className = `p-4 rounded-lg border-2 transition-colors duration-300 ${
          isValid 
            ? 'bg-green-50 border-green-200 text-green-800' 
            : 'bg-red-50 border-red-200 text-red-800'
        }`
        
        setTimeout(() => {
          resultElement.className = 'p-4 rounded-lg bg-slate-50 border border-slate-200'
        }, 2000)
      }
    } catch (error) {
      console.error('Verification failed:', error)
    } finally {
      setIsHashing(false)
    }
  }

  return (
    <div className="w-full max-w-4xl mx-auto bg-slate-800 rounded-lg shadow-lg border border-slate-700">
      <div className="p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="p-2 bg-blue-600 rounded-lg">
            <Hash className="w-6 h-6 text-white" />
          </div>
          <div>
            <h3 className="text-xl font-semibold text-white">
              Interactive Hash Demo
            </h3>
            <p className="text-slate-400 text-sm">
              Experience quantum-resistant hashing in real-time
            </p>
          </div>
        </div>

        {/* Input Section */}
        <div className="mb-6">
          <label htmlFor="hash-input" className="block text-sm font-medium text-slate-300 mb-2">
            Input Text
          </label>
          <textarea
            id="hash-input"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Enter text to hash..."
            className="w-full p-3 bg-slate-700 border border-slate-600 text-white rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none placeholder-slate-400"
            rows={3}
            maxLength={1000}
          />
          <div className="text-right text-xs text-slate-500 mt-1">
            {input.length}/1000 characters
          </div>
        </div>

        {/* Controls */}
        <div className="flex flex-wrap gap-3 mb-6">
          <Button 
            onClick={handleHash}
            disabled={isHashing || !input.trim()}
            className="bg-blue-600 hover:bg-blue-700"
          >
            {isHashing ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Hashing...
              </>
            ) : (
              <>
                <Hash className="w-4 h-4 mr-2" />
                Generate Hash
              </>
            )}
          </Button>
          
          {result && (
            <Button
              onClick={handleVerify}
              disabled={isHashing}
              variant="outline"
              className="border-green-600 text-green-600 hover:bg-green-50"
            >
              <Shield className="w-4 h-4 mr-2" />
              Verify Hash
            </Button>
          )}
        </div>

        {/* Results */}
        {result && (
          <div id="hash-result" className="p-4 rounded-lg bg-slate-700 border border-slate-600">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-slate-400" />
                <span className="text-sm text-slate-300">
                  Duration: <span className="font-mono font-medium text-white">{result.duration}ms</span>
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Zap className="w-4 h-4 text-slate-400" />
                <span className="text-sm text-slate-300">
                  Input Size: <span className="font-mono font-medium text-white">{result.inputSize} bytes</span>
                </span>
              </div>
            </div>
            
            <div className="mb-2">
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Hash Output ({result.algorithm})
              </label>
              <div className="p-3 bg-slate-800 border border-slate-600 rounded font-mono text-sm break-all text-blue-400">
                {result.hash}
              </div>
            </div>
            
            <button
              onClick={() => navigator.clipboard.writeText(result.hash)}
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              Copy to clipboard
            </button>
          </div>
        )}

        {/* Features */}
        <div className="mt-6 pt-6 border-t border-slate-600">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-center">
            <div className="p-3">
              <div className="text-lg font-semibold text-blue-400">⚡ Fast</div>
              <div className="text-sm text-slate-400">Sub-millisecond performance</div>
            </div>
            <div className="p-3">
              <div className="text-lg font-semibold text-green-400">🛡️ Secure</div>
              <div className="text-sm text-slate-400">Quantum-resistant design</div>
            </div>
            <div className="p-3">
              <div className="text-lg font-semibold text-purple-400">🎯 Scalable</div>
              <div className="text-sm text-slate-400">Enterprise-grade reliability</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default HashDemo

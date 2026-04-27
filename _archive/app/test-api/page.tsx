'use client';

import React, { useState } from 'react';
import PageLayout from '@/components/layout/PageLayout';

interface APIResponse {
  hash?: string;
  salt?: string;
  tiu?: number;
  meta?: {
    algo: string;
    iterations: number;
    durationMs: number;
    inputSize: number;
    quantumResistance: number;
  };
  error?: string;
}

export default function APITestPage() {
  const [input, setInput] = useState('Hello CodexHash!');
  const [result, setResult] = useState<APIResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const testAPI = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/hash', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          input,
          tiu: 0.618034,
          iterations: 16
        }),
      });
      
      const data = await response.json();
      setResult(data);
    } catch (error) {
      console.error('API test failed:', error);
      setResult({ error: 'Failed to call API' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageLayout className="min-h-screen bg-gradient-to-br from-gray-900 via-purple-900 to-gray-900 py-12 px-4">
      <div className="max-w-4xl mx-auto">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-white mb-4">CodexHash API Test</h1>
          <p className="text-gray-300 text-lg">
            Test the Harmonic Hash API with real-time quantum-resistant hashing
          </p>
        </div>

        <div className="bg-gray-800/50 backdrop-blur-sm rounded-2xl p-8 border border-gray-700/50 shadow-2xl">
          <div className="space-y-6">
            {/* Input Section */}
            <div>
              <label htmlFor="input" className="block text-sm font-medium text-gray-300 mb-2">
                Input Text to Hash
              </label>
              <textarea
                id="input"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                className="w-full px-4 py-3 bg-gray-900/50 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none"
                rows={3}
                placeholder="Enter text to hash..."
              />
            </div>

            {/* Test Button */}
            <button
              onClick={testAPI}
              disabled={loading}
              className="w-full bg-gradient-to-r from-purple-600 to-blue-600 hover:from-purple-700 hover:to-blue-700 disabled:from-gray-600 disabled:to-gray-700 text-white font-semibold py-3 px-6 rounded-lg transition-all duration-300 transform hover:scale-[1.02] disabled:scale-100 disabled:cursor-not-allowed"
            >
              {loading ? (
                <div className="flex items-center justify-center space-x-2">
                  <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
                  <span>Generating Hash...</span>
                </div>
              ) : (
                'Test Harmonic Hash API'
              )}
            </button>

            {/* Results Section */}
            {result && (
              <div className="mt-8 space-y-6">
                <h3 className="text-xl font-semibold text-white border-b border-gray-600 pb-2">
                  API Response
                </h3>
                
                {result.error ? (
                  <div className="bg-red-900/30 border border-red-700/50 rounded-lg p-4">
                    <div className="flex items-center space-x-2">
                      <svg className="w-5 h-5 text-red-400" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                      </svg>
                      <span className="text-red-300 font-medium">Error: {result.error}</span>
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {/* Hash Output */}
                    <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-600">
                      <label className="block text-sm font-medium text-gray-400 mb-2">Generated Hash</label>
                      <div className="bg-black/30 rounded p-3 font-mono text-sm text-green-400 break-all">
                        {result.hash}
                      </div>
                    </div>

                    {/* Salt */}
                    {result.salt && (
                      <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-600">
                        <label className="block text-sm font-medium text-gray-400 mb-2">Salt</label>
                        <div className="bg-black/30 rounded p-3 font-mono text-sm text-cyan-400 break-all">
                          {result.salt}
                        </div>
                      </div>
                    )}

                    {/* TIU */}
                    {result.tiu && (
                      <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-600">
                        <label className="block text-sm font-medium text-gray-400 mb-2">Time Inversion Unit (TIU)</label>
                        <div className="bg-black/30 rounded p-3 font-mono text-sm text-purple-400">
                          {result.tiu}
                        </div>
                      </div>
                    )}

                    {/* Metadata */}
                    {result.meta && (
                      <div className="bg-gray-900/50 rounded-lg p-4 border border-gray-600">
                        <label className="block text-sm font-medium text-gray-400 mb-3">Metadata</label>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
                          <div className="flex items-center justify-between">
                            <span className="text-gray-400">Algorithm:</span>
                            <span className="text-white">{result.meta.algo}</span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-gray-400">Iterations:</span>
                            <span className="text-blue-400">{result.meta.iterations}</span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-gray-400">Duration:</span>
                            <span className="text-yellow-400">{result.meta.durationMs}ms</span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-gray-400">Input Size:</span>
                            <span className="text-green-400">{result.meta.inputSize} bytes</span>
                          </div>
                          <div className="flex items-center justify-between">
                            <span className="text-gray-400">Quantum Resistance:</span>
                            <span className="text-purple-400">{result.meta.quantumResistance}/100</span>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Info Section */}
        <div className="mt-12 bg-gray-800/30 rounded-xl p-6 border border-gray-700/50">
          <h3 className="text-lg font-semibold text-white mb-4">About the CodexHash SDK</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Version:</span>
              <span className="text-white">1.0.0</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Algorithm:</span>
              <span className="text-white">HarmonicHash, QuantumHasher</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Physics Constants:</span>
              <span className="text-purple-400">Speed of Light, Planck Frequency, Golden Ratio</span>
            </div>
          </div>
        </div>
      </div>
    </PageLayout>
  );
}

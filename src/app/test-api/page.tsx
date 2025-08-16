'use client';

import React, { useState } from 'react';

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
    <div className="min-h-screen bg-gradient-to-br from-gray-900 via-purple-900 to-gray-900 py-12 px-4">
      <div className="max-w-4xl mx-auto">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-white mb-4">
            HarmonicHash API Test
          </h1>
          <p className="text-xl text-gray-300">
            Testing our @web3connected/codexhash package integration
          </p>
        </div>

        <div className="bg-gray-800/50 backdrop-blur-md rounded-lg border border-gray-700/50 p-8">
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Input Text
            </label>
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              className="w-full px-4 py-2 bg-gray-700 border border-gray-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
              placeholder="Enter text to hash..."
            />
          </div>

          <button
            onClick={testAPI}
            disabled={loading || !input.trim()}
            className="w-full bg-purple-600 hover:bg-purple-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium py-3 px-6 rounded-md transition-colors duration-200"
          >
            {loading ? 'Generating Hash...' : 'Test HarmonicHash API'}
          </button>

          {result && (
            <div className="mt-8">
              <h3 className="text-lg font-semibold text-white mb-4">API Response:</h3>
              <div className="bg-gray-900/50 rounded-md p-4 border border-gray-600">
                <pre className="text-green-400 text-sm overflow-x-auto whitespace-pre-wrap">
                  {JSON.stringify(result, null, 2)}
                </pre>
              </div>
              
              {result.hash && (
                <div className="mt-6 space-y-4">
                  <div className="bg-gray-700/30 rounded-md p-4">
                    <h4 className="text-white font-medium mb-2">Hash Details:</h4>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between">
                        <span className="text-gray-400">Algorithm:</span>
                        <span className="text-white">{result.meta?.algo}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Iterations:</span>
                        <span className="text-white">{result.meta?.iterations}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Duration:</span>
                        <span className="text-white">{result.meta?.durationMs}ms</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">Quantum Resistance:</span>
                        <span className="text-green-400">{((result.meta?.quantumResistance || 0) * 100).toFixed(1)}%</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-gray-400">TIU (Golden Ratio):</span>
                        <span className="text-yellow-400">{result.tiu}</span>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="mt-8 bg-gray-800/30 rounded-lg border border-gray-700/50 p-6">
          <h3 className="text-lg font-semibold text-white mb-4">Package Integration Status</h3>
          <div className="space-y-2 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Package:</span>
              <span className="text-green-400">@web3connected/codexhash</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Source:</span>
              <span className="text-blue-400">file:../NPMPackages/codexhash</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Classes Used:</span>
              <span className="text-white">HarmonicHash, QuantumHasher</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-gray-400">Physics Constants:</span>
              <span className="text-purple-400">Speed of Light, Planck Frequency, Golden Ratio</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

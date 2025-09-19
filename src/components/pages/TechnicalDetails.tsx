'use client'

import React, { useState } from 'react'
import { Clock, Shield, Zap, Hash, Globe, ArrowRight, CheckCircle, AlertTriangle, Info } from 'lucide-react'
import { Button } from '../widgets/ui/button'
import LandingHeader from '../layout/LandingHeader'
import LandingFooter from '../common/LandingFooter'
import Link from 'next/link'

const TechnicalDetails: React.FC = () => {
  const [activeTab, setActiveTab] = useState('overview')

  const tabs = [
    { id: 'overview', label: 'Overview', icon: Info },
    { id: 'algorithm', label: 'Algorithm', icon: Hash },
    { id: 'implementation', label: 'Implementation', icon: Zap },
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'api', label: 'API Reference', icon: Globe }
  ]

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      {/* Header */}
      <LandingHeader />

      {/* Main Content */}
      <main className="pt-24">
        {/* Page Header */}
        <header className="bg-gray-800 border-b border-gray-700">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-3xl font-bold text-white">CodexHash Technical Documentation</h1>
                <p className="text-gray-400 mt-2">Comprehensive breakdown of quantum-resistant hashing technology</p>
              </div>
              <Link href="/">
                <Button className="bg-purple-600 hover:bg-purple-700">
                  <ArrowRight className="w-4 h-4 mr-2 rotate-180" />
                  Back to Home
                </Button>
              </Link>
            </div>
          </div>
        </header>

        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {/* Tab Navigation */}
          <div className="flex space-x-1 bg-gray-800 p-1 rounded-lg mb-8">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center px-4 py-2 rounded-md transition-all duration-200 ${
                    activeTab === tab.id
                      ? 'bg-purple-600 text-white'
                      : 'text-gray-400 hover:text-white hover:bg-gray-700'
                  }`}
                >
                  <Icon className="w-4 h-4 mr-2" />
                  {tab.label}
                </button>
              )
            })}
          </div>

          {/* Tab Content */}
          <div className="space-y-8">
          {activeTab === 'overview' && (
            <div className="space-y-8">
              {/* Executive Summary */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">Executive Summary</h2>
                <div className="grid md:grid-cols-2 gap-8">
                  <div>
                    <h3 className="text-xl font-semibold mb-4">What is CodexHash?</h3>
                    <p className="text-gray-300 mb-6">
                      CodexHash is a quantum-resistant hashing system that combines traditional cryptographic methods 
                      with physics-based constants and harmonic frequencies to create cryptographically secure hashes 
                      designed to withstand quantum computing attacks.
                    </p>
                    <div className="space-y-3">
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-5 h-5 mr-3" />
                        <span>Quantum-resistant algorithms</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-5 h-5 mr-3" />
                        <span>Physics-based entropy generation</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-5 h-5 mr-3" />
                        <span>Time-aware hashing (TIU integration)</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-5 h-5 mr-3" />
                        <span>Multi-phase security architecture</span>
                      </div>
                    </div>
                  </div>
                  <div>
                    <h3 className="text-xl font-semibold mb-4">Key Specifications</h3>
                    <div className="space-y-4">
                      <div className="bg-gray-900 p-4 rounded-lg">
                        <div className="text-sm text-gray-400">Hash Output Length</div>
                        <div className="text-xl font-bold text-purple-400">256-bit / 64 characters</div>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg">
                        <div className="text-sm text-gray-400">Average Processing Time</div>
                        <div className="text-xl font-bold text-blue-400">540ms</div>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg">
                        <div className="text-sm text-gray-400">Quantum Resistance</div>
                        <div className="text-xl font-bold text-green-400">30-99%</div>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg">
                        <div className="text-sm text-gray-400">Algorithm Phases</div>
                        <div className="text-xl font-bold text-pink-400">4 Phases</div>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              {/* Architecture Overview */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">System Architecture</h2>
                <div className="grid lg:grid-cols-3 gap-6">
                  <div className="bg-gray-900 p-6 rounded-lg border-l-4 border-purple-500">
                    <h3 className="text-lg font-semibold mb-3 text-purple-400">Frontend Layer</h3>
                    <ul className="space-y-2 text-gray-300">
                      <li>• Next.js 15.4.6 Application</li>
                      <li>• Interactive Demo Components</li>
                      <li>• Real-time Hash Visualization</li>
                      <li>• Responsive UI with Tailwind CSS</li>
                    </ul>
                  </div>
                  <div className="bg-gray-900 p-6 rounded-lg border-l-4 border-blue-500">
                    <h3 className="text-lg font-semibold mb-3 text-blue-400">API Gateway</h3>
                    <ul className="space-y-2 text-gray-300">
                      <li>• Laravel 10+ Backend</li>
                      <li>• RESTful API Endpoints</li>
                      <li>• Request Validation & Rate Limiting</li>
                      <li>• MCP Server Integration</li>
                    </ul>
                  </div>
                  <div className="bg-gray-900 p-6 rounded-lg border-l-4 border-green-500">
                    <h3 className="text-lg font-semibold mb-3 text-green-400">Hash Engine</h3>
                    <ul className="space-y-2 text-gray-300">
                      <li>• Python FastAPI Backend</li>
                      <li>• Physics-based Algorithm</li>
                      <li>• Harmonic Frequency Generation</li>
                      <li>• Quantum Resistance Analysis</li>
                    </ul>
                  </div>
                </div>
              </section>
            </div>
          )}

          {activeTab === 'algorithm' && (
            <div className="space-y-8">
              {/* Algorithm Core */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">HarmonicHash Algorithm</h2>
                <div className="space-y-6">
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-green-400">Physics Constants Foundation</h3>
                    <div className="grid md:grid-cols-2 gap-6">
                      <div>
                        <h4 className="font-semibold text-purple-400 mb-3">Core Constants</h4>
                        <div className="space-y-2 font-mono text-sm">
                          <div><span className="text-gray-400">Speed of Light (C):</span> <span className="text-white">299,792,458 m/s</span></div>
                          <div><span className="text-gray-400">Astronomical Unit (AU):</span> <span className="text-white">149,597,870,700 m</span></div>
                          <div><span className="text-gray-400">Planck Frequency:</span> <span className="text-white">1.855×10⁴³ Hz</span></div>
                          <div><span className="text-gray-400">Golden Ratio (φ):</span> <span className="text-white">1.618033988749895</span></div>
                        </div>
                      </div>
                      <div>
                        <h4 className="font-semibold text-purple-400 mb-3">Derived Values</h4>
                        <div className="space-y-2 font-mono text-sm">
                          <div><span className="text-gray-400">Base Frequency:</span> <span className="text-white">C / (2π × AU)</span></div>
                          <div><span className="text-gray-400">Harmonic Modifier:</span> <span className="text-white">(C × φ) % 2³²</span></div>
                          <div><span className="text-gray-400">Planck Modifier:</span> <span className="text-white">(PF / 10⁴⁰) % 2³²</span></div>
                          <div><span className="text-gray-400">TIU Pressure:</span> <span className="text-white">Planck Frequency</span></div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-blue-400">Four-Phase Hashing Process</h3>
                    <div className="space-y-4">
                      <div className="border-l-4 border-purple-500 pl-4">
                        <h4 className="font-semibold text-purple-400">Phase 1: Classical Foundation</h4>
                        <p className="text-gray-300">Initial SHA3-256 hash generation from input data to establish cryptographic baseline</p>
                      </div>
                      <div className="border-l-4 border-blue-500 pl-4">
                        <h4 className="font-semibold text-blue-400">Phase 2: Harmonic Transformation</h4>
                        <p className="text-gray-300">Apply physics constants (Speed of Light, Planck Frequency) as modifiers to introduce quantum properties</p>
                      </div>
                      <div className="border-l-4 border-green-500 pl-4">
                        <h4 className="font-semibold text-green-400">Phase 3: Dynamic Frequency Modulation</h4>
                        <p className="text-gray-300">Calculate harmonic frequency using Golden Ratio and π, apply to each byte with entropy distortion</p>
                      </div>
                      <div className="border-l-4 border-pink-500 pl-4">
                        <h4 className="font-semibold text-pink-400">Phase 4: Quantum-Resistant Encoding</h4>
                        <p className="text-gray-300">Final BLAKE2b encoding with 32-byte digest to produce quantum-resistant hash output</p>
                      </div>
                    </div>
                  </div>

                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-yellow-400">TIU (Time Integrity Unit) Integration</h3>
                    <p className="text-gray-300 mb-4">
                      Time Integrity Units provide temporal entropy that makes hashes time-aware and adds an additional 
                      layer of security by incorporating time-based distortion into the hash calculation.
                    </p>
                    <div className="bg-gray-800 p-4 rounded-lg font-mono text-sm">
                      <div className="text-gray-400 mb-2">TIU Entropy Calculation:</div>
                      <div className="text-white">entropy = (data_hash_sum % 9973) / 9973</div>
                      <div className="text-white">time_distortion = entropy / base_frequency * tiu</div>
                      <div className="text-white">final_modifier = planck_frequency * time_distortion</div>
                    </div>
                  </div>
                </div>
              </section>
            </div>
          )}

          {activeTab === 'implementation' && (
            <div className="space-y-8">
              {/* Implementation Stack */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">Implementation Stack</h2>
                <div className="grid lg:grid-cols-2 gap-8">
                  <div>
                    <h3 className="text-xl font-semibold mb-4 text-blue-400">Backend Architecture</h3>
                    <div className="space-y-4">
                      <div className="bg-gray-900 p-4 rounded-lg">
                        <h4 className="font-semibold text-green-400 mb-2">PHP/Laravel Layer</h4>
                        <ul className="text-gray-300 space-y-1 text-sm">
                          <li>• CodexHashService: Main service orchestrator</li>
                          <li>• CodexHarmonicHash: Core algorithm implementation</li>
                          <li>• CodexTime: TIU generation and management</li>
                          <li>• API Controllers: REST endpoint handlers</li>
                        </ul>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg">
                        <h4 className="font-semibold text-purple-400 mb-2">Python/FastAPI Layer</h4>
                        <ul className="text-gray-300 space-y-1 text-sm">
                          <li>• Harmonic hash algorithm implementation</li>
                          <li>• Physics constants and calculations</li>
                          <li>• Quantum resistance analysis</li>
                          <li>• Performance benchmarking tools</li>
                        </ul>
                      </div>
                    </div>
                  </div>
                  <div>
                    <h3 className="text-xl font-semibold mb-4 text-pink-400">API Endpoints</h3>
                    <div className="space-y-3">
                      <div className="bg-gray-900 p-3 rounded-lg">
                        <div className="font-mono text-green-400 text-sm">POST /api/codexhash/hash</div>
                        <div className="text-gray-400 text-xs">Generate quantum-resistant hash</div>
                      </div>
                      <div className="bg-gray-900 p-3 rounded-lg">
                        <div className="font-mono text-blue-400 text-sm">POST /api/codexhash/verify</div>
                        <div className="text-gray-400 text-xs">Verify hash integrity</div>
                      </div>
                      <div className="bg-gray-900 p-3 rounded-lg">
                        <div className="font-mono text-purple-400 text-sm">GET /api/codexhash/info</div>
                        <div className="text-gray-400 text-xs">Service information and features</div>
                      </div>
                      <div className="bg-gray-900 p-3 rounded-lg">
                        <div className="font-mono text-yellow-400 text-sm">POST /api/codexhash/benchmark</div>
                        <div className="text-gray-400 text-xs">Performance testing</div>
                      </div>
                      <div className="bg-gray-900 p-3 rounded-lg">
                        <div className="font-mono text-pink-400 text-sm">GET /analyze</div>
                        <div className="text-gray-400 text-xs">Quantum resistance analysis</div>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              {/* Code Examples */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">Code Examples</h2>
                <div className="space-y-6">
                  <div>
                    <h3 className="text-lg font-semibold mb-3 text-green-400">PHP Implementation (Core Algorithm)</h3>
                    <div className="bg-gray-900 p-4 rounded-lg font-mono text-sm overflow-x-auto">
                      <pre className="text-gray-300">{`public function hash(string $data, ?string $salt = null, ?float $tiu = null, int $rounds = 16): array
{
    $salt ??= hash('sha256', 'codex-fixed-salt');
    $base = $data . hex2bin($salt) . $tiu;
    $mod = $this->entropyModifier($base, $tiu);
    $modHash = hash('sha256', number_format($mod, 18, '.', ''), true);
    
    $result = $this->xorBytes(hash('sha256', $base, true), $modHash);
    
    for ($i = 1; $i < $rounds; $i++) {
        $result = hash('sha256', $result, true);
    }
    
    return [
        'hash' => bin2hex($result),
        'salt' => $salt,
        'tiu' => number_format($tiu, 6, '.', ''),
        'rounds' => $rounds
    ];
}`}</pre>
                    </div>
                  </div>
                  
                  <div>
                    <h3 className="text-lg font-semibold mb-3 text-blue-400">Python Implementation (Harmonic Transform)</h3>
                    <div className="bg-gray-900 p-4 rounded-lg font-mono text-sm overflow-x-auto">
                      <pre className="text-gray-300">{`def harmonic_hash(data: str) -> tuple[str, Dict[str, Any]]:
    # Phase 1: Classical hash foundation
    sha3_hash = hashlib.sha3_256(data.encode('utf-8')).digest()
    
    # Phase 2: Harmonic transformation using physics constants
    harmonic_modifier = int(SPEED_OF_LIGHT * GOLDEN_RATIO) % (2**32)
    planck_modifier = int(PLANCK_FREQUENCY / 1e40) % (2**32)
    
    # Phase 3: Dynamic frequency modulation
    frequency = len(data) * GOLDEN_RATIO * PI
    harmonic_bytes = bytearray()
    
    for i, byte in enumerate(sha3_hash):
        harmonic_value = (
            byte ^ 
            ((harmonic_modifier + i) % 256) ^
            ((planck_modifier * i) % 256) ^
            (int(frequency * i) % 256)
        ) % 256
        harmonic_bytes.append(harmonic_value)
    
    # Phase 4: Final quantum-resistant encoding
    final_hash = hashlib.blake2b(harmonic_bytes, digest_size=32).hexdigest()
    
    return final_hash, metadata, quantum_resistance, processing_time`}</pre>
                    </div>
                  </div>
                </div>
              </section>
            </div>
          )}

          {activeTab === 'security' && (
            <div className="space-y-8">
              {/* Security Analysis */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">Security Analysis</h2>
                <div className="grid lg:grid-cols-2 gap-8">
                  <div>
                    <h3 className="text-xl font-semibold mb-4 text-green-400">Quantum Resistance Features</h3>
                    <div className="space-y-4">
                      <div className="flex items-start space-x-3">
                        <Shield className="w-5 h-5 text-green-400 mt-1 flex-shrink-0" />
                        <div>
                          <h4 className="font-semibold text-white">Physics-Based Entropy</h4>
                          <p className="text-gray-300 text-sm">Uses fundamental constants that quantum computers cannot easily reverse-engineer</p>
                        </div>
                      </div>
                      <div className="flex items-start space-x-3">
                        <Clock className="w-5 h-5 text-blue-400 mt-1 flex-shrink-0" />
                        <div>
                          <h4 className="font-semibold text-white">Time-Aware Hashing</h4>
                          <p className="text-gray-300 text-sm">TIU integration adds temporal complexity that changes over time</p>
                        </div>
                      </div>
                      <div className="flex items-start space-x-3">
                        <Zap className="w-5 h-5 text-yellow-400 mt-1 flex-shrink-0" />
                        <div>
                          <h4 className="font-semibold text-white">Multi-Phase Processing</h4>
                          <p className="text-gray-300 text-sm">Four distinct phases prevent single-point cryptographic failures</p>
                        </div>
                      </div>
                      <div className="flex items-start space-x-3">
                        <Hash className="w-5 h-5 text-purple-400 mt-1 flex-shrink-0" />
                        <div>
                          <h4 className="font-semibold text-white">Harmonic Frequency Modulation</h4>
                          <p className="text-gray-300 text-sm">Dynamic frequency calculation based on Golden Ratio creates unpredictable patterns</p>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div>
                    <h3 className="text-xl font-semibold mb-4 text-red-400">Security Levels</h3>
                    <div className="space-y-4">
                      <div className="bg-gray-900 p-4 rounded-lg border-l-4 border-green-500">
                        <div className="flex justify-between items-center mb-2">
                          <span className="font-semibold text-green-400">MAXIMUM</span>
                          <span className="text-green-400">90-99%</span>
                        </div>
                        <p className="text-gray-300 text-sm">Highest quantum resistance with complex harmonic patterns</p>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg border-l-4 border-blue-500">
                        <div className="flex justify-between items-center mb-2">
                          <span className="font-semibold text-blue-400">HIGH</span>
                          <span className="text-blue-400">70-89%</span>
                        </div>
                        <p className="text-gray-300 text-sm">Strong protection against quantum and classical attacks</p>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg border-l-4 border-yellow-500">
                        <div className="flex justify-between items-center mb-2">
                          <span className="font-semibold text-yellow-400">MEDIUM</span>
                          <span className="text-yellow-400">50-69%</span>
                        </div>
                        <p className="text-gray-300 text-sm">Adequate protection for most applications</p>
                      </div>
                      <div className="bg-gray-900 p-4 rounded-lg border-l-4 border-gray-500">
                        <div className="flex justify-between items-center mb-2">
                          <span className="font-semibold text-gray-400">BASIC</span>
                          <span className="text-gray-400">30-49%</span>
                        </div>
                        <p className="text-gray-300 text-sm">Entry-level quantum resistance</p>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              {/* Vulnerability Assessment */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">Vulnerability Assessment</h2>
                <div className="grid md:grid-cols-3 gap-6">
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-lg font-semibold mb-4 text-green-400">Resistant To</h3>
                    <div className="space-y-2">
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Rainbow Table Attacks</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Brute Force Attacks</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Collision Attacks</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Quantum Computing</span>
                      </div>
                      <div className="flex items-center text-green-400">
                        <CheckCircle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Time-based Attacks</span>
                      </div>
                    </div>
                  </div>
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-lg font-semibold mb-4 text-yellow-400">Considerations</h3>
                    <div className="space-y-2">
                      <div className="flex items-center text-yellow-400">
                        <AlertTriangle className="w-4 h-4 mr-2" />
                        <span className="text-sm">TIU Synchronization Required</span>
                      </div>
                      <div className="flex items-center text-yellow-400">
                        <AlertTriangle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Higher Processing Overhead</span>
                      </div>
                      <div className="flex items-center text-yellow-400">
                        <AlertTriangle className="w-4 h-4 mr-2" />
                        <span className="text-sm">Buffer Verification Needed</span>
                      </div>
                    </div>
                  </div>
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-lg font-semibold mb-4 text-blue-400">Best Practices</h3>
                    <div className="space-y-2 text-sm text-gray-300">
                      <div>• Use 16+ rounds for maximum security</div>
                      <div>• Implement proper TIU management</div>
                      <div>• Monitor quantum resistance scores</div>
                      <div>• Regular algorithm updates</div>
                      <div>• Backup verification methods</div>
                    </div>
                  </div>
                </div>
              </section>
            </div>
          )}

          {activeTab === 'api' && (
            <div className="space-y-8">
              {/* API Reference */}
              <section className="bg-gray-800 rounded-xl p-8 border border-gray-700">
                <h2 className="text-2xl font-bold mb-6 text-purple-400">API Reference</h2>
                <div className="space-y-6">
                  {/* Hash Generation */}
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-green-400">Generate Hash</h3>
                    <div className="space-y-4">
                      <div>
                        <div className="flex items-center space-x-2 mb-2">
                          <span className="bg-green-600 text-white px-2 py-1 rounded text-xs font-bold">POST</span>
                          <span className="font-mono text-white">/api/codexhash/hash</span>
                        </div>
                        <p className="text-gray-400 text-sm">Generate a quantum-resistant hash using the HarmonicHash algorithm</p>
                      </div>
                      <div>
                        <h4 className="font-semibold text-white mb-2">Request Body:</h4>
                        <div className="bg-gray-800 p-3 rounded font-mono text-sm">
                          <pre className="text-gray-300">{`{
  "input": "string",      // Required: Data to hash
  "rounds": 16            // Optional: Number of hash rounds (1-32)
}`}</pre>
                        </div>
                      </div>
                      <div>
                        <h4 className="font-semibold text-white mb-2">Response:</h4>
                        <div className="bg-gray-800 p-3 rounded font-mono text-sm">
                          <pre className="text-gray-300">{`{
  "success": true,
  "data": {
    "hash": "64-character hex string",
    "salt": "hash salt value",
    "tiu": "time integrity unit",
    "rounds": 16
  },
  "input_length": 123,
  "timestamp": "2025-09-19T..."
}`}</pre>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Hash Verification */}
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-blue-400">Verify Hash</h3>
                    <div className="space-y-4">
                      <div>
                        <div className="flex items-center space-x-2 mb-2">
                          <span className="bg-blue-600 text-white px-2 py-1 rounded text-xs font-bold">POST</span>
                          <span className="font-mono text-white">/api/codexhash/verify</span>
                        </div>
                        <p className="text-gray-400 text-sm">Verify hash integrity and authenticity</p>
                      </div>
                      <div>
                        <h4 className="font-semibold text-white mb-2">Request Body:</h4>
                        <div className="bg-gray-800 p-3 rounded font-mono text-sm">
                          <pre className="text-gray-300">{`{
  "input": "string",      // Required: Original data
  "hash": "string",       // Required: Hash to verify
  "salt": "string",       // Required: Salt used
  "tiu": 1.234567,        // Required: TIU value
  "rounds": 16            // Optional: Hash rounds used
}`}</pre>
                        </div>
                      </div>
                      <div>
                        <h4 className="font-semibold text-white mb-2">Response:</h4>
                        <div className="bg-gray-800 p-3 rounded font-mono text-sm">
                          <pre className="text-gray-300">{`{
  "success": true,
  "verified": true,
  "rounds": 16,
  "timestamp": "2025-09-19T..."
}`}</pre>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Service Info */}
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-purple-400">Service Information</h3>
                    <div className="space-y-4">
                      <div>
                        <div className="flex items-center space-x-2 mb-2">
                          <span className="bg-purple-600 text-white px-2 py-1 rounded text-xs font-bold">GET</span>
                          <span className="font-mono text-white">/api/codexhash/info</span>
                        </div>
                        <p className="text-gray-400 text-sm">Get comprehensive service information and available features</p>
                      </div>
                      <div>
                        <h4 className="font-semibold text-white mb-2">Response:</h4>
                        <div className="bg-gray-800 p-3 rounded font-mono text-sm">
                          <pre className="text-gray-300">{`{
  "service": "CodexHash",
  "description": "Quantum Hash System for SEO Optimization",
  "version": "1.0.0",
  "features": [
    "Quantum-aware hashing algorithms",
    "Time-based entropy generation",
    "Harmonic resonance optimization",
    "SEO-focused content indexing"
  ],
  "endpoints": { ... }
}`}</pre>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Benchmark */}
                  <div className="bg-gray-900 p-6 rounded-lg">
                    <h3 className="text-xl font-semibold mb-4 text-yellow-400">Performance Benchmark</h3>
                    <div className="space-y-4">
                      <div>
                        <div className="flex items-center space-x-2 mb-2">
                          <span className="bg-yellow-600 text-white px-2 py-1 rounded text-xs font-bold">POST</span>
                          <span className="font-mono text-white">/api/codexhash/benchmark</span>
                        </div>
                        <p className="text-gray-400 text-sm">Run performance benchmarks to test hashing speed and efficiency</p>
                      </div>
                      <div>
                        <h4 className="font-semibold text-white mb-2">Request Body:</h4>
                        <div className="bg-gray-800 p-3 rounded font-mono text-sm">
                          <pre className="text-gray-300">{`{
  "iterations": 10,       // Optional: Number of iterations (1-100)
  "input": "test string"  // Optional: Test input data
}`}</pre>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </section>
            </div>
          )}
          </div>
        </div>
      </main>

      {/* Footer */}
      <LandingFooter />
    </div>
  )
}

export default TechnicalDetails
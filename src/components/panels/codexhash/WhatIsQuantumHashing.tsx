import React from 'react'
import { ShieldCheck, Infinity, Zap, Lock } from 'lucide-react'

const WhatIsQuantumHashing = () => {
    return (
        <section className="py-20 bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
            <div className="container mx-auto px-6 max-w-4xl">

                {/* Title & Description */}
                <div className="mb-12 text-center">
                    <h2 className="text-4xl font-bold mb-4 text-blue-300">
                        What is Quantum Hashing?
                    </h2>
                    <p className="text-lg text-slate-300 leading-relaxed max-w-2xl mx-auto">
                        Quantum hashing is the next evolution in cryptographic security, engineered to
                        resist the power of quantum computers. Unlike traditional hash functions that
                        quantum algorithms can break, quantum-resistant methods ensure your data
                        remains secure in the post-quantum era.
                    </p>
                </div>

                {/* Key Features */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                    {/* Feature Card */}
                    <div className="p-6 bg-slate-800/60 rounded-xl border border-slate-700 hover:border-blue-500/40 transition-all">
                        <div className="flex items-center mb-3">
                            <ShieldCheck className="w-6 h-6 text-green-400 mr-2" />
                            <h3 className="text-lg font-semibold text-slate-100">Post-Quantum Security</h3>
                        </div>
                        <p className="text-slate-400 text-sm leading-relaxed">
                            Immune to Shor&apos;s and Grover&apos;s algorithms, built for quantum resilience.
                        </p>
                    </div>

                    <div className="p-6 bg-slate-800/60 rounded-xl border border-slate-700 hover:border-blue-500/40 transition-all">
                        <div className="flex items-center mb-3">
                            <Infinity className="w-6 h-6 text-blue-400 mr-2" />
                            <h3 className="text-lg font-semibold text-slate-100">Future-Proof</h3>
                        </div>
                        <p className="text-slate-400 text-sm leading-relaxed">
                            Secure and adaptable — designed to thrive in the quantum computing era.
                        </p>
                    </div>

                    <div className="p-6 bg-slate-800/60 rounded-xl border border-slate-700 hover:border-blue-500/40 transition-all">
                        <div className="flex items-center mb-3">
                            <Zap className="w-6 h-6 text-yellow-400 mr-2" />
                            <h3 className="text-lg font-semibold text-slate-100">High Performance</h3>
                        </div>
                        <p className="text-slate-400 text-sm leading-relaxed">
                            Optimized algorithms ensure speed, efficiency, and scalability.
                        </p>
                    </div>

                    <div className="p-6 bg-slate-800/60 rounded-xl border border-slate-700 hover:border-blue-500/40 transition-all">
                        <div className="flex items-center mb-3">
                            <Lock className="w-6 h-6 text-purple-400 mr-2" />
                            <h3 className="text-lg font-semibold text-slate-100">Cryptographic Strength</h3>
                        </div>
                        <p className="text-slate-400 text-sm leading-relaxed">
                            256-bit to 1024-bit security levels to match diverse use cases.
                        </p>
                    </div>
                </div>
            </div>
        </section>
    )
}

export default WhatIsQuantumHashing

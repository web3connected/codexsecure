import React from 'react'

const ComparisonTable = () => {
    return (
        <div className="mb-20 max-w-6xl mx-auto px-4 bg-slate-800/50">
            <h2 className="text-3xl font-bold mb-8 p-5 text-center text-cyan-400">Quantum vs Traditional Hashing</h2>
            <div className="overflow-x-auto">
                <table className="w-full bg-slate-800/50 rounded-lg border border-slate-700">
                    <thead>
                        <tr className="border-b border-slate-700">
                            <th className="px-6 py-4 text-left text-gray-300">Feature</th>
                            <th className="px-6 py-4 text-center text-red-400">Traditional Hash</th>
                            <th className="px-6 py-4 text-center text-yellow-400">Quantum Hash</th>
                            <th className="px-6 py-4 text-center text-purple-400">Harmonic Hash</th>
                        </tr>
                    </thead>
                    <tbody className="text-gray-300">
                        <tr className="border-b border-slate-700/50">
                            <td className="px-6 py-4 font-medium">Quantum Resistance</td>
                            <td className="px-6 py-4 text-center text-red-400">❌ Vulnerable</td>
                            <td className="px-6 py-4 text-center text-green-400">✅ Protected</td>
                            <td className="px-6 py-4 text-center text-purple-400">🌟 Advanced</td>
                        </tr>
                        <tr className="border-b border-slate-700/50">
                            <td className="px-6 py-4 font-medium">Mathematical Foundation</td>
                            <td className="px-6 py-4 text-center">Discrete Log</td>
                            <td className="px-6 py-4 text-center">Lattice-based</td>
                            <td className="px-6 py-4 text-center">Harmonic Principles</td>
                        </tr>
                        <tr className="border-b border-slate-700/50">
                            <td className="px-6 py-4 font-medium">Security Level</td>
                            <td className="px-6 py-4 text-center">128-256 bit</td>
                            <td className="px-6 py-4 text-center">256+ bit</td>
                            <td className="px-6 py-4 text-center">Sacred 12 Matrix</td>
                        </tr>
                        <tr className="border-b border-slate-700/50">
                            <td className="px-6 py-4 font-medium">Performance</td>
                            <td className="px-6 py-4 text-center text-green-400">Fast</td>
                            <td className="px-6 py-4 text-center text-yellow-400">Moderate</td>
                            <td className="px-6 py-4 text-center text-purple-400">Optimized</td>
                        </tr>
                        <tr>
                            <td className="px-6 py-4 font-medium">Future-Proof</td>
                            <td className="px-6 py-4 text-center text-red-400">No</td>
                            <td className="px-6 py-4 text-center text-green-400">Yes</td>
                            <td className="px-6 py-4 text-center text-purple-400">Beyond Future</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div >
    )
}

export default ComparisonTable
import React from 'react'
import { Target } from 'lucide-react'
import Image from 'next/image'

const ComparisonsPanel: React.FC = () => {
  return (
    <div className="min-h-full flex flex-col justify-center space-y-8 relative overflow-hidden" style={{
      padding: "100px"
    }}>
      {/* Enhanced Competitive Background */}
      <div className="absolute inset-0 bg-gradient-to-br from-red-900/20 via-orange-900/30 to-yellow-900/20"></div>
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,_var(--tw-gradient-stops))] from-red-500/10 via-transparent to-transparent"></div>
      
      {/* Web3 Security Background */}
      <div className="absolute inset-0 opacity-5">
        <Image
          src="/assets/images/web3-sicurezza.webp"
          alt="Web3 Security Comparison Background"
          fill
          className="object-cover"
        />
      </div>
      
      {/* Battle Elements */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute top-20 left-10 w-8 h-1 bg-red-500 rotate-45 opacity-20"></div>
        <div className="absolute top-40 right-20 w-6 h-1 bg-orange-500 -rotate-45 opacity-30"></div>
        <div className="absolute bottom-32 left-1/4 w-10 h-1 bg-yellow-500 rotate-12 opacity-25"></div>
        <div className="absolute top-60 right-1/3 w-4 h-4 border-2 border-red-500 rounded-full opacity-20"></div>
      </div>

      <div className="text-center mb-12 relative z-10">
        <div className="relative group mb-6">
          <div className="absolute inset-0 bg-gradient-to-r from-red-600 to-orange-600 rounded-full blur-xl opacity-30 group-hover:opacity-50 transition-opacity duration-300"></div>
          <Target className="w-16 h-16 text-red-600 mx-auto relative z-10 group-hover:scale-110 transition-transform duration-300 drop-shadow-2xl animate-pulse" />
        </div>
        <h2 className="text-4xl font-bold mb-4 text-slate-800 dark:text-slate-200 hover:scale-105 transition-transform duration-300">
          ⚔️ CodexHash vs Traditional Hashing
        </h2>
        <p className="text-xl text-slate-600 dark:text-slate-400 hover:text-slate-500 dark:hover:text-slate-300 transition-colors duration-300">
          See why Web3 builders choose CodexHash over legacy algorithms
        </p>
      </div>

      <div className="overflow-hidden rounded-2xl shadow-2xl relative z-10 group hover:shadow-3xl transition-shadow duration-300">
        <table className="w-full bg-white/95 dark:bg-slate-800/95 backdrop-blur-lg">
          <thead className="bg-gradient-to-r from-blue-500 via-purple-600 to-red-500 text-white">
            <tr>
              <th className="px-6 py-4 text-left font-bold">Feature</th>
              <th className="px-6 py-4 text-center font-bold hover:scale-105 transition-transform duration-300">🏆 CodexHash</th>
              <th className="px-6 py-4 text-center font-bold hover:scale-105 transition-transform duration-300">SHA-256</th>
              <th className="px-6 py-4 text-center font-bold hover:scale-105 transition-transform duration-300">Keccak-256</th>
              <th className="px-6 py-4 text-center font-bold hover:scale-105 transition-transform duration-300">Blake2b</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
            <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/70 transition-colors duration-300 group">
              <td className="px-6 py-4 font-semibold group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors duration-300">🛡️ Quantum Resistance</td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:scale-110 transition-transform duration-300">
                  ✅ Native
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 hover:scale-110 transition-transform duration-300">
                  ❌ Vulnerable
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 hover:scale-110 transition-transform duration-300">
                  ❌ Vulnerable
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 hover:scale-110 transition-transform duration-300">
                  ❌ Vulnerable
                </span>
              </td>
            </tr>
            <tr className="bg-slate-50 dark:bg-slate-800/50 hover:bg-slate-100 dark:hover:bg-slate-800/70 transition-colors duration-300 group">
              <td className="px-6 py-4 font-semibold group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors duration-300">⛽ Gas Efficiency</td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:scale-110 transition-transform duration-300">
                  ✅ Optimized
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 hover:scale-110 transition-transform duration-300">
                  ⚠️ Standard
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:scale-110 transition-transform duration-300">
                  ✅ Good
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 hover:scale-110 transition-transform duration-300">
                  ⚠️ High
                </span>
              </td>
            </tr>
            <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/70 transition-colors duration-300 group">
              <td className="px-6 py-4 font-semibold group-hover:text-green-600 dark:group-hover:text-green-400 transition-colors duration-300">🌐 Web3 Native</td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:scale-110 transition-transform duration-300">
                  ✅ Built-in
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 hover:scale-110 transition-transform duration-300">
                  ❌ Legacy
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 hover:scale-110 transition-transform duration-300">
                  ⚠️ Adapted
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 hover:scale-110 transition-transform duration-300">
                  ❌ Legacy
                </span>
              </td>
            </tr>
            <tr className="bg-slate-50 dark:bg-slate-800/50 hover:bg-slate-100 dark:hover:bg-slate-800/70 transition-colors duration-300 group">
              <td className="px-6 py-4 font-semibold group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors duration-300">🔒 Entropy Source</td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 hover:scale-110 transition-transform duration-300">
                  🌌 Physics
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200 hover:scale-110 transition-transform duration-300">
                  🔢 Math
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200 hover:scale-110 transition-transform duration-300">
                  🔢 Math
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200 hover:scale-110 transition-transform duration-300">
                  🔢 Math
                </span>
              </td>
            </tr>
            <tr className="hover:bg-slate-50 dark:hover:bg-slate-800/70 transition-colors duration-300 group">
              <td className="px-6 py-4 font-semibold group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors duration-300">📱 DApp Integration</td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:scale-110 transition-transform duration-300">
                  ✅ Seamless
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 hover:scale-110 transition-transform duration-300">
                  ⚠️ Complex
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 hover:scale-110 transition-transform duration-300">
                  ✅ Good
                </span>
              </td>
              <td className="px-6 py-4 text-center">
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 hover:scale-110 transition-transform duration-300">
                  ⚠️ Complex
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default ComparisonsPanel
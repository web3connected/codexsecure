import { Shield } from 'lucide-react'
import React from 'react'

const OpenSourceBadge = () => {
  return (
      <div className="inline-flex items-center gap-2 bg-blue-500/20 border border-blue-500/30 rounded-full px-4 py-2 mb-8">
          <Shield className="w-4 h-4 text-blue-400" />
          <span className="text-sm font-medium text-blue-100">
              Open Source Hashing Library
          </span>
      </div>
  )
}

export default OpenSourceBadge
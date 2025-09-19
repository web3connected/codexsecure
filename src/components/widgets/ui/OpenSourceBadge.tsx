import React from 'react'
import { Globe } from 'lucide-react'

const OpenSourceBadge: React.FC = () => {
  return (
    <div className="inline-flex items-center gap-2 px-4 py-2 bg-green-500/20 text-green-400 rounded-full text-sm font-medium mb-8">
      <Globe className="w-4 h-4" />
      Open Source
    </div>
  )
}

export default OpenSourceBadge
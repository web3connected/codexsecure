'use client'

import React, { useState } from 'react'
import { useCodexTheme } from '@/contexts/ThemeContext'

const ThemeSwitcher: React.FC = () => {
  const { codexTheme, setCodexTheme, themes } = useCodexTheme()
  const [isOpen, setIsOpen] = useState(false)

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="btn-theme-secondary px-4 py-2 rounded-lg flex items-center gap-2"
      >
        <i className="ph ph-palette"></i>
        <span className="hidden md:inline">Theme</span>
        <i className={`ph ph-caret-${isOpen ? 'up' : 'down'}`}></i>
      </button>

      {isOpen && (
        <div className="absolute top-full right-0 mt-2 w-48 card-theme rounded-lg shadow-lg z-50">
          <div className="p-2">
            {Object.entries(themes).map(([key, name]) => (
              <button
                key={key}
                onClick={() => {
                  setCodexTheme(key)
                  setIsOpen(false)
                }}
                className={`w-full text-left px-3 py-2 rounded hover:theme-bg-surface transition-colors ${
                  codexTheme === key ? 'theme-text theme-bg-primary' : 'theme-text-muted'
                }`}
              >
                {name}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default ThemeSwitcher

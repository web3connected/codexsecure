import { Shield, Menu, X } from 'lucide-react';
import React from 'react'

interface PanelType {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface SliderHeaderNavProps {
  panels: PanelType[];
  activePanel: string;
  setActivePanel: (panelId: string) => void;
  isMenuOpen: boolean;
  setIsMenuOpen: (open: boolean) => void;
}

const SliderHeaderNav: React.FC<SliderHeaderNavProps> = ({
  panels,
  activePanel,
  setActivePanel,
  isMenuOpen,
  setIsMenuOpen,
}) => {
  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-white/95 dark:bg-slate-900/95 backdrop-blur-lg border-b border-slate-200 dark:border-slate-800">
        <div className="container mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <Shield className="w-8 h-8 text-blue-600" />
              <span className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                CodexHash
              </span>
            </div>

            <nav className="hidden md:flex space-x-1">
              {panels.map((panel) => {
                const Icon = panel.icon
                return (
                  <button
                    key={panel.id}
                    onClick={() => setActivePanel(panel.id)}
                    className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-all duration-200 ${
                      activePanel === panel.id
                        ? 'bg-blue-500 text-white shadow-lg'
                        : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                    <span className="font-medium">{panel.label}</span>
                  </button>
                )
              })}
            </nav>

            <button
              onClick={() => setIsMenuOpen(!isMenuOpen)}
              className="md:hidden p-2 rounded-lg bg-slate-100 dark:bg-slate-800"
            >
              {isMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
            </button>
          </div>

          {/* Mobile Menu */}
          {isMenuOpen && (
            <div className="md:hidden mt-4 pb-4 border-t border-slate-200 dark:border-slate-700 pt-4">
              <div className="space-y-2">
                {panels.map((panel) => {
                  const Icon = panel.icon
                  return (
                    <button
                      key={panel.id}
                      onClick={() => {
                        setActivePanel(panel.id)
                        setIsMenuOpen(false)
                      }}
                      className={`w-full flex items-center space-x-3 px-4 py-3 rounded-lg transition-all duration-200 ${
                        activePanel === panel.id
                          ? 'bg-blue-500 text-white'
                          : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                      }`}
                    >
                      <Icon className="w-5 h-5" />
                      <span className="font-medium">{panel.label}</span>
                    </button>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>
  )
}

export default SliderHeaderNav
import React from 'react'

interface PageLayoutProps {
  children: React.ReactNode
  title?: string
  description?: string
  className?: string
}

/**
 * PageLayout - Inner page wrapper for CodexSecure
 * Note: Header/Footer are provided by ClientLayout in root layout.
 * This component provides consistent inner page structure only.
 */
export default function PageLayout({ children, title, description, className }: PageLayoutProps) {
  return (
    <div className={className ?? "min-h-screen bg-gradient-to-br from-slate-900 via-slate-900 to-slate-950"}>
      <main className="flex-1">
        {(title || description) && (
          <div className="container mx-auto px-4 py-12">
            {title && (
              <h1 className="text-4xl lg:text-5xl font-bold text-white mb-4">
                {title}
              </h1>
            )}
            {description && (
              <p className="text-xl text-slate-300 max-w-3xl">
                {description}
              </p>
            )}
          </div>
        )}
        {children}
      </main>
    </div>
  )
}

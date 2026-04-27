import React from 'react'
import Link from 'next/link'

interface Breadcrumb {
  label: string
  href?: string
}

interface TitleBarProps {
  title: string
  description?: string
  breadcrumbs?: Breadcrumb[]
}

/**
 * TitleBar - Page title section with optional description and breadcrumb navigation
 */
export default function TitleBar({ title, description, breadcrumbs }: TitleBarProps) {
  return (
    <div className="bg-slate-900 border-b border-slate-800">
      <div className="container mx-auto px-6 py-8 max-w-7xl">
        {breadcrumbs && breadcrumbs.length > 0 && (
          <nav className="flex items-center gap-2 text-sm text-slate-400 mb-4">
            {breadcrumbs.map((crumb, i) => (
              <React.Fragment key={i}>
                {i > 0 && <span className="text-slate-600">/</span>}
                {crumb.href ? (
                  <Link href={crumb.href} className="hover:text-white transition-colors">
                    {crumb.label}
                  </Link>
                ) : (
                  <span className="text-slate-300">{crumb.label}</span>
                )}
              </React.Fragment>
            ))}
          </nav>
        )}
        <h1 className="text-3xl lg:text-4xl font-bold text-white mb-3">{title}</h1>
        {description && (
          <p className="text-slate-400 max-w-3xl leading-relaxed">{description}</p>
        )}
      </div>
    </div>
  )
}

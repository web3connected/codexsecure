import React from 'react'

interface ContentLayoutProps {
    children: React.ReactNode
    className?: string
    title?: string
    subtitle?: string
    breadcrumbs?: Array<{ href?: string; label: string }>
}

const ContentLayout: React.FC<ContentLayoutProps> = ({ 
    children, 
    className = "py-12 px-6",
    title,
    subtitle,
    breadcrumbs
}) => {
    return (
        <main className={className}>
            <div className="container mx-auto max-w-4xl">
                {/* Breadcrumbs */}
                {breadcrumbs && (
                    <nav className="mb-8">
                        {breadcrumbs.map((crumb, index) => (
                            <React.Fragment key={index}>
                                {crumb.href ? (
                                    <a href={crumb.href} className="text-blue-400 hover:underline">
                                        {crumb.label}
                                    </a>
                                ) : (
                                    <span className="text-white">{crumb.label}</span>
                                )}
                                {index < breadcrumbs.length - 1 && (
                                    <span className="mx-2 text-slate-500">/</span>
                                )}
                            </React.Fragment>
                        ))}
                    </nav>
                )}

                {/* Page Header */}
                {(title || subtitle) && (
                    <div className="text-center mb-12">
                        {title && (
                            <h1 className="text-4xl font-bold text-white mb-4">{title}</h1>
                        )}
                        {subtitle && (
                            <p className="text-gray-300 text-lg">{subtitle}</p>
                        )}
                    </div>
                )}

                {/* Content */}
                {children}
            </div>
        </main>
    )
}

export default ContentLayout

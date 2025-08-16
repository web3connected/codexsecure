import React from 'react'

interface FullWidthLayoutProps {
    children: React.ReactNode
    className?: string
    includeBackground?: boolean
}

const FullWidthLayout: React.FC<FullWidthLayoutProps> = ({ 
    children, 
    className = "",
    includeBackground = true
}) => {
    const baseClasses = includeBackground 
        ? "min-h-screen bg-slate-900 text-white" 
        : "min-h-screen text-white"

    return (
        <div className={`${baseClasses} ${className}`}>
            {children}
        </div>
    )
}

export default FullWidthLayout

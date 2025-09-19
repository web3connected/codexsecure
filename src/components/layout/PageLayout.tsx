import React from 'react'
import MainHeader from '@/components/common/MainHeader'
import MainFooter from '@/components/common/MainFooter'

interface PageLayoutProps {
    children: React.ReactNode
    className?: string
    includeHeader?: boolean
    includeFooter?: boolean
}

const PageLayout: React.FC<PageLayoutProps> = ({ 
    children, 
    className = "min-h-screen bg-slate-900", 
    includeHeader = false,
    includeFooter = false 
}) => {
    return (
        <div className={className}>
            {includeHeader && <MainHeader />}
            <main className="flex-1">
                {children}
            </main>
            {includeFooter && <MainFooter />}
        </div>
    )
}

export default PageLayout

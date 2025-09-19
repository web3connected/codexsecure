import Link from 'next/link'
import React from 'react'
import WhatIsQuantumHashing from './WhatIsQuantumHashing'
import LawOfInversion from './LawOfInversion'
import ComparisonTable from './ComparisonTable'
import TechnicalImplementation from './TechnicalImplementation'
import CodexHashCallToAction from './CodexHashCallToAction'

const MainQuantumHashingContent = () => {
    return (
        <section className="py-16 px-6 ">
            <div className="max-w-6xl mx-auto">
                <div className="grid lg:grid-cols-2 gap-16 mb-20">
                    {/* WhatIsQuantumHashing */}
                    <WhatIsQuantumHashing />

                    {/* LawOfInversion */}
                    <LawOfInversion />
                </div>
            </div>

            {/* ComparisonTable */}
            <div className='bg-slate-300/50 p-20 rounded-lg border border-slate-700'>
                <ComparisonTable />
            </div>

            {/* TechnicalImplementation */}
            <TechnicalImplementation />

            {/* CallToAction */}
            <CodexHashCallToAction />        
        </section>
    )
}

export default MainQuantumHashingContent
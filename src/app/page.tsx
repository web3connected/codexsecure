import PageLayout from '@/components/layout/PageLayout'

import CodexHashHero from '@/components/codexhash/CodexHashHero'
import CorePrinciples from '@/components/codexhash/CorePrinciples'
import WhatIsQuantumHashing from '@/components/codexhash/WhatIsQuantumHashing'


export default function Home() {
  return (
    <PageLayout>
      <CodexHashHero />
      <WhatIsQuantumHashing />
      <CorePrinciples />
    
    </PageLayout>
  )
}

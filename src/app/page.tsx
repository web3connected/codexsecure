import PageLayout from '@/components/layout/PageLayout'

import CodexHashHero from '@/components/codexhash/CodexHashHero'
import TechnicalImplementation from '@/components/codexhash/TechnicalImplementation'
import CorePrinciples from '@/components/codexhash/CorePrinciples'


export default function Home() {
  return (
    <PageLayout>
      <CodexHashHero />

      <CorePrinciples />
    
    </PageLayout>
  )
}

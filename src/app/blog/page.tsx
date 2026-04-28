import Link from 'next/link';

export const metadata = {
  title: 'Blog | CodexSecure',
  description: 'Insights on zone-based security, route analysis, threat models, and distributed systems from the CodexSecure team.',
};

export default function BlogPage() {
  return (
    <main className="min-h-screen bg-secure-bg flex items-center justify-center px-6">
      <div className="max-w-2xl w-full text-center py-32">

        {/* Badge */}
        <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-secure-primary/30 bg-secure-primary/10 mb-10">
          <span className="w-2 h-2 rounded-full bg-secure-secondary animate-pulse" />
          <span className="text-secure-secondary text-xs font-semibold uppercase tracking-widest">
            Coming Soon
          </span>
        </div>

        {/* Heading */}
        <h1 className="text-5xl md:text-6xl font-brand font-black text-white mb-6 leading-tight">
          The{' '}
          <span className="text-transparent bg-clip-text bg-gradient-to-r from-secure-secondary to-secure-accent">
            CodexSecure
          </span>{' '}
          Blog
        </h1>

        {/* Subtext */}
        <p className="text-foreground/60 text-lg md:text-xl leading-relaxed mb-4 max-w-lg mx-auto">
          Deep dives on zone-based security, phase locking, route fingerprinting, and what it actually takes to protect distributed systems at scale.
        </p>
        <p className="text-foreground/40 text-sm mb-12">
          We&apos;re writing. Check back soon.
        </p>

        {/* Topic previews */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-14 text-left">
          {[
            { zone: 'Z1–Z4', topic: 'Public route security without overhead', color: 'border-secure-secondary/30 text-secure-secondary' },
            { zone: 'Z5–Z8', topic: 'Device trust and session phase locking', color: 'border-secure-primary/30 text-secure-primary' },
            { zone: 'Z9–Z12', topic: 'Ledger-grade enforcement for private APIs', color: 'border-secure-accent/30 text-secure-accent' },
          ].map(({ zone, topic, color }) => (
            <div
              key={zone}
              className={`rounded-xl border bg-white/[0.03] px-5 py-4 ${color.split(' ')[0]}`}
            >
              <span className={`text-xs font-bold uppercase tracking-widest ${color.split(' ')[1]} block mb-2`}>
                {zone}
              </span>
              <p className="text-foreground/70 text-sm leading-snug">{topic}</p>
            </div>
          ))}
        </div>

        {/* CTA links */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link
            href="/docs"
            className="px-6 py-3 rounded-lg bg-secure-primary text-white font-semibold text-sm hover:bg-secure-primary/90 transition-colors"
          >
            Read the Docs
          </Link>
          <Link
            href="/"
            className="px-6 py-3 rounded-lg border border-white/10 text-foreground/60 font-semibold text-sm hover:text-white hover:border-white/30 transition-colors"
          >
            Back to Home
          </Link>
        </div>
      </div>
    </main>
  );
}

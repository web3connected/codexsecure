"use client";

export default function CodexSecureFooter() {
  return (
    <footer className="bg-slate-900 border-t border-slate-800">
      <div className="container mx-auto px-4 py-12">
        <div className="grid md:grid-cols-4 gap-8">
          <div>
            <h3 className="text-white font-semibold mb-4">Product</h3>
            <ul className="space-y-2">
              <li><a href="/zones" className="text-slate-400 hover:text-purple-400 transition-colors">Security Zones</a></li>
              <li><a href="/pricing" className="text-slate-400 hover:text-purple-400 transition-colors">Pricing</a></li>
              <li><a href="/docs" className="text-slate-400 hover:text-purple-400 transition-colors">Documentation</a></li>
            </ul>
          </div>
          <div>
            <h3 className="text-white font-semibold mb-4">Company</h3>
            <ul className="space-y-2">
              <li><a href="/about" className="text-slate-400 hover:text-purple-400 transition-colors">About</a></li>
              <li><a href="/support" className="text-slate-400 hover:text-purple-400 transition-colors">Support</a></li>
            </ul>
          </div>
          <div>
            <h3 className="text-white font-semibold mb-4">Legal</h3>
            <ul className="space-y-2">
              <li><a href="/privacy" className="text-slate-400 hover:text-purple-400 transition-colors">Privacy</a></li>
              <li><a href="/terms" className="text-slate-400 hover:text-purple-400 transition-colors">Terms</a></li>
              <li><a href="/security" className="text-slate-400 hover:text-purple-400 transition-colors">Security</a></li>
            </ul>
          </div>
          <div>
            <h3 className="text-white font-semibold mb-4">Connect</h3>
            <p className="text-slate-400 text-sm mb-4">
              Enterprise security infrastructure
            </p>
          </div>
        </div>
        <div className="border-t border-slate-800 mt-8 pt-8 text-center text-slate-400 text-sm">
          <p>© {new Date().getFullYear()} CodexSecure. All rights reserved.</p>
        </div>
      </div>
    </footer>
  );
}

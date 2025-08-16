import type { Metadata } from "next";
import { ThemeProvider } from "@/contexts/ThemeContext";
import { LoadingProvider } from "@/contexts/LoadingContext";
import Script from "next/script";
import "./globals.css";

export const metadata: Metadata = {
  title: "Web3 Codex - Professional Blockchain Solutions",
  description: "Professional blockchain solutions, wallet connections, and DeFi integrations tailored for government and Fortune 500 companies.",
  icons: {
    icon: "/assets/icons/codexIcon.png",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" data-theme="dark" style={{colorScheme: "dark"}}>
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <link
          rel="shortcut icon"
          href="/assets/icons/codexIcon.png"
          type="image/x-icon"
        />
        
        {/* Phosphor Icons */}
        <link
          rel="stylesheet"
          type="text/css"
          href="https://cdn.jsdelivr.net/npm/@phosphor-icons/web@2.1.1/src/regular/style.css"
        />
        <link
          rel="stylesheet"
          type="text/css"
          href="https://cdn.jsdelivr.net/npm/@phosphor-icons/web@2.1.1/src/fill/style.css"
        />
        <link
          rel="stylesheet"
          type="text/css"
          href="https://cdn.jsdelivr.net/npm/@phosphor-icons/web@2.1.1/src/thin/style.css"
        />
      </head>
      <body className="dark bg-slate-900 text-white font-business">
        <ThemeProvider>
          <LoadingProvider>
            {children}
          </LoadingProvider>
        </ThemeProvider>
        
        {/* Only load scripts that actually exist */}
        <Script src="/assets/js/index.js" strategy="beforeInteractive" />
      </body>
    </html>
  );
}
  
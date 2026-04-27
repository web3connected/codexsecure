import type { Metadata } from "next";
import { Inter, Poppins, JetBrains_Mono, Orbitron } from "next/font/google";
import CodexSecureDataLayer from "./CodexSecureDataLayer";
import "./globals.css";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

const poppins = Poppins({
  subsets: ["latin"],
  weight: ["300", "400", "500", "600", "700"],
  variable: "--font-poppins",
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-jetbrains",
  display: "swap",
});

const orbitron = Orbitron({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "900"],
  variable: "--font-orbitron",
  display: "swap",
});

export const metadata: Metadata = {
  title: "CodexSecure - Zone-Based Security Platform",
  description: "Intelligent zone-based security, route analysis, and threat detection for distributed systems and blockchain applications.",
  keywords: "security zones, threat detection, route analysis, blockchain security, distributed systems, web3",
  icons: {
    icon: "/assets/icons/codexIcon.png",
  },
  openGraph: {
    title: "CodexSecure - Zone-Based Security",
    description: "Intelligent zone-based security for distributed systems",
    type: "website",
    locale: "en_US",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      data-theme="dark"
      style={{ colorScheme: "dark" }}
      className={`${inter.variable} ${poppins.variable} ${jetbrainsMono.variable} ${orbitron.variable}`}
    >
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
      </head>
      <body className="bg-secure-bg text-foreground font-body antialiased">
        <CodexSecureDataLayer>
          {children}
        </CodexSecureDataLayer>
      </body>
    </html>
  );
}

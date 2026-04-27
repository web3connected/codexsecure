import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
    './src/lib/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      screens: {
        'sm': '576px',
        'md': '768px',
        'lg': '992px',
        'xl': '1200px',
        'xxl': '1400px',
        '3xl': '1600px',
        '4xl': '1800px',
      },
      fontFamily: {
        'sans':     ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        'business': ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        'heading':  ['Poppins', 'Inter', 'sans-serif'],
        'mono':     ['JetBrains Mono', 'monospace'],
        'body':     ['Inter', 'system-ui', 'sans-serif'],
        'brand':    ['Orbitron', 'Poppins', 'sans-serif'],
      },
      colors: {
        // CodexSecure Brand Colors
        'secure-primary':   '#7C3AED',              // Violet — trust / security
        'secure-secondary': '#2DF4A1',              // Teal — confirmation / safe state
        'secure-accent':    '#00E4FF',              // Cyan — highlights / links
        'secure-bg':        '#0A0F1E',              // Near-black navy — base background
        'secure-surface':   '#0F1729',              // Elevated surface
        'secure-border':    'rgba(124, 58, 237, 0.2)', // Violet tinted border

        // SDK panel alias — shared panels reference hash-primary / hash-bg
        'hash-primary':     '#7C3AED',
        'hash-secondary':   '#2DF4A1',
        'hash-bg':          '#0A0F1E',

        // Semantic aliases
        'foreground':       '#E2E8F0',
        'background':       '#0A0F1E',

        // Background Overlays
        'bg1': 'rgba(255, 255, 255, 0.04)',
        'bg2': 'rgba(255, 255, 255, 0.08)',
        'bg3': 'rgba(255, 255, 255, 0.16)',
        'bg4': 'rgba(255, 255, 255, 0.02)',
        'bg5': 'rgba(255, 255, 255, 0.12)',

        // Legacy template colors (kept for archived component compatibility)
        'g900': '#000F09',
        'g300': '#2DF4A1',
      },
      padding: {
        '15': '60px',
        '25': '100px',
        '30': '120px',
      },
      boxShadow: {
        'glow':    '0 0 20px rgba(124, 58, 237, 0.15)',
        'glow-lg': '0 0 30px rgba(124, 58, 237, 0.25)',
        'glow-xl': '0 0 40px rgba(124, 58, 237, 0.35)',
      },
      animation: {
        'glow-pulse': 'glowPulse 2s ease-in-out infinite alternate',
        'fade-in':    'fadeIn 0.6s ease-out',
      },
    },
  },
  plugins: [],
}

export default config

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
        'sans': ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        'business': ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        'heading': ['Poppins', 'Inter', 'sans-serif'],
        'mono': ['JetBrains Mono', 'monospace'],
      },
      colors: {
        // Site Theme Colors (from Indiz) Template
        'g900': '#000F09',
        'g700': '#1C2924', 
        'g600': '#2E3A35',
        'g500': '#3B4642',
        'g400': '#4A5550',
        'g300': '#2DF4A1',
        'g80': '#949A98',
        'g50': '#BFC3C2',
        'g40': '#DEE0DF',
        'g20': '#F5F5F5',
        
        // Web3 Brand Colors
        'web3-primary': '#0e4774ff',
        'web3-secondary': '#282a35ff',
        'web3-accent': '#FF6B35',
        
        // Background Overlays
        'bg1': 'rgba(255, 255, 255, 0.04)',
        'bg2': 'rgba(255, 255, 255, 0.08)',
        'bg3': 'rgba(255, 255, 255, 0.16)',
        'bg4': 'rgba(255, 255, 255, 0.02)',
        'bg5': 'rgba(255, 255, 255, 0.12)',
      },
      padding: {
        '15': '60px',
        '25': '100px', 
        '30': '120px',
      },
      boxShadow: {
        'glow': '0 0 20px rgba(45, 244, 161, 0.1)',
        'glow-lg': '0 0 30px rgba(45, 244, 161, 0.2)',
        'glow-xl': '0 0 40px rgba(45, 244, 161, 0.3)',
      },
      animation: {
        'glow-pulse': 'glowPulse 2s ease-in-out infinite alternate',
        'fade-in': 'fadeIn 0.6s ease-out',
      },
    },
  },
  plugins: [],
}

export default config

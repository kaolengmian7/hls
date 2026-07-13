/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,tsx,js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        pixel: ['"VT323"', 'monospace'],
        body: ['"Noto Sans SC"', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
        // Keep as aliases so font-serif / font-display still work
        serif: ['"VT323"', 'monospace'],
        display: ['"VT323"', 'monospace'],
      },
      colors: {
        sdv: {
          bg: '#1a2e20',
          'panel-deep': '#3a5030',
          'panel-warm': '#c8a870',
          'panel-title': '#a07040',
          input: '#f0dfa0',
          border: '#5c3d24',
          'border-light': '#e8c898',
          'border-mid': '#a07840',
          gold: '#f0e080',
          'gold-dark': '#c89020',
          'btn-green': '#7ab848',
          'btn-green-dark': '#4a8828',
          'btn-green-border': '#2a5010',
          'btn-brown': '#8a6030',
          'btn-brown-dark': '#5c3d1a',
          'btn-brown-border': '#3a2010',
          'btn-red': '#c04030',
          'btn-red-dark': '#7a2018',
          'btn-red-border': '#4a1808',
          shadow: '#0a0806',
          'text-dark': '#3a200e',
          'text-light': '#f0e8c0',
          'text-nav': '#c8d8a0',
          'text-muted': '#7a5a38',
          active: '#e8d060',
          'active-border': '#c89020',
          grid: 'rgba(90,140,60,0.08)',
        },
      },
      boxShadow: {
        'sdv-panel': 'inset 0 0 0 2px #e8c898, inset 0 0 0 4px #a07840, 4px 4px 0 #0a0806',
        'sdv-panel-deep': 'inset 0 0 0 2px #2a3820, 4px 4px 0 #0a0806',
        'sdv-btn': 'inset 0 1px 0 rgba(255,255,255,0.25), inset 0 -2px 0 rgba(0,0,0,0.3), 3px 3px 0 #0a0806',
        'sdv-input': 'inset 2px 2px 0 #a07840, inset -1px -1px 0 #e8c898',
        'sdv-input-focus': 'inset 2px 2px 0 #a07840, inset -1px -1px 0 #e8c898, 0 0 0 2px #f0c840',
      },
      animation: {
        'pixel-fade': 'pixel-fade 0.3s ease-out',
        'pixel-slide': 'pixel-slide 0.25s ease-out',
        'pixel-pulse': 'pixel-pulse 1.4s ease-in-out infinite',
      },
      keyframes: {
        'pixel-fade': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'pixel-slide': {
          '0%': { transform: 'translateY(8px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        'pixel-pulse': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.55' },
        },
      },
    },
  },
  plugins: [],
};

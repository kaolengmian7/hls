/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,tsx,js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        pixel: ['"VT323"', 'monospace'],
        body: ['"Noto Sans SC"', '"PingFang SC"', '"Microsoft YaHei"', 'sans-serif'],
        serif: ['"VT323"', 'monospace'],
        display: ['"VT323"', 'monospace'],
      },
      colors: {
        hls: {
          // ── Background & Surface ──
          bg:               '#faf6ed',
          surface:          '#fffef9',
          'surface-active': '#fffdf0',
          'surface-nav':    '#a07040',
          // backward-compat aliases (used via batch-renamed Tailwind utilities)
          'panel-warm':     '#fffef9',
          'panel-deep':     '#a07040',
          'panel-title':    '#a07040',
          // ── Border ──
          border:           '#d4b870',
          'border-light':   '#ffffff',
          'border-mid':     '#c8a020',
          'border-active':  '#c8a020',
          'border-strong':  '#5c3d24',
          'border-subtle':  'rgba(160,112,64,0.3)',
          // ── Gold accent ──
          gold:             '#b88820',
          'gold-dark':      '#c89020',
          'gold-bright':    '#c89828',
          // ── Action buttons ──
          'btn-green':        '#5a8828',
          'btn-green-dark':   '#4a7820',
          'btn-green-border': '#2a5010',
          'btn-brown':        '#a07040',
          'btn-brown-dark':   '#7a5030',
          'btn-brown-border': '#5c3d24',
          'btn-red':          '#c04030',
          'btn-red-dark':     '#7a2018',
          'btn-red-border':   '#4a1808',
          // ── Text ──
          text:             '#2a1408',
          'text-dark':      '#2a1408',
          'text-light':     '#5c3820',   // was cream-on-dark; now secondary brown
          'text-secondary': '#5c3820',
          'text-nav':       '#9a7040',   // was green-on-dark; now muted brown
          'text-muted':     '#9a7040',
          'text-inverse':   '#fff8e8',
          // ── Misc ──
          shadow:           'rgba(60,30,8,0.12)',
          action:           '#5a8828',
          'action-border':  '#2a5010',
          danger:           '#c04030',
          active:           '#fffdf0',
          'active-border':  '#c8a020',
          grid:             'transparent',
        },
      },
      boxShadow: {
        'hls-panel':       'inset 0 1px 0 #ffffff, 3px 3px 0 rgba(60,30,8,0.12)',
        'hls-btn':         'inset 0 1px 0 rgba(255,255,255,0.2), inset 0 -2px 0 rgba(0,0,0,0.15), 3px 3px 0 rgba(60,30,8,0.12)',
        'hls-input':       'inset 1px 1px 0 #e8d8b0',
        'hls-input-focus': 'inset 1px 1px 0 #e8d8b0, 0 0 0 2px #c8a020',
      },
      animation: {
        'pixel-fade':  'pixel-fade 0.3s ease-out',
        'pixel-slide': 'pixel-slide 0.25s ease-out',
        'pixel-pulse': 'pixel-pulse 1.4s ease-in-out infinite',
      },
      keyframes: {
        'pixel-fade':  { '0%': { opacity: '0' }, '100%': { opacity: '1' } },
        'pixel-slide': { '0%': { transform: 'translateY(8px)', opacity: '0' }, '100%': { transform: 'translateY(0)', opacity: '1' } },
        'pixel-pulse': { '0%, 100%': { opacity: '1' }, '50%': { opacity: '0.55' } },
      },
    },
  },
  plugins: [],
};

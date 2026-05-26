/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts,tsx,js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        display: ['"Cinzel Decorative"', 'Cinzel', '"Noto Serif SC"', 'serif'],
        serif: ['Cinzel', 'Cardo', '"Noto Serif SC"', 'serif'],
        body: ['"Cormorant Garamond"', '"IM Fell English"', '"Noto Serif SC"', 'Georgia', 'serif'],
        rune: ['"Uncial Antiqua"', '"Cinzel Decorative"', 'serif'],
      },
      colors: {
        // 羊皮纸：从亮米白到深沉发黄
        parchment: {
          50: '#fbf3df',
          100: '#f4e6c2',
          200: '#e9d3a0',
          300: '#d8b87a',
          400: '#bf975a',
          500: '#9a7644',
          600: '#6f5430',
        },
        // 锻金：从烛火光到暗铜绿
        gold: {
          200: '#f3e2a4',
          300: '#e8c87c',
          400: '#d4af37',
          500: '#b8860b',
          600: '#8c6708',
          700: '#5e4505',
        },
        // 墨色 / 暗夜石：背景与重内容
        ink: {
          400: '#3a2b1f',
          500: '#251913',
          600: '#1a0f0a',
          700: '#100806',
          800: '#080403',
        },
        // 牛血酒红：危险与王权
        oxblood: {
          300: '#c13a3a',
          400: '#a02323',
          500: '#7f1d1d',
          600: '#5e1212',
          700: '#3d0a0a',
        },
        // 古林深绿：撤离 / 准备 / 成功
        forest: {
          300: '#88a04f',
          400: '#5a7a32',
          500: '#3f5a1f',
          600: '#2a3e14',
          700: '#1a2a0a',
        },
        // 王室紫：稀有 / 神秘
        royal: {
          400: '#7c3aed',
          500: '#6d28d9',
          600: '#4c1d95',
          700: '#2e1065',
        },
        // 余烬橙：老板 / 抽奖
        ember: {
          300: '#fdba74',
          400: '#fb923c',
          500: '#f97316',
          600: '#c2410c',
          700: '#7c2d12',
        },
        // 古蓝：护航 / 法师
        sapphire: {
          300: '#93c5fd',
          400: '#3b82f6',
          500: '#1d4ed8',
          600: '#1e3a8a',
          700: '#172554',
        },
        boss: {
          50: '#fff7ed',
          500: '#f97316',
          600: '#ea580c',
          700: '#c2410c',
        },
        escort: {
          50: '#eff6ff',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
      },
      boxShadow: {
        relic: '0 6px 24px -8px rgba(0,0,0,.65), inset 0 1px 0 rgba(244,230,194,.06)',
        torch:
          '0 0 0 1px rgba(212,175,55,.35), 0 12px 40px -12px rgba(212,175,55,.25), inset 0 1px 0 rgba(244,230,194,.08)',
        emboss: 'inset 0 1px 0 rgba(255,235,180,.12), inset 0 -2px 0 rgba(0,0,0,.45)',
      },
      animation: {
        'fade-in': 'fade-in .35s ease-out',
        'slide-up': 'slide-up .4s ease-out',
        pulse_soft: 'pulse_soft 1.6s ease-in-out infinite',
        flicker: 'flicker 2.8s ease-in-out infinite',
        shimmer: 'shimmer 3.2s linear infinite',
        'rune-spin': 'rune-spin 24s linear infinite',
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-up': {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        pulse_soft: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '.55' },
        },
        flicker: {
          '0%, 18%, 22%, 25%, 53%, 57%, 100%': {
            opacity: '1',
            filter: 'drop-shadow(0 0 6px rgba(255,180,80,.55))',
          },
          '20%, 24%, 55%': {
            opacity: '.8',
            filter: 'drop-shadow(0 0 2px rgba(255,180,80,.25))',
          },
        },
        shimmer: {
          '0%': { 'background-position': '-200% 0' },
          '100%': { 'background-position': '200% 0' },
        },
        'rune-spin': {
          from: { transform: 'rotate(0deg)' },
          to: { transform: 'rotate(360deg)' },
        },
      },
    },
  },
  plugins: [],
};

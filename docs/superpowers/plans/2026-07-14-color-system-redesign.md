# Color System Redesign (HLS Light Theme) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the dark SDV pixel theme (`sdv-*` tokens) with a semantic light cream theme (`hls-*` tokens) across the entire codebase.

**Architecture:** Three-layer change — (1) define new `hls` color tokens in tailwind.config.js, (2) rewrite main.css with new `hls-*` component classes, (3) batch-rename all `sdv-` references to `hls-` in Vue files and index.html. Aliases handle the two merged classes (`panel-deep`, `panel-title-light`) so the batch rename is purely mechanical.

**Tech Stack:** Vue 3, Tailwind CSS v3, Vite, Noto Sans SC (Google Fonts)

---

### Task 1: Update index.html

**Files:**
- Modify: `index.html`

- [ ] **Step 1: Update meta theme-color and Google Fonts, fix body classes**

Replace the entire `<head>` section and body tag with:

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no" />
    <meta name="theme-color" content="#faf6ed" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
      href="https://fonts.googleapis.com/css2?family=VT323&family=Noto+Sans+SC:wght@400;500;700&display=swap"
      rel="stylesheet"
    />
    <title>花栗鼠的神秘商店</title>
  </head>
  <body class="bg-hls-bg text-hls-text">
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

Changes: `theme-color` `#1a2e20` → `#faf6ed`; Fonts add `wght@400;500;700` (adds 500 weight needed for buttons); body classes `bg-sdv-bg text-sdv-text-light` → `bg-hls-bg text-hls-text`.

- [ ] **Step 2: Commit**

```bash
git add index.html
git commit -m "refactor(theme): update index.html for hls light theme"
```

---

### Task 2: Update tailwind.config.js

**Files:**
- Modify: `tailwind.config.js`

- [ ] **Step 1: Replace colors and boxShadow with hls tokens**

Replace the entire file content with:

```js
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
```

- [ ] **Step 2: Commit**

```bash
git add tailwind.config.js
git commit -m "refactor(theme): replace sdv color tokens with hls light palette"
```

---

### Task 3: Rewrite src/shared/styles/main.css

**Files:**
- Modify: `src/shared/styles/main.css`

- [ ] **Step 1: Replace entire file with hls light theme component classes**

Replace the entire file with:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* =========================================================================
   HLS Light Theme
   ========================================================================= */

@layer base {
  html,
  body,
  #app {
    @apply h-full;
  }

  body {
    @apply font-body antialiased;
    color: #2a1408;
    background-color: #faf6ed;
    min-height: 100vh;
  }

  ::selection {
    background-color: rgba(184, 136, 32, 0.25);
    color: #2a1408;
  }

  ::-webkit-scrollbar { width: 10px; height: 10px; }
  ::-webkit-scrollbar-track { background: #e8d8b0; }
  ::-webkit-scrollbar-thumb {
    background: linear-gradient(180deg, #a07040, #7a5030);
    border-radius: 2px;
    border: 2px solid #e8d8b0;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: linear-gradient(180deg, #c89020, #a07040);
  }

  input, textarea { caret-color: #b88820; }

  h1, h2, h3, h4, h5 {
    font-family: 'VT323', monospace;
    color: #b88820;
    letter-spacing: 0.08em;
  }
}

@layer components {

  /* ---------- Panel ---------- */
  .hls-panel {
    background: #fffef9;
    border: 4px solid #d4b870;
    box-shadow: inset 0 1px 0 #ffffff, 4px 4px 0 rgba(60, 30, 8, 0.12);
    position: relative;
  }

  .hls-panel-deep {
    @apply hls-panel;
  }

  /* ---------- Panel title bar ---------- */
  .hls-panel-title {
    background: #a07040;
    border-bottom: 3px solid #5c3d24;
    box-shadow: inset 0 -2px 0 rgba(0, 0, 0, 0.15), inset 0 1px 0 rgba(255, 255, 255, 0.15);
    font-family: 'VT323', monospace;
    font-size: 1.25rem;
    letter-spacing: 0.12em;
    color: #fff8e8;
    padding: 12px 16px 8px;
  }

  .hls-panel-title-light {
    @apply hls-panel-title;
  }

  /* ---------- Buttons ---------- */
  .hls-btn {
    @apply inline-flex items-center justify-center gap-2 font-body font-medium tracking-wide
      px-5 py-2.5 transition-transform duration-75 active:translate-y-px
      disabled:cursor-not-allowed disabled:opacity-50
      focus:outline-none focus:ring-2 focus:ring-hls-gold/50 focus:ring-offset-2 focus:ring-offset-hls-bg;
    font-size: 0.9375rem;
    color: #f0f8e8;
    background: linear-gradient(180deg, #6a9838 0%, #5a8828 100%);
    border: 3px solid #2a5010;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.2), inset 0 -2px 0 rgba(0, 0, 0, 0.15), 3px 3px 0 rgba(60, 30, 8, 0.12);
    text-shadow: 1px 1px 0 #1a3808;
  }
  .hls-btn:hover:not(:disabled) {
    transform: translate(-1px, -1px);
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.2), inset 0 -2px 0 rgba(0, 0, 0, 0.15), 4px 4px 0 rgba(60, 30, 8, 0.12);
  }

  .hls-btn-secondary {
    @apply hls-btn;
    background: linear-gradient(180deg, #b08040 0%, #a07040 100%);
    border-color: #5c3d24;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.15), inset 0 -2px 0 rgba(0, 0, 0, 0.15), 3px 3px 0 rgba(60, 30, 8, 0.12);
    text-shadow: 1px 1px 0 rgba(0, 0, 0, 0.3);
    color: #fff8e8;
  }
  .hls-btn-secondary:hover:not(:disabled) {
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.15), inset 0 -2px 0 rgba(0, 0, 0, 0.15), 4px 4px 0 rgba(60, 30, 8, 0.12);
  }

  .hls-btn-danger {
    @apply hls-btn;
    background: linear-gradient(180deg, #c04030 0%, #7a2018 100%);
    border-color: #4a1808;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.15), inset 0 -2px 0 rgba(0, 0, 0, 0.15), 3px 3px 0 rgba(60, 30, 8, 0.12);
    text-shadow: 1px 1px 0 #200808;
    color: #ffe8e0;
  }
  .hls-btn-danger:hover:not(:disabled) {
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.15), inset 0 -2px 0 rgba(0, 0, 0, 0.15), 4px 4px 0 rgba(60, 30, 8, 0.12);
  }

  .hls-btn-ghost {
    @apply inline-flex items-center justify-center gap-2 font-body font-medium tracking-wide text-sm
      px-3 py-2 transition-colors duration-150
      disabled:cursor-not-allowed disabled:opacity-50
      focus:outline-none;
    color: #5c3820;
    border: 2px solid rgba(160, 112, 64, 0.4);
    background: transparent;
  }
  .hls-btn-ghost:hover:not(:disabled) {
    border-color: rgba(160, 112, 64, 0.7);
    color: #2a1408;
    background: rgba(212, 184, 112, 0.15);
  }

  /* ---------- Input ---------- */
  .hls-input {
    @apply w-full font-body text-base tracking-normal
      placeholder:text-hls-text-muted
      focus:outline-none transition-shadow;
    color: #2a1408;
    background: #fffef9;
    border: 2px solid #d4b870;
    box-shadow: inset 1px 1px 0 #e8d8b0;
    padding: 8px 12px;
  }
  .hls-input:focus {
    box-shadow: inset 1px 1px 0 #e8d8b0, 0 0 0 2px #c8a020;
  }

  /* ---------- Badge ---------- */
  .hls-badge {
    @apply inline-block font-body font-medium tracking-wide;
    font-size: 0.75rem;
    color: #f0f8e8;
    background: #5a8828;
    border: 2px solid #2a5010;
    box-shadow: 1px 1px 0 rgba(60, 30, 8, 0.15);
    padding: 2px 10px;
  }
  .hls-badge-gold  { background: #c89020; border-color: #7a5010; color: #fff8d0; }
  .hls-badge-gray  { background: #c8b090; border-color: #8a6838; color: #5c3820; }
  .hls-badge-red   { background: #a03020; border-color: #601808; color: #ffd8c8; }
  .hls-badge-teal  { background: #2a7060; border-color: #185040; color: #b0f0e0; }
  .hls-badge-amber { background: #c07018; border-color: #804808; color: #ffe8a0; }

  /* ---------- Nav header ---------- */
  .hls-nav {
    background: #a07040;
    border-bottom: 3px solid #5c3d24;
    box-shadow: 0 2px 0 rgba(60, 30, 8, 0.2), inset 0 1px 0 rgba(255, 255, 255, 0.1);
    padding: 10px 24px;
  }

  /* ---------- Menu row ---------- */
  .hls-menu-row {
    @apply flex items-center gap-3 transition-transform duration-75;
    min-height: 52px;
    padding: 10px 14px;
    background: #fffef9;
    border: 2px solid #d4b870;
    box-shadow: inset 0 1px 0 #ffffff, inset 0 -1px 0 #e0c888, 2px 2px 0 rgba(60, 30, 8, 0.12);
  }
  .hls-menu-row:hover:not([disabled]):not(.disabled) {
    transform: translateX(3px);
    background: #fffdf0;
  }
  .hls-menu-row.active {
    background: #fffdf0;
    border-color: #c8a020;
    box-shadow: inset 0 1px 0 #fffce8, inset 0 -1px 0 #c8a020, 2px 2px 0 rgba(60, 30, 8, 0.12);
  }
  .hls-menu-row.disabled,
  .hls-menu-row[disabled] {
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* ---------- Divider ---------- */
  .hls-divider {
    height: 3px;
    background: linear-gradient(90deg, transparent, #c8a870 20%, #c8a870 80%, transparent);
    margin: 8px 0;
  }
  .hls-divider-label {
    @apply flex items-center gap-3 font-body text-sm tracking-widest uppercase text-hls-text-muted;
  }
  .hls-divider-label::before,
  .hls-divider-label::after {
    content: '';
    flex: 1;
    height: 2px;
    background: linear-gradient(90deg, transparent, rgba(160, 112, 64, 0.4), transparent);
  }

  /* ---------- Toast ---------- */
  .hls-toast {
    @apply font-body tracking-wide;
    color: #fff8e8;
    border: 2px solid rgba(160, 112, 64, 0.4);
    box-shadow: 2px 2px 0 rgba(60, 30, 8, 0.15);
    padding: 8px 16px;
    font-size: 0.9rem;
  }
  .hls-toast--success { background: linear-gradient(180deg, #5a8c3a 0%, #3a6020 100%); }
  .hls-toast--error   { background: linear-gradient(180deg, #a03020 0%, #601808 100%); }
  .hls-toast--warn    { background: linear-gradient(180deg, #b07018 0%, #7a4808 100%); }
  .hls-toast--info    { background: linear-gradient(180deg, #2a4838 0%, #1a3020 100%); }

  /* ---------- Coin tag ---------- */
  .coin-tag {
    font-family: 'VT323', monospace;
    color: #b88820;
    font-size: 1rem;
  }

  /* ---------- Stat panel ---------- */
  .hls-stat {
    background: #fffef9;
    border: 2px solid #d4b870;
    box-shadow: inset 0 1px 0 #ffffff, 2px 2px 0 rgba(60, 30, 8, 0.12);
    padding: 10px 14px;
  }

}

@layer utilities {
  .font-pixel { font-family: 'VT323', monospace; }
  .text-shadow-pixel { text-shadow: 1px 1px 0 rgba(0, 0, 0, 0.4); }
  .border-pixel { image-rendering: pixelated; }
}
```

- [ ] **Step 2: Commit**

```bash
git add src/shared/styles/main.css
git commit -m "refactor(theme): rewrite main.css with hls light component classes"
```

---

### Task 4: Batch rename sdv- → hls- in all Vue files

**Files:**
- Modify: all `*.vue` files under `src/`

This single sed pass renames every `sdv-` occurrence (component classes and Tailwind utilities) to `hls-`. The aliases in main.css and tailwind.config.js ensure that structurally-renamed tokens (`panel-deep`, `panel-title-light`, `text-light`, `text-nav`) still resolve correctly.

- [ ] **Step 1: Run the rename**

```bash
find /Users/kratos/hls/src -name "*.vue" -exec sed -i '' 's/sdv-/hls-/g' {} +
```

- [ ] **Step 2: Verify the rename succeeded**

```bash
grep -r "sdv-" /Users/kratos/hls/src --include="*.vue" | wc -l
```

Expected output: `0`

If non-zero, run to see what remains:

```bash
grep -rn "sdv-" /Users/kratos/hls/src --include="*.vue"
```

Fix any residual manually.

- [ ] **Step 3: Commit**

```bash
git add src/
git commit -m "refactor(theme): batch rename sdv- to hls- in all Vue files"
```

---

### Task 5: Visual verification and hardcoded color fixes

**Files:**
- Modify: any Vue file with hardcoded dark hex colors

- [ ] **Step 1: Start dev server**

```bash
npm run dev
```

Open http://localhost:5173 in browser.

- [ ] **Step 2: Check each page in order**

For each page, look for: dark backgrounds surviving on the page, illegible text (cream/white text on cream background), mismatched border colors.

| URL | What to look for |
|---|---|
| `/` (LandingPage) | Cream background, brown nav, gold title, cream cards with gold borders |
| `/login` | Panel should be `#fffef9` with brown title bar |
| `/profile` | Panels light, headings gold |
| `/admin` | Table rows light, panel titles brown |
| `/shop` (ShopHomePage) | Light cards, brown/gold accents |
| `/shop/room/:id` (RoomPage) | Player slots light |
| `/shop/play/:id` (PlayPage) | Lottery panel light |

- [ ] **Step 3: Fix hardcoded dark colors**

Search for remaining dark hex values that weren't covered by the rename:

```bash
grep -rn "#1a2e20\|#3a5030\|#0d1c10\|#2a3820\|#1a2810\|#1a2808" /Users/kratos/hls/src --include="*.vue"
```

For each match, replace with the semantic equivalent:
- `#1a2e20` (old body bg) → `#faf6ed`
- `#3a5030` (old deep panel) → `#a07040` or `#fffef9` depending on context
- `#0d1c10` (old scrollbar track) → `#e8d8b0`
- `#2a3820`, `#1a2810`, `#1a2808` (old dark green variants) → `#faf6ed`

- [ ] **Step 4: Fix hardcoded light-on-dark text colors**

Search for cream/light text colors that were designed for dark backgrounds:

```bash
grep -rn "#f0e8c0\|#c8d8a0\|#f0e080\|rgba(255,255,255,0.06)" /Users/kratos/hls/src --include="*.vue"
```

For each match, replace with readable equivalents:
- `#f0e8c0` (old cream text) → `#2a1408` or `#5c3820`
- `#c8d8a0` (old green nav text) → `#9a7040`
- `#f0e080` (old bright gold) → `#b88820`
- `rgba(255,255,255,0.06)` (faint white inset) → remove or replace with `rgba(0,0,0,0.04)`

- [ ] **Step 5: Commit fixes**

```bash
git add src/
git commit -m "fix(theme): replace hardcoded dark colors with hls light equivalents"
```

---

### Task 6: Final type-check and cleanup

**Files:**
- None (verification only)

- [ ] **Step 1: Run type check**

```bash
npm run typecheck
```

Expected: no errors. (Token renames are CSS class strings, not TypeScript types, so no type errors are expected.)

- [ ] **Step 2: Final visual pass**

With dev server running, navigate through:
- The full shop game flow (ShopHomePage → RoomPage → PlayPage)
- ConfirmDialog (trigger any delete/confirm action)
- SliderCaptcha (go to login page)
- Toast messages (any action that triggers a toast in the shop)

Confirm all components render with the cream background and no leftover dark surfaces.

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "feat(theme): complete hls light color system migration"
```

# 全站像素风 UI 改版（星露谷物语风格）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将全站 UI 从 RPG 羊皮纸风格统一改为星露谷物语像素游戏风格，覆盖所有页面与共享组件。

**Architecture:** 全局替换策略——更新 `main.css`（SDV 全局变量 + 组件类）和 `tailwind.config.js`（SDV 色板 + VT323 字体），然后逐文件替换 `rpg-*`/`card`/`btn-*`/`parchment-*`/`gold-*` 等旧类名为新 `sdv-*` 类名。无需路由隔离。

**Tech Stack:** Vue 3, Tailwind CSS v3, VT323 pixel font (Google Fonts), Noto Sans SC

---

## 文件结构

| 文件 | 操作 |
|------|------|
| `index.html` | 替换字体链接，更新 theme-color，更新 body class |
| `tailwind.config.js` | 新增 SDV 色板，更新字体族 |
| `src/shared/styles/main.css` | 完全替换为 SDV 主题 |
| `src/shared/components/AppHeader.vue` | SDV 导航栏样式 |
| `src/shared/components/AuthWidget.vue` | SDV 按钮/下拉菜单样式 |
| `src/shared/components/ConfirmDialog.vue` | SDV 面板/按钮样式 |
| `src/shared/components/Avatar.vue` | SDV 头像边框 |
| `src/pages/LandingPage.vue` | SDV 首页，菜单行游戏卡片 |
| `src/pages/LoginPage.vue` | SDV 暖面板表单 |
| `src/pages/ProfilePage.vue` | SDV 设置面板 |
| `src/pages/AdminPage.vue` | SDV 表格/弹窗 |
| `src/pages/AuthCallbackPage.vue` | SDV 加载状态 |
| `src/games/teach/pages/TeachPage.vue` | SDV 占位页 |
| `src/games/audition/pages/AuditionPage.vue` | SDV 占位页 |
| `src/games/poker/pages/PokerPage.vue` | SDV 占位页 |
| `src/games/shop/ShopLayout.vue` | 移除图片背景，继承全局 SDV bg |
| `src/games/shop/pages/ShopHomePage.vue` | SDV 面板 + 输入框 |
| `src/games/shop/pages/RoomPage.vue` | SDV 角色卡片/面板 |
| `src/games/shop/pages/PlayPage.vue` | SDV 游戏面板 |
| `src/games/shop/components/PlayerSlot.vue` | SDV 玩家槽 |
| `src/games/shop/components/ToastContainer.vue` | SDV toast 样式 |
| `src/games/shop/components/FeedStream.vue` | SDV 消息行 |
| `src/games/shop/components/EquipmentShop.vue` | SDV 装备列表 |
| `src/games/shop/components/LotteryPanel.vue` | SDV 抽奖面板 |
| `src/games/shop/components/LotteryRecordModal.vue` | SDV 模态框 |
| `src/games/shop/components/SettlementModal.vue` | SDV 结算模态框 |
| `src/games/shop/components/SettlementRecordModal.vue` | SDV 模态框 |

---

## Task 1: Foundation — 字体、全局 CSS、Tailwind 配置

**Files:**
- Modify: `index.html`
- Modify: `tailwind.config.js`
- Modify: `src/shared/styles/main.css`

- [ ] **Step 1: 更新 `index.html`**

替换整个文件内容：

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no" />
    <meta name="theme-color" content="#1a2e20" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
      href="https://fonts.googleapis.com/css2?family=VT323&family=Noto+Sans+SC:wght@400;700&display=swap"
      rel="stylesheet"
    />
    <title>花栗鼠的神秘商店</title>
  </head>
  <body class="bg-sdv-bg text-sdv-text-light">
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 2: 更新 `tailwind.config.js`**

替换整个文件内容：

```js
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
```

- [ ] **Step 3: 替换 `src/shared/styles/main.css`**

替换整个文件内容：

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* =========================================================================
   SDV / Stardew Valley Pixel Theme
   ========================================================================= */

@layer base {
  html,
  body,
  #app {
    @apply h-full;
  }

  body {
    @apply font-body antialiased text-sdv-text-light;
    background-color: #1a2e20;
    background-image:
      repeating-linear-gradient(0deg, transparent, transparent 23px, rgba(90,140,60,0.08) 23px, rgba(90,140,60,0.08) 24px),
      repeating-linear-gradient(90deg, transparent, transparent 23px, rgba(90,140,60,0.08) 23px, rgba(90,140,60,0.08) 24px),
      radial-gradient(ellipse at 35% 0%, rgba(60,120,60,0.2) 0%, transparent 55%);
    background-attachment: fixed;
    background-size: 24px 24px, 24px 24px, 100% 100%;
    min-height: 100vh;
  }

  ::selection {
    background-color: rgba(122, 184, 72, 0.4);
    color: #f0e8c0;
  }

  ::-webkit-scrollbar { width: 10px; height: 10px; }
  ::-webkit-scrollbar-track { background: #0d1c10; }
  ::-webkit-scrollbar-thumb {
    background: linear-gradient(180deg, #5c3d24, #3a2010);
    border-radius: 2px;
    border: 2px solid #0d1c10;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: linear-gradient(180deg, #8a6030, #5c3d24);
  }

  input, textarea { caret-color: #f0e080; }

  h1, h2, h3, h4, h5 {
    font-family: 'VT323', monospace;
    color: #f0e8c0;
    letter-spacing: 0.08em;
  }
}

@layer components {

  /* ---------- Panel: warm wood ---------- */
  .sdv-panel {
    background: #c8a870;
    border: 4px solid #5c3d24;
    box-shadow: inset 0 0 0 2px #e8c898, inset 0 0 0 4px #a07840, 4px 4px 0 #0a0806;
    position: relative;
  }

  /* ---------- Panel: deep forest ---------- */
  .sdv-panel-deep {
    background: #3a5030;
    border: 4px solid #2a3820;
    box-shadow: inset 0 0 0 2px rgba(255,255,255,0.06), 4px 4px 0 #0a0806;
    position: relative;
  }

  /* ---------- Panel title bar ---------- */
  .sdv-panel-title {
    background: #a07040;
    border-bottom: 3px solid #5c3d24;
    box-shadow: inset 0 -2px 0 #3a2010, inset 0 1px 0 #e8c080;
    font-family: 'VT323', monospace;
    font-size: 1.25rem;
    letter-spacing: 0.12em;
    color: #3a200e;
    text-shadow: 1px 1px 0 #e8c080;
    padding: 12px 16px 8px;
  }

  .sdv-panel-title-light {
    background: #3a5030;
    border-bottom: 3px solid #2a3820;
    box-shadow: inset 0 -2px 0 #1a2810, inset 0 1px 0 rgba(255,240,180,0.15);
    font-family: 'VT323', monospace;
    font-size: 1.25rem;
    letter-spacing: 0.12em;
    color: #f0e8c0;
    text-shadow: 1px 1px 0 #1a2808;
    padding: 12px 16px 8px;
  }

  /* ---------- Buttons ---------- */
  .sdv-btn {
    @apply inline-flex items-center justify-center gap-2 font-pixel tracking-widest uppercase
      px-5 py-2.5 transition-transform duration-75 active:translate-y-px
      disabled:cursor-not-allowed disabled:opacity-50
      focus:outline-none focus:ring-2 focus:ring-sdv-gold/50 focus:ring-offset-2 focus:ring-offset-sdv-bg;
    font-size: 1.1rem;
    color: #f4f0d0;
    background: linear-gradient(180deg, #7ab848 0%, #4a8828 100%);
    border: 3px solid #2a5010;
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.25), inset 0 -2px 0 rgba(0,0,0,0.3), 3px 3px 0 #0a0806;
    text-shadow: 1px 1px 0 #1a3808;
  }
  .sdv-btn:hover:not(:disabled) {
    transform: translate(-1px, -1px);
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.25), inset 0 -2px 0 rgba(0,0,0,0.3), 4px 4px 0 #0a0806;
  }

  .sdv-btn-secondary {
    @apply sdv-btn;
    background: linear-gradient(180deg, #8a6030 0%, #5c3d1a 100%);
    border-color: #3a2010;
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.15), inset 0 -2px 0 rgba(0,0,0,0.3), 3px 3px 0 #0a0806;
    text-shadow: 1px 1px 0 #1a0a00;
  }
  .sdv-btn-secondary:hover:not(:disabled) {
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.15), inset 0 -2px 0 rgba(0,0,0,0.3), 4px 4px 0 #0a0806;
  }

  .sdv-btn-danger {
    @apply sdv-btn;
    background: linear-gradient(180deg, #c04030 0%, #7a2018 100%);
    border-color: #4a1808;
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.15), inset 0 -2px 0 rgba(0,0,0,0.3), 3px 3px 0 #0a0806;
    text-shadow: 1px 1px 0 #200808;
  }
  .sdv-btn-danger:hover:not(:disabled) {
    box-shadow: inset 0 1px 0 rgba(255,255,255,0.15), inset 0 -2px 0 rgba(0,0,0,0.3), 4px 4px 0 #0a0806;
  }

  .sdv-btn-ghost {
    @apply inline-flex items-center justify-center gap-2 font-pixel tracking-wider uppercase text-sm
      px-3 py-2 transition-colors duration-150
      disabled:cursor-not-allowed disabled:opacity-50
      focus:outline-none;
    color: #c8d8a0;
    border: 2px solid rgba(90,140,60,0.4);
    background: rgba(26,46,32,0.6);
  }
  .sdv-btn-ghost:hover:not(:disabled) {
    border-color: rgba(122,184,72,0.7);
    color: #f0e8c0;
    background: rgba(58,80,48,0.7);
  }

  /* ---------- Input ---------- */
  .sdv-input {
    @apply w-full font-pixel text-lg tracking-wide
      placeholder:text-sdv-text-muted
      focus:outline-none transition-shadow;
    color: #2a1408;
    background: #f0dfa0;
    border: 3px solid #5c3d24;
    box-shadow: inset 2px 2px 0 #a07840, inset -1px -1px 0 #e8c898;
    padding: 8px 12px;
  }
  .sdv-input:focus {
    box-shadow: inset 2px 2px 0 #a07840, inset -1px -1px 0 #e8c898, 0 0 0 2px #f0c840;
  }

  /* ---------- Badge ---------- */
  .sdv-badge {
    @apply inline-block font-pixel tracking-wider uppercase;
    font-size: 0.85rem;
    color: #f0f4d0;
    background: #7ab848;
    border: 2px solid #2a5010;
    box-shadow: 1px 1px 0 #0a0806;
    padding: 2px 10px;
  }
  .sdv-badge-gold {
    background: #c89020;
    border-color: #7a5010;
    color: #fff8d0;
  }
  .sdv-badge-gray {
    background: #6a5030;
    border-color: #3a2810;
    color: #a88860;
  }
  .sdv-badge-red {
    background: #a03020;
    border-color: #601808;
    color: #ffd8c8;
  }
  .sdv-badge-teal {
    background: #2a7060;
    border-color: #185040;
    color: #b0f0e0;
  }
  .sdv-badge-amber {
    background: #c07018;
    border-color: #804808;
    color: #ffe8a0;
  }

  /* ---------- Nav header ---------- */
  .sdv-nav {
    background: #3a5030;
    border-bottom: 4px solid #2a3820;
    box-shadow: 0 3px 0 #1a2810, inset 0 1px 0 rgba(255,255,255,0.06);
    padding: 10px 24px;
  }

  /* ---------- Menu row ---------- */
  .sdv-menu-row {
    @apply flex items-center gap-3 transition-transform duration-75;
    min-height: 52px;
    padding: 10px 14px;
    background: #d4b878;
    border: 2px solid rgba(122,80,48,0.65);
    box-shadow: inset 0 1px 0 #e8c888, inset 0 -1px 0 #9a6838, 2px 2px 0 #1a1208;
  }
  .sdv-menu-row:hover:not([disabled]) {
    transform: translateX(3px);
    background: #e0c880;
  }
  .sdv-menu-row.active {
    background: #e8d060;
    border-color: #c89020;
    box-shadow: inset 0 1px 0 #ffe880, inset 0 -1px 0 #a87020, 2px 2px 0 #1a1208;
  }
  .sdv-menu-row.disabled,
  .sdv-menu-row[disabled] {
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* ---------- Divider ---------- */
  .sdv-divider {
    height: 3px;
    background: linear-gradient(90deg, transparent, #7a5030 20%, #7a5030 80%, transparent);
    margin: 8px 0;
  }
  .sdv-divider-label {
    @apply flex items-center gap-3 font-pixel text-sm tracking-widest uppercase text-sdv-text-muted;
  }
  .sdv-divider-label::before,
  .sdv-divider-label::after {
    content: '';
    flex: 1;
    height: 2px;
    background: linear-gradient(90deg, transparent, rgba(122,80,48,0.6), transparent);
  }

  /* ---------- Toast ---------- */
  .sdv-toast {
    @apply font-body tracking-wide;
    color: #f0e8c0;
    border: 2px solid rgba(122,80,48,0.6);
    box-shadow: 2px 2px 0 #0a0806;
    padding: 8px 16px;
    font-size: 0.9rem;
  }
  .sdv-toast--success { background: linear-gradient(180deg, #5a8c3a 0%, #3a6020 100%); }
  .sdv-toast--error   { background: linear-gradient(180deg, #a03020 0%, #601808 100%); }
  .sdv-toast--warn    { background: linear-gradient(180deg, #b07018 0%, #7a4808 100%); }
  .sdv-toast--info    { background: linear-gradient(180deg, #2a4838 0%, #1a3020 100%); }

  /* ---------- Stat panel ---------- */
  .sdv-stat {
    background: #c8a870;
    border: 2px solid #7a5030;
    box-shadow: inset 0 1px 0 #e8c888, 2px 2px 0 #1a1208;
    padding: 10px 14px;
  }

}

@layer utilities {
  .font-pixel { font-family: 'VT323', monospace; }
  .text-shadow-pixel { text-shadow: 1px 1px 0 rgba(0,0,0,0.8); }
  .border-pixel { image-rendering: pixelated; }
}
```

- [ ] **Step 4: 启动开发服务器检查全局背景是否变为深绿色网格**

```bash
npm run dev
```

访问 http://localhost:5173，页面背景应为 `#1a2e20` 深森林绿 + 24px 像素网格纹理。若仍是深棕色，检查 `main.css` import 是否生效。

- [ ] **Step 5: Commit**

```bash
git add index.html tailwind.config.js src/shared/styles/main.css
git commit -m "feat(theme): replace RPG style with Stardew Valley pixel theme foundation"
```

---

## Task 2: 共享组件 — AppHeader, AuthWidget, ConfirmDialog, Avatar

**Files:**
- Modify: `src/shared/components/AppHeader.vue`
- Modify: `src/shared/components/AuthWidget.vue`
- Modify: `src/shared/components/ConfirmDialog.vue`
- Modify: `src/shared/components/Avatar.vue`

- [ ] **Step 1: 替换 `AppHeader.vue` template**

```vue
<template>
  <nav class="sdv-nav flex justify-between items-center">
    <div class="flex items-center gap-6">
      <span class="font-pixel text-base tracking-widest text-sdv-gold">✦ 花栗鼠俱乐部</span>
      <slot />
    </div>
    <AuthWidget />
  </nav>
</template>
```

- [ ] **Step 2: 替换 `AuthWidget.vue` template**

```vue
<template>
  <div ref="wrapperRef" class="relative">
    <button
      v-if="!auth.user"
      class="sdv-btn-ghost text-sm px-4 py-1.5"
      @click="router.push('/login')"
    >
      登录
    </button>

    <div v-else>
      <button
        class="flex items-center gap-2 hover:opacity-80 transition-opacity"
        @click="open = !open"
      >
        <Avatar :seed="String(auth.user.id)" :name="auth.user.nickname" size="sm" />
        <span class="font-pixel text-sm text-sdv-text-nav tracking-wide max-w-[80px] truncate">
          {{ auth.user.nickname || '用户' }}
        </span>
      </button>

      <div
        v-if="open"
        class="absolute right-0 top-full mt-2 min-w-[120px] sdv-panel-deep py-1 z-50"
      >
        <button
          class="w-full text-left font-pixel text-sm text-sdv-text-nav hover:text-sdv-gold px-4 py-2 transition-colors"
          @click="router.push('/profile'); open = false"
        >
          个人管理
        </button>
        <button
          class="w-full text-left font-pixel text-sm text-sdv-text-nav hover:text-sdv-gold px-4 py-2 transition-colors"
          @click="auth.logout()"
        >
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 3: 替换 `ConfirmDialog.vue` template**

```vue
<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
      @click.self="!persistent && emit('cancel')"
    >
      <div class="sdv-panel w-[92%] max-w-sm animate-pixel-slide">
        <div class="sdv-panel-title text-center">{{ title }}</div>
        <div class="p-6 text-center">
          <p class="font-body text-sm text-sdv-text-dark leading-relaxed">
            {{ message }}
          </p>
          <div class="mt-6 flex gap-3 justify-center">
            <button v-if="!noCancel" class="sdv-btn-secondary px-5 py-2" @click="emit('cancel')">
              {{ cancelText ?? '稍候' }}
            </button>
            <button :class="danger ? 'sdv-btn-danger' : 'sdv-btn'" class="px-5 py-2" @click="emit('confirm')">
              {{ confirmText ?? '确认' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
```

- [ ] **Step 4: 替换 `Avatar.vue` scoped style**

将 `<style scoped>` 块替换为：

```css
<style scoped>
.avatar-portrait {
  position: relative;
  border: 3px solid #5c3d24;
  box-shadow:
    inset 0 0 0 1px #e8c898,
    2px 2px 0 #0a0806;
  image-rendering: pixelated;
}
</style>
```

同时将 template 中 `class="avatar-portrait rounded-full ...font-display..."`  的 `font-display` 改为 `font-pixel`，移除 `rounded-full`（SDV 头像方形）：

```vue
<template>
  <div
    class="avatar-portrait flex items-center justify-center font-pixel font-bold text-white select-none"
    :class="sizeCls"
    :style="{ backgroundColor: bg }"
  >
    {{ letter }}
  </div>
</template>
```

- [ ] **Step 5: 开发服务器中访问任意页面，检查导航栏显示为深绿背景 + 金色 Logo，ConfirmDialog 触发时显示暖木色面板**

- [ ] **Step 6: Commit**

```bash
git add src/shared/components/AppHeader.vue src/shared/components/AuthWidget.vue \
        src/shared/components/ConfirmDialog.vue src/shared/components/Avatar.vue
git commit -m "feat(theme): apply SDV pixel style to shared components"
```

---

## Task 3: LandingPage

**Files:**
- Modify: `src/pages/LandingPage.vue`

- [ ] **Step 1: 替换 `LandingPage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen flex flex-col">

    <!-- 顶部导航 -->
    <AppHeader>
      <div class="flex gap-6 font-pixel text-xs tracking-widest text-sdv-text-nav/60">
        <a href="#games" class="hover:text-sdv-gold transition-colors">玩法</a>
        <span>微信服务号</span>
      </div>
    </AppHeader>

    <!-- Hero -->
    <header class="flex-none px-6 pt-14 pb-10 text-center select-none border-b border-sdv-border/30">
      <div class="font-pixel text-sm mb-4 text-sdv-gold/80 tracking-widest animate-pixel-pulse">
        ✦ 花栗鼠俱乐部 · 三角洲行动 ✦
      </div>
      <h1 class="font-pixel text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em] text-sdv-gold">
        花栗鼠俱乐部
      </h1>
      <div class="mt-5 mx-auto max-w-xs sdv-divider" />
      <p class="mt-5 font-body text-sm sm:text-base text-sdv-text-light tracking-wide leading-relaxed">
        开一家最有温度的三角洲陪玩店
      </p>
      <p class="mt-2 font-body text-xs text-sdv-text-nav/60 tracking-widest">
        原创 1V1 定制教学单 · 神秘商店趣味单
      </p>
    </header>

    <!-- 玩法卡片 -->
    <main id="games" class="flex-1 px-4 py-10">
      <div class="mx-auto max-w-lg">
        <div class="font-pixel text-xs tracking-[0.4em] uppercase text-sdv-text-nav/50 mb-6 text-center">
          ❖ 我们的玩法 ❖
        </div>
        <div class="flex flex-col gap-3">
          <button
            v-for="game in games"
            :key="game.key"
            class="sdv-menu-row text-left"
            :class="[
              game.active ? 'active' : 'disabled',
            ]"
            @click="game.active && router.push(game.route)"
          >
            <div class="text-2xl shrink-0">{{ game.icon }}</div>
            <div class="flex-1 min-w-0">
              <div class="font-pixel text-base tracking-wider text-sdv-text-dark">
                {{ game.title }}
              </div>
              <div class="font-body text-xs leading-relaxed text-sdv-text-muted whitespace-pre-line mt-0.5">
                {{ game.desc }}
              </div>
            </div>
            <div class="shrink-0">
              <span v-if="game.active" class="sdv-badge">▶ 进入</span>
              <span v-else class="sdv-badge sdv-badge-gray">即将推出</span>
            </div>
          </button>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="text-center font-pixel text-sm tracking-[0.4em] uppercase text-sdv-text-nav/40 py-5 border-t border-sdv-border/20">
      ✥ 微信服务号 · 花栗鼠俱乐部 ✥
    </footer>

  </div>
</template>
```

- [ ] **Step 2: 访问 http://localhost:5173/，检查首页显示 SDV 样式菜单行，神秘商店行高亮金色，其余行置灰**

- [ ] **Step 3: Commit**

```bash
git add src/pages/LandingPage.vue
git commit -m "feat(theme): apply SDV pixel style to LandingPage"
```

---

## Task 4: LoginPage

**Files:**
- Modify: `src/pages/LoginPage.vue`

- [ ] **Step 1: 替换 `LoginPage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <div class="w-full max-w-xs">
      <div class="sdv-panel animate-pixel-slide">
        <div class="sdv-panel-title text-center">花栗鼠俱乐部</div>
        <div class="p-6 space-y-4">
          <p v-if="error" class="font-body text-sm text-red-400 text-center">{{ error }}</p>

          <div>
            <label class="font-pixel text-sm text-sdv-text-dark tracking-wider block mb-1">账号</label>
            <input
              v-model="username"
              type="text"
              autocomplete="username"
              placeholder="hlsXXXX"
              class="sdv-input w-full"
              @keydown.enter="!loading && handleLogin()"
            />
          </div>

          <div>
            <label class="font-pixel text-sm text-sdv-text-dark tracking-wider block mb-1">密码</label>
            <input
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="••••••"
              class="sdv-input w-full"
              @keydown.enter="!loading && handleLogin()"
            />
          </div>

          <button
            class="sdv-btn w-full py-2.5 mt-2"
            :disabled="loading || !username || !password"
            @click="handleLogin"
          >
            {{ loading ? '登录中…' : '▶ 登录' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 访问 http://localhost:5173/login，检查显示暖木色面板 + 深棕边框 + 像素字体，输入框为米黄色背景**

- [ ] **Step 3: Commit**

```bash
git add src/pages/LoginPage.vue
git commit -m "feat(theme): apply SDV pixel style to LoginPage"
```

---

## Task 5: ProfilePage + AuthCallbackPage

**Files:**
- Modify: `src/pages/ProfilePage.vue`
- Modify: `src/pages/AuthCallbackPage.vue`

- [ ] **Step 1: 替换 `ProfilePage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen px-4 py-8 max-w-md mx-auto space-y-6">
    <h1 class="font-pixel text-2xl tracking-widest text-sdv-gold">个人管理</h1>

    <div v-if="auth.user" class="font-body text-xs text-sdv-text-nav/60">
      账号：{{ auth.user.username }}
    </div>

    <!-- 修改昵称 -->
    <div class="sdv-panel">
      <div class="sdv-panel-title">修改昵称</div>
      <div class="p-5 space-y-3">
        <p v-if="nicknameError" class="font-body text-xs text-red-400">{{ nicknameError }}</p>
        <p v-if="nicknameSuccess" class="font-body text-xs text-green-400">修改成功</p>
        <input v-model="nickname" type="text" placeholder="昵称" class="sdv-input w-full" />
        <button
          class="sdv-btn py-2 px-5"
          :disabled="nicknameLoading || !nickname.trim()"
          @click="updateNickname"
        >
          {{ nicknameLoading ? '保存中…' : '▶ 保存昵称' }}
        </button>
      </div>
    </div>

    <!-- 修改密码 -->
    <div class="sdv-panel">
      <div class="sdv-panel-title">修改密码</div>
      <div class="p-5 space-y-3">
        <p v-if="passwordError" class="font-body text-xs text-red-400">{{ passwordError }}</p>
        <p v-if="passwordSuccess" class="font-body text-xs text-green-400">密码修改成功</p>
        <input v-model="oldPassword" type="password" placeholder="旧密码" class="sdv-input w-full" />
        <input v-model="newPassword" type="password" placeholder="新密码（至少6位）" class="sdv-input w-full" />
        <input v-model="confirmPassword" type="password" placeholder="确认新密码" class="sdv-input w-full" />
        <button
          class="sdv-btn-secondary py-2 px-5"
          :disabled="passwordLoading || !oldPassword || !newPassword || !confirmPassword"
          @click="updatePassword"
        >
          {{ passwordLoading ? '修改中…' : '▶ 修改密码' }}
        </button>
      </div>
    </div>

    <!-- 店长专属 -->
    <div v-if="auth.user?.role === 'owner'" class="sdv-panel">
      <div class="p-5">
        <button class="sdv-btn-secondary w-full py-2.5" @click="router.push('/admin')">
          ▶ 进入后台管理
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 替换 `AuthCallbackPage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen flex items-center justify-center">
    <p class="font-pixel text-lg tracking-widest text-sdv-text-nav animate-pixel-pulse">正在跳转…</p>
  </div>
</template>
```

- [ ] **Step 3: Commit**

```bash
git add src/pages/ProfilePage.vue src/pages/AuthCallbackPage.vue
git commit -m "feat(theme): apply SDV pixel style to ProfilePage and AuthCallbackPage"
```

---

## Task 6: AdminPage

**Files:**
- Modify: `src/pages/AdminPage.vue`

- [ ] **Step 1: 替换 `AdminPage.vue` template**

```vue
<template>
  <div class="min-h-screen px-4 py-8 max-w-3xl mx-auto space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="font-pixel text-2xl tracking-widest text-sdv-gold">后台管理</h1>
      <button class="sdv-btn py-2 px-5" @click="openCreate">▶ 创建用户</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-sdv-text-nav/60 animate-pixel-pulse">加载中…</p>

    <div v-if="!loading && !listError" class="sdv-panel overflow-hidden">
      <div class="sdv-panel-title">用户列表</div>
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b-2 border-sdv-border/50">
            <th class="text-left px-4 py-3 font-pixel text-sdv-text-dark tracking-wider font-normal">ID</th>
            <th class="text-left px-4 py-3 font-pixel text-sdv-text-dark tracking-wider font-normal">账号</th>
            <th class="text-left px-4 py-3 font-pixel text-sdv-text-dark tracking-wider font-normal">昵称</th>
            <th class="text-left px-4 py-3 font-pixel text-sdv-text-dark tracking-wider font-normal">角色</th>
            <th class="text-left px-4 py-3 font-pixel text-sdv-text-dark tracking-wider font-normal">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="u in users"
            :key="u.id"
            class="border-b border-sdv-border/25 last:border-0"
          >
            <td class="px-4 py-3 font-body text-sdv-text-muted">{{ u.id }}</td>
            <td class="px-4 py-3 font-body text-sdv-text-dark">{{ u.username }}</td>
            <td class="px-4 py-3 font-body text-sdv-text-dark">{{ u.nickname }}</td>
            <td class="px-4 py-3"><span class="sdv-badge sdv-badge-gold">{{ roleLabels[u.role] }}</span></td>
            <td class="px-4 py-3">
              <div class="flex gap-2">
                <button class="sdv-btn-secondary text-xs py-1 px-3" @click="askResetPassword(u)">重置密码</button>
                <button class="sdv-btn-secondary text-xs py-1 px-3" @click="askResetNickname(u)">重置昵称</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create User Modal -->
    <transition name="fade">
      <div
        v-if="showCreate"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
        @click.self="showCreate = false"
      >
        <div class="sdv-panel w-[92%] max-w-sm animate-pixel-slide">
          <div class="sdv-panel-title text-center">创建用户</div>
          <div class="p-6 space-y-4">

            <div v-if="createResult" class="space-y-3">
              <p class="font-body text-sm text-sdv-text-dark text-center">创建成功！</p>
              <div class="sdv-stat space-y-2">
                <p class="font-body text-sm text-sdv-text-muted">账号：<span class="font-pixel text-sdv-text-dark">{{ createResult.username }}</span></p>
                <p class="font-body text-sm text-sdv-text-muted">初始密码：<span class="font-pixel text-sdv-gold-dark">{{ createResult.password }}</span></p>
              </div>
              <button class="sdv-btn w-full py-2" @click="showCreate = false">关闭</button>
            </div>

            <template v-else>
              <p v-if="createError" class="font-body text-xs text-red-400 text-center">{{ createError }}</p>
              <input v-model="createNickname" type="text" placeholder="昵称（可选）" class="sdv-input w-full" />
              <select v-model="createRole" class="sdv-input w-full">
                <option value="customer">客户</option>
                <option value="service">客服</option>
                <option value="enforcer">打手</option>
                <option value="owner">店长</option>
              </select>
              <div class="flex gap-3 pt-1">
                <button class="flex-1 sdv-btn-secondary py-2" @click="showCreate = false">取消</button>
                <button class="flex-1 sdv-btn py-2 disabled:opacity-40" :disabled="createLoading" @click="submitCreate">
                  {{ createLoading ? '创建中…' : '▶ 创建' }}
                </button>
              </div>
            </template>

          </div>
        </div>
      </div>
    </transition>

    <p v-if="confirmError" class="font-body text-sm text-red-400 text-center">{{ confirmError }}</p>

    <ConfirmDialog
      :open="confirmOpen"
      :title="confirmTitle"
      :message="confirmMessage"
      confirm-text="确认"
      cancel-text="取消"
      danger
      @confirm="handleConfirm"
      @cancel="confirmOpen = false"
    />
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add src/pages/AdminPage.vue
git commit -m "feat(theme): apply SDV pixel style to AdminPage"
```

---

## Task 7: 占位游戏页面 — Teach, Audition, Poker

**Files:**
- Modify: `src/games/teach/pages/TeachPage.vue`
- Modify: `src/games/audition/pages/AuditionPage.vue`
- Modify: `src/games/poker/pages/PokerPage.vue`

三个文件结构相同，仅图标和标题不同。

- [ ] **Step 1: 替换 `TeachPage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="sdv-panel max-w-sm w-full mx-4 animate-pixel-slide">
        <div class="sdv-panel-title text-center">1V1 定制教学</div>
        <div class="p-8 text-center">
          <div class="text-5xl mb-4">🎓</div>
          <p class="font-body text-sm text-sdv-text-muted mb-6">即将推出，敬请期待</p>
          <span class="sdv-badge sdv-badge-gray mb-6 inline-block">即将推出</span>
          <div class="mt-4">
            <button class="sdv-btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 替换 `AuditionPage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="sdv-panel max-w-sm w-full mx-4 animate-pixel-slide">
        <div class="sdv-panel-title text-center">试音</div>
        <div class="p-8 text-center">
          <div class="text-5xl mb-4">🎵</div>
          <p class="font-body text-sm text-sdv-text-muted mb-6">即将推出，敬请期待</p>
          <span class="sdv-badge sdv-badge-gray mb-6 inline-block">即将推出</span>
          <div class="mt-4">
            <button class="sdv-btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 3: 替换 `PokerPage.vue` 完整 template**

```vue
<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="sdv-panel max-w-sm w-full mx-4 animate-pixel-slide">
        <div class="sdv-panel-title text-center">摸红扑克牌</div>
        <div class="p-8 text-center">
          <div class="text-5xl mb-4">🃏</div>
          <p class="font-body text-sm text-sdv-text-muted mb-6">即将推出，敬请期待</p>
          <span class="sdv-badge sdv-badge-gray mb-6 inline-block">即将推出</span>
          <div class="mt-4">
            <button class="sdv-btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 4: Commit**

```bash
git add src/games/teach/pages/TeachPage.vue src/games/audition/pages/AuditionPage.vue \
        src/games/poker/pages/PokerPage.vue
git commit -m "feat(theme): apply SDV pixel style to placeholder game pages"
```

---

## Task 8: ShopLayout + ShopHomePage

**Files:**
- Modify: `src/games/shop/ShopLayout.vue`
- Modify: `src/games/shop/pages/ShopHomePage.vue`

- [ ] **Step 1: 替换 `ShopLayout.vue` — 移除图片背景，继承全局 SDV 背景**

```vue
<template>
  <div class="flex-1 flex flex-col">
    <router-view />
  </div>
</template>
```

- [ ] **Step 2: 替换 `ShopHomePage.vue` 完整 template**

```vue
<template>
  <div class="flex-1 flex flex-col">
    <AppHeader />
    <ToastContainer />

    <header class="px-6 pt-12 pb-4 text-center select-none">
      <div class="font-pixel text-sm mb-3 text-sdv-gold/80 tracking-widest animate-pixel-pulse">
        ✦ 花栗鼠俱乐部 · 原创玩法 ✦
      </div>
      <h1 class="font-pixel text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em] text-sdv-gold">
        花栗鼠的神秘商店
      </h1>
      <div class="mt-4 mx-auto max-w-xs sdv-divider" />
    </header>

    <main class="flex-1 flex items-center justify-center px-4 py-6">
      <div class="sdv-panel w-full max-w-md animate-pixel-slide">
        <div class="sdv-panel-title text-center">开店 / 入店</div>
        <div class="p-7 sm:p-9 space-y-5">

          <div>
            <label class="font-pixel text-sm text-sdv-text-dark tracking-wider block mb-1">
              商店保底
            </label>
            <div class="relative">
              <input
                :value="guaranteeInput"
                class="sdv-input w-full pr-10 text-right"
                :class="{ 'ring-2 ring-red-400': !guaranteeValid }"
                inputmode="numeric"
                maxlength="6"
                :placeholder="String(DEFAULT_GUARANTEED_AMOUNT)"
                @input="onGuaranteeInput"
              />
              <span class="pointer-events-none absolute inset-y-0 right-3 flex items-center font-pixel text-sdv-text-dark">W</span>
            </div>
            <p class="font-body text-xs mt-1.5" :class="guaranteeValid ? 'text-sdv-text-muted' : 'text-red-400'">
              <template v-if="guaranteeValid">
                当前保底：<span class="font-pixel text-sdv-gold-dark">{{ guaranteeNumber }}W</span>
                · 可填 {{ MIN_GUARANTEED_AMOUNT }} ~ {{ MAX_GUARANTEED_AMOUNT }}
              </template>
              <template v-else>
                保底请填 {{ MIN_GUARANTEED_AMOUNT }} ~ {{ MAX_GUARANTEED_AMOUNT }} 之间的整数。
              </template>
            </p>
          </div>

          <button class="sdv-btn w-full py-3" :disabled="creating || !guaranteeValid" @click="handleCreate">
            <span v-if="creating">商店布置中…</span>
            <span v-else>⚜ 开一家神秘商店</span>
          </button>

          <div class="sdv-divider-label"><span>OR</span></div>

          <div>
            <label class="font-pixel text-sm text-sdv-text-dark tracking-wider block mb-1">
              商店口令（六位）
            </label>
            <input
              :value="codeInput"
              class="sdv-input w-full tracking-[.55em] text-center text-2xl"
              placeholder="✦ ✦ ✦ ✦ ✦ ✦"
              inputmode="numeric"
              maxlength="6"
              @input="onCodeInput"
              @keyup.enter="handleJoin"
            />
          </div>

          <button class="sdv-btn-secondary w-full py-3" :disabled="joining" @click="handleJoin">
            <span v-if="joining">正在入店…</span>
            <span v-else>✶ 进入已有商店</span>
          </button>

        </div>
      </div>
    </main>

    <footer class="text-center font-pixel text-sm tracking-[0.4em] uppercase text-sdv-text-nav/40 py-5">
      ✥ 微信服务号 · 花栗鼠俱乐部 ✥
    </footer>
  </div>
</template>
```

- [ ] **Step 3: 访问 http://localhost:5173/shop，检查神秘商店首页显示 SDV 风格**

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/ShopLayout.vue src/games/shop/pages/ShopHomePage.vue
git commit -m "feat(theme): apply SDV pixel style to ShopLayout and ShopHomePage"
```

---

## Task 9: RoomPage + PlayerSlot

**Files:**
- Modify: `src/games/shop/pages/RoomPage.vue`
- Modify: `src/games/shop/components/PlayerSlot.vue`

- [ ] **Step 1: 替换 `PlayerSlot.vue`**

```vue
<template>
  <div
    class="sdv-panel-deep flex flex-col items-center gap-3 p-5 min-h-[180px] transition-all duration-200"
    :class="{
      'ring-2 ring-sdv-gold/60': frameClass === 'is-self',
      'ring-2 ring-sdv-btn-green/60': frameClass === 'is-ready',
      'opacity-60': !player,
    }"
  >
    <template v-if="player">
      <div class="relative">
        <Avatar :seed="player.id" :name="player.nickname" size="lg" />
        <span
          v-if="player.ready"
          class="absolute -bottom-1 -right-1 w-6 h-6 flex items-center justify-center font-pixel text-xs bg-sdv-btn-green border-2 border-sdv-btn-green-border text-white"
          title="已准备"
        >✓</span>
      </div>
      <div class="text-center">
        <div class="font-pixel text-base text-sdv-text-light flex items-center justify-center gap-1.5">
          <span v-if="player.isHost" title="房主" class="text-sdv-gold">👑</span>
          <span class="truncate max-w-[140px]">{{ player.nickname }}</span>
          <span v-if="isSelf" class="sdv-badge sdv-badge-gold !text-xs py-0 px-2">我</span>
        </div>
        <div class="font-body text-xs text-sdv-text-nav/60 mt-1">
          ✦ {{ (player.id ?? '').slice(-6) }} ✦
        </div>
      </div>
      <div class="flex gap-2 items-center mt-auto flex-wrap justify-center">
        <span class="sdv-badge" :class="roleClass">
          {{ player.role ? ROLE_LABEL[player.role] : '未选角色' }}
        </span>
        <span class="sdv-badge" :class="player.ready ? 'sdv-badge-teal' : 'sdv-badge-gray'">
          {{ player.ready ? '已准备' : '未准备' }}
        </span>
      </div>
    </template>
    <template v-else>
      <div class="flex-1 flex flex-col items-center justify-center gap-2 text-sdv-text-nav/50">
        <div class="font-pixel text-3xl animate-pixel-pulse">✦</div>
        <div class="font-pixel text-sm tracking-widest">空位等待加入…</div>
      </div>
    </template>
  </div>
</template>

<style scoped></style>
```

同时在 `<script setup>` 中更新 `roleClass` computed：

```ts
const roleClass = computed(() => {
  if (!props.player?.role) return 'sdv-badge-gray';
  return props.player.role === 'BOSS' ? 'sdv-badge-amber' : 'sdv-badge-teal';
});
```

- [ ] **Step 2: 替换 `RoomPage.vue` template 中的所有 RPG 类名**

关键替换对照：

| 旧类名 | 新类名 |
|--------|--------|
| `card rpg-corners` | `sdv-panel` |
| `rpg-title text-2xl sm:text-3xl` | `font-pixel text-2xl sm:text-3xl text-sdv-gold` |
| `rpg-title text-xl` | `font-pixel text-xl text-sdv-gold` |
| `rpg-title text-2xl tracking-[.4em] font-display` | `font-pixel text-2xl tracking-[.4em] text-sdv-gold` |
| `rpg-divider` | `sdv-divider` |
| `rune-mark text-3xl animate-flicker` | `font-pixel text-3xl animate-pixel-pulse` |
| `font-serif text-[10px] tracking-[0.32em] uppercase text-gold-300/70` | `font-pixel text-xs text-sdv-gold/70` |
| `btn-ghost` | `sdv-btn-ghost` |
| `btn-ghost text-oxblood-300 ...` | `sdv-btn-danger` |
| `btn btn-danger` | `sdv-btn-danger` |
| `btn btn-primary` | `sdv-btn` |
| `btn-boss` | `sdv-btn text-sdv-gold` (在 `role-card.is-active-boss` 语境下) |
| `role-card` | `sdv-menu-row` |
| `role-card.is-active-escort` | `sdv-menu-row active` |
| `role-card.is-active-boss` | `sdv-menu-row active` |
| `badge` | `sdv-badge` |
| `font-body text-parchment-300/80` | `font-body text-sdv-text-light/80` |
| `font-body text-parchment-400/80` | `font-body text-sdv-text-muted` |
| `font-display tracking-[0.2em] text-sapphire-300 text-xl uppercase` | `font-pixel text-xl text-sdv-text-nav` |
| `font-display tracking-[0.2em] text-ember-400 text-xl uppercase` | `font-pixel text-xl text-sdv-gold` |
| `border-gold-500/20 bg-ink-700/60 backdrop-blur` | `border-sdv-border/20 bg-sdv-panel-deep/60 backdrop-blur` |
| `rounded-md px-3 py-1.5 ... border border-gold-500/40 bg-ink-700/60` | `sdv-stat` |
| `text-forest-300 font-bold` | `font-pixel text-sdv-btn-green font-bold` |
| `text-gold-300 font-bold` | `font-pixel text-sdv-gold font-bold` |

完整替换 `RoomPage.vue` template（加载状态卡片部分）：

```html
<!-- 加载状态 -->
<div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center">
  <ToastContainer />
  <div class="sdv-panel p-10 text-center max-w-sm">
    <div class="sdv-panel-title text-center">重返神秘商店</div>
    <div class="p-6">
      <div class="font-pixel text-3xl text-sdv-gold mb-3 animate-pixel-pulse">✶</div>
      <div class="font-body text-sm text-sdv-text-dark">正在根据商店口令返回花栗鼠的神秘商店…</div>
      <div class="mt-4 font-pixel text-xs tracking-[0.32em] uppercase text-sdv-text-muted">
        口令 · {{ code }}
      </div>
    </div>
  </div>
</div>
```

- [ ] **Step 3: 访问商店房间页，检查角色选择卡片、玩家槽、按钮均显示 SDV 风格**

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/pages/RoomPage.vue src/games/shop/components/PlayerSlot.vue
git commit -m "feat(theme): apply SDV pixel style to RoomPage and PlayerSlot"
```

---

## Task 10: 商店基础组件 — ToastContainer, FeedStream, EquipmentShop, LotteryRecordModal

**Files:**
- Modify: `src/games/shop/components/ToastContainer.vue`
- Modify: `src/games/shop/components/FeedStream.vue`
- Modify: `src/games/shop/components/EquipmentShop.vue`
- Modify: `src/games/shop/components/LotteryRecordModal.vue`

- [ ] **Step 1: 替换 `ToastContainer.vue` scoped styles**

将 `<style scoped>` 完整替换（template 保持结构，只改 class 名）：

Template 中 `class="rpg-toast px-4 py-2 rounded-md text-sm font-serif tracking-wider pointer-events-auto"` 改为 `class="sdv-toast pointer-events-auto"`；`:class="\`rpg-toast--${t.type}\`"` 改为 `:class="\`sdv-toast--${t.type}\`"`；移除 `<span class="rune-mark mr-1">` 包裹，保留内层文字。

```vue
<template>
  <div class="fixed top-6 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 pointer-events-none">
    <transition-group name="toast" tag="div" class="flex flex-col gap-2 items-center">
      <div
        v-for="t in roomStore.toasts"
        :key="t.id"
        class="sdv-toast pointer-events-auto"
        :class="`sdv-toast--${t.type}`"
      >
        <template v-if="t.type === 'success'">✓ </template>
        <template v-else-if="t.type === 'error'">✗ </template>
        <template v-else-if="t.type === 'warn'">⚠ </template>
        <template v-else>✦ </template>
        {{ t.message }}
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from { opacity: 0; transform: translateY(-10px); }
.toast-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
```

- [ ] **Step 2: 替换 `FeedStream.vue` 旧 RPG 类名**

读取 `FeedStream.vue` 后，将所有 `card`、`rpg-*`、`parchment-*`、`gold-*`、`font-serif`、`font-body`、`badge` 按以下规则替换：

| 旧 | 新 |
|----|----|
| `card` / `card rpg-corners` | `sdv-panel-deep` |
| `font-serif ... text-gold-300` | `font-pixel text-sdv-gold` |
| `font-body text-parchment-300/...` | `font-body text-sdv-text-light/...` |
| `badge` | `sdv-badge` |
| `text-parchment-400/60` | `text-sdv-text-nav/60` |
| `border-gold-500/30` | `border-sdv-border/30` |
| `bg-ink-700/60` | `bg-sdv-bg/60` |

- [ ] **Step 3: 替换 `EquipmentShop.vue` 旧 RPG 类名**

读取后按相同规则替换，装备列表行改为 `sdv-menu-row`，购买按钮改为 `sdv-btn`。

- [ ] **Step 4: 替换 `LotteryRecordModal.vue`**

读取后，将外层 modal 容器改为 SDV 面板：

```html
<!-- modal wrapper -->
<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4">
  <div class="sdv-panel w-[92%] max-w-lg max-h-[80vh] flex flex-col animate-pixel-slide">
    <div class="sdv-panel-title">抽奖记录</div>
    <div class="flex-1 overflow-y-auto p-5">
      <!-- record rows: sdv-menu-row -->
    </div>
    <div class="p-4 border-t border-sdv-border/30">
      <button class="sdv-btn-secondary w-full py-2" @click="emit('close')">关闭</button>
    </div>
  </div>
</div>
```

- [ ] **Step 5: Commit**

```bash
git add src/games/shop/components/ToastContainer.vue src/games/shop/components/FeedStream.vue \
        src/games/shop/components/EquipmentShop.vue src/games/shop/components/LotteryRecordModal.vue
git commit -m "feat(theme): apply SDV pixel style to shop utility components"
```

---

## Task 11: LotteryPanel + Settlement 组件 + PlayPage

**Files:**
- Modify: `src/games/shop/components/LotteryPanel.vue`
- Modify: `src/games/shop/components/SettlementModal.vue`
- Modify: `src/games/shop/components/SettlementRecordModal.vue`
- Modify: `src/games/shop/pages/PlayPage.vue`

- [ ] **Step 1: 替换 `LotteryPanel.vue` 外层容器与控件**

LotteryPanel 有复杂的滚动动画（纯 canvas/JS 驱动），只改外层容器和控件样式，不改动画核心逻辑。

读取文件后找到外层 template 结构，将：
- 外层 `<div class="card rpg-corners ...">` → `<div class="sdv-panel ...">`
- 奖品条区域的 `<div class="scroll-panel ...">` → `<div class="sdv-panel ...">`  
- 抽奖按钮 `btn-primary` → `sdv-btn`
- `coin-tag` → `font-pixel text-sdv-gold`
- `font-serif text-... text-gold-300` → `font-pixel text-sdv-gold`
- `text-parchment-300/80` → `text-sdv-text-light/80`
- `badge` → `sdv-badge`
- `rpg-title` → `font-pixel text-xl text-sdv-gold`

- [ ] **Step 2: 替换 `SettlementModal.vue` 外层容器**

该文件 531 行，结构为：modals、表格、按钮。关键替换：

```html
<!-- 外层 modal -->
<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4">
  <div class="sdv-panel w-[95%] max-w-2xl max-h-[90vh] flex flex-col animate-pixel-slide">
    <div class="sdv-panel-title">本轮结算</div>
    <div class="flex-1 overflow-y-auto">
      <!-- content -->
    </div>
    <div class="p-4 flex gap-3 justify-end border-t border-sdv-border/30">
      <button class="sdv-btn-secondary py-2 px-5" @click="emit('close')">关闭</button>
      <button class="sdv-btn py-2 px-5" @click="confirmSettle">确认结算</button>
    </div>
  </div>
</div>
```

将表格内 `scroll-panel`、`rpg-title`、`gold-*`、`parchment-*`、`forest-*`、`ember-*`、`btn-*`、`badge` 按前述规则全部替换。

- [ ] **Step 3: 替换 `SettlementRecordModal.vue`**

结构同 LotteryRecordModal，替换外层容器为 SDV 面板，标题 `sdv-panel-title`，记录行 `sdv-menu-row`。

- [ ] **Step 4: 替换 `PlayPage.vue`**

PlayPage 主要是布局容器和控件，不改动游戏逻辑。关键替换：

```html
<!-- 加载状态 -->
<div class="sdv-panel p-10 text-center max-w-sm">
  <div class="sdv-panel-title text-center">重返神秘商店</div>
  <div class="p-6">
    <div class="font-pixel text-3xl text-sdv-gold mb-3 animate-pixel-pulse">✶</div>
    <div class="font-body text-sm text-sdv-text-dark">正在根据商店口令返回花栗鼠的神秘商店…</div>
  </div>
</div>

<!-- sticky header -->
<header class="px-4 sm:px-8 py-3 sticky top-0 z-10 backdrop-blur sdv-panel-deep border-b border-sdv-border/25">
  <div class="flex items-center justify-between gap-3 flex-wrap">
    <div class="flex items-center gap-3 min-w-0">
      <div class="font-pixel text-2xl text-sdv-gold animate-pixel-pulse">⚜</div>
      <div class="min-w-0">
        <div class="font-pixel text-base sm:text-lg leading-tight text-sdv-gold">花栗鼠的神秘商店</div>
        <div class="font-pixel text-xs tracking-[0.3em] uppercase text-sdv-gold/60 mt-0.5">口令 · {{ code }}</div>
      </div>
    </div>
    <button class="sdv-btn-danger text-xs py-1.5 px-3" @click="confirmLeave">离开</button>
  </div>
</header>

<!-- 左侧边栏 -->
<aside class="order-2 lg:order-1">
  <div class="sdv-panel-deep p-4 lg:sticky lg:top-24 flex flex-col gap-4">

    <!-- 本局数据 -->
    <div>
      <div class="font-pixel text-xs tracking-[0.2em] uppercase text-sdv-gold/60 mb-2">⚜ 本局数据</div>
      <div class="grid grid-cols-2 gap-2">
        <div class="sdv-stat text-center" title="商店保底">
          <div class="font-pixel text-xs uppercase tracking-[0.2em] text-sdv-gold/80">保底</div>
          <div class="font-pixel text-base text-sdv-gold mt-0.5">
            {{ roomStore.guaranteedAmount }}<span class="text-sdv-gold/80 ml-0.5 text-xs">W</span>
          </div>
        </div>
        <div class="sdv-stat text-center" title="历次撤离成功累计扣减额">
          <div class="font-pixel text-xs uppercase tracking-[0.2em] text-sdv-btn-green/70">得吃</div>
          <div class="font-pixel text-base text-sdv-btn-green mt-0.5">
            {{ roomStore.evacuatedTotal }}<span class="text-sdv-btn-green/80 ml-0.5 text-xs">W</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 操作 -->
    <div>
      <div class="font-pixel text-xs tracking-[0.2em] uppercase text-sdv-gold/60 mb-2">🗂 操作</div>
      <div class="flex flex-wrap gap-2">
        <button v-if="canSettle" class="sdv-btn py-1.5 px-3 text-xs" @click="showSettlement = true">⚖ 本轮结算</button>
        <button v-if="canSettle" class="sdv-btn-danger py-1.5 px-3 text-xs" @click="handleCloseGame">结单</button>
        <button class="sdv-btn-ghost py-1.5 px-3 text-xs" @click="showSettlementHistory = true">结算记录</button>
        <button class="sdv-btn-ghost py-1.5 px-3 text-xs" @click="showLottery = true">抽奖记录</button>
      </div>
    </div>

    <div class="sdv-divider" />

    <!-- 店内成员 -->
    <div>
      <div class="font-pixel text-xs tracking-[0.2em] uppercase text-sdv-gold/60 mb-2">👥 店内成员</div>
      <div class="grid grid-cols-3 gap-2 lg:grid-cols-1">
        <div
          v-for="p in roster"
          :key="p.id"
          class="sdv-panel-deep p-2 flex flex-col lg:flex-row items-center gap-1.5 lg:gap-3 lg:p-2.5 transition-colors"
          :class="p.id === roomStore.selfPlayerId ? 'ring-1 ring-sdv-gold/60' : ''"
        >
          <div class="relative shrink-0">
            <Avatar :seed="p.id" :name="p.nickname" size="md" />
            <span
              v-if="!p.connected"
              class="absolute -bottom-1 -right-1 w-3.5 h-3.5 bg-sdv-btn-red border border-sdv-bg"
              title="失联中"
            />
          </div>
          <div class="flex-1 min-w-0 w-full">
            <div class="font-pixel text-sm text-sdv-text-light flex flex-wrap items-center gap-1 justify-center lg:justify-start">
              <span v-if="p.isHost" class="text-sdv-gold shrink-0" title="房主">👑</span>
              <span class="flex-1 min-w-0 truncate text-xs lg:text-sm lg:break-all lg:whitespace-normal">{{ p.nickname }}</span>
              <span v-if="p.id === roomStore.selfPlayerId" class="sdv-badge sdv-badge-gold text-xs py-0 px-2 shrink-0">我</span>
            </div>
            <div class="flex items-center gap-1.5 mt-1 justify-center lg:justify-start">
              <span class="sdv-badge text-xs py-0 px-2" :class="p.role === 'BOSS' ? 'sdv-badge-amber' : 'sdv-badge-teal'">
                {{ p.role ? ROLE_LABEL[p.role] : '未择阶' }}
              </span>
            </div>
            <div class="font-pixel text-xs mt-1 text-sdv-gold">
              💰 {{ p.coin }}
            </div>
          </div>
        </div>
      </div>
    </div>

  </div>
</aside>

<!-- 中央区域（未选角色占位） -->
<div class="sdv-panel p-10 text-center">
  <div class="sdv-panel-title text-center">正在确认角色</div>
  <div class="p-6">
    <div class="font-pixel text-sdv-text-dark text-sm animate-pixel-pulse">正在确认您的店内角色…</div>
  </div>
</div>
```

- [ ] **Step 5: 运行开发服务器，进入商店游戏全流程测试（首页 → 创建/加入房间 → 角色选择 → 游戏页）**

```bash
npm run dev
```

检查：
- 抽奖转盘动画正常（动画逻辑未改动，只改外层容器）
- 结算弹窗显示 SDV 面板样式
- Toast 通知显示 SDV 样式
- 离开/结单操作对话框显示 SDV 面板

- [ ] **Step 6: Commit**

```bash
git add src/games/shop/components/LotteryPanel.vue \
        src/games/shop/components/SettlementModal.vue \
        src/games/shop/components/SettlementRecordModal.vue \
        src/games/shop/pages/PlayPage.vue
git commit -m "feat(theme): apply SDV pixel style to LotteryPanel, settlement modals, and PlayPage"
```

---

## 完成验证清单

- [ ] 首页 (`/`) — SDV 菜单行，神秘商店高亮
- [ ] 登录页 (`/login`) — 暖木色面板 + 像素字体
- [ ] 个人页 (`/profile`) — SDV 设置面板
- [ ] 后台管理 (`/admin`) — SDV 表格 + 弹窗（店长账号登录）
- [ ] 占位页 (`/teach`, `/audition`, `/poker`) — SDV 占位面板
- [ ] 商店首页 (`/shop`) — SDV 面板，开店/入店
- [ ] 准备大厅 (`/shop/:code`) — SDV 角色选择、玩家槽
- [ ] 游戏页 (`/shop/:code/play`) — SDV 全局，抽奖动画正常，结算弹窗正常
- [ ] 全站导航栏 — 深绿背景 + 金色 Logo
- [ ] Toast 通知 — SDV 风格（绿/红/橙/暗绿）
- [ ] ConfirmDialog — SDV 暖面板 + 按钮

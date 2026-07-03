# Global Auth Header Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a persistent top-right login entry to every page — shows "登录" button when unauthenticated, avatar + nickname + logout dropdown when authenticated.

**Architecture:** Two new shared components (`AuthWidget.vue`, `AppHeader.vue`) are added to `src/shared/components/`. `App.vue` calls `auth.init()` on mount to restore login state from localStorage. Each of the 7 target pages imports and renders `AppHeader` at the top of its template.

**Tech Stack:** Vue 3 (Composition API, `<script setup>`), Pinia (`useAuthStore`), Vue Router, Tailwind CSS

---

## File Structure

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `src/shared/components/AuthWidget.vue` | Renders login button or user avatar+nickname+dropdown |
| Create | `src/shared/components/AppHeader.vue` | Shared nav bar: brand left, optional slot, AuthWidget right |
| Modify | `src/App.vue` | Call `auth.init()` on mount |
| Modify | `src/pages/LandingPage.vue` | Replace `<nav>` with `<AppHeader>` + slot for existing links |
| Modify | `src/games/shop/pages/ShopHomePage.vue` | Add `<AppHeader />` at top of template |
| Modify | `src/games/shop/pages/RoomPage.vue` | Wrap template roots in single div, add `<AppHeader />` |
| Modify | `src/games/shop/pages/PlayPage.vue` | Wrap template roots in single div, add `<AppHeader />` |
| Modify | `src/games/teach/pages/TeachPage.vue` | Restructure layout, add `<AppHeader />` |
| Modify | `src/games/audition/pages/AuditionPage.vue` | Restructure layout, add `<AppHeader />` |
| Modify | `src/games/poker/pages/PokerPage.vue` | Restructure layout, add `<AppHeader />` |

---

## Task 1: Create AuthWidget.vue

**Files:**
- Create: `src/shared/components/AuthWidget.vue`

- [ ] **Step 1: Create the file**

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Avatar from '@shared/components/Avatar.vue'

const auth = useAuthStore()
const router = useRouter()
const open = ref(false)
const wrapperRef = ref<HTMLElement | null>(null)

function handleOutsideClick(e: MouseEvent) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', handleOutsideClick))
onUnmounted(() => document.removeEventListener('click', handleOutsideClick))
</script>

<template>
  <div ref="wrapperRef" class="relative">
    <button
      v-if="!auth.user"
      class="font-serif text-xs tracking-widest text-gold-300 border border-gold-400/40 px-3 py-1.5 rounded hover:border-gold-300 transition-colors"
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
        <span class="font-serif text-xs text-parchment-300 tracking-wide max-w-[80px] truncate">
          {{ auth.user.nickname }}
        </span>
      </button>

      <div
        v-if="open"
        class="absolute right-0 top-full mt-2 min-w-[100px] card border border-white/10 rounded-lg py-1 z-50"
      >
        <button
          class="w-full text-left font-serif text-xs text-parchment-300/80 hover:text-gold-300 px-4 py-2 transition-colors"
          @click="auth.logout()"
        >
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors related to `AuthWidget.vue`

- [ ] **Step 3: Commit**

```bash
git add src/shared/components/AuthWidget.vue
git commit -m "feat: add AuthWidget component for login/logout UI"
```

---

## Task 2: Create AppHeader.vue

**Files:**
- Create: `src/shared/components/AppHeader.vue`

- [ ] **Step 1: Create the file**

```vue
<script setup lang="ts">
import AuthWidget from './AuthWidget.vue'
</script>

<template>
  <nav class="flex justify-between items-center px-6 py-4 border-b border-white/5">
    <div class="flex items-center gap-6">
      <span class="rpg-subtitle text-sm tracking-widest text-gold-300">✦ 花栗鼠俱乐部</span>
      <slot />
    </div>
    <AuthWidget />
  </nav>
</template>
```

- [ ] **Step 2: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add src/shared/components/AppHeader.vue
git commit -m "feat: add AppHeader shared nav component"
```

---

## Task 3: Initialize auth state on app mount

**Files:**
- Modify: `src/App.vue`

Current content of `src/App.vue`:
```vue
<script setup lang="ts">
import { onMounted } from 'vue';
import { usePlayerStore } from '@/shared/stores/player';

const playerStore = usePlayerStore();

onMounted(() => {
  playerStore.ensureIdentity();
});
</script>
```

- [ ] **Step 1: Add `auth.init()` to `onMounted`**

Replace the `<script setup>` block in `src/App.vue` with:

```vue
<script setup lang="ts">
import { onMounted } from 'vue';
import { usePlayerStore } from '@/shared/stores/player';
import { useAuthStore } from '@/stores/auth';

const playerStore = usePlayerStore();
const auth = useAuthStore();

onMounted(() => {
  playerStore.ensureIdentity();
  auth.init();
});
</script>
```

- [ ] **Step 2: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add src/App.vue
git commit -m "feat: restore auth state from localStorage on app mount"
```

---

## Task 4: Update LandingPage.vue

**Files:**
- Modify: `src/pages/LandingPage.vue`

The existing `<nav>` block (lines 46–52):
```html
<nav class="flex justify-between items-center px-6 py-4 border-b border-white/5">
  <span class="rpg-subtitle text-sm tracking-widest text-gold-300">✦ 花栗鼠俱乐部</span>
  <div class="flex gap-6 font-serif text-xs tracking-widest text-parchment-400/60">
    <a href="#games" class="hover:text-gold-300 transition-colors">玩法</a>
    <span>微信服务号</span>
  </div>
</nav>
```

- [ ] **Step 1: Add `AppHeader` import to `<script setup>`**

In `src/pages/LandingPage.vue`, add to the script block:

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';

const router = useRouter();
// ... rest of existing code unchanged
```

- [ ] **Step 2: Replace `<nav>` with `<AppHeader>`**

Replace the existing `<nav>` block with:

```html
<AppHeader>
  <div class="flex gap-6 font-serif text-xs tracking-widest text-parchment-400/60">
    <a href="#games" class="hover:text-gold-300 transition-colors">玩法</a>
    <span>微信服务号</span>
  </div>
</AppHeader>
```

- [ ] **Step 3: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add src/pages/LandingPage.vue
git commit -m "feat: add AppHeader with auth widget to LandingPage"
```

---

## Task 5: Update ShopHomePage.vue

**Files:**
- Modify: `src/games/shop/pages/ShopHomePage.vue`

The template root starts at line 174:
```html
<template>
  <div class="flex-1 flex flex-col">

    <header class="px-6 pt-12 pb-4 text-center select-none">
```

- [ ] **Step 1: Add `AppHeader` import**

In the `<script setup>` block of `src/games/shop/pages/ShopHomePage.vue`, add:

```ts
import AppHeader from '@shared/components/AppHeader.vue';
```

- [ ] **Step 2: Add `<AppHeader />` to template**

Directly after `<div class="flex-1 flex flex-col">`, add `<AppHeader />`:

```html
<template>
  <div class="flex-1 flex flex-col">
    <AppHeader />

    <header class="px-6 pt-12 pb-4 text-center select-none">
```

- [ ] **Step 3: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/pages/ShopHomePage.vue
git commit -m "feat: add AppHeader to ShopHomePage"
```

---

## Task 6: Update RoomPage.vue

**Files:**
- Modify: `src/games/shop/pages/RoomPage.vue`

The template currently has two root elements (a loading-state div and a main-content div). Wrap them in a single root div and prepend `<AppHeader />`.

- [ ] **Step 1: Add `AppHeader` import**

In the `<script setup>` block of `src/games/shop/pages/RoomPage.vue`, add:

```ts
import AppHeader from '@shared/components/AppHeader.vue';
```

- [ ] **Step 2: Wrap template and add header**

Replace the opening of the template:

```html
<template>
  <div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center">
```

with:

```html
<template>
  <div class="flex-1 flex flex-col">
    <AppHeader />

    <div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center">
```

And add one closing `</div>` at the end of the template (before `</template>`), to close the new outer wrapper div.

- [ ] **Step 3: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/pages/RoomPage.vue
git commit -m "feat: add AppHeader to RoomPage"
```

---

## Task 7: Update PlayPage.vue

**Files:**
- Modify: `src/games/shop/pages/PlayPage.vue`

Same pattern as RoomPage: two root elements, wrap in outer div, add `<AppHeader />`.

- [ ] **Step 1: Add `AppHeader` import**

In the `<script setup>` block of `src/games/shop/pages/PlayPage.vue`, add:

```ts
import AppHeader from '@shared/components/AppHeader.vue';
```

- [ ] **Step 2: Wrap template and add header**

Replace:

```html
<template>
  <div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center min-h-screen">
```

with:

```html
<template>
  <div class="flex-1 flex flex-col min-h-screen">
    <AppHeader />

    <div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center">
```

Replace:

```html
  <div v-else class="flex-1 flex flex-col min-h-screen">
```

with:

```html
    <div v-else class="flex-1 flex flex-col">
```

And add one closing `</div>` at the end before `</template>` to close the outer wrapper.

- [ ] **Step 3: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/pages/PlayPage.vue
git commit -m "feat: add AppHeader to PlayPage"
```

---

## Task 8: Update TeachPage, AuditionPage, PokerPage

**Files:**
- Modify: `src/games/teach/pages/TeachPage.vue`
- Modify: `src/games/audition/pages/AuditionPage.vue`
- Modify: `src/games/poker/pages/PokerPage.vue`

All three pages currently use this layout pattern:
```html
<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <div class="text-center select-none">
      ...
    </div>
  </div>
</template>
```

Restructure to put `<AppHeader />` at the top and center the inner content in the remaining space.

- [ ] **Step 1: Update TeachPage.vue**

Replace the entire file content of `src/games/teach/pages/TeachPage.vue`:

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="text-center select-none">
        <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
        <div class="text-4xl mb-4">🎓</div>
        <h1 class="rpg-title text-3xl font-bold mb-3">1V1 定制教学</h1>
        <div class="mx-auto max-w-xs rpg-divider mb-6" />
        <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
        <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Update AuditionPage.vue**

Replace the entire file content of `src/games/audition/pages/AuditionPage.vue`:

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="text-center select-none">
        <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
        <div class="text-4xl mb-4">🎵</div>
        <h1 class="rpg-title text-3xl font-bold mb-3">试音</h1>
        <div class="mx-auto max-w-xs rpg-divider mb-6" />
        <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
        <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 3: Update PokerPage.vue**

Replace the entire file content of `src/games/poker/pages/PokerPage.vue`:

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="text-center select-none">
        <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
        <div class="text-4xl mb-4">🃏</div>
        <h1 class="rpg-title text-3xl font-bold mb-3">摸红扑克牌</h1>
        <div class="mx-auto max-w-xs rpg-divider mb-6" />
        <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
        <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 4: Run typecheck**

```bash
npm run typecheck
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add src/games/teach/pages/TeachPage.vue src/games/audition/pages/AuditionPage.vue src/games/poker/pages/PokerPage.vue
git commit -m "feat: add AppHeader to Teach, Audition, and Poker pages"
```

---

## Task 9: Smoke test in browser

- [ ] **Step 1: Start dev server**

```bash
npm run dev
```

- [ ] **Step 2: Verify unauthenticated state**

Open `http://localhost:5173`. Check:
- Top-right shows「登录」button on every page (/, /shop, /teach, /audition, /poker)
- Clicking「登录」navigates to `/login`

- [ ] **Step 3: Verify authenticated state**

Complete WeChat login. Then check:
- Top-right shows avatar + nickname
- Clicking opens dropdown with「退出登录」
- Clicking「退出登录」clears the session and redirects to `/login`
- Refreshing the page keeps the logged-in state (auth.init() restores from localStorage)

- [ ] **Step 4: Verify game pages**

Navigate to `/shop` → create/join a room → confirm AppHeader appears on RoomPage and PlayPage without breaking existing game UI.

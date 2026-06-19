# 花栗鼠俱乐部官网改版 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 hls.com 改为花栗鼠俱乐部官方主页，神秘商店迁移至 `/shop`，并建立按玩法模块化的目录结构。

**Architecture:** 新增 `src/games/` 目录，每个玩法完全自包含（pages + components + net + game + stores + router）；跨玩法公共代码放 `src/shared/`；`src/pages/LandingPage.vue` 为新首页。文件迁移采用「先建后删」策略：新位置文件创建并更新完 import 后，最后统一删除旧文件。

**Tech Stack:** Vue 3 + TypeScript + Vite + Pinia + Tailwind CSS（vue-router 4）

---

## 文件变更全览

| 操作 | 文件 |
|---|---|
| **新建** | `src/pages/LandingPage.vue` |
| **新建** | `src/games/shop/router.ts` |
| **新建** | `src/games/teach/pages/TeachPage.vue`, `src/games/teach/router.ts` |
| **新建** | `src/games/audition/pages/AuditionPage.vue`, `src/games/audition/router.ts` |
| **新建** | `src/games/poker/pages/PokerPage.vue`, `src/games/poker/router.ts` |
| **迁移** | `src/utils/*` → `src/shared/utils/*` |
| **迁移** | `src/styles/main.css` → `src/shared/styles/main.css` |
| **迁移** | `src/stores/player.ts` → `src/shared/stores/player.ts` |
| **迁移** | `src/components/Avatar.vue` → `src/shared/components/Avatar.vue` |
| **迁移** | `src/components/ConfirmDialog.vue` → `src/shared/components/ConfirmDialog.vue` |
| **迁移** | `src/net/*` → `src/games/shop/net/*` |
| **迁移** | `src/game/*` → `src/games/shop/game/*` |
| **迁移** | `src/stores/room.ts` → `src/games/shop/stores/room.ts` |
| **迁移** | `src/components/ToastContainer.vue` → `src/games/shop/components/ToastContainer.vue` |
| **迁移** | `src/components/{EquipmentShop,FeedStream,LotteryPanel,LotteryRecordModal,PlayerSlot,SettlementModal,SettlementRecordModal}.vue` → `src/games/shop/components/` |
| **迁移** | `src/pages/HomePage.vue` → `src/games/shop/pages/ShopHomePage.vue` |
| **迁移** | `src/pages/RoomPage.vue` → `src/games/shop/pages/RoomPage.vue` |
| **迁移** | `src/pages/PlayPage.vue` → `src/games/shop/pages/PlayPage.vue` |
| **修改** | `vite.config.ts`（新增别名） |
| **修改** | `tsconfig.json`（新增路径） |
| **修改** | `src/router/index.ts`（全部重写） |
| **修改** | `src/main.ts`（更新 styles 导入） |
| **删除** | 旧 `src/components/`, `src/pages/`, `src/net/`, `src/game/`, `src/stores/`, `src/utils/`, `src/styles/` |

---

## Task 1: 添加路径别名

**Files:**
- Modify: `vite.config.ts`
- Modify: `tsconfig.json`

- [ ] **Step 1: 更新 vite.config.ts**

将 `resolve.alias` 改为：

```ts
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { fileURLToPath, URL } from 'node:url';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@shop': fileURLToPath(new URL('./src/games/shop', import.meta.url)),
      '@shared': fileURLToPath(new URL('./src/shared', import.meta.url)),
    },
  },
  server: {
    host: true,
    port: 5173,
  },
});
```

- [ ] **Step 2: 更新 tsconfig.json 的 paths**

将 `compilerOptions.paths` 改为：

```json
"paths": {
  "@/*": ["src/*"],
  "@shop/*": ["src/games/shop/*"],
  "@shared/*": ["src/shared/*"]
}
```

- [ ] **Step 3: 验证**

```bash
cd /Users/kratos/hls && npm run typecheck
```

期望：无错误（此时还没动任何源文件，不应报错）

- [ ] **Step 4: 提交**

```bash
git add vite.config.ts tsconfig.json
git commit -m "chore: add @shop and @shared path aliases"
```

---

## Task 2: 创建目录结构

**Files:**
- Create directories only（无文件内容）

- [ ] **Step 1: 批量建目录**

```bash
mkdir -p /Users/kratos/hls/src/shared/utils
mkdir -p /Users/kratos/hls/src/shared/stores
mkdir -p /Users/kratos/hls/src/shared/components
mkdir -p /Users/kratos/hls/src/shared/styles
mkdir -p /Users/kratos/hls/src/games/shop/pages
mkdir -p /Users/kratos/hls/src/games/shop/components
mkdir -p /Users/kratos/hls/src/games/shop/net
mkdir -p /Users/kratos/hls/src/games/shop/game
mkdir -p /Users/kratos/hls/src/games/shop/stores
mkdir -p /Users/kratos/hls/src/games/teach/pages
mkdir -p /Users/kratos/hls/src/games/audition/pages
mkdir -p /Users/kratos/hls/src/games/poker/pages
```

---

## Task 3: 迁移 shared/utils 与 shared/styles

这三个文件没有内部 `@/` 导入，直接复制即可。

**Files:**
- Create: `src/shared/utils/id.ts`
- Create: `src/shared/utils/storage.ts`
- Create: `src/shared/utils/avatar.ts`
- Create: `src/shared/styles/main.css`

- [ ] **Step 1: 复制文件**

```bash
cp /Users/kratos/hls/src/utils/id.ts /Users/kratos/hls/src/shared/utils/id.ts
cp /Users/kratos/hls/src/utils/storage.ts /Users/kratos/hls/src/shared/utils/storage.ts
cp /Users/kratos/hls/src/utils/avatar.ts /Users/kratos/hls/src/shared/utils/avatar.ts
cp /Users/kratos/hls/src/styles/main.css /Users/kratos/hls/src/shared/styles/main.css
```

- [ ] **Step 2: 提交**

```bash
git add src/shared/
git commit -m "chore: copy shared utils and styles to src/shared"
```

---

## Task 4: 迁移 shared/stores/player.ts

**Files:**
- Create: `src/shared/stores/player.ts`

- [ ] **Step 1: 复制并更新 import 路径**

```bash
cp /Users/kratos/hls/src/stores/player.ts /Users/kratos/hls/src/shared/stores/player.ts
```

在 `src/shared/stores/player.ts` 中将：
```ts
import { generatePlayerId } from '@/utils/id';
import { storage } from '@/utils/storage';
```
改为：
```ts
import { generatePlayerId } from '@shared/utils/id';
import { storage } from '@shared/utils/storage';
```

- [ ] **Step 2: 提交**

```bash
git add src/shared/stores/player.ts
git commit -m "chore: copy player store to src/shared/stores"
```

---

## Task 5: 迁移 shared/components（Avatar、ConfirmDialog）

**Files:**
- Create: `src/shared/components/Avatar.vue`
- Create: `src/shared/components/ConfirmDialog.vue`

- [ ] **Step 1: 复制 ConfirmDialog（无内部导入，直接复制）**

```bash
cp /Users/kratos/hls/src/components/ConfirmDialog.vue /Users/kratos/hls/src/shared/components/ConfirmDialog.vue
```

- [ ] **Step 2: 复制并更新 Avatar.vue**

```bash
cp /Users/kratos/hls/src/components/Avatar.vue /Users/kratos/hls/src/shared/components/Avatar.vue
```

在 `src/shared/components/Avatar.vue` 中将：
```ts
import { avatarColor, avatarLetter } from '@/utils/avatar';
```
改为：
```ts
import { avatarColor, avatarLetter } from '@shared/utils/avatar';
```

- [ ] **Step 3: 提交**

```bash
git add src/shared/components/
git commit -m "chore: copy Avatar and ConfirmDialog to src/shared/components"
```

---

## Task 6: 迁移 shop/net/

`ws.ts`、`api.ts` 内部已使用相对导入（`./config`、`./protocol`），移动后无需改动。`client.ts` 有一个 `@/utils/id` 需更新。

**Files:**
- Create: `src/games/shop/net/config.ts`
- Create: `src/games/shop/net/protocol.ts`
- Create: `src/games/shop/net/ws.ts`
- Create: `src/games/shop/net/api.ts`
- Create: `src/games/shop/net/client.ts`

- [ ] **Step 1: 复制 config.ts 和 protocol.ts（无内部导入）**

```bash
cp /Users/kratos/hls/src/net/config.ts /Users/kratos/hls/src/games/shop/net/config.ts
cp /Users/kratos/hls/src/net/protocol.ts /Users/kratos/hls/src/games/shop/net/protocol.ts
```

- [ ] **Step 2: 复制 ws.ts 和 api.ts（相对导入不需改）**

```bash
cp /Users/kratos/hls/src/net/ws.ts /Users/kratos/hls/src/games/shop/net/ws.ts
cp /Users/kratos/hls/src/net/api.ts /Users/kratos/hls/src/games/shop/net/api.ts
```

- [ ] **Step 3: 复制并更新 client.ts**

```bash
cp /Users/kratos/hls/src/net/client.ts /Users/kratos/hls/src/games/shop/net/client.ts
```

在 `src/games/shop/net/client.ts` 中将：
```ts
import { shortId } from '@/utils/id';
```
改为：
```ts
import { shortId } from '@shared/utils/id';
```

- [ ] **Step 4: 提交**

```bash
git add src/games/shop/net/
git commit -m "chore: copy shop net layer to src/games/shop/net"
```

---

## Task 7: 迁移 shop/game/

**Files:**
- Create: `src/games/shop/game/equipment.ts`
- Create: `src/games/shop/game/lottery.ts`
- Create: `src/games/shop/game/rules.ts`

- [ ] **Step 1: 复制 equipment.ts 和 lottery.ts（无内部导入）**

```bash
cp /Users/kratos/hls/src/game/equipment.ts /Users/kratos/hls/src/games/shop/game/equipment.ts
cp /Users/kratos/hls/src/game/lottery.ts /Users/kratos/hls/src/games/shop/game/lottery.ts
```

- [ ] **Step 2: 复制并更新 rules.ts**

```bash
cp /Users/kratos/hls/src/game/rules.ts /Users/kratos/hls/src/games/shop/game/rules.ts
```

在 `src/games/shop/game/rules.ts` 中将：
```ts
import type { Player, Role } from '@/net/protocol';
```
改为：
```ts
import type { Player, Role } from '../net/protocol';
```

- [ ] **Step 3: 提交**

```bash
git add src/games/shop/game/
git commit -m "chore: copy shop game logic to src/games/shop/game"
```

---

## Task 8: 迁移 shop/stores/room.ts

**Files:**
- Create: `src/games/shop/stores/room.ts`

- [ ] **Step 1: 复制并更新 import 路径**

```bash
cp /Users/kratos/hls/src/stores/room.ts /Users/kratos/hls/src/games/shop/stores/room.ts
```

在 `src/games/shop/stores/room.ts` 中更新以下导入：

```ts
// 将
import { GameClient } from '@/net/client';
import type {
  EventMap,
  EventType,
  IntentMap,
  IntentType,
  LotteryRecord,
  Player,
  RoomSnapshot,
} from '@/net/protocol';
import { usePlayerStore } from './player';

// 改为
import { GameClient } from '../net/client';
import type {
  EventMap,
  EventType,
  IntentMap,
  IntentType,
  LotteryRecord,
  Player,
  RoomSnapshot,
} from '../net/protocol';
import { usePlayerStore } from '@shared/stores/player';
```

- [ ] **Step 2: 提交**

```bash
git add src/games/shop/stores/room.ts
git commit -m "chore: copy shop room store to src/games/shop/stores"
```

---

## Task 9: 迁移 shop/components/

**Files:**
- Create: `src/games/shop/components/ToastContainer.vue`
- Create: `src/games/shop/components/Avatar.vue`（注意：这里不再存在，见说明）
- Create: `src/games/shop/components/PlayerSlot.vue`
- Create: `src/games/shop/components/FeedStream.vue`
- Create: `src/games/shop/components/EquipmentShop.vue`
- Create: `src/games/shop/components/LotteryPanel.vue`
- Create: `src/games/shop/components/LotteryRecordModal.vue`
- Create: `src/games/shop/components/SettlementModal.vue`
- Create: `src/games/shop/components/SettlementRecordModal.vue`

> **说明：** `Avatar.vue` 已在 Task 5 放入 `shared/components/`，shop 组件直接从 `@shared/components/Avatar.vue` 导入，不再在 shop/components/ 里重复存放。

- [ ] **Step 1: 复制并更新 ToastContainer.vue**

```bash
cp /Users/kratos/hls/src/components/ToastContainer.vue /Users/kratos/hls/src/games/shop/components/ToastContainer.vue
```

将 `@/stores/room` → `../stores/room`

- [ ] **Step 2: 复制并更新 PlayerSlot.vue**

```bash
cp /Users/kratos/hls/src/components/PlayerSlot.vue /Users/kratos/hls/src/games/shop/components/PlayerSlot.vue
```

更新导入：
```ts
// 将
import Avatar from '@/components/Avatar.vue';
import { ROLE_LABEL } from '@/game/rules';
import type { Player } from '@/net/protocol';

// 改为
import Avatar from '@shared/components/Avatar.vue';
import { ROLE_LABEL } from '../game/rules';
import type { Player } from '../net/protocol';
```

- [ ] **Step 3: 复制并更新 FeedStream.vue**

```bash
cp /Users/kratos/hls/src/components/FeedStream.vue /Users/kratos/hls/src/games/shop/components/FeedStream.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';
import { getEquipment, getWeapon } from '@/game/equipment';

// 改为
import { useRoomStore } from '../stores/room';
import { getEquipment, getWeapon } from '../game/equipment';
```

- [ ] **Step 4: 复制并更新 EquipmentShop.vue**

```bash
cp /Users/kratos/hls/src/components/EquipmentShop.vue /Users/kratos/hls/src/games/shop/components/EquipmentShop.vue
```

更新导入：
```ts
// 将
import { EQUIPMENT_LIST, type Equipment } from '@/game/equipment';
import { useRoomStore } from '@/stores/room';

// 改为
import { EQUIPMENT_LIST, type Equipment } from '../game/equipment';
import { useRoomStore } from '../stores/room';
```

- [ ] **Step 5: 复制并更新 LotteryPanel.vue**

```bash
cp /Users/kratos/hls/src/components/LotteryPanel.vue /Users/kratos/hls/src/games/shop/components/LotteryPanel.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';
import { ... } from '@/game/lottery';
import type { LotteryItem, LotteryRecord } from '@/net/protocol';

// 改为
import { useRoomStore } from '../stores/room';
import { ... } from '../game/lottery';
import type { LotteryItem, LotteryRecord } from '../net/protocol';
```

（`...` 表示保持原有具名导入不变，只替换路径部分）

- [ ] **Step 6: 复制并更新 LotteryRecordModal.vue**

```bash
cp /Users/kratos/hls/src/components/LotteryRecordModal.vue /Users/kratos/hls/src/games/shop/components/LotteryRecordModal.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';
import { ... } from '@/game/lottery';

// 改为
import { useRoomStore } from '../stores/room';
import { ... } from '../game/lottery';
```

- [ ] **Step 7: 复制并更新 SettlementModal.vue**

```bash
cp /Users/kratos/hls/src/components/SettlementModal.vue /Users/kratos/hls/src/games/shop/components/SettlementModal.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';
import type { SettlementOutcome } from '@/net/protocol';
import { ROLE_LABEL } from '@/game/rules';
import Avatar from '@/components/Avatar.vue';

// 改为
import { useRoomStore } from '../stores/room';
import type { SettlementOutcome } from '../net/protocol';
import { ROLE_LABEL } from '../game/rules';
import Avatar from '@shared/components/Avatar.vue';
```

- [ ] **Step 8: 复制并更新 SettlementRecordModal.vue**

```bash
cp /Users/kratos/hls/src/components/SettlementRecordModal.vue /Users/kratos/hls/src/games/shop/components/SettlementRecordModal.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';

// 改为
import { useRoomStore } from '../stores/room';
```

- [ ] **Step 9: 提交**

```bash
git add src/games/shop/components/
git commit -m "chore: copy shop components to src/games/shop/components"
```

---

## Task 10: 迁移 shop/pages/

**Files:**
- Create: `src/games/shop/pages/ShopHomePage.vue`
- Create: `src/games/shop/pages/RoomPage.vue`
- Create: `src/games/shop/pages/PlayPage.vue`

- [ ] **Step 1: 复制并更新 ShopHomePage.vue（原 HomePage.vue）**

```bash
cp /Users/kratos/hls/src/pages/HomePage.vue /Users/kratos/hls/src/games/shop/pages/ShopHomePage.vue
```

更新导入：
```ts
// 将
import { isValidRoomCode } from '@/utils/id';
import { MAX_NICKNAME_LENGTH, usePlayerStore } from '@/stores/player';
import { useRoomStore } from '@/stores/room';
import { createRoom, queryRoom, ApiError, ... } from '@/net/api';
import ToastContainer from '@/components/ToastContainer.vue';

// 改为
import { isValidRoomCode } from '@shared/utils/id';
import { MAX_NICKNAME_LENGTH, usePlayerStore } from '@shared/stores/player';
import { useRoomStore } from '../stores/room';
import { createRoom, queryRoom, ApiError, ... } from '../net/api';
import ToastContainer from '../components/ToastContainer.vue';
```

- [ ] **Step 2: 复制并更新 RoomPage.vue**

```bash
cp /Users/kratos/hls/src/pages/RoomPage.vue /Users/kratos/hls/src/games/shop/pages/RoomPage.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';
import { NICKNAME_PRESETS, composeNickname, usePlayerStore } from '@/stores/player';
import PlayerSlot from '@/components/PlayerSlot.vue';
import ToastContainer from '@/components/ToastContainer.vue';
import ConfirmDialog from '@/components/ConfirmDialog.vue';
import { MAX_PLAYERS, summarizeRoster } from '@/game/rules';

// 改为
import { useRoomStore } from '../stores/room';
import { NICKNAME_PRESETS, composeNickname, usePlayerStore } from '@shared/stores/player';
import PlayerSlot from '../components/PlayerSlot.vue';
import ToastContainer from '../components/ToastContainer.vue';
import ConfirmDialog from '@shared/components/ConfirmDialog.vue';
import { MAX_PLAYERS, summarizeRoster } from '../game/rules';
```

- [ ] **Step 3: 复制并更新 PlayPage.vue**

```bash
cp /Users/kratos/hls/src/pages/PlayPage.vue /Users/kratos/hls/src/games/shop/pages/PlayPage.vue
```

更新导入：
```ts
// 将
import { useRoomStore } from '@/stores/room';
import { ROLE_LABEL } from '@/game/rules';
import Avatar from '@/components/Avatar.vue';
import LotteryPanel from '@/components/LotteryPanel.vue';
import EquipmentShop from '@/components/EquipmentShop.vue';
import FeedStream from '@/components/FeedStream.vue';
import LotteryRecordModal from '@/components/LotteryRecordModal.vue';
import SettlementModal from '@/components/SettlementModal.vue';
import SettlementRecordModal from '@/components/SettlementRecordModal.vue';
import ConfirmDialog from '@/components/ConfirmDialog.vue';
import ToastContainer from '@/components/ToastContainer.vue';
import { usePlayerStore } from '@/stores/player';

// 改为
import { useRoomStore } from '../stores/room';
import { ROLE_LABEL } from '../game/rules';
import Avatar from '@shared/components/Avatar.vue';
import LotteryPanel from '../components/LotteryPanel.vue';
import EquipmentShop from '../components/EquipmentShop.vue';
import FeedStream from '../components/FeedStream.vue';
import LotteryRecordModal from '../components/LotteryRecordModal.vue';
import SettlementModal from '../components/SettlementModal.vue';
import SettlementRecordModal from '../components/SettlementRecordModal.vue';
import ConfirmDialog from '@shared/components/ConfirmDialog.vue';
import ToastContainer from '../components/ToastContainer.vue';
import { usePlayerStore } from '@shared/stores/player';
```

- [ ] **Step 4: 更新 shop 页面内的路由 name 引用**

原路由名 `'room'`、`'play'` 已改为 `'shop-room'`、`'shop-play'`（见 Task 11）。需在已复制的 shop 页面中全局替换：

在 `src/games/shop/pages/ShopHomePage.vue` 中：
```ts
// 将所有
router.push({ name: 'room', ... })
// 改为
router.push({ name: 'shop-room', ... })
```

在 `src/games/shop/pages/RoomPage.vue` 中：
```ts
// 将所有
router.push({ name: 'play', ... })
// 改为
router.push({ name: 'shop-play', ... })
```

（`'home'` → `'home'` 不变，首页路由名保持不变）

- [ ] **Step 5: 提交**

```bash
git add src/games/shop/pages/
git commit -m "chore: copy shop pages to src/games/shop/pages"
```

---

## Task 11: 创建 shop/router.ts

**Files:**
- Create: `src/games/shop/router.ts`

- [ ] **Step 1: 创建 shop 路由文件**

```ts
// src/games/shop/router.ts
import type { RouteRecordRaw } from 'vue-router';

const shopRoutes: RouteRecordRaw[] = [
  {
    path: '/shop',
    name: 'shop-home',
    component: () => import('./pages/ShopHomePage.vue'),
  },
  {
    path: '/shop/room/:code',
    name: 'shop-room',
    component: () => import('./pages/RoomPage.vue'),
    props: true,
  },
  {
    path: '/shop/room/:code/play',
    name: 'shop-play',
    component: () => import('./pages/PlayPage.vue'),
    props: true,
  },
];

export default shopRoutes;
```

- [ ] **Step 2: 提交**

```bash
git add src/games/shop/router.ts
git commit -m "chore: add shop router"
```

---

## Task 12: 创建占位玩法页面与路由

**Files:**
- Create: `src/games/teach/pages/TeachPage.vue`
- Create: `src/games/teach/router.ts`
- Create: `src/games/audition/pages/AuditionPage.vue`
- Create: `src/games/audition/router.ts`
- Create: `src/games/poker/pages/PokerPage.vue`
- Create: `src/games/poker/router.ts`

- [ ] **Step 1: 创建 TeachPage.vue**

```vue
<!-- src/games/teach/pages/TeachPage.vue -->
<script setup lang="ts">
import { useRouter } from 'vue-router';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <div class="text-center select-none">
      <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
      <div class="text-4xl mb-4">🎓</div>
      <h1 class="rpg-title text-3xl font-bold mb-3">1V1 定制教学</h1>
      <div class="mx-auto max-w-xs rpg-divider mb-6" />
      <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
      <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 创建 teach/router.ts**

```ts
// src/games/teach/router.ts
import type { RouteRecordRaw } from 'vue-router';

const teachRoutes: RouteRecordRaw[] = [
  {
    path: '/teach',
    name: 'teach',
    component: () => import('./pages/TeachPage.vue'),
  },
];

export default teachRoutes;
```

- [ ] **Step 3: 创建 AuditionPage.vue**

```vue
<!-- src/games/audition/pages/AuditionPage.vue -->
<script setup lang="ts">
import { useRouter } from 'vue-router';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <div class="text-center select-none">
      <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
      <div class="text-4xl mb-4">🎵</div>
      <h1 class="rpg-title text-3xl font-bold mb-3">试音</h1>
      <div class="mx-auto max-w-xs rpg-divider mb-6" />
      <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
      <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
    </div>
  </div>
</template>
```

- [ ] **Step 4: 创建 audition/router.ts**

```ts
// src/games/audition/router.ts
import type { RouteRecordRaw } from 'vue-router';

const auditionRoutes: RouteRecordRaw[] = [
  {
    path: '/audition',
    name: 'audition',
    component: () => import('./pages/AuditionPage.vue'),
  },
];

export default auditionRoutes;
```

- [ ] **Step 5: 创建 PokerPage.vue**

```vue
<!-- src/games/poker/pages/PokerPage.vue -->
<script setup lang="ts">
import { useRouter } from 'vue-router';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <div class="text-center select-none">
      <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
      <div class="text-4xl mb-4">🃏</div>
      <h1 class="rpg-title text-3xl font-bold mb-3">摸红扑克牌</h1>
      <div class="mx-auto max-w-xs rpg-divider mb-6" />
      <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
      <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
    </div>
  </div>
</template>
```

- [ ] **Step 6: 创建 poker/router.ts**

```ts
// src/games/poker/router.ts
import type { RouteRecordRaw } from 'vue-router';

const pokerRoutes: RouteRecordRaw[] = [
  {
    path: '/poker',
    name: 'poker',
    component: () => import('./pages/PokerPage.vue'),
  },
];

export default pokerRoutes;
```

- [ ] **Step 7: 提交**

```bash
git add src/games/teach/ src/games/audition/ src/games/poker/
git commit -m "feat: add placeholder pages for teach, audition, poker"
```

---

## Task 13: 创建 LandingPage.vue（官方首页）

**Files:**
- Create: `src/pages/LandingPage.vue`

- [ ] **Step 1: 创建首页**

```vue
<!-- src/pages/LandingPage.vue -->
<script setup lang="ts">
import { useRouter } from 'vue-router';

const router = useRouter();

const games = [
  {
    key: 'shop',
    icon: '🎲',
    title: '神秘商店',
    desc: '趣味单 · 三人对局\n抽奖 · 装备 · 结算',
    route: '/shop',
    active: true,
  },
  {
    key: 'teach',
    icon: '🎓',
    title: '1V1 定制教学',
    desc: '专属教练 · 针对性训练\n技术进阶',
    route: '/teach',
    active: false,
  },
  {
    key: 'audition',
    icon: '🎵',
    title: '试音',
    desc: '与花栗鼠一起玩\n找到最合拍的搭档',
    route: '/audition',
    active: false,
  },
  {
    key: 'poker',
    icon: '🃏',
    title: '摸红扑克牌',
    desc: '原创趣味玩法\n手气决定一切',
    route: '/poker',
    active: false,
  },
] as const;
</script>

<template>
  <div class="min-h-screen flex flex-col">

    <!-- 顶部导航 -->
    <nav class="flex justify-between items-center px-6 py-4 border-b border-white/5">
      <span class="rpg-subtitle text-sm tracking-widest text-gold-300">✦ 花栗鼠俱乐部</span>
      <div class="flex gap-6 font-serif text-xs tracking-widest text-parchment-400/60">
        <a href="#games" class="hover:text-gold-300 transition-colors">玩法</a>
        <span>微信服务号</span>
      </div>
    </nav>

    <!-- Hero -->
    <header class="flex-none px-6 pt-14 pb-10 text-center select-none border-b border-white/5">
      <div class="rpg-subtitle text-[10px] sm:text-xs mb-4 animate-flicker tracking-widest">
        ✦ 花栗鼠俱乐部 · 三角洲行动 ✦
      </div>
      <h1 class="rpg-title text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em]">
        花栗鼠俱乐部
      </h1>
      <div class="mt-5 mx-auto max-w-xs rpg-divider" />
      <p class="mt-5 font-serif text-sm sm:text-base text-gold-300 tracking-wide leading-relaxed">
        开一家最有温度的三角洲陪玩店
      </p>
      <p class="mt-2 font-body text-xs text-parchment-400/60 tracking-widest">
        原创 1V1 定制教学单 · 神秘商店趣味单
      </p>
    </header>

    <!-- 玩法卡片 -->
    <main id="games" class="flex-1 px-4 py-10">
      <div class="mx-auto max-w-lg">
        <div class="font-serif text-[10px] tracking-[0.4em] uppercase text-parchment-400/50 mb-6 text-center">
          <span class="rune-mark mr-2">❖</span>我们的玩法<span class="rune-mark ml-2">❖</span>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <button
            v-for="game in games"
            :key="game.key"
            class="card rpg-corners text-left p-5 transition-all duration-200"
            :class="game.active
              ? 'border-gold-400/60 hover:border-gold-300 hover:shadow-gold cursor-pointer'
              : 'border-white/8 opacity-60 cursor-pointer hover:opacity-75'"
            @click="router.push(game.route)"
          >
            <div class="text-2xl mb-3">{{ game.icon }}</div>
            <div
              class="font-serif text-sm font-bold mb-2 leading-snug"
              :class="game.active ? 'text-parchment-100' : 'text-parchment-400/70'"
            >
              {{ game.title }}
            </div>
            <div class="font-body text-[10px] leading-relaxed whitespace-pre-line"
              :class="game.active ? 'text-parchment-300/60' : 'text-parchment-400/40'"
            >
              {{ game.desc }}
            </div>
            <div class="mt-3">
              <span
                v-if="game.active"
                class="inline-block font-serif text-[10px] text-gold-300 tracking-wider"
              >立即开始 →</span>
              <span
                v-else
                class="inline-block font-serif text-[10px] text-parchment-400/35 tracking-wider"
              >即将推出</span>
            </div>
          </button>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="text-center font-serif text-[11px] tracking-[0.4em] uppercase text-parchment-400/40 py-5 border-t border-white/5">
      <span class="rune-mark mr-2">✥</span>微信服务号 · 花栗鼠俱乐部<span class="rune-mark ml-2">✥</span>
    </footer>

  </div>
</template>
```

- [ ] **Step 2: 提交**

```bash
git add src/pages/LandingPage.vue
git commit -m "feat: add LandingPage as official club homepage"
```

---

## Task 14: 更新 src/router/index.ts

**Files:**
- Modify: `src/router/index.ts`

- [ ] **Step 1: 重写路由文件**

```ts
// src/router/index.ts
import { createRouter, createWebHistory } from 'vue-router';
import shopRoutes from '@/games/shop/router';
import teachRoutes from '@/games/teach/router';
import auditionRoutes from '@/games/audition/router';
import pokerRoutes from '@/games/poker/router';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/LandingPage.vue'),
    },
    ...shopRoutes,
    ...teachRoutes,
    ...auditionRoutes,
    ...pokerRoutes,
    { path: '/:catchAll(.*)', redirect: '/' },
  ],
});
```

- [ ] **Step 2: 验证 typecheck（旧文件仍存在，不应报错）**

```bash
cd /Users/kratos/hls && npm run typecheck
```

期望：无错误

- [ ] **Step 3: 提交**

```bash
git add src/router/index.ts
git commit -m "feat: update router to use modular game routes"
```

---

## Task 15: 更新 src/main.ts

**Files:**
- Modify: `src/main.ts`

- [ ] **Step 1: 更新 styles 导入路径**

将：
```ts
import './styles/main.css';
```
改为：
```ts
import '@shared/styles/main.css';
```

- [ ] **Step 2: 提交**

```bash
git add src/main.ts
git commit -m "chore: update main.ts styles import to shared path"
```

---

## Task 16: 删除旧文件，最终验证

**Files:**
- Delete: `src/components/`（整个目录）
- Delete: `src/pages/HomePage.vue`, `src/pages/RoomPage.vue`, `src/pages/PlayPage.vue`
- Delete: `src/net/`
- Delete: `src/game/`
- Delete: `src/stores/`
- Delete: `src/utils/`
- Delete: `src/styles/`

- [ ] **Step 1: 删除旧目录**

```bash
rm -rf /Users/kratos/hls/src/components
rm -rf /Users/kratos/hls/src/net
rm -rf /Users/kratos/hls/src/game
rm -rf /Users/kratos/hls/src/stores
rm -rf /Users/kratos/hls/src/utils
rm -rf /Users/kratos/hls/src/styles
rm -f /Users/kratos/hls/src/pages/HomePage.vue
rm -f /Users/kratos/hls/src/pages/RoomPage.vue
rm -f /Users/kratos/hls/src/pages/PlayPage.vue
```

- [ ] **Step 2: 运行 typecheck**

```bash
cd /Users/kratos/hls && npm run typecheck
```

期望：无错误。若有报错，根据错误信息找到漏更新的 import 路径并修复。

- [ ] **Step 3: 运行构建验证**

```bash
cd /Users/kratos/hls && npm run build
```

期望：构建成功，`dist/` 目录生成。

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "refactor: remove old src directories after full migration to games/shared structure"
```

---

## 完成验证清单

构建成功后，通过 `npm run preview` 启动预览，手动检查以下路由：

- [ ] `/` → 显示花栗鼠俱乐部首页，四个玩法卡片可见
- [ ] 点击"神秘商店"卡片 → 跳转到 `/shop`，显示商店创建/加入界面
- [ ] `/shop/room/123456`（任意 6 位数字）→ 不报 JS 错误（会因 WS 连接失败而回首页，这是正常行为）
- [ ] `/teach` → 显示"即将推出"占位页，"返回首页"按钮可用
- [ ] `/audition` → 同上
- [ ] `/poker` → 同上
- [ ] `/any-invalid-route` → 重定向到 `/`

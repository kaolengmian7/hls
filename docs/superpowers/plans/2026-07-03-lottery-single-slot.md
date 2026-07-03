# 神秘商店老虎机单格大图 + CSGO 动效 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将老虎机从 3 行窗口改为单格全幅（240px），加入 CSGO 风格「快速飞过 1s → 极慢减速 3s」动效，开奖停稳后显示脉冲光晕 + 四角 ✦ 装饰。

**Architecture:** 只修改 `LotteryPanel.vue` 一个文件，分两步：先改 `<script>` 中的动画参数和缓动函数，再改 `<template>` 中的图片尺寸、遮罩高度、结果框样式、角装饰和 CSS 动画。抽奖流程（`drawOne` / `tick` / `beginLanding` / `watch`）逻辑不变。

**Tech Stack:** Vue 3 `<script setup>`, Tailwind CSS, requestAnimationFrame

---

## 文件

- Modify: `src/games/shop/components/LotteryPanel.vue`（全部改动集中在此一个文件）

---

### Task 1: 更新动画参数与缓动函数

**Files:**
- Modify: `src/games/shop/components/LotteryPanel.vue:15-22, 58-60, 130`

- [ ] **Step 1: 替换常量块（第 15–22 行）**

将以下 7 个常量全部替换：

```ts
// 旧
const ITEM_H = 96;
const VISIBLE_ROWS = 3;
const CENTER_ROW = Math.floor(VISIBLE_ROWS / 2);
const VIEWPORT_H = ITEM_H * VISIBLE_ROWS;
const SPIN_SPEED = ITEM_H * 17;
const MIN_SPIN_MS = 700;
const LAND_MS = 2300;
const LAND_ITEMS = 16;

// 新
const ITEM_H = 240;
const VISIBLE_ROWS = 1;
const CENTER_ROW = Math.floor(VISIBLE_ROWS / 2); // → 0，不用改
const VIEWPORT_H = ITEM_H * VISIBLE_ROWS;         // → 240，不用改
const SPIN_SPEED = ITEM_H * 22;
const MIN_SPIN_MS = 1000;
const LAND_MS = 3000;
const LAND_ITEMS = 22;
```

- [ ] **Step 2: 将 `easeOutQuart` 改为 `easeOutSextic`（第 58–60 行）**

```ts
// 旧
function easeOutQuart(t: number): number {
  return 1 - Math.pow(1 - t, 4);
}

// 新
function easeOutSextic(t: number): number {
  return 1 - Math.pow(1 - t, 6);
}
```

- [ ] **Step 3: 更新 `tick` 中的缓动调用（第 130 行）**

```ts
// 旧
offset.value = landStartOffset + (landTargetOffset - landStartOffset) * easeOutQuart(t);

// 新
offset.value = landStartOffset + (landTargetOffset - landStartOffset) * easeOutSextic(t);
```

- [ ] **Step 4: 确认 TypeScript 无报错**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

预期：无输出（0 errors）

- [ ] **Step 5: Commit**

```bash
git add src/games/shop/components/LotteryPanel.vue
git commit -m "feat(LotteryPanel): single-slot params + easeOutSextic for CSGO-style decel"
```

---

### Task 2: 更新模板——图片、遮罩、结果框、四角装饰、CSS 动画

**Files:**
- Modify: `src/games/shop/components/LotteryPanel.vue:254-311`（template 段）+ 新增 `<style>` 块

- [ ] **Step 1: 放大图片与占位符（第 254–266 行）**

```html
<!-- 旧 -->
<img
  v-if="row.imageUrl && !failedImages.has(row.imageUrl)"
  :src="row.imageUrl"
  :alt="row.name"
  style="width:64px;height:64px;border-radius:10px;object-fit:contain;"
  @error="failedImages.add(row.imageUrl!)"
/>
<div
  v-else
  style="width:64px;height:64px;border-radius:10px;background:rgba(255,255,255,0.06);display:flex;align-items:center;justify-content:center;font-size:30px;"
>
  🎁
</div>

<!-- 新 -->
<img
  v-if="row.imageUrl && !failedImages.has(row.imageUrl)"
  :src="row.imageUrl"
  :alt="row.name"
  style="width:160px;height:160px;border-radius:16px;object-fit:contain;"
  @error="failedImages.add(row.imageUrl!)"
/>
<div
  v-else
  style="width:160px;height:160px;border-radius:16px;background:rgba(255,255,255,0.06);display:flex;align-items:center;justify-content:center;font-size:80px;"
>
  🎁
</div>
```

- [ ] **Step 2: 缩小上下遮罩至 28px（第 270–283 行）**

```html
<!-- 旧：height: ITEM_H + 'px'（= 240px，会完全遮住唯一的格子） -->
<div
  class="pointer-events-none absolute inset-x-0 top-0"
  :style="{
    height: ITEM_H + 'px',
    background: 'linear-gradient(180deg, #1a0f0a 5%, rgba(26,15,10,0))',
  }"
/>
<div
  class="pointer-events-none absolute inset-x-0 bottom-0"
  :style="{
    height: ITEM_H + 'px',
    background: 'linear-gradient(0deg, #1a0f0a 5%, rgba(26,15,10,0))',
  }"
/>

<!-- 新：固定 28px，只盖边缘 -->
<div
  class="pointer-events-none absolute inset-x-0 top-0"
  style="height:28px;background:linear-gradient(180deg,#1a0f0a,rgba(26,15,10,0));"
/>
<div
  class="pointer-events-none absolute inset-x-0 bottom-0"
  style="height:28px;background:linear-gradient(0deg,#1a0f0a,rgba(26,15,10,0));"
/>
```

- [ ] **Step 3: 增强结果框并加入四角 ✦ 装饰（第 286–311 行）**

```html
<!-- 旧 -->
<div
  class="pointer-events-none absolute inset-x-0 transition-all duration-300"
  :style="{
    top: CENTER_ROW * ITEM_H + 'px',
    height: ITEM_H + 'px',
    border: revealed
      ? '1px solid rgba(212,175,55,0.85)'
      : '1px solid rgba(212,175,55,0.35)',
    boxShadow: revealed
      ? 'inset 0 0 26px rgba(249,115,22,0.35), 0 0 18px rgba(212,175,55,0.45)'
      : 'inset 0 0 18px rgba(0,0,0,0.45)',
    background: revealed ? 'rgba(249,115,22,0.08)' : 'transparent',
  }"
>
  <span
    class="absolute left-1 top-1/2 -translate-y-1/2 text-gold-400/80 text-xs"
    :class="spinning ? 'animate-flicker' : ''"
    >▶</span
  >
  <span
    class="absolute right-1 top-1/2 -translate-y-1/2 text-gold-400/80 text-xs"
    :class="spinning ? 'animate-flicker' : ''"
    >◀</span
  >
</div>

<!-- 新 -->
<div
  class="pointer-events-none absolute inset-x-0 transition-all duration-300"
  :class="{ 'lottery-reveal-pulse': revealed }"
  :style="{
    top: CENTER_ROW * ITEM_H + 'px',
    height: ITEM_H + 'px',
    border: revealed
      ? '1.5px solid rgba(212,175,55,0.95)'
      : '1px solid rgba(212,175,55,0.35)',
    boxShadow: revealed
      ? 'inset 0 0 56px rgba(249,115,22,0.4), 0 0 36px rgba(212,175,55,0.55)'
      : 'inset 0 0 18px rgba(0,0,0,0.45)',
    background: revealed ? 'rgba(249,115,22,0.06)' : 'transparent',
  }"
>
  <span
    class="absolute left-1 top-1/2 -translate-y-1/2 text-gold-400/80 text-xs"
    :class="spinning ? 'animate-flicker' : ''"
    >▶</span
  >
  <span
    class="absolute right-1 top-1/2 -translate-y-1/2 text-gold-400/80 text-xs"
    :class="spinning ? 'animate-flicker' : ''"
    >◀</span
  >
  <!-- 四角装饰：revealed 时淡入 -->
  <span
    class="absolute text-gold-400 text-sm transition-opacity duration-500"
    :style="{ opacity: revealed ? 0.9 : 0, top: '8px', left: '8px' }"
    >✦</span
  >
  <span
    class="absolute text-gold-400 text-sm transition-opacity duration-500"
    :style="{ opacity: revealed ? 0.9 : 0, top: '8px', right: '8px' }"
    >✦</span
  >
  <span
    class="absolute text-gold-400 text-sm transition-opacity duration-500"
    :style="{ opacity: revealed ? 0.9 : 0, bottom: '8px', left: '8px' }"
    >✦</span
  >
  <span
    class="absolute text-gold-400 text-sm transition-opacity duration-500"
    :style="{ opacity: revealed ? 0.9 : 0, bottom: '8px', right: '8px' }"
    >✦</span
  >
</div>
```

- [ ] **Step 4: 在文件末尾添加 `<style>` 块**

在 `</template>` 之后追加：

```html
<style scoped>
.lottery-reveal-pulse {
  animation: lotteryRevealPulse 1.8s ease-in-out infinite;
}

@keyframes lotteryRevealPulse {
  0%, 100% {
    box-shadow: inset 0 0 56px rgba(249, 115, 22, 0.4), 0 0 36px rgba(212, 175, 55, 0.55);
  }
  50% {
    box-shadow: inset 0 0 80px rgba(249, 115, 22, 0.6), 0 0 64px rgba(212, 175, 55, 0.9);
  }
}
</style>
```

- [ ] **Step 5: 确认 TypeScript 无报错**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

预期：无输出（0 errors）

- [ ] **Step 6: 启动开发服务器，手动验证**

```bash
cd /Users/kratos/hls && npm run dev
```

进入神秘商店页面，点击「开启神秘商店」，确认：
1. 单格大图（160px）居中显示
2. 快速飞过约 1 秒
3. 减速约 3 秒，最后几格极慢
4. 停稳后边框变金、四角 ✦ 淡入、框持续脉冲光晕

- [ ] **Step 7: Commit**

```bash
git add src/games/shop/components/LotteryPanel.vue
git commit -m "feat(LotteryPanel): single-slot 240px display with CSGO decel and reveal glow"
```

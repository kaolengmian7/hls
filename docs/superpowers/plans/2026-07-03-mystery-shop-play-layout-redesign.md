# Mystery Shop Play Page Layout Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the PlayPage layout so game stats (保底/得吃) and action buttons move from the header into the left sidebar, the three-column proportions become balanced, and the feed stream fills available height.

**Architecture:** All changes are purely template/CSS — no logic, store, or protocol changes. `PlayPage.vue` receives the bulk of the edits (header strip-down, grid widths, left aside rebuild). `FeedStream.vue` gets a one-line container class fix. No new components needed.

**Tech Stack:** Vue 3 (Composition API), Tailwind CSS utility classes, existing `rpg-*` / `btn-*` / `badge` / `coin-tag` design tokens, `roomStore` (already imported).

---

### Task 1: Adjust grid column widths and mobile order

**Files:**
- Modify: `src/games/shop/pages/PlayPage.vue` (line 171, 172, 248)

- [ ] **Step 1: Update the `<main>` grid class**

Find line 171 in `PlayPage.vue`:
```html
<main class="flex-1 px-4 sm:px-8 py-6 grid grid-cols-1 lg:grid-cols-[260px_1fr_340px] gap-6 max-w-7xl mx-auto w-full">
```
Replace with:
```html
<main class="flex-1 px-4 sm:px-8 py-6 grid grid-cols-1 lg:grid-cols-[220px_1fr_260px] gap-6 max-w-7xl mx-auto w-full">
```

- [ ] **Step 2: Fix left aside mobile order**

Find line 172:
```html
<aside class="order-2 lg:order-1">
```
The left panel should render after the main content on mobile (so the slot machine is first). This is already correct (`order-2` on mobile, `order-1` on desktop). No change needed — verify it matches the above.

- [ ] **Step 3: Fix right aside order**

Find line 248:
```html
<aside class="order-3">
```
This is already correct. Verify it matches.

- [ ] **Step 4: Run dev server and confirm the three-column proportions look balanced**

```bash
npm run dev
```
Open the play page (you'll need a running game session). Confirm: left ≈ 220px, center expands to fill, right ≈ 260px.

- [ ] **Step 5: Commit**

```bash
git add src/games/shop/pages/PlayPage.vue
git commit -m "feat: tighten sidebar widths to [220px_1fr_260px] for balanced layout"
```

---

### Task 2: Simplify the header — remove stats and action buttons

**Files:**
- Modify: `src/games/shop/pages/PlayPage.vue` (lines 130–168)

The right half of the header currently contains two stat badges (保底, 得吃) and four buttons (本轮结算, 结算记录, 抽奖记录, 离开). Keep only 离开; the rest move to the left aside in Task 3.

- [ ] **Step 1: Replace the header's right-side `<div>` block**

Find this block in the header (starts around line 130):
```html
          <div class="flex items-center gap-2 flex-wrap">
            <div
              class="rounded-md px-2.5 py-1 leading-tight text-right border border-gold-500/40 bg-ink-700/70"
              :title="`商店保底`"
            >
              <div class="font-serif text-[10px] uppercase tracking-[0.3em] text-gold-300/80">
                保底
              </div>
              <div class="text-sm font-bold font-serif text-gold-200">
                {{ roomStore.guaranteedAmount }}<span class="text-gold-300/80 ml-0.5">W</span>
              </div>
            </div>
            <div
              class="rounded-md px-2.5 py-1 leading-tight text-right border border-forest-400/40 bg-ink-700/70"
              :title="`历次撤离成功累计扣减额`"
            >
              <div class="font-serif text-[10px] uppercase tracking-[0.3em] text-forest-300/70">
                得吃
              </div>
              <div class="text-sm font-bold font-serif text-forest-300">
                {{ roomStore.evacuatedTotal }}<span class="text-forest-300/80 ml-0.5">W</span>
              </div>
            </div>
            <button
              v-if="canSettle"
              class="btn-boss py-1.5 px-3 text-[11px]"
              @click="showSettlement = true"
            >
              ⚖ 本轮结算
            </button>
            <button class="btn-ghost text-[11px]" @click="showSettlementHistory = true">
              结算记录
            </button>
            <button class="btn-ghost text-[11px]" @click="showLottery = true">抽奖记录</button>
            <button class="btn-ghost text-oxblood-300 hover:!text-oxblood-300 text-[11px]" @click="showLeaveConfirm = true">
              离开
            </button>
          </div>
```

Replace the entire block with:
```html
          <div class="flex items-center gap-2">
            <button class="btn-ghost text-oxblood-300 hover:!text-oxblood-300 text-[11px]" @click="showLeaveConfirm = true">
              离开
            </button>
          </div>
```

- [ ] **Step 2: Verify the header renders cleanly**

In the browser, the header should now show only: ⚜ 花栗鼠的神秘商店 · 口令 · XXXX | 离开

- [ ] **Step 3: Commit**

```bash
git add src/games/shop/pages/PlayPage.vue
git commit -m "feat: strip stats and action buttons from header"
```

---

### Task 3: Rebuild left aside — stats, operations, and members

**Files:**
- Modify: `src/games/shop/pages/PlayPage.vue` (the `<aside class="order-2 lg:order-1">` block)

Replace the entire contents of the left aside. The card now has three sections separated by dividers: 本局数据 → 操作 → 店内成员.

- [ ] **Step 1: Replace the left aside inner content**

Find the left aside block:
```html
        <aside class="order-2 lg:order-1">
          <div class="card rpg-corners p-4 lg:sticky lg:top-24">
            <h3 class="rpg-title text-base flex items-center gap-2">
              <span class="rune-mark">⚜</span>店内成员
            </h3>
            <div class="rpg-divider my-3" />
            <div class="space-y-2.5">
              <div
                v-for="p in roster"
                :key="p.id"
                class="rounded-md p-2.5 flex items-center gap-3 border transition-colors"
                :class="
                  p.id === roomStore.selfPlayerId
                    ? 'border-gold-400/70 bg-gold-500/10'
                    : 'border-gold-500/25 bg-ink-700/60'
                "
              >
                <div class="relative shrink-0">
                  <Avatar :seed="p.id" :name="p.nickname" size="md" />
                  <span
                    v-if="!p.connected"
                    class="absolute -bottom-1 -right-1 w-3.5 h-3.5 rounded-full bg-oxblood-500 border border-ink-700"
                    title="失联中"
                  />
                </div>
                <div class="flex-1 min-w-0">
                  <div class="font-serif font-semibold text-parchment-50 flex flex-wrap items-center gap-1">
                    <span v-if="p.isHost" class="text-gold-300 shrink-0" title="房主">👑</span>
                    <span class="break-all">{{ p.nickname }}</span>
                    <span
                      v-if="p.id === roomStore.selfPlayerId"
                      class="badge !text-gold-200 !border-gold-400/60 text-[10px] shrink-0"
                    >
                      我
                    </span>
                  </div>
                  <div class="flex items-center gap-1.5 mt-1">
                    <span
                      class="badge text-[10px]"
                      :class="
                        p.role === 'BOSS'
                          ? '!border-ember-400/60 !text-ember-300'
                          : '!border-sapphire-300/60 !text-sapphire-300'
                      "
                    >
                      {{ p.role ? ROLE_LABEL[p.role] : '未择阶' }}
                    </span>
                  </div>
                  <div class="coin-tag text-sm mt-1.5">
                    <span>💰</span> {{ p.coin }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </aside>
```

Replace with:
```html
        <aside class="order-2 lg:order-1">
          <div class="card rpg-corners p-4 lg:sticky lg:top-24 flex flex-col gap-4">

            <!-- 本局数据 -->
            <div>
              <div class="font-serif text-[10px] tracking-[0.2em] uppercase text-gold-300/60 mb-2">
                <span class="rune-mark mr-1">⚜</span>本局数据
              </div>
              <div class="grid grid-cols-2 gap-2">
                <div
                  class="rounded-md px-2.5 py-2 text-center border border-gold-500/40 bg-ink-700/60"
                  title="商店保底"
                >
                  <div class="font-serif text-[10px] uppercase tracking-[0.2em] text-gold-300/80">保底</div>
                  <div class="text-base font-bold font-serif text-gold-200 mt-0.5">
                    {{ roomStore.guaranteedAmount }}<span class="text-gold-300/80 ml-0.5 text-[11px]">W</span>
                  </div>
                </div>
                <div
                  class="rounded-md px-2.5 py-2 text-center border border-forest-400/40 bg-ink-700/60"
                  title="历次撤离成功累计扣减额"
                >
                  <div class="font-serif text-[10px] uppercase tracking-[0.2em] text-forest-300/70">得吃</div>
                  <div class="text-base font-bold font-serif text-forest-300 mt-0.5">
                    {{ roomStore.evacuatedTotal }}<span class="text-forest-300/80 ml-0.5 text-[11px]">W</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 操作 -->
            <div>
              <div class="font-serif text-[10px] tracking-[0.2em] uppercase text-gold-300/60 mb-2">
                🗂 操作
              </div>
              <div class="flex flex-wrap gap-2">
                <button
                  v-if="canSettle"
                  class="btn-boss py-1.5 px-3 text-[11px]"
                  @click="showSettlement = true"
                >
                  ⚖ 本轮结算
                </button>
                <button class="btn-ghost text-[11px]" @click="showSettlementHistory = true">结算记录</button>
                <button class="btn-ghost text-[11px]" @click="showLottery = true">抽奖记录</button>
              </div>
            </div>

            <div class="rpg-divider" />

            <!-- 店内成员 -->
            <div>
              <div class="font-serif text-[10px] tracking-[0.2em] uppercase text-gold-300/60 mb-2">
                👥 店内成员
              </div>
              <div class="grid grid-cols-3 gap-2 lg:grid-cols-1">
                <div
                  v-for="p in roster"
                  :key="p.id"
                  class="rounded-md p-2 border transition-colors flex flex-col lg:flex-row items-center gap-1.5 lg:gap-3 lg:p-2.5"
                  :class="
                    p.id === roomStore.selfPlayerId
                      ? 'border-gold-400/70 bg-gold-500/10'
                      : 'border-gold-500/25 bg-ink-700/60'
                  "
                >
                  <div class="relative shrink-0">
                    <Avatar :seed="p.id" :name="p.nickname" size="md" />
                    <span
                      v-if="!p.connected"
                      class="absolute -bottom-1 -right-1 w-3.5 h-3.5 rounded-full bg-oxblood-500 border border-ink-700"
                      title="失联中"
                    />
                  </div>
                  <div class="flex-1 min-w-0 w-full">
                    <div class="font-serif font-semibold text-parchment-50 flex flex-wrap items-center gap-1 justify-center lg:justify-start">
                      <span v-if="p.isHost" class="text-gold-300 shrink-0" title="房主">👑</span>
                      <span class="truncate text-xs lg:text-sm lg:break-all lg:whitespace-normal">{{ p.nickname }}</span>
                      <span
                        v-if="p.id === roomStore.selfPlayerId"
                        class="badge !text-gold-200 !border-gold-400/60 text-[10px] shrink-0"
                      >
                        我
                      </span>
                    </div>
                    <div class="flex items-center gap-1.5 mt-1 justify-center lg:justify-start">
                      <span
                        class="badge text-[10px]"
                        :class="
                          p.role === 'BOSS'
                            ? '!border-ember-400/60 !text-ember-300'
                            : '!border-sapphire-300/60 !text-sapphire-300'
                        "
                      >
                        {{ p.role ? ROLE_LABEL[p.role] : '未择阶' }}
                      </span>
                    </div>
                    <div class="coin-tag text-xs mt-1 lg:text-sm">
                      <span>💰</span> {{ p.coin }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

          </div>
        </aside>
```

- [ ] **Step 2: Verify in browser — desktop**

On desktop (≥ 1024px):
- Left aside shows: 保底/得吃 side by side → 操作按钮 → divider → 成员竖排
- 「本轮结算」only visible when you're host and game is PLAYING

- [ ] **Step 3: Verify in browser — mobile**

Shrink browser to ≤ 375px width (or use DevTools mobile mode):
- Page order: center (slot/equipment) first, then the left panel (stacked), then feed
- Inside the left panel, member cards are 3-column grid (horizontal), each card is vertical (avatar on top, name + role + coin below)

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/pages/PlayPage.vue
git commit -m "feat: merge stats and action buttons into left sidebar panel"
```

---

### Task 4: Fix FeedStream height to fill available space

**Files:**
- Modify: `src/games/shop/components/FeedStream.vue` (lines 25–27)

Currently the feed's scroll container has `max-h-[420px]`, leaving blank space on tall screens. We make it flex-fill instead, and add sticky positioning to match the left aside.

- [ ] **Step 1: Update the outer card container in FeedStream.vue**

Find the root element (line 25):
```html
  <div class="card rpg-corners p-5 flex flex-col">
```
Replace with:
```html
  <div class="card rpg-corners p-5 flex flex-col lg:sticky lg:top-24">
```

- [ ] **Step 2: Update the scroll container**

Find (line 35):
```html
    <div class="flex-1 overflow-y-auto max-h-[420px] pr-1 space-y-2">
```
Replace with:
```html
    <div class="flex-1 overflow-y-auto max-h-[60vh] lg:max-h-[calc(100vh-8rem)] pr-1 space-y-2">
```

> Note: `max-h-[60vh]` caps the feed on mobile so it doesn't push content off screen. On desktop (`lg:`), `calc(100vh-8rem)` accounts for the sticky header (~4rem) and some breathing room. This is better than `max-h-none` which would cause the feed to grow unbounded if there are many events.

- [ ] **Step 3: Verify in browser — desktop tall screen**

On a tall window (e.g., 1080px+), the feed should now extend further down and show more events without a large gap below.

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/components/FeedStream.vue
git commit -m "feat: make feed stream fill available height with sticky positioning"
```

---

## Acceptance Checklist

Run through these after all tasks complete:

- [ ] Desktop ≥ 1024px: three columns visible, proportions `220px | 1fr | 260px`
- [ ] Left aside: 保底 + 得吃 shown as 2-col grid at top
- [ ] Left aside: 操作 buttons below stats; 「本轮结算」only visible to host during PLAYING
- [ ] Left aside: 店内成员 below divider, cards vertically stacked
- [ ] Header: only ⚜ title + 口令 + 离开 button — no stat badges, no record buttons
- [ ] Mobile ≤ 768px: slot machine / equipment first, then panel, then feed
- [ ] Mobile: member cards display in 3-column grid (compact vertical cards)
- [ ] Feed stream: no large blank area below last event on tall screens
- [ ] Feed stream: sticky on desktop, scrolls independently when content overflows

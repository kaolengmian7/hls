<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { useRoomStore } from '../stores/room';
import {
  SINGLE_DRAW_COST,
  formatGuaranteeShift,
  formatValueW,
} from '../game/lottery';
import type { LotteryItem, LotteryRecord } from '../net/protocol';

const roomStore = useRoomStore();

/* ------------------------------ 老虎机参数 ------------------------------ */

const ITEM_H = 240; // 每格高度（px）
const VISIBLE_ROWS = 1; // 窗口可见行数（奇数，中间行为结果行）
const CENTER_ROW = Math.floor(VISIBLE_ROWS / 2);
const VIEWPORT_H = ITEM_H * VISIBLE_ROWS;
const SPIN_SPEED = ITEM_H * 22; // 滚动速度（px/s），制造高速模糊
const MIN_SPIN_MS = 1000; // 即使结果秒回，也至少滚动这么久
const LAND_MS = 3000; // 减速着陆时长
const LAND_ITEMS = 22; // 着陆阶段额外滚过的格数
const RECYCLE_AT = 20; // 顶部消耗超过这么多格就回收，保持位移有界
const RESULT_TIMEOUT_MS = 6000; // 抽奖被拒/无响应的安全复位

/* ------------------------------ 响应式状态 ------------------------------ */

type Phase = 'idle' | 'spinning' | 'landing' | 'revealed';

const phase = ref<Phase>('idle');
const rows = ref<LotteryItem[]>([]);
const offset = ref(0); // 卷轴向上滚动的像素位移
const result = ref<LotteryRecord | null>(null);
const failedImages = reactive(new Set<string>());

const myCoin = computed(() => roomStore.me?.coin ?? 0);
const canDraw = computed(
  () =>
    (phase.value === 'idle' || phase.value === 'revealed') &&
    myCoin.value >= SINGLE_DRAW_COST,
);
const spinning = computed(() => phase.value === 'spinning' || phase.value === 'landing');
const stripStyle = computed(() => ({ transform: `translateY(${-offset.value}px)` }));

/* ------------------------------ 动画内部状态 ------------------------------ */

let fillerPool: LotteryItem[] = [];
let rafId = 0;
let timeoutId = 0;
let lastTs = 0;
let spinStartTs = 0;
let landStartTs = 0;
let landStartOffset = 0;
let landTargetOffset = 0;
let awaiting = false;
let landing = false;

function easeOutSextic(t: number): number {
  return 1 - Math.pow(1 - t, 6);
}

function randItem(pool: LotteryItem[]): LotteryItem {
  return pool[Math.floor(Math.random() * pool.length)]!;
}

function makeFillers(pool: LotteryItem[], n: number): LotteryItem[] {
  return Array.from({ length: n }, () => randItem(pool));
}

function buildFillerPool(): LotteryItem[] {
  const seen = new Set<string>();
  const pool: LotteryItem[] = [];
  for (const rec of roomStore.lotteryRecords) {
    const it = rec.item;
    if (it && !seen.has(it.id)) {
      seen.add(it.id);
      pool.push(it);
    }
  }
  for (const it of roomStore.lotteryPool) {
    if (!seen.has(it.id)) {
      seen.add(it.id);
      pool.push(it);
    }
  }
  return pool;
}

/* ------------------------------ 抽奖主流程 ------------------------------ */

function drawOne(): void {
  if (myCoin.value < SINGLE_DRAW_COST) {
    roomStore.pushToast('warn', '鼠币不足，板板暂时开不了神秘货柜');
    return;
  }
  if (!canDraw.value) return;

  cancelAnim();
  fillerPool = buildFillerPool();
  rows.value = makeFillers(fillerPool, 60);
  offset.value = 0;
  result.value = null;
  phase.value = 'spinning';
  awaiting = true;
  landing = false;
  spinStartTs = performance.now();
  lastTs = spinStartTs;
  timeoutId = window.setTimeout(onTimeout, RESULT_TIMEOUT_MS);
  rafId = requestAnimationFrame(tick);

  roomStore.sendIntent('DRAW_LOTTERY', { times: 1 });
}

function tick(ts: number): void {
  const dt = Math.min((ts - lastTs) / 1000, 0.05);
  lastTs = ts;

  if (!landing) {
    offset.value += SPIN_SPEED * dt;
    const consumed = Math.floor(offset.value / ITEM_H);
    if (consumed >= RECYCLE_AT) {
      rows.value = [...rows.value.slice(consumed), ...makeFillers(fillerPool, consumed)];
      offset.value -= consumed * ITEM_H;
    }
    if (result.value && ts - spinStartTs >= MIN_SPIN_MS) {
      beginLanding(ts);
    }
  } else {
    const t = Math.min((ts - landStartTs) / LAND_MS, 1);
    offset.value = landStartOffset + (landTargetOffset - landStartOffset) * easeOutSextic(t);
    if (t >= 1) {
      offset.value = landTargetOffset;
      phase.value = 'revealed';
      cancelAnim();
      // 卷轴停稳、老板已看到本次结果 —— 此时才放出小报童对该次抽奖的播报。
      roomStore.releaseSelfLotteryReveal();
      return;
    }
  }
  rafId = requestAnimationFrame(tick);
}

/** 锁定中奖物品所在格子，并计算减速到中间行的目标位移。 */
function beginLanding(ts: number): void {
  landing = true;
  landStartTs = ts;
  landStartOffset = offset.value;

  const baseIndex = Math.floor(offset.value / ITEM_H);
  const wonIndex = baseIndex + LAND_ITEMS;
  const need = wonIndex + VISIBLE_ROWS + 2;
  const next = rows.value.slice();
  while (next.length < need) next.push(randItem(fillerPool));
  next[wonIndex] = result.value!.item;
  rows.value = next;

  // item j 落在中间行时：offset = (j - CENTER_ROW) * ITEM_H
  landTargetOffset = (wonIndex - CENTER_ROW) * ITEM_H;
}

function onTimeout(): void {
  // 抽奖被服务端拒绝（如保底超限）时不会收到 LOTTERY_DRAWN：复位卷轴。
  if (phase.value === 'spinning' && !result.value) {
    awaiting = false;
    cancelAnim();
    phase.value = 'idle';
    rows.value = [];
    offset.value = 0;
  }
}

function cancelAnim(): void {
  if (rafId) cancelAnimationFrame(rafId);
  rafId = 0;
  if (timeoutId) window.clearTimeout(timeoutId);
  timeoutId = 0;
}

// 老板本人收到私有 LOTTERY_DRAWN 后，交给卷轴在最短滚动时间后减速着陆。
watch(
  () => roomStore.latestLotteryReveal,
  (next) => {
    if (!awaiting) return;
    if (!next || next.length === 0) return;
    awaiting = false;
    if (timeoutId) {
      window.clearTimeout(timeoutId);
      timeoutId = 0;
    }
    result.value = next[0]!;
  },
  { deep: false },
);

onBeforeUnmount(() => {
  cancelAnim();
  // 兜底：若在卷轴动画途中离开（如切换到补给架视图），避免被暂存的播报永久隐藏。
  roomStore.releaseSelfLotteryReveal();
});

const revealed = computed(() => phase.value === 'revealed' && result.value !== null);
</script>

<template>
  <div class="card rpg-corners p-6 relative overflow-hidden">
    <div
      class="pointer-events-none absolute -top-24 -right-24 w-72 h-72 rounded-full bg-ember-500/15 blur-3xl"
    />
    <div
      class="pointer-events-none absolute -bottom-32 -left-20 w-72 h-72 rounded-full bg-royal-500/10 blur-3xl"
    />
    <div class="relative">
      <div class="flex items-center justify-between mb-4 gap-2">
        <h3 class="rpg-title text-xl">
          <span class="rune-mark mr-1">⚜</span>板板专属 · 神秘商店
        </h3>
        <span class="badge !border-ember-400/60 !text-ember-300 animate-flicker">
          <span class="rune-mark mr-1">★</span>神秘商店
        </span>
      </div>
      <div class="rpg-divider mb-5" />

      <!-- 老虎机卷轴 -->
      <div
        class="rounded-md relative overflow-hidden"
        :style="{
          border: '1px solid rgba(212, 175, 55, 0.45)',
          background:
            'radial-gradient(120% 80% at 50% 0%, rgba(249, 115, 22, 0.18), transparent 60%), linear-gradient(180deg, rgba(0, 0, 0, 0.45), rgba(0, 0, 0, 0.65)), #1a0f0a',
          boxShadow: 'inset 0 0 24px rgba(0, 0, 0, 0.6), inset 0 0 0 1px rgba(0, 0, 0, 0.45)',
        }"
      >
        <!-- 空闲态：开奖前的提示 -->
        <div
          v-if="phase === 'idle' && rows.length === 0"
          class="flex flex-col items-center justify-center text-center px-6"
          :style="{ height: VIEWPORT_H + 'px' }"
        >
          <div class="rune-mark text-3xl text-gold-400/70 animate-flicker">✶</div>
          <div class="font-body text-parchment-300/80 text-sm mt-2">
            点击开启神秘货柜，看看板板这次摸到什么好货。
          </div>
        </div>

        <!-- 卷轴态 -->
        <div v-else class="relative" :style="{ height: VIEWPORT_H + 'px' }">
          <div :style="stripStyle">
            <div
              v-for="(row, idx) in rows"
              :key="idx"
              class="flex items-center justify-center"
              :style="{ height: ITEM_H + 'px' }"
            >
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
            </div>
          </div>

          <!-- 上下渐隐遮罩，强化"窗口"观感 -->
          <div
            class="pointer-events-none absolute inset-x-0 top-0"
            style="height:28px;background:linear-gradient(180deg,#1a0f0a,rgba(26,15,10,0));"
          />
          <div
            class="pointer-events-none absolute inset-x-0 bottom-0"
            style="height:28px;background:linear-gradient(0deg,#1a0f0a,rgba(26,15,10,0));"
          />

          <!-- 中间结果指示线 -->
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
        </div>
      </div>

      <!-- 开奖详情：减速停稳后展示价值与保底变化 -->
      <div v-if="revealed" class="text-center mt-4 animate-slide-up">
        <div class="font-serif text-xs text-gold-300/80 tracking-[0.32em] uppercase mb-1">
          ✦ 本次开出的红 ✦
        </div>
        <div class="text-xs font-serif text-parchment-300/80">
          <span class="text-forest-300 font-semibold">
            价值 {{ formatValueW(result!.item.value) }}
          </span>
        </div>
        <div
          v-if="result!.guaranteeDelta > 0"
          class="text-[11px] mt-1.5 font-serif text-gold-200 tracking-wider"
        >
          <span class="rune-mark mr-1">⚜</span>
          保底
          {{
            formatGuaranteeShift(
              result!.guaranteeDelta,
              result!.guaranteeBefore,
              result!.guaranteeAfter,
            )
          }}
        </div>
      </div>

      <div class="mt-5">
        <button class="btn-boss w-full py-3" :disabled="!canDraw" @click="drawOne">
          <template v-if="spinning">
            <span class="rune-mark animate-flicker">🎰</span> 货柜滚动中…
          </template>
          <template v-else>
            <span class="rune-mark">🎁</span> 开启神秘商店 · {{ SINGLE_DRAW_COST }} 鼠币
          </template>
        </button>
      </div>
      <p class="font-body text-[12px] text-parchment-300/75 mt-3 leading-relaxed text-center">
        按照市场价加入保底
      </p>
    </div>
  </div>
</template>

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

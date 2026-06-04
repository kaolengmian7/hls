<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useRoomStore } from '@/stores/room';
import {
  RARITY_COLOR,
  SINGLE_DRAW_COST,
  formatGuaranteeShift,
  formatValueW,
  type Rarity,
} from '@/game/lottery';
import type { LotteryItem, LotteryRecord } from '@/net/protocol';

const roomStore = useRoomStore();

/* ------------------------------ 老虎机参数 ------------------------------ */

const ITEM_H = 84; // 每格高度（px）
const VISIBLE_ROWS = 3; // 窗口可见行数（奇数，中间行为结果行）
const CENTER_ROW = Math.floor(VISIBLE_ROWS / 2);
const VIEWPORT_H = ITEM_H * VISIBLE_ROWS;
const SPIN_SPEED = ITEM_H * 17; // 滚动速度（px/s），制造高速模糊
const MIN_SPIN_MS = 700; // 即使结果秒回，也至少滚动这么久
const RECYCLE_AT = 20; // 顶部消耗超过这么多格就回收，保持位移有界
const RESULT_TIMEOUT_MS = 6000; // 抽奖被拒/无响应的安全复位

/**
 * 稀有度分级着陆演出（建议 7）：价值越高，减速越慢、悬念越久、特效越炸。
 *   - ms：减速着陆时长；items：着陆阶段额外滚过的格数（越多越"拉扯"）。
 */
type Tier = 'high' | 'medium' | 'low';
const LAND_TUNING: Record<Tier, { ms: number; items: number }> = {
  high: { ms: 3200, items: 26 },
  medium: { ms: 2400, items: 18 },
  low: { ms: 1700, items: 12 },
};

/** near-miss（建议 8）：非高价值结果时，按此概率制造"差一点中大奖"的擦肩演出。 */
const NEAR_MISS_CHANCE = 0.38;

/** near-miss 用的"诱饵"大奖名称（仅表现，value=0，不参与任何结算）。 */
const NEAR_MISS_TEASERS: LotteryItem[] = [
  { id: 'teaser-ocean', name: '海洋之泪', rarity: 'HIGH', value: 0 },
  { id: 'teaser-africa', name: '非洲之心', rarity: 'HIGH', value: 0 },
  { id: 'teaser-crown', name: '万金泪冠', rarity: 'HIGH', value: 0 },
  { id: 'teaser-reactor', name: '微型反应炉', rarity: 'HIGH', value: 0 },
];

/** 把后端稀有度（含历史 COMMON/RARE/EPIC/LEGENDARY）归一到三档演出强度。 */
function tierOf(rarity: string): Tier {
  if (rarity === 'HIGH' || rarity === 'EPIC' || rarity === 'LEGENDARY') return 'high';
  if (rarity === 'MEDIUM' || rarity === 'RARE') return 'medium';
  return 'low';
}

/** 各档位的特效配色（光柱 / 爆裂光晕 / 中奖线辉光）。 */
const TIER_FX: Record<Tier, { core: string; halo: string; line: string }> = {
  high: {
    core: 'rgba(249,179,66,0.9)',
    halo: 'rgba(249,115,22,0.55)',
    line: 'rgba(212,175,55,0.95)',
  },
  medium: {
    core: 'rgba(192,132,252,0.85)',
    halo: 'rgba(168,85,247,0.45)',
    line: 'rgba(168,85,247,0.85)',
  },
  low: {
    core: 'rgba(125,211,252,0.8)',
    halo: 'rgba(56,189,248,0.4)',
    line: 'rgba(56,189,248,0.8)',
  },
};

/**
 * 仅用于"滚动模糊"的装饰条目，**不是**权威奖池（无真实价值/权重）。
 * 真正开出的奖品始终来自服务端 LOTTERY_DRAWN；这里只是让卷轴看起来热闹。
 * 历史抽奖记录里的真实奖品会优先用于填充，装饰条目作为新房间的兜底。
 */
const DECOR_FILLERS: LotteryItem[] = [
  { id: 'decor-1', name: '海洋之泪', rarity: 'HIGH', value: 0 },
  { id: 'decor-2', name: '非洲之心', rarity: 'HIGH', value: 0 },
  { id: 'decor-3', name: '便携军用雷达', rarity: 'HIGH', value: 0 },
  { id: 'decor-4', name: '万金泪冠', rarity: 'HIGH', value: 0 },
  { id: 'decor-5', name: '军用无人机', rarity: 'MEDIUM', value: 0 },
  { id: 'decor-6', name: '万足金条', rarity: 'MEDIUM', value: 0 },
  { id: 'decor-7', name: '医疗机器人', rarity: 'MEDIUM', value: 0 },
  { id: 'decor-8', name: '笔记本电脑', rarity: 'MEDIUM', value: 0 },
  { id: 'decor-9', name: '劳力士', rarity: 'LOW', value: 0 },
  { id: 'decor-10', name: '鱼子酱', rarity: 'LOW', value: 0 },
  { id: 'decor-11', name: '鎏金卡牌', rarity: 'LOW', value: 0 },
  { id: 'decor-12', name: '恒星敏感器', rarity: 'LOW', value: 0 },
];

/* ------------------------------ 响应式状态 ------------------------------ */

type Phase = 'idle' | 'spinning' | 'landing' | 'revealed';

const phase = ref<Phase>('idle');
const rows = ref<LotteryItem[]>([]);
const offset = ref(0); // 卷轴向上滚动的像素位移
const result = ref<LotteryRecord | null>(null);
const reelShake = ref(false); // 高价值停稳瞬间的面板抖动
const nearMissActive = ref(false); // 本次着陆是否触发了"差一点"演出

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
let landDurationMs = LAND_TUNING.medium.ms; // 当前着陆时长，按稀有度设定
let landEase: (t: number) => number = easeOutQuart; // 当前着陆缓动
let shakeTimer = 0;

function easeOutQuart(t: number): number {
  return 1 - Math.pow(1 - t, 4);
}

// 高价值用更"黏"的五次缓动，末段近乎蠕动，把悬念拉满。
function easeOutQuint(t: number): number {
  return 1 - Math.pow(1 - t, 5);
}

function randItem(pool: LotteryItem[]): LotteryItem {
  return pool[Math.floor(Math.random() * pool.length)]!;
}

function makeFillers(pool: LotteryItem[], n: number): LotteryItem[] {
  return Array.from({ length: n }, () => randItem(pool));
}

/** 优先用本房间真实的历史奖品填充卷轴，不足时用装饰条目兜底。 */
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
  for (const it of DECOR_FILLERS) {
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
  reelShake.value = false;
  nearMissActive.value = false;
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
    const t = Math.min((ts - landStartTs) / landDurationMs, 1);
    offset.value = landStartOffset + (landTargetOffset - landStartOffset) * landEase(t);
    if (t >= 1) {
      offset.value = landTargetOffset;
      phase.value = 'revealed';
      onRevealed();
      cancelAnim();
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

  // 建议 7：按稀有度决定着陆时长 / 滚动格数 / 缓动曲线。
  const tier = tierOf(result.value!.item.rarity);
  const tuning = LAND_TUNING[tier];
  landDurationMs = tuning.ms;
  landEase = tier === 'high' ? easeOutQuint : easeOutQuart;

  const baseIndex = Math.floor(offset.value / ITEM_H);
  const wonIndex = baseIndex + tuning.items;
  const need = wonIndex + VISIBLE_ROWS + 2;
  const next = rows.value.slice();
  while (next.length < need) next.push(randItem(fillerPool));
  next[wonIndex] = result.value!.item;

  // 建议 8：未中高价值时，有概率在中奖格正上方塞一个顶级"诱饵"，
  // 让它擦着中奖线滑过、最终停在上一格 —— 制造"就差一点"的紧张感。
  nearMissActive.value =
    tier !== 'high' && wonIndex - 1 >= 0 && Math.random() < NEAR_MISS_CHANCE;
  if (nearMissActive.value) {
    next[wonIndex - 1] = randItem(NEAR_MISS_TEASERS);
  }
  rows.value = next;

  // item j 落在中间行时：offset = (j - CENTER_ROW) * ITEM_H
  landTargetOffset = (wonIndex - CENTER_ROW) * ITEM_H;
}

/** 停稳瞬间：高价值触发面板抖动，配合模板里的光柱/爆裂演出。 */
function onRevealed(): void {
  if (!result.value) return;
  if (tierOf(result.value.item.rarity) === 'high') {
    reelShake.value = true;
    if (shakeTimer) window.clearTimeout(shakeTimer);
    shakeTimer = window.setTimeout(() => {
      reelShake.value = false;
      shakeTimer = 0;
    }, 600);
  }
}

function onTimeout(): void {
  // 抽奖被服务端拒绝（如保底超限）时不会收到 LOTTERY_DRAWN：复位卷轴。
  if (phase.value === 'spinning' && !result.value) {
    awaiting = false;
    cancelAnim();
    phase.value = 'idle';
    rows.value = [];
    offset.value = 0;
    reelShake.value = false;
    nearMissActive.value = false;
  }
}

function cancelAnim(): void {
  if (rafId) cancelAnimationFrame(rafId);
  rafId = 0;
  if (timeoutId) window.clearTimeout(timeoutId);
  timeoutId = 0;
  if (shakeTimer) window.clearTimeout(shakeTimer);
  shakeTimer = 0;
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

onBeforeUnmount(cancelAnim);

const revealed = computed(() => phase.value === 'revealed' && result.value !== null);
const revealTier = computed<Tier>(() =>
  result.value ? tierOf(result.value.item.rarity) : 'low',
);
const fx = computed(() => TIER_FX[revealTier.value]);
const isHighReveal = computed(() => revealed.value && revealTier.value === 'high');
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
        :class="reelShake ? 'animate-reel-shake' : ''"
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
              class="flex flex-col items-center justify-center"
              :style="{ height: ITEM_H + 'px' }"
            >
              <div
                class="font-display text-2xl font-bold tracking-wide text-shadow-relic leading-none"
                :class="RARITY_COLOR[row.rarity as Rarity] ?? 'text-parchment-100'"
              >
                {{ row.name }}
              </div>
            </div>
          </div>

          <!-- 上下渐隐遮罩，强化"窗口"观感 -->
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

          <!-- 中间结果指示线（停稳后辉光按稀有度变色） -->
          <div
            class="pointer-events-none absolute inset-x-0 transition-all duration-300"
            :style="{
              top: CENTER_ROW * ITEM_H + 'px',
              height: ITEM_H + 'px',
              border: revealed
                ? '1px solid ' + fx.line
                : '1px solid rgba(212,175,55,0.35)',
              boxShadow: revealed
                ? 'inset 0 0 28px ' + fx.halo + ', 0 0 20px ' + fx.line
                : 'inset 0 0 18px rgba(0,0,0,0.45)',
              background: revealed ? fx.halo.replace(/0\.\d+\)$/, '0.1)') : 'transparent',
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

          <!-- 建议 7：停稳瞬间的分级特效（光柱 + 爆裂光晕），key 绑结果以重放 -->
          <template v-if="revealed">
            <div
              :key="'beam-' + result!.id"
              class="pointer-events-none absolute inset-x-0 animate-beam-rise"
              :style="{
                top: CENTER_ROW * ITEM_H + 'px',
                height: ITEM_H + 'px',
                transformOrigin: 'center',
                background:
                  'linear-gradient(90deg, transparent, ' +
                  fx.core +
                  ', transparent)',
                filter: 'blur(2px)',
                mixBlendMode: 'screen',
              }"
            />
            <div
              :key="'burst-' + result!.id"
              class="pointer-events-none absolute left-1/2 animate-crit-burst"
              :style="{
                top: (CENTER_ROW + 0.5) * ITEM_H + 'px',
                width: ITEM_H * 3 + 'px',
                height: ITEM_H * 3 + 'px',
                marginLeft: -(ITEM_H * 1.5) + 'px',
                marginTop: -(ITEM_H * 1.5) + 'px',
                borderRadius: '9999px',
                background:
                  'radial-gradient(circle, ' +
                  fx.core +
                  ' 0%, ' +
                  fx.halo +
                  ' 35%, transparent 70%)',
                mixBlendMode: 'screen',
              }"
            />
          </template>

          <!-- 建议 8：near-miss 提示 —— 大奖擦肩停在中奖线上一格 -->
          <div
            v-if="revealed && nearMissActive"
            :key="'miss-' + result!.id"
            class="pointer-events-none absolute inset-x-0 flex items-center justify-end pr-3 animate-near-miss"
            :style="{ top: (CENTER_ROW - 1) * ITEM_H + 'px', height: ITEM_H + 'px' }"
          >
            <span
              class="font-serif text-[11px] tracking-[0.2em] text-ember-300/90 px-2 py-0.5 rounded"
              :style="{
                background: 'rgba(0,0,0,0.55)',
                border: '1px solid rgba(249,115,22,0.5)',
                boxShadow: '0 0 12px rgba(249,115,22,0.35)',
              }"
            >
              差 一 点 ！
            </span>
          </div>
        </div>
      </div>

      <!-- 开奖详情：减速停稳后展示价值与保底变化 -->
      <div v-if="revealed" class="text-center mt-4 animate-slide-up">
        <div
          class="font-serif text-xs tracking-[0.32em] uppercase mb-1"
          :class="isHighReveal ? 'text-ember-300 animate-flicker' : 'text-gold-300/80'"
        >
          <template v-if="isHighReveal">✦✦ 极品出货 ✦✦</template>
          <template v-else>✦ 本次开出的红 ✦</template>
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

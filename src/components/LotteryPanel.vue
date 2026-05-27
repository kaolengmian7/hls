<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoomStore } from '@/stores/room';
import {
  RARITY_COLOR,
  RARITY_LABEL,
  SINGLE_DRAW_COST,
  formatGuaranteeShift,
  formatValueW,
  type Rarity,
} from '@/game/lottery';
import type { LotteryRecord } from '@/net/protocol';

const roomStore = useRoomStore();
const animating = ref(false);
const showcase = ref<LotteryRecord[]>([]);

const myCoin = computed(() => roomStore.me?.coin ?? 0);

// 监听最新抽奖记录变化（仅当抽奖人是自己时弹出动画）
watch(
  () => roomStore.lotteryRecords,
  (next, prev) => {
    if (!prev || prev === next) return;
    const newCount = next.length - prev.length;
    if (newCount <= 0) return;
    const fresh = next.slice(0, newCount);
    if (fresh.length && fresh[0]!.playerId === roomStore.selfPlayerId) {
      showcase.value = fresh;
      animating.value = true;
      setTimeout(() => {
        animating.value = false;
      }, 1800);
    }
  },
  { deep: false },
);

function drawOne(): void {
  if (myCoin.value < SINGLE_DRAW_COST) {
    roomStore.pushToast('warn', '金币不足，老板暂时开不了神秘货柜');
    return;
  }
  roomStore.sendIntent('DRAW_LOTTERY', { times: 1 });
}
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
          <span class="rune-mark mr-1">⚜</span>老板专属 · 神秘货柜
        </h3>
        <span class="badge !border-ember-400/60 !text-ember-300 animate-flicker">
          <span class="rune-mark mr-1">★</span>仅老板可开启
        </span>
      </div>
      <div class="rpg-divider mb-5" />

      <div
        class="rounded-md p-6 min-h-[150px] flex items-center justify-center transition-all relative overflow-hidden"
        style="
          border: 1px solid rgba(212, 175, 55, 0.45);
          background:
            radial-gradient(120% 80% at 50% 0%, rgba(249, 115, 22, 0.18), transparent 60%),
            linear-gradient(180deg, rgba(0, 0, 0, 0.45), rgba(0, 0, 0, 0.65)),
            #1a0f0a;
          box-shadow:
            inset 0 0 24px rgba(0, 0, 0, 0.6),
            inset 0 0 0 1px rgba(0, 0, 0, 0.45);
        "
      >
        <div v-if="animating" class="text-center">
          <div class="rune-mark text-4xl animate-rune-spin inline-block">✶</div>
          <div class="rpg-title text-2xl mt-2 animate-flicker">神秘货柜开启中</div>
        </div>
        <div v-else-if="showcase.length >= 1" class="text-center animate-slide-up">
          <div class="font-serif text-xs text-gold-300/80 tracking-[0.32em] uppercase mb-2">
            ✦ 本次开出的货品 ✦
          </div>
          <div
            class="font-display text-3xl font-bold tracking-wide text-shadow-relic"
            :class="RARITY_COLOR[showcase[0]!.item.rarity as Rarity] ?? 'text-parchment-100'"
          >
            {{ showcase[0]!.item.name }}
          </div>
          <div class="text-xs mt-2 font-serif text-parchment-300/80">
            <span class="tracking-widest uppercase">
              {{ RARITY_LABEL[showcase[0]!.item.rarity as Rarity] ?? showcase[0]!.item.rarity }}
            </span>
            <span class="mx-1 text-gold-400/60">·</span>
            <span class="text-forest-300 font-semibold">
              价值 {{ formatValueW(showcase[0]!.item.value) }}
            </span>
          </div>
          <div
            v-if="showcase[0]!.guaranteeDelta > 0"
            class="text-[11px] mt-2 font-serif text-gold-200 tracking-wider"
          >
            <span class="rune-mark mr-1">⚜</span>
            库藏 {{ formatGuaranteeShift(showcase[0]!.guaranteeDelta, showcase[0]!.guaranteeBefore, showcase[0]!.guaranteeAfter) }}
          </div>
        </div>
        <div v-else class="text-center">
          <div class="rune-mark text-3xl text-gold-400/70 animate-flicker">✶</div>
          <div class="font-body text-parchment-300/80 text-sm mt-2">
            点击开启神秘货柜，看看老板这次摸到什么好货。
          </div>
        </div>
      </div>

      <div class="mt-5">
        <button
          class="btn-boss w-full py-3"
          :disabled="myCoin < SINGLE_DRAW_COST"
          @click="drawOne"
        >
          <span class="rune-mark">🎁</span> 开启神秘货柜 · {{ SINGLE_DRAW_COST }} 金
        </button>
      </div>
      <p class="font-body text-[12px] text-parchment-300/75 mt-3 leading-relaxed text-center">
        开柜结果由服务端随机生成并同步给全员，可在「货柜记录」中查看。
      </p>
    </div>
  </div>
</template>

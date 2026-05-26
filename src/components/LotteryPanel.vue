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
    roomStore.pushToast('warn', '金币不足');
    return;
  }
  roomStore.sendIntent('DRAW_LOTTERY', { times: 1 });
}
</script>

<template>
  <div class="card p-6 relative overflow-hidden">
    <div
      class="pointer-events-none absolute -top-20 -right-20 w-72 h-72 rounded-full bg-orange-500/10 blur-3xl"
    />
    <div class="relative">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-lg font-semibold text-slate-50">抽奖机 · 老板专属</h3>
        <span class="badge bg-orange-500/20 text-orange-300">幸运在你这边</span>
      </div>

      <div class="rounded-2xl border border-orange-400/20 bg-slate-900/60 p-6 min-h-[140px] flex items-center justify-center transition-all">
        <div v-if="animating" class="text-3xl font-bold animate-pulse_soft text-amber-300">
          ✨ 正在揭晓 ✨
        </div>
        <div v-else-if="showcase.length >= 1" class="text-center animate-slide-up">
          <div class="text-sm text-slate-400 mb-1">本次抽中</div>
          <div
            class="text-2xl font-bold"
            :class="RARITY_COLOR[showcase[0]!.item.rarity as Rarity] ?? 'text-slate-200'"
          >
            {{ showcase[0]!.item.name }}
          </div>
          <div class="text-xs mt-1 text-slate-400">
            {{ RARITY_LABEL[showcase[0]!.item.rarity as Rarity] ?? showcase[0]!.item.rarity }}
            ·
            <span class="text-emerald-300">价值 {{ formatValueW(showcase[0]!.item.value) }}</span>
          </div>
          <div
            v-if="showcase[0]!.guaranteeDelta > 0"
            class="text-[11px] mt-1 text-amber-300"
          >
            保底 {{ formatGuaranteeShift(showcase[0]!.guaranteeDelta, showcase[0]!.guaranteeBefore, showcase[0]!.guaranteeAfter) }}
          </div>
        </div>
        <div v-else class="text-slate-500 text-sm text-center">
          点击下方按钮开始抽奖
        </div>
      </div>

      <div class="mt-5">
        <button
          class="btn-boss w-full"
          :disabled="myCoin < SINGLE_DRAW_COST"
          @click="drawOne"
        >
          单抽 · {{ SINGLE_DRAW_COST }} 币
        </button>
      </div>
      <p class="text-[11px] text-slate-500 mt-3 leading-relaxed">
        抽奖结果由服务端裁决并广播，所有玩家可在「抽奖记录」中查看。
      </p>
    </div>
  </div>
</template>

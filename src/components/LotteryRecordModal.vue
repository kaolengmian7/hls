<script setup lang="ts">
import { computed } from 'vue';
import { useRoomStore } from '@/stores/room';
import {
  RARITY_COLOR,
  RARITY_LABEL,
  formatGuaranteeShift,
  formatValueW,
  type Rarity,
} from '@/game/lottery';

defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();

const roomStore = useRoomStore();

function nameOf(playerId: string): string {
  return roomStore.players.find((p) => p.id === playerId)?.nickname ?? playerId.slice(-4);
}

function format(ts: number): string {
  const d = new Date(ts);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`;
}

const records = computed(() => roomStore.lotteryRecords);
</script>

<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      @click.self="emit('close')"
    >
      <div class="card w-full max-w-2xl max-h-[80vh] flex flex-col animate-slide-up">
        <div class="flex items-center justify-between p-5 border-b border-white/10">
          <h3 class="text-lg font-semibold text-slate-50">抽奖记录</h3>
          <button class="btn-ghost py-1 px-3" @click="emit('close')">关闭</button>
        </div>
        <div class="flex-1 overflow-y-auto p-5">
          <table class="w-full text-sm">
            <thead class="text-slate-400 text-left">
              <tr class="border-b border-white/10">
                <th class="py-2 font-medium">时间</th>
                <th class="py-2 font-medium">抽奖人</th>
                <th class="py-2 font-medium">消耗</th>
                <th class="py-2 font-medium">物品</th>
                <th class="py-2 font-medium">稀有度</th>
                <th class="py-2 font-medium">价值</th>
                <th class="py-2 font-medium">保底变化</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="r in records"
                :key="r.id"
                class="border-b border-white/5 hover:bg-slate-900/40"
              >
                <td class="py-2 text-slate-300">{{ format(r.at) }}</td>
                <td class="py-2 text-slate-200">{{ nameOf(r.playerId) }}</td>
                <td class="py-2 text-amber-400">{{ r.cost }}</td>
                <td
                  class="py-2 font-medium"
                  :class="RARITY_COLOR[r.item.rarity as Rarity] ?? 'text-slate-200'"
                >
                  {{ r.item.name }}
                </td>
                <td class="py-2 text-slate-300">
                  {{ RARITY_LABEL[r.item.rarity as Rarity] ?? r.item.rarity }}
                </td>
                <td class="py-2 text-emerald-300">{{ formatValueW(r.item.value) }}</td>
                <td class="py-2 text-amber-300 whitespace-nowrap">
                  {{ formatGuaranteeShift(r.guaranteeDelta, r.guaranteeBefore, r.guaranteeAfter) || '—' }}
                </td>
              </tr>
              <tr v-if="records.length === 0">
                <td colspan="7" class="py-8 text-center text-slate-500">尚无抽奖记录</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

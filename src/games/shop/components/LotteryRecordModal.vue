<script setup lang="ts">
import { computed } from 'vue';
import { useRoomStore } from '../stores/room';
import {
  RARITY_COLOR,
  RARITY_LABEL,
  formatGuaranteeShift,
  formatValueW,
  type Rarity,
} from '../game/lottery';

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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
      @click.self="emit('close')"
    >
      <div class="hls-panel w-[92%] max-w-lg max-h-[80vh] flex flex-col animate-pixel-slide">
        <div class="hls-panel-title">抽奖记录</div>
        <div class="flex-1 overflow-y-auto p-5">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left">
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">时辰</th>
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">开启人</th>
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">耗金</th>
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">所得</th>
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">阶位</th>
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">价值</th>
                <th class="py-2 font-body font-medium text-[11px] tracking-[0.18em] uppercase text-hls-gold/85">库藏变动</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="r in records"
                :key="r.id"
                class="hls-menu-row border-t border-hls-border/30 hover:bg-hls-bg/60 transition-colors"
              >
                <td class="py-2 font-body text-hls-text-light/90">{{ format(r.at) }}</td>
                <td class="py-2 font-body text-hls-text-light">{{ nameOf(r.playerId) }}</td>
                <td class="py-2 coin-tag">💰 {{ r.cost }}</td>
                <td
                  class="py-2 font-body font-bold"
                  :class="RARITY_COLOR[r.item.rarity as Rarity] ?? 'text-hls-text-light'"
                >
                  {{ r.item.name }}
                </td>
                <td class="py-2 font-body text-hls-text-light/80">
                  {{ RARITY_LABEL[r.item.rarity as Rarity] ?? r.item.rarity }}
                </td>
                <td class="py-2 text-hls-btn-green font-body">{{ formatValueW(r.item.value) }}</td>
                <td class="py-2 text-hls-gold whitespace-nowrap font-body">
                  {{ formatGuaranteeShift(r.guaranteeDelta, r.guaranteeBefore, r.guaranteeAfter) || '—' }}
                </td>
              </tr>
              <tr v-if="records.length === 0">
                <td colspan="7" class="py-10 text-center">
                  <div class="text-2xl text-hls-gold/60 animate-pixel-pulse">✶</div>
                  <div class="font-body text-hls-text-light/75 mt-2">还没有抽奖记录</div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="p-4 border-t border-hls-border/30">
          <button class="hls-btn-secondary w-full py-2" @click="emit('close')">关闭</button>
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

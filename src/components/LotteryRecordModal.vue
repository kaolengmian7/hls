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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      @click.self="emit('close')"
    >
      <div class="card rpg-corners w-full max-w-2xl max-h-[82vh] flex flex-col animate-slide-up">
        <div class="flex items-center justify-between p-5 border-b border-gold-500/25">
          <h3 class="rpg-title text-xl">
            <span class="rune-mark mr-1">⚜</span>圣匣秘录
          </h3>
          <button class="btn-ghost py-1.5 px-3" @click="emit('close')">合卷</button>
        </div>
        <div class="flex-1 overflow-y-auto p-5">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left">
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">时辰</th>
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">启匣人</th>
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">耗金</th>
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">所得</th>
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">阶位</th>
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">价值</th>
                <th class="py-2 font-serif text-[11px] tracking-[0.18em] uppercase text-gold-300/85">库藏变动</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="r in records"
                :key="r.id"
                class="border-t border-gold-500/15 hover:bg-ink-500/40 transition-colors"
              >
                <td class="py-2 font-serif text-parchment-300/90">{{ format(r.at) }}</td>
                <td class="py-2 font-serif text-parchment-100">{{ nameOf(r.playerId) }}</td>
                <td class="py-2 coin-tag">💰 {{ r.cost }}</td>
                <td
                  class="py-2 font-serif font-bold"
                  :class="RARITY_COLOR[r.item.rarity as Rarity] ?? 'text-parchment-100'"
                >
                  {{ r.item.name }}
                </td>
                <td class="py-2 font-body text-parchment-200/80">
                  {{ RARITY_LABEL[r.item.rarity as Rarity] ?? r.item.rarity }}
                </td>
                <td class="py-2 text-forest-300 font-serif">{{ formatValueW(r.item.value) }}</td>
                <td class="py-2 text-gold-200 whitespace-nowrap font-serif">
                  {{ formatGuaranteeShift(r.guaranteeDelta, r.guaranteeBefore, r.guaranteeAfter) || '—' }}
                </td>
              </tr>
              <tr v-if="records.length === 0">
                <td colspan="7" class="py-10 text-center">
                  <div class="rune-mark text-2xl text-gold-300/60 animate-flicker">✶</div>
                  <div class="font-body text-parchment-300/75 mt-2">圣匣未启，秘录尚空</div>
                </td>
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

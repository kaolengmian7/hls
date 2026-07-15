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
      <div class="hls-panel w-[94%] max-w-3xl max-h-[80vh] flex flex-col animate-pixel-slide">
        <div class="hls-panel-title">抽奖记录</div>
        <div class="flex-1 overflow-y-auto p-5">
          <div v-if="records.length === 0" class="py-10 text-center">
            <div class="text-2xl text-hls-gold/60 animate-pixel-pulse">✶</div>
            <div class="mt-2 font-body text-hls-text-light/75">还没有抽奖记录</div>
          </div>
          <ul v-else class="space-y-3">
            <li
              v-for="r in records"
              :key="r.id"
              class="lottery-record-card"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="min-w-0 space-y-2">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-body text-[11px] tracking-[0.18em] uppercase text-hls-gold/75">
                      ✦ {{ format(r.at) }}
                    </span>
                    <span class="hls-badge hls-badge-gold text-[11px]">
                      {{ RARITY_LABEL[r.item.rarity as Rarity] ?? r.item.rarity }}
                    </span>
                  </div>
                  <div
                    class="font-body text-lg font-bold break-words"
                    :class="RARITY_COLOR[r.item.rarity as Rarity] ?? 'text-hls-text-light'"
                  >
                    {{ r.item.name }}
                  </div>
                </div>
                <div class="record-chip self-start">
                  <span class="text-hls-text-muted">💰</span>
                  <span class="coin-tag leading-none">{{ r.cost }}</span>
                </div>
              </div>

              <div class="mt-3 grid gap-2 sm:grid-cols-3">
                <div class="record-meta-card">
                  <div class="record-meta-label">开启人</div>
                  <div class="record-meta-value break-all">{{ nameOf(r.playerId) }}</div>
                </div>
                <div class="record-meta-card">
                  <div class="record-meta-label">价值</div>
                  <div class="record-meta-value text-hls-btn-green">
                    {{ formatValueW(r.item.value) }}
                  </div>
                </div>
                <div class="record-meta-card sm:col-span-3">
                  <div class="record-meta-label">库藏变动</div>
                  <div class="record-meta-value text-hls-gold break-words">
                    {{
                      formatGuaranteeShift(r.guaranteeDelta, r.guaranteeBefore, r.guaranteeAfter) || '—'
                    }}
                  </div>
                </div>
              </div>
            </li>
          </ul>
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

.lottery-record-card {
  border: 2px solid #d4b870;
  box-shadow: inset 0 1px 0 #ffffff, inset 0 -1px 0 #e0c888, 2px 2px 0 rgba(60, 30, 8, 0.12);
  background: #fffef9;
  padding: 14px;
}

.record-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 2px solid rgba(212, 184, 112, 0.55);
  background: rgba(212, 184, 112, 0.14);
  padding: 4px 10px;
}

.record-meta-card {
  background: rgba(212, 184, 112, 0.08);
  border: 1px solid rgba(160, 112, 64, 0.22);
  padding: 10px 12px;
}

.record-meta-label {
  font-family: 'Source Han Sans CN', sans-serif;
  font-size: 11px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(160, 112, 64, 0.85);
}

.record-meta-value {
  margin-top: 4px;
  font-family: 'Source Han Sans CN', sans-serif;
  font-size: 14px;
  font-weight: 600;
  color: #3f2412;
}
</style>

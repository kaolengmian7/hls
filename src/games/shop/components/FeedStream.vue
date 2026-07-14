<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoomStore } from '../stores/room';
import { getEquipment, getWeapon } from '../game/equipment';

const roomStore = useRoomStore();
const PAGE_SIZE = 5;

function nameOf(playerId: string): string {
  return roomStore.players.find((p) => p.id === playerId)?.nickname ?? playerId.slice(-4);
}

function format(ts: number): string {
  const d = new Date(ts);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`;
}

function asStr(v: unknown): string {
  return v == null ? '' : String(v);
}

const events = computed(() => roomStore.feed);
const currentPage = ref(1);
const pageCount = computed(() => Math.max(1, Math.ceil(events.value.length / PAGE_SIZE)));
const pagedEvents = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE;
  return events.value.slice(start, start + PAGE_SIZE);
});

watch(
  () => events.value.length,
  () => {
    if (currentPage.value > pageCount.value) {
      currentPage.value = pageCount.value;
    }
  },
);

function prevPage(): void {
  if (currentPage.value > 1) currentPage.value -= 1;
}

function nextPage(): void {
  if (currentPage.value < pageCount.value) currentPage.value += 1;
}
</script>

<template>
  <div class="hls-panel-deep p-5 flex flex-col lg:sticky lg:top-24">
    <div class="flex items-center mb-3">
      <h3 class="font-body font-bold text-hls-gold text-lg">
        ⚜ 神秘商店 · 小报童
      </h3>
    </div>
    <div class="hls-divider mb-3" />
    <div class="flex-1 min-h-0">
      <transition-group name="feed" class="flex flex-col gap-2">
        <div
          v-for="ev in pagedEvents"
          :key="ev.id"
          class="feed-item rounded-md px-3 py-2 text-sm font-body"
        >
          <div class="font-body text-[10px] text-hls-gold/70 mb-0.5">
            ✦ {{ format(ev.at) }}
          </div>
          <template v-if="ev.kind === 'PLAYER_JOINED'">
            <span class="font-body font-semibold text-hls-btn-green">「{{ asStr(ev.payload.nickname) || nameOf(asStr(ev.payload.playerId)) }}」</span>进入了神秘商店。
          </template>

          <template v-else-if="ev.kind === 'PLAYER_LEFT'">
            <span class="font-body font-semibold text-red-400">「{{ nameOf(asStr(ev.payload.playerId)) }}」</span>离开了神秘商店。
          </template>

          <template v-else-if="ev.kind === 'LOTTERY'">
            <div><span class="font-body font-semibold text-hls-gold">「板板」</span>在神秘商店摸出了<span class="font-body font-bold text-hls-text-light">‹{{ asStr(ev.payload.itemName) }}›</span></div>
            <div v-if="Number(ev.payload.itemValue) > 0 || Number(ev.payload.guaranteeDelta) > 0" class="feed-detail">
              <span v-if="Number(ev.payload.itemValue) > 0" class="text-hls-btn-green">价值 {{ Number(ev.payload.itemValue) }}W</span>
              <span v-if="Number(ev.payload.guaranteeDelta) > 0" class="text-hls-gold/80">保底 +{{ Number(ev.payload.guaranteeDelta) }}W · {{ Number(ev.payload.guaranteeBefore) }}→{{ Number(ev.payload.guaranteeAfter) }}</span>
            </div>
          </template>

          <template v-else-if="ev.kind === 'PURCHASE'">
            <div><span class="font-body font-semibold text-hls-text-nav">「{{ nameOf(asStr(ev.payload.playerId)) }}」</span>拿走了 <span class="font-body font-bold text-hls-text-light">‹{{ getEquipment(asStr(ev.payload.equipmentId))?.name ?? asStr(ev.payload.equipmentId) }}›</span></div>
            <div v-if="ev.payload.pick" class="feed-detail">
              <span class="text-hls-btn-green">{{ asStr(ev.payload.pickName) || getWeapon(asStr(ev.payload.pick))?.name || asStr(ev.payload.pick) }}</span>
            </div>
          </template>

          <template v-else-if="ev.kind === 'GAME_STARTED'">
            <span class="font-body text-hls-gold text-base font-bold">⚜ 神秘商店正式开张！</span>
          </template>

          <template v-else-if="ev.kind === 'SETTLEMENT'">
            <div><span class="font-body font-bold" :class="ev.payload.outcome === 'EVACUATED' ? 'text-hls-btn-green' : 'text-red-400'">{{ ev.payload.outcome === 'EVACUATED' ? '✓ 成功撤离' : '✗ 炸单翻车' }}</span></div>
            <div v-if="Number(ev.payload.guaranteeDelta) > 0" class="feed-detail">
              <span class="text-hls-gold/80">库藏 {{ asStr(ev.payload.guaranteeBefore) }}W → {{ asStr(ev.payload.guaranteeAfter) }}W</span>
            </div>
          </template>

          <template v-else>
            <span class="font-body text-hls-text-muted">{{ ev.kind }}</span>
          </template>
        </div>
      </transition-group>
      <div v-if="events.length === 0" class="text-center py-8">
        <div class="text-2xl text-hls-gold/60 animate-pixel-pulse">✶</div>
        <div class="font-body text-hls-text-light/70 text-sm mt-2">
          还没有新的小报童消息。
        </div>
      </div>
    </div>
    <div v-if="events.length > 0" class="mt-3 pt-3 border-t border-hls-border/20 flex items-center justify-between gap-2">
      <button class="hls-btn-secondary px-3 py-1.5 text-[11px]" :disabled="currentPage === 1" @click="prevPage">
        上一页
      </button>
      <div class="font-body text-[11px] text-hls-gold-dark font-bold">
        第 {{ currentPage }} / {{ pageCount }} 页
      </div>
      <button class="hls-btn-secondary px-3 py-1.5 text-[11px]" :disabled="currentPage === pageCount" @click="nextPage">
        下一页
      </button>
    </div>
  </div>
</template>

<style scoped>
.feed-item {
  background:
    linear-gradient(180deg, rgba(212, 175, 55, 0.05), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, rgba(232, 208, 168, 0.7), rgba(232, 208, 168, 0.7));
  border: 1px solid rgba(212, 175, 55, 0.18);
  color: #2a1408;
}
.feed-detail {
  display: flex;
  flex-wrap: wrap;
  gap: 0 0.6rem;
  margin-top: 3px;
  font-size: 11px;
  opacity: 0.9;
}
.feed-enter-active {
  transition: all 0.3s ease;
}
.feed-enter-from {
  opacity: 0;
  transform: translateX(10px);
}
</style>

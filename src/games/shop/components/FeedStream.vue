<script setup lang="ts">
import { computed } from 'vue';
import { useRoomStore } from '../stores/room';
import { getEquipment, getWeapon } from '../game/equipment';

const props = defineProps<{
  panelHeight?: number | null;
}>();

const roomStore = useRoomStore();

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

function isBossCovered(payload: Record<string, unknown>): boolean {
  return payload.outcome === 'EVACUATED'
    && Number(payload.guaranteeDelta) > 0
    && Number(payload.guaranteeBefore) === Number(payload.guaranteeAfter);
}

const events = computed(() => roomStore.feed);
</script>

<template>
  <div
    class="hls-panel-deep p-5 flex min-h-[24rem] flex-col lg:sticky lg:top-24"
    :style="props.panelHeight ? { height: `${props.panelHeight}px` } : undefined"
  >
    <div class="flex items-center mb-3">
      <h3 class="font-body font-bold text-hls-gold text-xl">
        ⚜ 神秘商店 · 小报童
      </h3>
    </div>
    <div class="hls-divider mb-3" />
    <div class="feed-scroll flex-1 min-h-0 overflow-y-auto pr-1">
      <transition-group name="feed" tag="div" class="flex flex-col gap-3">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="feed-item rounded-md px-3 py-2 text-base font-body"
        >
          <div class="font-body text-xs text-hls-gold/70 mb-0.5">
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
            <span class="font-body text-hls-gold text-lg font-bold">⚜ 神秘商店正式开张！</span>
          </template>

          <template v-else-if="ev.kind === 'SETTLEMENT'">
            <div><span class="font-body font-bold" :class="ev.payload.outcome === 'EVACUATED' ? 'text-hls-btn-green' : 'text-red-400'">{{ ev.payload.outcome === 'EVACUATED' ? '✓ 成功撤离' : '✗ 炸单翻车' }}</span></div>
            <div v-if="Number(ev.payload.guaranteeDelta) > 0" class="feed-detail">
              <span class="text-hls-gold/80">库藏 {{ asStr(ev.payload.guaranteeBefore) }}W → {{ asStr(ev.payload.guaranteeAfter) }}W</span>
              <span v-if="isBossCovered(ev.payload)" class="text-hls-btn-green">老板兜底 · 得吃 +{{ Number(ev.payload.guaranteeDelta) }}W</span>
            </div>
          </template>

          <template v-else>
            <span class="font-body text-hls-text-muted">{{ ev.kind }}</span>
          </template>
        </div>
      </transition-group>
      <div v-if="events.length === 0" class="text-center py-8">
        <div class="text-2xl text-hls-gold/60 animate-pixel-pulse">✶</div>
        <div class="font-body text-hls-text-light/70 text-base mt-2">
          还没有新的小报童消息。
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed-scroll {
  scrollbar-width: thin;
  scrollbar-color: rgba(212, 175, 55, 0.45) rgba(0, 0, 0, 0.14);
}
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
  font-size: 13px;
  opacity: 0.9;
}
.feed-enter-active {
  transition: all 0.3s ease;
}
.feed-enter-from {
  opacity: 0;
  transform: translateX(10px);
}
.feed-scroll::-webkit-scrollbar {
  width: 6px;
}
.feed-scroll::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.14);
  border-radius: 999px;
}
.feed-scroll::-webkit-scrollbar-thumb {
  background: rgba(212, 175, 55, 0.45);
  border-radius: 999px;
}
</style>

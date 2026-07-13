<script setup lang="ts">
import { computed } from 'vue';
import { useRoomStore } from '../stores/room';
import { getEquipment, getWeapon } from '../game/equipment';

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

const events = computed(() => roomStore.feed);
</script>

<template>
  <div class="sdv-panel-deep p-5 flex flex-col lg:sticky lg:top-24">
    <div class="flex items-center justify-between mb-3">
      <h3 class="font-pixel text-sdv-gold text-lg">
        ⚜ 神秘商店 · 小报童
      </h3>
      <span class="font-pixel text-[11px] tracking-[0.2em] uppercase text-sdv-gold/70">
        {{ events.length }} 缄
      </span>
    </div>
    <div class="sdv-divider mb-3" />
    <div class="flex-1 overflow-y-auto max-h-[60vh] lg:max-h-[calc(100vh-8rem)] pr-1 space-y-2">
      <transition-group name="feed">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="feed-item rounded-md px-3 py-2 text-sm font-body"
        >
          <div class="font-pixel text-[10px] tracking-widest text-sdv-gold/70 mb-0.5">
            ✦ {{ format(ev.at) }}
          </div>
          <template v-if="ev.kind === 'PLAYER_JOINED'">
            <span class="font-pixel font-semibold text-sdv-btn-green">「{{ asStr(ev.payload.nickname) || nameOf(asStr(ev.payload.playerId)) }}」</span>进入了神秘商店。
          </template>

          <template v-else-if="ev.kind === 'PLAYER_LEFT'">
            <span class="font-pixel font-semibold text-red-400">「{{ nameOf(asStr(ev.payload.playerId)) }}」</span>离开了神秘商店。
          </template>

          <template v-else-if="ev.kind === 'LOTTERY'">
            <div><span class="font-pixel font-semibold text-sdv-gold">「板板」</span>在神秘商店摸出了<span class="font-pixel font-bold text-sdv-text-light">‹{{ asStr(ev.payload.itemName) }}›</span></div>
            <div v-if="Number(ev.payload.itemValue) > 0 || Number(ev.payload.guaranteeDelta) > 0" class="feed-detail">
              <span v-if="Number(ev.payload.itemValue) > 0" class="text-sdv-btn-green">价值 {{ Number(ev.payload.itemValue) }}W</span>
              <span v-if="Number(ev.payload.guaranteeDelta) > 0" class="text-sdv-gold/80">保底 +{{ Number(ev.payload.guaranteeDelta) }}W · {{ Number(ev.payload.guaranteeBefore) }}→{{ Number(ev.payload.guaranteeAfter) }}</span>
            </div>
          </template>

          <template v-else-if="ev.kind === 'PURCHASE'">
            <div><span class="font-pixel font-semibold text-sdv-text-nav">「{{ nameOf(asStr(ev.payload.playerId)) }}」</span>拿走了 <span class="font-pixel font-bold text-sdv-text-light">‹{{ getEquipment(asStr(ev.payload.equipmentId))?.name ?? asStr(ev.payload.equipmentId) }}›</span></div>
            <div v-if="ev.payload.pick" class="feed-detail">
              <span class="text-sdv-btn-green">{{ asStr(ev.payload.pickName) || getWeapon(asStr(ev.payload.pick))?.name || asStr(ev.payload.pick) }}</span>
            </div>
          </template>

          <template v-else-if="ev.kind === 'GAME_STARTED'">
            <span class="font-pixel text-sdv-gold text-base font-bold tracking-widest">⚜ 神秘商店正式开张！</span>
          </template>

          <template v-else-if="ev.kind === 'SETTLEMENT'">
            <div><span class="font-pixel font-bold tracking-wider" :class="ev.payload.outcome === 'EVACUATED' ? 'text-sdv-btn-green' : 'text-red-400'">{{ ev.payload.outcome === 'EVACUATED' ? '✓ 成功撤离' : '✗ 炸单翻车' }}</span></div>
            <div v-if="Number(ev.payload.guaranteeDelta) > 0" class="feed-detail">
              <span class="text-sdv-gold/80">库藏 {{ asStr(ev.payload.guaranteeBefore) }}W → {{ asStr(ev.payload.guaranteeAfter) }}W</span>
            </div>
          </template>

          <template v-else>
            <span class="font-body text-sdv-text-muted font-pixel">{{ ev.kind }}</span>
          </template>
        </div>
      </transition-group>
      <div v-if="events.length === 0" class="text-center py-8">
        <div class="text-2xl text-sdv-gold/60 animate-pixel-pulse">✶</div>
        <div class="font-body text-sdv-text-light/70 text-sm mt-2">
          还没有新的小报童消息。
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed-item {
  background:
    linear-gradient(180deg, rgba(212, 175, 55, 0.05), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, rgba(20, 12, 8, 0.7), rgba(10, 6, 4, 0.7));
  border: 1px solid rgba(212, 175, 55, 0.18);
  color: #f4e6c2;
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

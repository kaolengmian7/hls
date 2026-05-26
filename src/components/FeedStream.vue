<script setup lang="ts">
import { computed } from 'vue';
import { useRoomStore } from '@/stores/room';
import { getEquipment, getWeapon } from '@/game/equipment';

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
  <div class="card rpg-corners p-5 flex flex-col">
    <div class="flex items-center justify-between mb-3">
      <h3 class="rpg-title text-lg">
        <span class="rune-mark mr-1">⚜</span>店内传话筒
      </h3>
      <span class="font-serif text-[11px] tracking-[0.2em] uppercase text-gold-300/70">
        {{ events.length }} 条吆喝
      </span>
    </div>
    <div class="rpg-divider mb-3" />
    <div class="flex-1 overflow-y-auto max-h-[420px] pr-1 space-y-2">
      <transition-group name="feed">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="feed-item rounded-md px-3 py-2 text-sm font-body"
        >
          <div class="font-serif text-[10px] tracking-widest text-gold-300/70 mb-0.5">
            ✦ {{ format(ev.at) }}
          </div>
          <template v-if="ev.kind === 'PLAYER_JOINED'">
            <span class="font-serif font-semibold text-forest-300">
              「{{ asStr(ev.payload.nickname) || nameOf(asStr(ev.payload.playerId)) }}」
            </span>
            踏入本店。
          </template>
          <template v-else-if="ev.kind === 'PLAYER_LEFT'">
            <span class="font-serif font-semibold text-oxblood-300">
              「{{ nameOf(asStr(ev.payload.playerId)) }}」
            </span>
            悄然离店。
          </template>
          <template v-else-if="ev.kind === 'LOTTERY'">
            <span class="font-serif font-semibold text-gold-200">
              「{{ nameOf(asStr(ev.payload.playerId)) }}」
            </span>
            扣开神秘货柜 ——
            <span class="font-serif font-bold text-royal-400">
              ‹{{ asStr(ev.payload.itemName) }}›
            </span>
            <span
              v-if="Number(ev.payload.itemValue) > 0"
              class="text-forest-300 ml-1"
            >
              · 价值 {{ Number(ev.payload.itemValue) }}W
            </span>
            <span
              v-if="Number(ev.payload.guaranteeDelta) > 0"
              class="text-gold-200 ml-1"
            >
              · 保底 +{{ Number(ev.payload.guaranteeDelta) }}W
              （{{ Number(ev.payload.guaranteeBefore) }} → {{ Number(ev.payload.guaranteeAfter) }}）
            </span>
          </template>
          <template v-else-if="ev.kind === 'PURCHASE'">
            <span class="font-serif font-semibold text-sapphire-300">
              「{{ nameOf(asStr(ev.payload.playerId)) }}」
            </span>
            自军械架入手
            <span class="font-serif font-bold text-parchment-50">
              ‹{{ getEquipment(asStr(ev.payload.equipmentId))?.name ?? asStr(ev.payload.equipmentId) }}›
            </span>
            <template v-if="ev.payload.pick">
              <span class="text-gold-300/70 mx-1">·</span>
              <span class="font-serif font-semibold text-forest-300">
                {{ asStr(ev.payload.pickName) || getWeapon(asStr(ev.payload.pick))?.name || asStr(ev.payload.pick) }}
              </span>
            </template>
          </template>
          <template v-else-if="ev.kind === 'GAME_STARTED'">
            <span class="rpg-title text-base font-bold tracking-widest">
              ⚜ 本店今夜开张！
            </span>
          </template>
          <template v-else-if="ev.kind === 'SETTLEMENT'">
            <span
              class="font-serif font-bold tracking-wider"
              :class="
                ev.payload.outcome === 'EVACUATED'
                  ? 'text-forest-300'
                  : 'text-oxblood-300'
              "
            >
              【盘账】{{ ev.payload.outcome === 'EVACUATED' ? '顺利出仓' : '炸单覆灭' }}
            </span>
            <span v-if="Number(ev.payload.guaranteeDelta) > 0" class="text-gold-200 ml-1">
              · 保底 {{ asStr(ev.payload.guaranteeBefore) }}W →
              {{ asStr(ev.payload.guaranteeAfter) }}W
            </span>
          </template>
          <template v-else>
            <span class="text-parchment-400/80 font-serif italic">{{ ev.kind }}</span>
          </template>
        </div>
      </transition-group>
      <div v-if="events.length === 0" class="text-center py-8">
        <div class="rune-mark text-2xl text-gold-300/60 animate-flicker">✶</div>
        <div class="font-body italic text-parchment-300/70 text-sm mt-2">
          传话筒静默，静候本店开张。
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
.feed-enter-active {
  transition: all 0.3s ease;
}
.feed-enter-from {
  opacity: 0;
  transform: translateX(10px);
}
</style>

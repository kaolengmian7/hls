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
  <div class="card p-5 flex flex-col">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-lg font-semibold text-slate-50">实时播报</h3>
      <span class="text-xs text-slate-500">{{ events.length }} 条</span>
    </div>
    <div class="flex-1 overflow-y-auto max-h-[420px] pr-1 space-y-2">
      <transition-group name="feed">
        <div
          v-for="ev in events"
          :key="ev.id"
          class="rounded-lg bg-slate-900/60 border border-white/5 px-3 py-2 text-sm"
        >
          <div class="text-[11px] text-slate-500 mb-0.5">{{ format(ev.at) }}</div>
          <template v-if="ev.kind === 'PLAYER_JOINED'">
            <span class="text-emerald-300">[{{ asStr(ev.payload.nickname) || nameOf(asStr(ev.payload.playerId)) }}]</span> 加入了房间
          </template>
          <template v-else-if="ev.kind === 'PLAYER_LEFT'">
            <span class="text-rose-300">[{{ nameOf(asStr(ev.payload.playerId)) }}]</span> 离开了房间
          </template>
          <template v-else-if="ev.kind === 'LOTTERY'">
            <span class="text-amber-300">[{{ nameOf(asStr(ev.payload.playerId)) }}]</span>
            抽中了
            <span class="text-purple-300 font-semibold">[{{ asStr(ev.payload.itemName) }}]</span>
            <span
              v-if="Number(ev.payload.itemValue) > 0"
              class="text-emerald-300 ml-1"
            >
              · 价值 {{ Number(ev.payload.itemValue) }}W
            </span>
            <span
              v-if="Number(ev.payload.guaranteeDelta) > 0"
              class="text-amber-300 ml-1"
            >
              · 保底 +{{ Number(ev.payload.guaranteeDelta) }}W（{{ Number(ev.payload.guaranteeBefore) }}
              → {{ Number(ev.payload.guaranteeAfter) }}）
            </span>
          </template>
          <template v-else-if="ev.kind === 'PURCHASE'">
            <span class="text-blue-300">[{{ nameOf(asStr(ev.payload.playerId)) }}]</span>
            购买了
            <span class="text-slate-200 font-semibold">
              [{{ getEquipment(asStr(ev.payload.equipmentId))?.name ?? asStr(ev.payload.equipmentId) }}]
            </span>
            <template v-if="ev.payload.pick">
              <span class="text-slate-400">·</span>
              <span class="text-emerald-300 font-semibold">
                {{ asStr(ev.payload.pickName) || getWeapon(asStr(ev.payload.pick))?.name || asStr(ev.payload.pick) }}
              </span>
            </template>
          </template>
          <template v-else-if="ev.kind === 'GAME_STARTED'">
            <span class="text-emerald-300 font-semibold">游戏已开始！</span>
          </template>
          <template v-else-if="ev.kind === 'SETTLEMENT'">
            <span
              :class="
                ev.payload.outcome === 'EVACUATED'
                  ? 'text-emerald-300 font-semibold'
                  : 'text-rose-300 font-semibold'
              "
            >
              [本局结算]
              {{ ev.payload.outcome === 'EVACUATED' ? '撤离成功' : '炸单' }}
            </span>
            <span v-if="Number(ev.payload.guaranteeDelta) > 0" class="text-amber-300 ml-1">
              · 保底 {{ asStr(ev.payload.guaranteeBefore) }}W →
              {{ asStr(ev.payload.guaranteeAfter) }}W
            </span>
          </template>
          <template v-else>
            <span class="text-slate-400">{{ ev.kind }}</span>
          </template>
        </div>
      </transition-group>
      <div v-if="events.length === 0" class="text-sm text-slate-500 py-6 text-center">
        暂无事件
      </div>
    </div>
  </div>
</template>

<style scoped>
.feed-enter-active {
  transition: all 0.25s ease;
}
.feed-enter-from {
  opacity: 0;
  transform: translateX(8px);
}
</style>

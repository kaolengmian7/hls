<script setup lang="ts">
/**
 * 本局结算历史记录 Modal（所有玩家可见）。
 *
 * 数据源：服务端权威 `STATE_SNAPSHOT.settlementRecords`，
 * 列表已由服务端按时间倒序维护（最新在前），无需前端再排序。
 */
import { computed } from 'vue';
import { useRoomStore } from '@/stores/room';

defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();

const roomStore = useRoomStore();

function nameOf(playerId: string): string {
  return roomStore.players.find((p) => p.id === playerId)?.nickname ?? playerId.slice(-4);
}

function pad(n: number): string {
  return String(n).padStart(2, '0');
}

function format(ts: number): string {
  const d = new Date(ts);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

const records = computed(() => roomStore.settlementRecords);
</script>

<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      @click.self="emit('close')"
    >
      <div class="card rpg-corners w-full max-w-3xl max-h-[82vh] flex flex-col animate-slide-up">
        <div class="flex items-center justify-between p-5 border-b border-gold-500/25">
          <h3 class="rpg-title text-xl">
            <span class="rune-mark mr-1">⚜</span>征战编年史
          </h3>
          <button class="btn-ghost py-1.5 px-3" @click="emit('close')">合卷</button>
        </div>
        <div class="flex-1 overflow-y-auto p-5">
          <div v-if="records.length === 0" class="text-center py-10">
            <div class="rune-mark text-2xl text-gold-300/60 animate-flicker">✶</div>
            <div class="font-body text-parchment-300/75 mt-2">编年史尚未起笔</div>
          </div>
          <ul v-else class="space-y-4">
            <li
              v-for="r in records"
              :key="r.id"
              class="rounded-md p-4 chronicle-entry"
            >
              <div class="flex items-center justify-between mb-3 gap-2 flex-wrap">
                <div class="flex items-center gap-2">
                  <span class="font-serif text-[11px] tracking-widest text-gold-300/70">
                    ✦ {{ format(r.at) }}
                  </span>
                  <span
                    class="badge"
                    :class="
                      r.outcome === 'EVACUATED'
                        ? '!border-forest-400/70 !text-forest-300'
                        : '!border-oxblood-300/70 !text-oxblood-300'
                    "
                  >
                    {{ r.outcome === 'EVACUATED' ? '凯旋归来' : '群星陨落' }}
                  </span>
                </div>
                <div class="font-serif text-xs text-gold-200">
                  库藏 {{ r.guaranteeBefore }}W →
                  <span
                    :class="
                      r.outcome === 'EVACUATED' ? 'text-forest-300' : 'text-oxblood-300'
                    "
                  >
                    {{ r.guaranteeAfter }}W
                  </span>
                  <span v-if="r.guaranteeDelta > 0" class="ml-1 text-parchment-300/80">
                    ({{ r.outcome === 'EVACUATED' ? '−' : '+' }}{{ r.guaranteeDelta }}W)
                  </span>
                </div>
              </div>
              <table class="w-full text-xs">
                <thead>
                  <tr class="text-left">
                    <th class="py-1 font-serif tracking-[0.18em] uppercase text-gold-300/80">同袍</th>
                    <th class="py-1 font-serif tracking-[0.18em] uppercase text-gold-300/80">斩敌</th>
                    <th class="py-1 font-serif tracking-[0.18em] uppercase text-gold-300/80">小红</th>
                    <th class="py-1 font-serif tracking-[0.18em] uppercase text-gold-300/80">6 格</th>
                    <th class="py-1 font-serif tracking-[0.18em] uppercase text-gold-300/80">9+ 格</th>
                    <th class="py-1 font-serif tracking-[0.18em] uppercase text-gold-300/80 text-right">所得</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="pp in r.perPlayer"
                    :key="pp.playerId"
                    class="border-t border-gold-500/15"
                  >
                    <td class="py-1.5 font-serif text-parchment-100">{{ nameOf(pp.playerId) }}</td>
                    <td class="py-1.5 text-parchment-300/90">{{ pp.kills }}</td>
                    <td class="py-1.5 text-parchment-300/90">{{ pp.smallReds }}</td>
                    <td class="py-1.5 text-parchment-300/90">{{ pp.sixGrids }}</td>
                    <td class="py-1.5 text-parchment-300/90">{{ pp.nineGrids }}</td>
                    <td
                      class="py-1.5 text-right font-serif font-bold"
                      :class="pp.coinDelta > 0 ? 'text-forest-300' : 'text-parchment-400/70'"
                    >
                      +{{ pp.coinDelta }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </li>
          </ul>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.chronicle-entry {
  background:
    linear-gradient(180deg, rgba(212, 175, 55, 0.05), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #1a0f0a, #120804);
  border: 1px solid rgba(212, 175, 55, 0.25);
  box-shadow: inset 0 1px 0 rgba(244, 230, 194, 0.06);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

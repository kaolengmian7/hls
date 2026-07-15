<script setup lang="ts">
/**
 * 本局结算历史记录 Modal（所有玩家可见）。
 *
 * 数据源：服务端权威 `STATE_SNAPSHOT.settlementRecords`，
 * 列表已由服务端按时间倒序维护（最新在前），无需前端再排序。
 */
import { computed } from 'vue';
import { useRoomStore } from '../stores/room';

defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();

const roomStore = useRoomStore();

function nameOf(playerId: string): string {
  return roomStore.players.find((p) => p.id === playerId)?.nickname ?? playerId.slice(-4);
}

function isBossCovered(guaranteeBefore: number, guaranteeAfter: number, guaranteeDelta: number, outcome: string): boolean {
  return outcome === 'EVACUATED' && guaranteeDelta > 0 && guaranteeBefore === guaranteeAfter;
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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
      @click.self="emit('close')"
    >
      <div class="hls-panel w-[92%] max-w-lg max-h-[80vh] flex flex-col animate-pixel-slide">
        <div class="hls-panel-title">结算记录</div>
        <div class="flex-1 overflow-y-auto p-5">
          <div v-if="records.length === 0" class="text-center py-10">
            <div class="text-2xl text-hls-gold/60 animate-pixel-pulse">✶</div>
            <div class="font-body text-hls-text-light/75 mt-2">暂时还没有结算记录</div>
          </div>
          <ul v-else class="space-y-4">
            <li
              v-for="r in records"
              :key="r.id"
              class="hls-menu-row rounded-md p-4"
            >
              <div class="flex items-center justify-between mb-3 gap-2 flex-wrap">
                <div class="flex items-center gap-2">
                  <span class="font-body text-[11px] text-hls-gold/70">
                    ✦ {{ format(r.at) }}
                  </span>
                  <span
                    class="hls-badge"
                    :class="
                      r.outcome === 'EVACUATED'
                        ? '!border-hls-btn-green/70 !text-white'
                        : 'hls-badge-red !text-white'
                    "
                  >
                    {{ r.outcome === 'EVACUATED' ? '撤离成功' : '炸单翻车' }}
                  </span>
                </div>
                <div class="font-body text-xs text-hls-gold">
                  库藏 {{ r.guaranteeBefore }}W →
                  <span
                    :class="
                      r.outcome === 'EVACUATED' ? 'text-hls-btn-green' : 'text-red-400'
                    "
                  >
                    {{ r.guaranteeAfter }}W
                  </span>
                  <span
                    v-if="isBossCovered(r.guaranteeBefore, r.guaranteeAfter, r.guaranteeDelta, r.outcome)"
                    class="ml-1 text-hls-btn-green"
                  >
                    (老板兜底，得吃 +{{ r.guaranteeDelta }}W)
                  </span>
                  <span
                    v-else-if="r.guaranteeDelta > 0"
                    class="ml-1 text-hls-text-muted"
                  >
                    ({{ r.outcome === 'EVACUATED' ? '−' : '+' }}{{ r.guaranteeDelta }}W)
                  </span>
                </div>
              </div>
              <table class="w-full text-xs">
                <thead>
                  <tr class="text-left">
                    <th class="py-1 font-body font-medium tracking-[0.18em] uppercase text-hls-gold/80">成员</th>
                    <th class="py-1 font-body font-medium tracking-[0.18em] uppercase text-hls-gold/80">击杀</th>
                    <th class="py-1 font-body font-medium tracking-[0.18em] uppercase text-hls-gold/80">小红</th>
                    <th class="py-1 font-body font-medium tracking-[0.18em] uppercase text-hls-gold/80">6 格</th>
                    <th class="py-1 font-body font-medium tracking-[0.18em] uppercase text-hls-gold/80">9+ 格</th>
                    <th class="py-1 font-body font-medium tracking-[0.18em] uppercase text-hls-gold/80 text-right">所得</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="pp in r.perPlayer"
                    :key="pp.playerId"
                    class="border-t border-hls-border/15"
                  >
                    <td class="py-1.5 font-body text-hls-text-light">{{ nameOf(pp.playerId) }}</td>
                    <td class="py-1.5 text-hls-text-light/90">{{ pp.kills }}</td>
                    <td class="py-1.5 text-hls-text-light/90">{{ pp.smallReds }}</td>
                    <td class="py-1.5 text-hls-text-light/90">{{ pp.sixGrids }}</td>
                    <td class="py-1.5 text-hls-text-light/90">{{ pp.nineGrids }}</td>
                    <td
                      class="py-1.5 text-right font-body font-bold"
                      :class="pp.coinDelta > 0 ? 'text-hls-btn-green' : 'text-hls-text-muted'"
                    >
                      +{{ pp.coinDelta }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </li>
          </ul>
        </div>
        <div class="p-4 flex justify-end border-t border-hls-border/30">
          <button class="hls-btn-secondary py-1.5 px-4" @click="emit('close')">关闭</button>
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

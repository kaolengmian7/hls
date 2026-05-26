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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      @click.self="emit('close')"
    >
      <div class="card w-full max-w-3xl max-h-[80vh] flex flex-col animate-slide-up">
        <div class="flex items-center justify-between p-5 border-b border-white/10">
          <h3 class="text-lg font-semibold text-slate-50">本局结算记录</h3>
          <button class="btn-ghost py-1 px-3" @click="emit('close')">关闭</button>
        </div>
        <div class="flex-1 overflow-y-auto p-5">
          <div v-if="records.length === 0" class="text-center text-slate-500 py-8">
            尚无结算记录
          </div>
          <ul v-else class="space-y-3">
            <li
              v-for="r in records"
              :key="r.id"
              class="rounded-xl border border-white/10 bg-slate-900/50 p-4"
            >
              <div class="flex items-center justify-between mb-2 gap-2 flex-wrap">
                <div class="flex items-center gap-2">
                  <span class="text-[11px] text-slate-500">{{ format(r.at) }}</span>
                  <span
                    class="badge text-[11px]"
                    :class="
                      r.outcome === 'EVACUATED'
                        ? 'bg-emerald-500/20 text-emerald-300'
                        : 'bg-rose-500/20 text-rose-300'
                    "
                  >
                    {{ r.outcome === 'EVACUATED' ? '撤离成功' : '炸单' }}
                  </span>
                </div>
                <div class="text-xs text-amber-300 font-mono">
                  保底 {{ r.guaranteeBefore }}W →
                  <span
                    :class="
                      r.outcome === 'EVACUATED' ? 'text-emerald-300' : 'text-rose-300'
                    "
                  >
                    {{ r.guaranteeAfter }}W
                  </span>
                  <span v-if="r.guaranteeDelta > 0" class="ml-1 text-slate-400">
                    ({{ r.outcome === 'EVACUATED' ? '−' : '+' }}{{ r.guaranteeDelta }}W)
                  </span>
                </div>
              </div>
              <table class="w-full text-xs">
                <thead class="text-slate-500">
                  <tr class="text-left">
                    <th class="py-1 font-medium">玩家</th>
                    <th class="py-1 font-medium">击杀</th>
                    <th class="py-1 font-medium">小红</th>
                    <th class="py-1 font-medium">6 格</th>
                    <th class="py-1 font-medium">9+ 格</th>
                    <th class="py-1 font-medium text-right">本次币</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="pp in r.perPlayer"
                    :key="pp.playerId"
                    class="border-t border-white/5"
                  >
                    <td class="py-1.5 text-slate-300">{{ nameOf(pp.playerId) }}</td>
                    <td class="py-1.5 text-slate-400">{{ pp.kills }}</td>
                    <td class="py-1.5 text-slate-400">{{ pp.smallReds }}</td>
                    <td class="py-1.5 text-slate-400">{{ pp.sixGrids }}</td>
                    <td class="py-1.5 text-slate-400">{{ pp.nineGrids }}</td>
                    <td
                      class="py-1.5 text-right font-mono"
                      :class="pp.coinDelta > 0 ? 'text-emerald-400' : 'text-slate-500'"
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
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

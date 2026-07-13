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

function isEscort(pp: { role?: string; playerId: string }): boolean {
  // 优先用记录里冻结的角色；历史记录无该字段时回落到当前房间角色。
  const role = pp.role ?? roomStore.players.find((p) => p.id === pp.playerId)?.role;
  return role === 'ESCORT';
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
                  <span class="font-pixel text-[11px] tracking-widest text-hls-gold/70">
                    ✦ {{ format(r.at) }}
                  </span>
                  <span
                    class="hls-badge"
                    :class="
                      r.outcome === 'EVACUATED'
                        ? '!border-hls-btn-green/70 !text-hls-btn-green'
                        : '!border-red-400/70 !text-red-400'
                    "
                  >
                    {{ r.outcome === 'EVACUATED' ? '撤离成功' : '炸单翻车' }}
                  </span>
                </div>
                <div class="font-pixel text-xs text-hls-gold">
                  库藏 {{ r.guaranteeBefore }}W →
                  <span
                    :class="
                      r.outcome === 'EVACUATED' ? 'text-hls-btn-green' : 'text-red-400'
                    "
                  >
                    {{ r.guaranteeAfter }}W
                  </span>
                  <span v-if="r.guaranteeDelta > 0" class="ml-1 text-hls-text-muted">
                    ({{ r.outcome === 'EVACUATED' ? '−' : '+' }}{{ r.guaranteeDelta }}W)
                  </span>
                </div>
              </div>
              <table class="w-full text-xs">
                <thead>
                  <tr class="text-left">
                    <th class="py-1 font-pixel tracking-[0.18em] uppercase text-hls-gold/80">成员</th>
                    <th class="py-1 font-pixel tracking-[0.18em] uppercase text-hls-gold/80">击杀</th>
                    <th class="py-1 font-pixel tracking-[0.18em] uppercase text-hls-gold/80">小红</th>
                    <th class="py-1 font-pixel tracking-[0.18em] uppercase text-hls-gold/80">6 格</th>
                    <th class="py-1 font-pixel tracking-[0.18em] uppercase text-hls-gold/80">9+ 格</th>
                    <th class="py-1 font-pixel tracking-[0.18em] uppercase text-hls-gold/80 text-right">所得</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="pp in r.perPlayer"
                    :key="pp.playerId"
                    class="border-t border-hls-border/15"
                  >
                    <td class="py-1.5 font-pixel text-hls-text-light">{{ nameOf(pp.playerId) }}</td>
                    <td class="py-1.5 text-hls-text-light/90">{{ pp.kills }}</td>
                    <template v-if="isEscort(pp)">
                      <td colspan="3" class="py-1.5 text-hls-text-light/90">
                        <span class="hls-badge !border-hls-text-nav/50 !text-hls-text-nav text-[10px] mr-1">护航</span>
                        救助板板：
                        <span :class="pp.rescue ? 'text-hls-btn-green' : 'text-hls-text-muted'">
                          {{ pp.rescue ? '是 (+1)' : '否' }}
                        </span>
                      </td>
                    </template>
                    <template v-else>
                      <td class="py-1.5 text-hls-text-light/90">{{ pp.smallReds }}</td>
                      <td class="py-1.5 text-hls-text-light/90">{{ pp.sixGrids }}</td>
                      <td class="py-1.5 text-hls-text-light/90">{{ pp.nineGrids }}</td>
                    </template>
                    <td
                      class="py-1.5 text-right font-pixel font-bold"
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

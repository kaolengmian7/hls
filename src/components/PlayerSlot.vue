<script setup lang="ts">
import { computed } from 'vue';
import Avatar from '@/components/Avatar.vue';
import { ROLE_LABEL } from '@/game/rules';
import type { Player } from '@/net/protocol';

const props = defineProps<{
  player: Player | null;
  isSelf?: boolean;
}>();

const roleClass = computed(() => {
  if (!props.player?.role) return 'bg-slate-700/60 text-slate-300';
  return props.player.role === 'BOSS'
    ? 'bg-orange-500/15 text-orange-300 border border-orange-400/30'
    : 'bg-blue-500/15 text-blue-300 border border-blue-400/30';
});
</script>

<template>
  <div
    class="card p-5 flex flex-col items-center gap-3 transition-all duration-150 min-h-[180px]"
    :class="player ? 'border-white/20' : 'opacity-60 border-dashed'"
  >
    <template v-if="player">
      <div class="flex items-center gap-2">
        <Avatar :seed="player.id" :name="player.nickname" size="lg" />
      </div>
      <div class="text-center">
        <div class="text-base font-semibold text-slate-50 flex items-center justify-center gap-1">
          <span v-if="player.isHost" title="房主" class="text-amber-400">👑</span>
          <span class="truncate max-w-[140px]">{{ player.nickname }}</span>
          <span v-if="isSelf" class="badge bg-indigo-500/20 text-indigo-300">我</span>
        </div>
        <div class="text-xs text-slate-400 mt-1">{{ (player.id ?? '').slice(-6) }}</div>
      </div>
      <div class="flex gap-2 items-center">
        <span class="badge" :class="roleClass">
          {{ player.role ? ROLE_LABEL[player.role] : '未选角' }}
        </span>
        <span
          class="badge"
          :class="
            player.ready
              ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-400/30'
              : 'bg-slate-600/40 text-slate-300'
          "
        >
          {{ player.ready ? '已准备' : '未准备' }}
        </span>
      </div>
    </template>
    <template v-else>
      <div class="flex-1 flex items-center justify-center text-slate-500 text-sm">
        等待加入...
      </div>
    </template>
  </div>
</template>

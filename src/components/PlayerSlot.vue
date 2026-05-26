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
  if (!props.player?.role) return '';
  return props.player.role === 'BOSS'
    ? '!border-ember-400/60 !text-ember-300'
    : '!border-sapphire-300/60 !text-sapphire-300';
});

const frameClass = computed(() => {
  if (!props.player) return 'opacity-70';
  if (props.player.ready) return 'is-ready';
  if (props.isSelf) return 'is-self';
  return '';
});
</script>

<template>
  <div
    class="player-slot card rpg-corners p-5 flex flex-col items-center gap-3 transition-all duration-200 min-h-[200px]"
    :class="frameClass"
  >
    <template v-if="player">
      <div class="relative">
        <Avatar :seed="player.id" :name="player.nickname" size="lg" />
        <span
          v-if="player.ready"
          class="absolute -bottom-1 -right-1 w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold bg-forest-500 border border-gold-400 text-parchment-50"
          title="已签契约"
        >
          ✓
        </span>
      </div>
      <div class="text-center">
        <div class="font-serif text-base font-semibold text-parchment-50 flex items-center justify-center gap-1.5">
          <span v-if="player.isHost" title="掌柜" class="text-gold-300">👑</span>
          <span class="truncate max-w-[140px]">{{ player.nickname }}</span>
          <span v-if="isSelf" class="badge !text-gold-200 !border-gold-400/60">我</span>
        </div>
        <div class="font-serif italic text-[11px] text-parchment-400/80 mt-1 tracking-wider">
          ✦ {{ (player.id ?? '').slice(-6) }} ✦
        </div>
      </div>
      <div class="flex gap-2 items-center mt-auto">
        <span class="badge" :class="roleClass">
          {{ player.role ? ROLE_LABEL[player.role] : '未占位' }}
        </span>
        <span
          class="badge"
          :class="player.ready ? '!border-forest-400/70 !text-forest-300' : ''"
        >
          {{ player.ready ? '已签约' : '未签约' }}
        </span>
      </div>
    </template>
    <template v-else>
      <div class="flex-1 flex flex-col items-center justify-center gap-2 text-parchment-400/70">
        <div class="rune-mark text-2xl animate-flicker">✦</div>
        <div class="font-serif italic text-sm tracking-widest">空位待客…</div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.player-slot.is-self {
  border-color: rgba(232, 200, 124, 0.7);
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.6),
    0 22px 50px -22px rgba(0, 0, 0, 0.85),
    inset 0 1px 0 rgba(244, 230, 194, 0.1),
    0 0 18px -2px rgba(212, 175, 55, 0.35);
}
.player-slot.is-ready {
  border-color: rgba(90, 122, 50, 0.85);
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.6),
    0 22px 50px -22px rgba(0, 0, 0, 0.85),
    inset 0 1px 0 rgba(244, 230, 194, 0.1),
    0 0 22px -4px rgba(90, 122, 50, 0.55);
}
</style>

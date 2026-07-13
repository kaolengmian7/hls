<script setup lang="ts">
import { computed } from 'vue';
import Avatar from '@shared/components/Avatar.vue';
import { ROLE_LABEL } from '../game/rules';
import type { Player } from '../net/protocol';

const props = defineProps<{
  player: Player | null;
  isSelf?: boolean;
}>();

const roleClass = computed(() => {
  if (!props.player?.role) return 'sdv-badge-gray';
  return props.player.role === 'BOSS' ? 'sdv-badge-amber' : 'sdv-badge-teal';
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
    class="sdv-panel-deep flex flex-col items-center gap-3 p-5 min-h-[180px] transition-all duration-200"
    :class="{
      'ring-2 ring-sdv-gold/60': frameClass === 'is-self',
      'ring-2 ring-sdv-btn-green/60': frameClass === 'is-ready',
      'opacity-60': !player,
    }"
  >
    <template v-if="player">
      <div class="relative">
        <Avatar :seed="player.id" :name="player.nickname" size="lg" />
        <span
          v-if="player.ready"
          class="absolute -bottom-1 -right-1 w-6 h-6 flex items-center justify-center font-pixel text-xs bg-sdv-btn-green border-2 border-sdv-btn-green-border text-white"
          title="已准备"
        >✓</span>
      </div>
      <div class="text-center">
        <div class="font-pixel text-base text-sdv-text-light flex items-center justify-center gap-1.5">
          <span v-if="player.isHost" title="房主" class="text-sdv-gold">👑</span>
          <span class="truncate max-w-[140px]">{{ player.nickname }}</span>
          <span v-if="isSelf" class="sdv-badge sdv-badge-gold !text-xs py-0 px-2">我</span>
        </div>
        <div class="font-body text-xs text-sdv-text-nav/60 mt-1">
          ✦ {{ (player.id ?? '').slice(-6) }} ✦
        </div>
      </div>
      <div class="flex gap-2 items-center mt-auto flex-wrap justify-center">
        <span class="sdv-badge" :class="roleClass">
          {{ player.role ? ROLE_LABEL[player.role] : '未选角色' }}
        </span>
        <span class="sdv-badge" :class="player.ready ? 'sdv-badge-teal' : 'sdv-badge-gray'">
          {{ player.ready ? '已准备' : '未准备' }}
        </span>
      </div>
    </template>
    <template v-else>
      <div class="flex-1 flex flex-col items-center justify-center gap-2 text-sdv-text-nav/50">
        <div class="font-pixel text-3xl animate-pixel-pulse">✦</div>
        <div class="font-pixel text-sm tracking-widest">空位等待加入…</div>
      </div>
    </template>
  </div>
</template>

<style scoped></style>

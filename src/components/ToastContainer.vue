<script setup lang="ts">
import { useRoomStore } from '@/stores/room';

const roomStore = useRoomStore();
</script>

<template>
  <div class="fixed top-6 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 pointer-events-none">
    <transition-group name="toast" tag="div" class="flex flex-col gap-2 items-center">
      <div
        v-for="t in roomStore.toasts"
        :key="t.id"
        class="px-4 py-2 rounded-xl shadow-xl text-sm font-medium pointer-events-auto backdrop-blur"
        :class="{
          'bg-emerald-500/90 text-white': t.type === 'success',
          'bg-rose-500/90 text-white': t.type === 'error',
          'bg-amber-500/90 text-white': t.type === 'warn',
          'bg-slate-700/90 text-slate-50': t.type === 'info',
        }"
      >
        {{ t.message }}
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.2s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateY(-8px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>

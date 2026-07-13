<script setup lang="ts">
import { useRoomStore } from '../stores/room';

const roomStore = useRoomStore();
</script>

<template>
  <div class="fixed top-6 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 pointer-events-none">
    <transition-group name="toast" tag="div" class="flex flex-col gap-2 items-center">
      <div
        v-for="t in roomStore.toasts"
        :key="t.id"
        class="sdv-toast pointer-events-auto"
        :class="`sdv-toast--${t.type}`"
      >
        <template v-if="t.type === 'success'">✓ </template>
        <template v-else-if="t.type === 'error'">✗ </template>
        <template v-else-if="t.type === 'warn'">⚠ </template>
        <template v-else>✦ </template>
        {{ t.message }}
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all 0.2s ease; }
.toast-enter-from { opacity: 0; transform: translateY(-10px); }
.toast-leave-to { opacity: 0; transform: translateY(-4px); }
</style>

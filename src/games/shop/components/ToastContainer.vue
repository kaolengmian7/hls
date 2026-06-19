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
        class="rpg-toast px-4 py-2 rounded-md text-sm font-serif tracking-wider pointer-events-auto"
        :class="`rpg-toast--${t.type}`"
      >
        <span class="rune-mark mr-1">
          <template v-if="t.type === 'success'">✓</template>
          <template v-else-if="t.type === 'error'">✗</template>
          <template v-else-if="t.type === 'warn'">⚠</template>
          <template v-else>✦</template>
        </span>
        {{ t.message }}
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.rpg-toast {
  border: 1px solid rgba(212, 175, 55, 0.6);
  box-shadow:
    inset 0 1px 0 rgba(255, 235, 180, 0.18),
    inset 0 -2px 0 rgba(0, 0, 0, 0.4),
    0 14px 40px -10px rgba(0, 0, 0, 0.7);
  text-shadow: 0 1px 0 rgba(0, 0, 0, 0.5);
  color: #fff7e0;
}
.rpg-toast--success {
  background: linear-gradient(180deg, #5a7a32 0%, #3f5a1f 100%);
}
.rpg-toast--error {
  background: linear-gradient(180deg, #a02323 0%, #5e1212 100%);
}
.rpg-toast--warn {
  background: linear-gradient(180deg, #c2410c 0%, #7c2d12 100%);
}
.rpg-toast--info {
  background: linear-gradient(180deg, #2a1a10 0%, #120804 100%);
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.2s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>

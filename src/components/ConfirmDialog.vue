<script setup lang="ts">
defineProps<{
  open: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
}>();
const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();
</script>

<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      @click.self="emit('cancel')"
    >
      <div class="card rpg-corners w-[92%] max-w-sm p-7 animate-slide-up text-center">
        <div class="rune-mark text-2xl text-gold-300 animate-flicker">⚜</div>
        <h3 class="rpg-title text-xl mt-1 mb-2">{{ title }}</h3>
        <div class="mx-auto max-w-[60%] rpg-divider mb-3" />
        <p class="font-body text-sm text-parchment-200/90 leading-relaxed">
          {{ message }}
        </p>
        <div class="mt-6 flex gap-3 justify-center">
          <button class="btn-ghost px-4 py-2" @click="emit('cancel')">
            {{ cancelText ?? '稍候' }}
          </button>
          <button :class="danger ? 'btn-danger' : 'btn-primary'" @click="emit('confirm')">
            {{ confirmText ?? '应允' }}
          </button>
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

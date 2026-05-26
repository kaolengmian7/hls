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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      @click.self="emit('cancel')"
    >
      <div class="card w-[92%] max-w-sm p-6 animate-slide-up">
        <h3 class="text-lg font-semibold mb-2 text-slate-50">{{ title }}</h3>
        <p class="text-sm text-slate-300 leading-relaxed">{{ message }}</p>
        <div class="mt-5 flex gap-3 justify-end">
          <button class="btn-ghost" @click="emit('cancel')">
            {{ cancelText ?? '取消' }}
          </button>
          <button :class="danger ? 'btn-danger' : 'btn-primary'" @click="emit('confirm')">
            {{ confirmText ?? '确认' }}
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

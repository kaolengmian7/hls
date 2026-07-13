<script setup lang="ts">
defineProps<{
  open: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  /** 为 true 时背板点击不触发 cancel */
  persistent?: boolean;
  /** 为 true 时隐藏取消按钮 */
  noCancel?: boolean;
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
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
      @click.self="!persistent && emit('cancel')"
    >
      <div class="sdv-panel w-[92%] max-w-sm animate-pixel-slide">
        <div class="sdv-panel-title text-center">{{ title }}</div>
        <div class="p-6 text-center">
          <p class="font-body text-sm text-sdv-text-dark leading-relaxed">
            {{ message }}
          </p>
          <div class="mt-6 flex gap-3 justify-center">
            <button v-if="!noCancel" class="sdv-btn-secondary px-5 py-2" @click="emit('cancel')">
              {{ cancelText ?? '稍候' }}
            </button>
            <button :class="danger ? 'sdv-btn-danger' : 'sdv-btn'" class="px-5 py-2" @click="emit('confirm')">
              {{ confirmText ?? '确认' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

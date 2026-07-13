<script setup lang="ts">
import { ref, computed } from 'vue'

const emit = defineEmits<{ (e: 'pass'): void }>()

const TRACK_W = 280
const KNOB_W = 44
const MAX_X = TRACK_W - KNOB_W

const passed = ref(false)
const shaking = ref(false)
const sliderX = ref(0)

const fillWidth = computed(() => sliderX.value + KNOB_W)

function onPointerDown(e: PointerEvent) {
  if (passed.value) return
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  const startClientX = e.clientX - sliderX.value

  function onMove(ev: PointerEvent) {
    const x = Math.min(Math.max(0, ev.clientX - startClientX), MAX_X)
    sliderX.value = x
    if (x / MAX_X >= 0.9) {
      sliderX.value = MAX_X
      passed.value = true
      emit('pass')
    }
  }

  function onUp() {
    if (!passed.value) sliderX.value = 0
    ;(e.currentTarget as HTMLElement)?.removeEventListener('pointermove', onMove)
    ;(e.currentTarget as HTMLElement)?.removeEventListener('pointerup', onUp)
  }

  ;(e.currentTarget as HTMLElement).addEventListener('pointermove', onMove)
  ;(e.currentTarget as HTMLElement).addEventListener('pointerup', onUp)
}

function shake() {
  if (shaking.value) return
  shaking.value = true
  setTimeout(() => { shaking.value = false }, 500)
}

function reset() {
  passed.value = false
  sliderX.value = 0
}

defineExpose({ shake, reset, passed })
</script>

<template>
  <div :class="['relative select-none', { shake: shaking }]" :style="{ width: TRACK_W + 'px' }">
    <div class="relative rounded border border-white/20 bg-white/5 overflow-hidden" style="height: 44px;">
      <!-- fill bar -->
      <div
        class="absolute left-0 top-0 h-full pointer-events-none transition-colors duration-200"
        :class="passed ? 'bg-green-500/30' : 'bg-sdv-gold/20'"
        :style="{ width: fillWidth + 'px' }"
      />
      <!-- label -->
      <span
        class="absolute inset-0 flex items-center justify-center text-xs font-pixel tracking-widest pointer-events-none"
        :class="passed ? 'text-green-400' : 'text-sdv-text-muted/70'"
      >
        {{ passed ? '验证通过 ✓' : '拖动滑块完成验证' }}
      </span>
      <!-- knob -->
      <div
        class="absolute top-0 h-full flex items-center justify-center border-r border-white/20 bg-white/10 cursor-grab active:cursor-grabbing touch-none"
        :style="{ width: KNOB_W + 'px', left: sliderX + 'px' }"
        @pointerdown.prevent="onPointerDown"
      >
        <span class="text-sdv-gold text-base select-none">›</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.shake {
  animation: captcha-shake 0.45s ease-in-out;
}
@keyframes captcha-shake {
  0%, 100% { transform: translateX(0); }
  20%       { transform: translateX(-8px); }
  45%       { transform: translateX(8px); }
  65%       { transform: translateX(-5px); }
  85%       { transform: translateX(4px); }
}
</style>

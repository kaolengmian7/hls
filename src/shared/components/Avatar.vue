<script setup lang="ts">
import { computed } from 'vue';
import { avatarColor, avatarLetter } from '@shared/utils/avatar';

const props = withDefaults(
  defineProps<{
    seed: string;
    name: string;
    size?: 'sm' | 'md' | 'lg';
  }>(),
  { size: 'md' },
);

const bg = computed(() => avatarColor(props.seed));
const letter = computed(() => avatarLetter(props.name));
const sizeCls = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'w-8 h-8 text-sm';
    case 'lg':
      return 'w-16 h-16 text-2xl';
    default:
      return 'w-12 h-12 text-lg';
  }
});
</script>

<template>
  <div
    class="avatar-portrait flex items-center justify-center font-pixel font-bold text-white shadow-inner select-none"
    :class="sizeCls"
    :style="{ backgroundColor: bg }"
  >
    {{ letter }}
  </div>
</template>

<style scoped>
.avatar-portrait {
  position: relative;
  border: 3px solid #5c3d24;
  box-shadow:
    inset 0 0 0 1px #e8c898,
    2px 2px 0 #0a0806;
  image-rendering: pixelated;
}
</style>

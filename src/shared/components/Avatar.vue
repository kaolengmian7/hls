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
    class="avatar-portrait rounded-full flex items-center justify-center font-display font-bold text-white shadow-inner select-none"
    :class="sizeCls"
    :style="{ backgroundColor: bg }"
  >
    {{ letter }}
  </div>
</template>

<style scoped>
.avatar-portrait {
  position: relative;
  border: 2px solid rgba(212, 175, 55, 0.65);
  box-shadow:
    inset 0 0 8px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(0, 0, 0, 0.6),
    0 0 0 3px rgba(20, 12, 8, 0.7),
    0 4px 10px rgba(0, 0, 0, 0.5);
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.55);
}
</style>

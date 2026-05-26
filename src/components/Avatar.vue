<script setup lang="ts">
import { computed } from 'vue';
import { avatarColor, avatarLetter } from '@/utils/avatar';

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
    class="rounded-full flex items-center justify-center font-bold text-white shadow-inner select-none"
    :class="sizeCls"
    :style="{ backgroundColor: bg }"
  >
    {{ letter }}
  </div>
</template>

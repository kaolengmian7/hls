<script setup lang="ts">
import { onMounted } from 'vue';
import { usePlayerStore } from '@/shared/stores/player';

const playerStore = usePlayerStore();

onMounted(() => {
  playerStore.ensureIdentity();
});
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <router-view v-slot="{ Component }">
      <transition name="fade" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

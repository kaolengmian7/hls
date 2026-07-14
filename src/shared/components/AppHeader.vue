<script setup lang="ts">
import { RouterLink } from 'vue-router'
import AuthWidget from './AuthWidget.vue'
</script>

<template>
  <nav class="hls-nav flex justify-between items-center">
    <div class="flex items-center gap-6">
      <RouterLink
        to="/"
        class="inline-flex items-center rounded-full border-2 border-black/70 bg-white/80 px-4 py-2 font-body text-base font-semibold tracking-widest text-black transition hover:-translate-y-0.5 hover:border-black hover:bg-white"
      >
        ✦ 花栗鼠俱乐部
      </RouterLink>
      <slot />
    </div>
    <AuthWidget />
  </nav>
</template>

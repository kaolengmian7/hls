<script setup lang="ts">
import AuthWidget from './AuthWidget.vue'
</script>

<template>
  <nav class="hls-nav flex justify-between items-center">
    <div class="flex items-center gap-6">
      <span class="font-pixel text-base tracking-widest text-hls-gold">✦ 花栗鼠俱乐部</span>
      <slot />
    </div>
    <AuthWidget />
  </nav>
</template>

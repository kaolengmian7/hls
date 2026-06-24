<script setup lang="ts">
import AuthWidget from './AuthWidget.vue'
</script>

<template>
  <nav class="flex justify-between items-center px-6 py-4 border-b border-white/5">
    <div class="flex items-center gap-6">
      <span class="rpg-subtitle text-sm tracking-widest text-gold-300">✦ 花栗鼠俱乐部</span>
      <slot />
    </div>
    <AuthWidget />
  </nav>
</template>

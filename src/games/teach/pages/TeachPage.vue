<script setup lang="ts">
import { useRouter } from 'vue-router';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <div class="text-center select-none">
      <div class="rpg-subtitle text-[10px] mb-4 animate-flicker">✦ 花栗鼠俱乐部 ✦</div>
      <div class="text-4xl mb-4">🎓</div>
      <h1 class="rpg-title text-3xl font-bold mb-3">1V1 定制教学</h1>
      <div class="mx-auto max-w-xs rpg-divider mb-6" />
      <p class="text-parchment-300/70 font-body text-sm mb-8">即将推出，敬请期待</p>
      <button class="btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
    </div>
  </div>
</template>

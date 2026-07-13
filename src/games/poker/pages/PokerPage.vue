<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <AppHeader />
    <div class="flex-1 flex flex-col items-center justify-center">
      <div class="sdv-panel max-w-sm w-full mx-4 animate-pixel-slide">
        <div class="sdv-panel-title text-center">摸红扑克牌</div>
        <div class="p-8 text-center">
          <div class="text-5xl mb-4">🃏</div>
          <p class="font-body text-sm text-sdv-text-muted mb-6">即将推出，敬请期待</p>
          <span class="sdv-badge sdv-badge-gray mb-6 inline-block">即将推出</span>
          <div class="mt-4">
            <button class="sdv-btn-secondary px-6 py-2" @click="router.push('/')">← 返回首页</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

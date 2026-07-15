<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';

const router = useRouter();

const games = [
  {
    key: 'shop',
    icon: '🎲',
    title: '神秘商店',
    route: '/shop',
    active: true,
  },
  {
    key: 'teach',
    icon: '🎓',
    title: '1V1 定制教学',
    route: '/teach',
    active: false,
  },
  {
    key: 'audition',
    icon: '🎵',
    title: '试音',
    route: '/audition',
    active: false,
  },
  {
    key: 'poker',
    icon: '🃏',
    title: '摸红扑克牌',
    route: '/poker',
    active: false,
  },
] as const;
</script>

<template>
  <div class="min-h-screen flex flex-col">

    <!-- 顶部导航 -->
    <AppHeader>
      <div class="flex gap-6 font-body text-xs text-hls-text-nav/60">
        <a href="#games" class="hover:text-hls-gold transition-colors">玩法</a>
        <span>微信服务号</span>
      </div>
    </AppHeader>

    <!-- Hero -->
    <header class="flex-none px-6 pt-10 pb-6 text-center select-none">
      <div class="font-pixel text-sm mb-4 text-hls-gold/80 tracking-widest animate-pixel-pulse">
        ✦ 花栗鼠俱乐部 · 三角洲行动 ✦
      </div>
      <h1 class="font-pixel text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em] text-hls-gold">
        花栗鼠俱乐部
      </h1>
      <div class="mt-4 mx-auto max-w-xs hls-divider" />
    </header>

    <!-- 玩法卡片 -->
    <main id="games" class="flex-1 px-4 pt-6 pb-10">
      <div class="mx-auto max-w-lg">
        <div class="flex flex-col gap-3">
          <button
            v-for="game in games"
            :key="game.key"
            class="hls-menu-row text-left"
            :class="[
              game.active ? 'active' : 'disabled',
            ]"
            :disabled="!game.active"
            @click="router.push(game.route)"
          >
            <div class="text-2xl shrink-0">{{ game.icon }}</div>
            <div class="flex-1 min-w-0">
              <div class="font-body font-bold text-base text-hls-text-dark">
                {{ game.title }}
              </div>
            </div>
            <div class="shrink-0">
              <span v-if="game.active" class="hls-badge">▶ 进入</span>
              <span v-else class="hls-badge hls-badge-gray">即将推出</span>
            </div>
          </button>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="text-center font-pixel text-sm tracking-[0.4em] uppercase text-hls-text-nav/40 py-5 border-t border-hls-border/20">
      ✥ 微信服务号 · 花栗鼠俱乐部 ✥
    </footer>

  </div>
</template>

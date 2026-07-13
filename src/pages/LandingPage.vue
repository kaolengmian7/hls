<script setup lang="ts">
import { useRouter } from 'vue-router';
import AppHeader from '@shared/components/AppHeader.vue';

const router = useRouter();

const games = [
  {
    key: 'shop',
    icon: '🎲',
    title: '神秘商店',
    desc: '趣味单 · 三人对局\n抽奖 · 装备 · 结算',
    route: '/shop',
    active: true,
  },
  {
    key: 'teach',
    icon: '🎓',
    title: '1V1 定制教学',
    desc: '专属教练 · 针对性训练\n技术进阶',
    route: '/teach',
    active: false,
  },
  {
    key: 'audition',
    icon: '🎵',
    title: '试音',
    desc: '与花栗鼠一起玩\n找到最合拍的搭档',
    route: '/audition',
    active: false,
  },
  {
    key: 'poker',
    icon: '🃏',
    title: '摸红扑克牌',
    desc: '原创趣味玩法\n手气决定一切',
    route: '/poker',
    active: false,
  },
] as const;
</script>

<template>
  <div class="min-h-screen flex flex-col">

    <!-- 顶部导航 -->
    <AppHeader>
      <div class="flex gap-6 font-pixel text-xs tracking-widest text-sdv-text-nav/60">
        <a href="#games" class="hover:text-sdv-gold transition-colors">玩法</a>
        <span>微信服务号</span>
      </div>
    </AppHeader>

    <!-- Hero -->
    <header class="flex-none px-6 pt-14 pb-10 text-center select-none border-b border-sdv-border/30">
      <div class="font-pixel text-sm mb-4 text-sdv-gold/80 tracking-widest animate-pixel-pulse">
        ✦ 花栗鼠俱乐部 · 三角洲行动 ✦
      </div>
      <h1 class="font-pixel text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em] text-sdv-gold">
        花栗鼠俱乐部
      </h1>
      <div class="mt-5 mx-auto max-w-xs sdv-divider" />
      <p class="mt-5 font-body text-sm sm:text-base text-sdv-text-light tracking-wide leading-relaxed">
        开一家最有温度的三角洲陪玩店
      </p>
      <p class="mt-2 font-body text-xs text-sdv-text-nav/60 tracking-widest">
        原创 1V1 定制教学单 · 神秘商店趣味单
      </p>
    </header>

    <!-- 玩法卡片 -->
    <main id="games" class="flex-1 px-4 py-10">
      <div class="mx-auto max-w-lg">
        <div class="font-pixel text-xs tracking-[0.4em] uppercase text-sdv-text-nav/50 mb-6 text-center">
          ❖ 我们的玩法 ❖
        </div>
        <div class="flex flex-col gap-3">
          <button
            v-for="game in games"
            :key="game.key"
            class="sdv-menu-row text-left"
            :class="[
              game.active ? 'active' : 'disabled',
            ]"
            @click="game.active && router.push(game.route)"
          >
            <div class="text-2xl shrink-0">{{ game.icon }}</div>
            <div class="flex-1 min-w-0">
              <div class="font-pixel text-base tracking-wider text-sdv-text-dark">
                {{ game.title }}
              </div>
              <div class="font-body text-xs leading-relaxed text-sdv-text-muted whitespace-pre-line mt-0.5">
                {{ game.desc }}
              </div>
            </div>
            <div class="shrink-0">
              <span v-if="game.active" class="sdv-badge">▶ 进入</span>
              <span v-else class="sdv-badge sdv-badge-gray">即将推出</span>
            </div>
          </button>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="text-center font-pixel text-sm tracking-[0.4em] uppercase text-sdv-text-nav/40 py-5 border-t border-sdv-border/20">
      ✥ 微信服务号 · 花栗鼠俱乐部 ✥
    </footer>

  </div>
</template>

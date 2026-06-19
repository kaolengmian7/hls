<script setup lang="ts">
import { useRouter } from 'vue-router';

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
    <nav class="flex justify-between items-center px-6 py-4 border-b border-white/5">
      <span class="rpg-subtitle text-sm tracking-widest text-gold-300">✦ 花栗鼠俱乐部</span>
      <div class="flex gap-6 font-serif text-xs tracking-widest text-parchment-400/60">
        <a href="#games" class="hover:text-gold-300 transition-colors">玩法</a>
        <span>微信服务号</span>
      </div>
    </nav>

    <!-- Hero -->
    <header class="flex-none px-6 pt-14 pb-10 text-center select-none border-b border-white/5">
      <div class="rpg-subtitle text-[10px] sm:text-xs mb-4 animate-flicker tracking-widest">
        ✦ 花栗鼠俱乐部 · 三角洲行动 ✦
      </div>
      <h1 class="rpg-title text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em]">
        花栗鼠俱乐部
      </h1>
      <div class="mt-5 mx-auto max-w-xs rpg-divider" />
      <p class="mt-5 font-serif text-sm sm:text-base text-gold-300 tracking-wide leading-relaxed">
        开一家最有温度的三角洲陪玩店
      </p>
      <p class="mt-2 font-body text-xs text-parchment-400/60 tracking-widest">
        原创 1V1 定制教学单 · 神秘商店趣味单
      </p>
    </header>

    <!-- 玩法卡片 -->
    <main id="games" class="flex-1 px-4 py-10">
      <div class="mx-auto max-w-lg">
        <div class="font-serif text-[10px] tracking-[0.4em] uppercase text-parchment-400/50 mb-6 text-center">
          <span class="rune-mark mr-2">❖</span>我们的玩法<span class="rune-mark ml-2">❖</span>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <button
            v-for="game in games"
            :key="game.key"
            class="card rpg-corners text-left p-5 transition-all duration-200"
            :class="game.active
              ? 'border-gold-400/60 hover:border-gold-300 hover:shadow-gold cursor-pointer'
              : 'border-white/8 opacity-60 cursor-pointer hover:opacity-75'"
            @click="router.push(game.route)"
          >
            <div class="text-2xl mb-3">{{ game.icon }}</div>
            <div
              class="font-serif text-sm font-bold mb-2 leading-snug"
              :class="game.active ? 'text-parchment-100' : 'text-parchment-400/70'"
            >
              {{ game.title }}
            </div>
            <div class="font-body text-[10px] leading-relaxed whitespace-pre-line"
              :class="game.active ? 'text-parchment-300/60' : 'text-parchment-400/40'"
            >
              {{ game.desc }}
            </div>
            <div class="mt-3">
              <span
                v-if="game.active"
                class="inline-block font-serif text-[10px] text-gold-300 tracking-wider"
              >立即开始 →</span>
              <span
                v-else
                class="inline-block font-serif text-[10px] text-parchment-400/35 tracking-wider"
              >即将推出</span>
            </div>
          </button>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="text-center font-serif text-[11px] tracking-[0.4em] uppercase text-parchment-400/40 py-5 border-t border-white/5">
      <span class="rune-mark mr-2">✥</span>微信服务号 · 花栗鼠俱乐部<span class="rune-mark ml-2">✥</span>
    </footer>

  </div>
</template>

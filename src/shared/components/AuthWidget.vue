<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Avatar from '@shared/components/Avatar.vue'

const auth = useAuthStore()
const router = useRouter()
const open = ref(false)
const wrapperRef = ref<HTMLElement | null>(null)

function handleOutsideClick(e: MouseEvent) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', handleOutsideClick))
onUnmounted(() => document.removeEventListener('click', handleOutsideClick))
</script>

<template>
  <div ref="wrapperRef" class="relative">
    <button
      v-if="!auth.user"
      class="hls-btn-ghost text-sm px-4 py-1.5"
      @click="router.push('/login')"
    >
      登录
    </button>

    <div v-else>
      <button
        class="flex items-center gap-2 hover:opacity-80 transition-opacity"
        @click="open = !open"
      >
        <Avatar :seed="String(auth.user.id)" :name="auth.user.nickname" size="sm" />
        <span class="font-body text-sm text-hls-text-nav max-w-[80px] truncate">
          {{ auth.user.nickname || '用户' }}
        </span>
      </button>

      <div
        v-if="open"
        class="absolute right-0 top-full mt-2 min-w-[120px] hls-panel-deep py-1 z-50"
      >
        <button
          v-if="auth.user.role === 'customer'"
          class="w-full text-left font-body text-sm text-hls-text-nav hover:text-hls-gold px-4 py-2 transition-colors"
          @click="router.push('/admin'); open = false"
        >
          我的教学
        </button>
        <button
          class="w-full text-left font-body text-sm text-hls-text-nav hover:text-hls-gold px-4 py-2 transition-colors"
          @click="router.push('/profile'); open = false"
        >
          个人管理
        </button>
        <button
          class="w-full text-left font-body text-sm text-hls-text-nav hover:text-hls-gold px-4 py-2 transition-colors"
          @click="auth.logout()"
        >
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>

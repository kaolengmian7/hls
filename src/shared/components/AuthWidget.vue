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
      class="font-serif text-xs tracking-widest text-gold-300 border border-gold-400/40 px-3 py-1.5 rounded hover:border-gold-300 transition-colors"
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
        <span class="font-serif text-xs text-parchment-300 tracking-wide max-w-[80px] truncate">
          {{ auth.user.nickname || '用户' }}
        </span>
      </button>

      <div
        v-if="open"
        class="absolute right-0 top-full mt-2 min-w-[100px] card border border-white/10 rounded-lg py-1 z-50"
      >
        <button
          class="w-full text-left font-serif text-xs text-parchment-300/80 hover:text-gold-300 px-4 py-2 transition-colors"
          @click="auth.logout()"
        >
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>

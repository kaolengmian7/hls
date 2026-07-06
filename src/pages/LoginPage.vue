<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import axios from 'axios'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value.trim(), password.value)
    const redirect = (route.query.redirect as string | undefined) || '/'
    router.push(redirect)
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      error.value = '账号或密码错误'
    } else {
      error.value = '登录失败，请重试'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <div class="w-full max-w-xs space-y-4">
      <p v-if="error" class="text-red-400 text-sm text-center">{{ error }}</p>

      <input
        v-model="username"
        type="text"
        autocomplete="username"
        placeholder="账号（hlsXXXX）"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        @keydown.enter="!loading && handleLogin()"
      />

      <input
        v-model="password"
        type="password"
        autocomplete="current-password"
        placeholder="密码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        @keydown.enter="!loading && handleLogin()"
      />

      <button
        class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        :disabled="loading || !username || !password"
        @click="handleLogin"
      >
        {{ loading ? '登录中…' : '登录' }}
      </button>
    </div>
  </div>
</template>

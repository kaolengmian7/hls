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
    <div class="w-full max-w-xs">
      <div class="hls-panel animate-pixel-slide">
        <div class="hls-panel-title text-center">花栗鼠俱乐部</div>
        <div class="p-6 space-y-4">
          <p v-if="error" class="font-body text-sm text-red-400 text-center">{{ error }}</p>

          <div>
            <label class="font-pixel text-sm text-hls-text-dark tracking-wider block mb-1">账号</label>
            <input
              v-model="username"
              type="text"
              autocomplete="username"
              placeholder="hlsXXXX"
              class="hls-input w-full"
              @keydown.enter="!loading && handleLogin()"
            />
          </div>

          <div>
            <label class="font-pixel text-sm text-hls-text-dark tracking-wider block mb-1">密码</label>
            <input
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="••••••"
              class="hls-input w-full"
              @keydown.enter="!loading && handleLogin()"
            />
          </div>

          <button
            class="hls-btn w-full py-2.5 mt-2"
            :disabled="loading || !username || !password"
            @click="handleLogin"
          >
            {{ loading ? '登录中…' : '▶ 登录' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

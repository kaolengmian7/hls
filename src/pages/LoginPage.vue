<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useWechatQR } from '../composables/useWechatQR'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const errorMsg = computed(() => {
  const e = route.query.error as string | undefined
  if (e === 'wechat_code_invalid') return '微信授权已过期，请重新扫码'
  if (e === 'wechat_timeout') return '微信服务超时，请稍后重试'
  if (e === 'server_error') return '服务器异常，请稍后重试'
  if (e === 'missing_token') return '登录凭证缺失，请重试'
  return ''
})

const { iframeUrl, status, start } = useWechatQR(async (token) => {
  auth.setToken(token)
  await auth.init()
  router.push('/')
})

onMounted(() => start())
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <p v-if="errorMsg" class="mb-6 text-red-400 text-sm">{{ errorMsg }}</p>

    <template v-if="status === 'expired'">
      <p class="text-parchment-400/60 text-sm mb-4">二维码已过期</p>
      <button class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300" @click="start">
        刷新二维码
      </button>
    </template>

    <template v-else-if="status === 'error'">
      <p class="text-red-400 text-sm mb-4">加载失败，请重试</p>
      <button class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300" @click="start">
        重试
      </button>
    </template>

    <template v-else>
      <div class="border border-white/10 rounded-lg p-3 bg-white">
        <iframe
          v-if="iframeUrl"
          :src="iframeUrl"
          scrolling="no"
          frameborder="0"
          sandbox="allow-scripts allow-top-navigation allow-same-origin"
          style="width: 300px; height: 400px; display: block"
        />
        <div
          v-else
          style="width: 300px; height: 400px"
          class="flex items-center justify-center"
        >
          <span class="text-gray-400 text-sm">加载中…</span>
        </div>
      </div>
      <p class="mt-4 font-body text-xs text-parchment-400/50 tracking-widest">
        使用微信扫码登录 · 花栗鼠俱乐部
      </p>
    </template>
  </div>
</template>

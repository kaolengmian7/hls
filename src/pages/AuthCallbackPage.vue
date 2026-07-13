<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

onMounted(async () => {
  const token = route.query.token as string | undefined
  if (token) {
    auth.setToken(token)
    await auth.init()
    router.replace('/')
  } else {
    router.replace('/login?error=missing_token')
  }
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center">
    <p class="font-pixel text-lg tracking-widest text-sdv-text-nav animate-pixel-pulse">正在跳转…</p>
  </div>
</template>

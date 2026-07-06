<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import http from '../http'
import axios from 'axios'

const auth = useAuthStore()
const router = useRouter()

const nickname = ref(auth.user?.nickname ?? '')
const nicknameLoading = ref(false)
const nicknameError = ref('')
const nicknameSuccess = ref(false)

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordLoading = ref(false)
const passwordError = ref('')
const passwordSuccess = ref(false)

async function updateNickname() {
  nicknameError.value = ''
  nicknameSuccess.value = false
  nicknameLoading.value = true
  try {
    await http.put('/user/nickname', { nickname: nickname.value.trim() })
    if (auth.user) auth.user.nickname = nickname.value.trim()
    nicknameSuccess.value = true
  } catch {
    nicknameError.value = '修改失败，请重试'
  } finally {
    nicknameLoading.value = false
  }
}

async function updatePassword() {
  passwordError.value = ''
  passwordSuccess.value = false
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = '两次输入的新密码不一致'
    return
  }
  if (newPassword.value.length < 6) {
    passwordError.value = '新密码至少6位'
    return
  }
  passwordLoading.value = true
  try {
    await http.put('/user/password', {
      old_password: oldPassword.value,
      new_password: newPassword.value,
    })
    oldPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordSuccess.value = true
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      passwordError.value = '旧密码错误'
    } else {
      passwordError.value = '修改失败，请重试'
    }
  } finally {
    passwordLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen px-4 py-8 max-w-md mx-auto space-y-8">
    <h1 class="rpg-title text-xl tracking-widest">个人管理</h1>

    <div v-if="auth.user" class="text-xs text-parchment-400/60 font-serif">
      账号：{{ auth.user.username }}
    </div>

    <!-- 修改昵称 -->
    <div class="card border border-white/10 rounded-lg p-6 space-y-4">
      <h2 class="font-serif text-sm text-gold-300 tracking-wide">修改昵称</h2>
      <p v-if="nicknameError" class="text-red-400 text-xs">{{ nicknameError }}</p>
      <p v-if="nicknameSuccess" class="text-green-400 text-xs">修改成功</p>
      <input
        v-model="nickname"
        type="text"
        placeholder="昵称"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <button
        class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 transition-colors"
        :disabled="nicknameLoading || !nickname.trim()"
        @click="updateNickname"
      >
        {{ nicknameLoading ? '保存中…' : '保存昵称' }}
      </button>
    </div>

    <!-- 修改密码 -->
    <div class="card border border-white/10 rounded-lg p-6 space-y-4">
      <h2 class="font-serif text-sm text-gold-300 tracking-wide">修改密码</h2>
      <p v-if="passwordError" class="text-red-400 text-xs">{{ passwordError }}</p>
      <p v-if="passwordSuccess" class="text-green-400 text-xs">密码修改成功</p>
      <input
        v-model="oldPassword"
        type="password"
        placeholder="旧密码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <input
        v-model="newPassword"
        type="password"
        placeholder="新密码（至少6位）"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <input
        v-model="confirmPassword"
        type="password"
        placeholder="确认新密码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <button
        class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 transition-colors"
        :disabled="passwordLoading || !oldPassword || !newPassword || !confirmPassword"
        @click="updatePassword"
      >
        {{ passwordLoading ? '修改中…' : '修改密码' }}
      </button>
    </div>

    <!-- 店长专属 -->
    <div v-if="auth.user?.role === 'owner'" class="card border border-gold-400/20 rounded-lg p-6">
      <button
        class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 transition-colors"
        @click="router.push('/admin')"
      >
        进入后台管理
      </button>
    </div>
  </div>
</template>

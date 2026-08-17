<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import http from '../http'
import axios from 'axios'

const auth = useAuthStore()
const router = useRouter()
const adminRoles = ['owner', 'service', 'enforcer']

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
  <div class="min-h-screen px-4 py-8 max-w-md mx-auto space-y-6">
    <h1 class="font-body font-bold text-2xl text-hls-gold">个人管理</h1>

    <div v-if="auth.user" class="font-body text-xs text-hls-text-nav/60">
      账号：{{ auth.user.username }}
    </div>

    <!-- 修改昵称 -->
    <div class="hls-panel">
      <div class="hls-panel-title">修改昵称</div>
      <div class="p-5 space-y-3">
        <p v-if="nicknameError" class="font-body text-xs text-red-400">{{ nicknameError }}</p>
        <p v-if="nicknameSuccess" class="font-body text-xs text-green-400">修改成功</p>
        <input v-model="nickname" type="text" placeholder="昵称" class="hls-input w-full" />
        <button
          class="hls-btn py-2 px-5"
          :disabled="nicknameLoading || !nickname.trim()"
          @click="updateNickname"
        >
          {{ nicknameLoading ? '保存中…' : '▶ 保存昵称' }}
        </button>
      </div>
    </div>

    <!-- 修改密码 -->
    <div class="hls-panel">
      <div class="hls-panel-title">修改密码</div>
      <div class="p-5 space-y-3">
        <p v-if="passwordError" class="font-body text-xs text-red-400">{{ passwordError }}</p>
        <p v-if="passwordSuccess" class="font-body text-xs text-green-400">密码修改成功</p>
        <input v-model="oldPassword" type="password" placeholder="旧密码" class="hls-input w-full" />
        <input v-model="newPassword" type="password" placeholder="新密码（至少6位）" class="hls-input w-full" />
        <input v-model="confirmPassword" type="password" placeholder="确认新密码" class="hls-input w-full" />
        <button
          class="hls-btn-secondary py-2 px-5"
          :disabled="passwordLoading || !oldPassword || !newPassword || !confirmPassword"
          @click="updatePassword"
        >
          {{ passwordLoading ? '修改中…' : '▶ 修改密码' }}
        </button>
      </div>
    </div>

    <!-- 后台入口 -->
    <div v-if="auth.user && adminRoles.includes(auth.user.role)" class="hls-panel">
      <div class="p-5">
        <button class="hls-btn-secondary w-full py-2.5" @click="router.push('/admin')">
          ▶ 进入后台管理
        </button>
      </div>
    </div>
  </div>
</template>

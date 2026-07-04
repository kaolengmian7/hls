<script setup lang="ts">
import { usePhoneLogin } from '../composables/usePhoneLogin'

const { phone, code, verifying, error, verifyCode } = usePhoneLogin()
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <div class="w-full max-w-xs space-y-4">
      <p v-if="error" class="text-red-400 text-sm text-center">{{ error }}</p>

      <input
        v-model="phone"
        type="tel"
        maxlength="11"
        autocomplete="tel"
        placeholder="请输入手机号"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        @keydown.enter="code.length === phone.length && !verifying && verifyCode()"
      />

      <input
        v-model="code"
        type="text"
        inputmode="numeric"
        :maxlength="phone.length || 11"
        autocomplete="one-time-code"
        placeholder="请输入验证码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60 tracking-[0.4em] text-center"
        @keydown.enter="code.length === phone.length && !verifying && verifyCode()"
      />

      <button
        class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        :disabled="verifying || phone.length < 9 || code.length !== phone.length"
        @click="verifyCode"
      >
        {{ verifying ? '验证中…' : '登录' }}
      </button>

      <p class="text-center text-xs text-parchment-400/40 font-serif">
        测试账号：900006001 &nbsp;·&nbsp; 验证码与手机号相同
      </p>
    </div>
  </div>
</template>

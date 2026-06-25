<script setup lang="ts">
import { ref } from 'vue'
import SliderCaptcha from '@shared/components/SliderCaptcha.vue'
import { usePhoneLogin } from '../composables/usePhoneLogin'

const captchaRef = ref<InstanceType<typeof SliderCaptcha> | null>(null)
const captchaPassed = ref(false)

const {
  step, phone, code, cooldown, sending, verifying, error,
  sendCode, verifyCode, maskedPhone, backToPhone,
} = usePhoneLogin()

async function handleSend() {
  if (!captchaPassed.value) {
    captchaRef.value?.shake()
    return
  }
  await sendCode(captchaPassed.value)
  captchaRef.value?.reset()
  captchaPassed.value = false
}

function handleBack() {
  captchaRef.value?.reset()
  captchaPassed.value = false
  backToPhone()
}
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <div class="w-full max-w-xs space-y-4">

      <p v-if="error" class="text-red-400 text-sm text-center">{{ error }}</p>

      <!-- Step 1: enter phone -->
      <template v-if="step === 'phone'">
        <SliderCaptcha
          ref="captchaRef"
          class="mx-auto"
          @pass="captchaPassed = true"
        />

        <input
          v-model="phone"
          type="tel"
          maxlength="11"
          autocomplete="tel"
          placeholder="请输入手机号"
          class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        />

        <button
          class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          :disabled="sending || phone.length !== 11"
          @click="handleSend"
        >
          {{ sending ? '发送中…' : '发送验证码' }}
        </button>
      </template>

      <!-- Step 2: enter code -->
      <template v-else>
        <p class="text-center font-body text-sm text-parchment-400/60">
          验证码已发至 {{ maskedPhone(phone) }}
        </p>

        <input
          v-model="code"
          type="text"
          inputmode="numeric"
          maxlength="6"
          autocomplete="one-time-code"
          placeholder="请输入6位验证码"
          class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60 tracking-[0.4em] text-center"
        />

        <button
          class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          :disabled="verifying || code.length !== 6"
          @click="verifyCode"
        >
          {{ verifying ? '验证中…' : '登录' }}
        </button>

        <button
          class="w-full font-serif text-xs py-1 transition-colors"
          :class="cooldown > 0 ? 'text-parchment-400/30 cursor-not-allowed' : 'text-parchment-400/50 hover:text-gold-300'"
          :disabled="cooldown > 0"
          @click="handleBack"
        >
          {{ cooldown > 0 ? `重新发送 (${cooldown}s)` : '重新发送' }}
        </button>
      </template>

    </div>
  </div>
</template>

import { ref, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import http from '../http'
import axios from 'axios'

export type LoginStep = 'phone' | 'code'

interface SendResp {
  ok?: boolean
  error?: string
  retry_after?: number
}

interface VerifyResp {
  token?: string
  error?: string
}

const ERROR_LABELS: Record<string, string> = {
  cooldown:     '请稍后再试',
  phone_limit:  '该号码今日发送已达上限',
  ip_limit:     '操作过于频繁，请明日再试',
  global_limit: '服务繁忙，请明日再试',
  invalid_code: '验证码错误，请重新输入',
  code_expired: '验证码已过期，请重新发送',
  send_failed:  '短信发送失败，请稍后重试',
}

export function usePhoneLogin() {
  const router = useRouter()
  const route = useRoute()
  const auth = useAuthStore()

  const step = ref<LoginStep>('phone')
  const phone = ref('')
  const code = ref('')
  const cooldown = ref(0)
  const sending = ref(false)
  const verifying = ref(false)
  const error = ref('')

  let cooldownTimer: ReturnType<typeof setInterval> | null = null

  function startCooldown(seconds: number) {
    cooldown.value = seconds
    if (cooldownTimer) clearInterval(cooldownTimer)
    cooldownTimer = setInterval(() => {
      cooldown.value = Math.max(0, cooldown.value - 1)
      if (cooldown.value === 0) {
        clearInterval(cooldownTimer!)
        cooldownTimer = null
      }
    }, 1000)
  }

  onUnmounted(() => {
    if (cooldownTimer) clearInterval(cooldownTimer)
  })

  async function sendCode(captchaPassed: boolean) {
    error.value = ''
    if (!captchaPassed) return
    sending.value = true
    try {
      await http.post<SendResp>('/auth/sms/send', {
        phone: phone.value,
        captcha: 'passed',
      })
      step.value = 'code'
      startCooldown(60)
    } catch (e: unknown) {
      if (axios.isAxiosError(e)) {
        const data = e.response?.data as SendResp | undefined
        if (data?.error === 'cooldown' && data.retry_after) {
          startCooldown(data.retry_after)
          error.value = `请 ${data.retry_after} 秒后再试`
        } else {
          error.value = ERROR_LABELS[data?.error ?? ''] ?? '发送失败，请稍后重试'
        }
      } else {
        error.value = '发送失败，请稍后重试'
      }
    } finally {
      sending.value = false
    }
  }

  async function verifyCode() {
    error.value = ''
    verifying.value = true
    try {
      const res = await http.post<VerifyResp>('/auth/sms/verify', {
        phone: phone.value,
        code: code.value,
      })
      if (res.data.token) {
        auth.setToken(res.data.token)
        try { await auth.init() } catch { /* token valid, profile fetch failed — navigate anyway */ }
        const redirect = (route.query.redirect as string | undefined) || '/'
        router.push(redirect)
      } else {
        error.value = '登录失败，请重试'
      }
    } catch (e: unknown) {
      if (axios.isAxiosError(e)) {
        const data = e.response?.data as VerifyResp | undefined
        error.value = ERROR_LABELS[data?.error ?? ''] ?? '登录失败，请重试'
      } else {
        error.value = '登录失败，请重试'
      }
    } finally {
      verifying.value = false
    }
  }

  function maskedPhone(p: string): string {
    if (p.length !== 11) return p
    return p.slice(0, 3) + '****' + p.slice(7)
  }

  function backToPhone() {
    step.value = 'phone'
    code.value = ''
    error.value = ''
  }

  return {
    step,
    phone,
    code,
    cooldown,
    sending,
    verifying,
    error,
    sendCode,
    verifyCode,
    maskedPhone,
    backToPhone,
  }
}

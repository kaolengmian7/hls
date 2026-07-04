import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import http from '../http'
import axios from 'axios'

interface VerifyResp {
  token?: string
  error?: string
}

const ERROR_LABELS: Record<string, string> = {
  invalid_code:   '验证码错误，请重新输入',
  code_expired:   '验证码已过期',
  invalid_phone:  '手机号格式不正确',
  invalid_body:   '请求格式错误',
  server_error:   '服务器内部错误，请检查服务端日志',
}

export function usePhoneLogin() {
  const router = useRouter()
  const route = useRoute()
  const auth = useAuthStore()

  const phone = ref('')
  const code = ref('')
  const verifying = ref(false)
  const error = ref('')

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
        try { await auth.init() } catch { /* navigate anyway */ }
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

  return { phone, code, verifying, error, verifyCode }
}

import { defineStore } from 'pinia'
import { ref } from 'vue'
import http from '../http'

export interface AuthUser {
  id: number
  nickname: string
  avatar_url: string
}

const TOKEN_KEY = 'hls_auth_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<AuthUser | null>(null)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
  }

  async function init() {
    if (!token.value) return
    try {
      const res = await http.get<AuthUser>('/auth/me')
      user.value = res.data
    } catch {
      token.value = null
      localStorage.removeItem(TOKEN_KEY)
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    window.location.href = '/login'
  }

  return { token, user, setToken, init, logout }
})

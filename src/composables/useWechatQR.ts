import { ref, onUnmounted } from 'vue'
import http from '../http'

type QrStatus = 'loading' | 'pending' | 'done' | 'expired' | 'error'

interface QrStartResp {
  state: string
  iframe_url: string
}

interface QrStatusResp {
  status: 'pending' | 'done' | 'expired'
  token?: string
}

export function useWechatQR(onToken: (token: string) => void) {
  const iframeUrl = ref('')
  const status = ref<QrStatus>('loading')
  let timer: ReturnType<typeof setInterval> | null = null

  function stopPolling() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  async function start() {
    stopPolling()
    status.value = 'loading'
    iframeUrl.value = ''
    try {
      const res = await http.get<QrStartResp>('/auth/wechat/qr/start')
      iframeUrl.value = res.data.iframe_url
      const state = res.data.state
      status.value = 'pending'

      timer = setInterval(async () => {
        try {
          const r = await http.get<QrStatusResp>(`/auth/wechat/qr/status?state=${state}`)
          if (r.data.status === 'done' || r.data.status === 'expired') {
            stopPolling()
            status.value = r.data.status
          }
          if (r.data.status === 'done' && r.data.token) {
            onToken(r.data.token)
          }
        } catch {
          stopPolling()
          status.value = 'error'
        }
      }, 2000)
    } catch {
      status.value = 'error'
    }
  }

  onUnmounted(() => stopPolling())

  return { iframeUrl, status, start }
}

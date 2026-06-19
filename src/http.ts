import axios from 'axios'

const http = axios.create({
  baseURL: (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '/api',
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('hls_auth_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('hls_auth_token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  },
)

export default http

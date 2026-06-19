import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import '@shared/styles/main.css'
import { useAuthStore } from './stores/auth'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)

router.isReady().then(async () => {
  const auth = useAuthStore()
  await auth.init()
  app.mount('#app')
})

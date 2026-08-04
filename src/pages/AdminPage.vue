<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import UsersPanel from './admin/UsersPanel.vue'
import OrdersPanel from './admin/OrdersPanel.vue'
import MyOrdersPanel from './admin/MyOrdersPanel.vue'

type TabKey = 'orders' | 'users' | 'my-orders'

const auth = useAuthStore()

interface TabDef { key: TabKey; label: string; roles: string[] }
const allTabs: TabDef[] = [
  { key: 'orders', label: '订单管理', roles: ['owner', 'service'] },
  { key: 'users', label: '用户管理', roles: ['owner'] },
  { key: 'my-orders', label: '我的订单', roles: ['enforcer'] },
]

const tabs = computed(() =>
  allTabs.filter((t) => auth.user && t.roles.includes(auth.user.role)),
)
const active = ref<TabKey>('orders')

onMounted(() => {
  if (tabs.value.length && !tabs.value.find((t) => t.key === active.value)) {
    active.value = tabs.value[0].key
  }
})
</script>

<template>
  <div class="min-h-screen flex">
    <aside class="w-48 shrink-0 border-r-2 border-hls-border/50 p-4 space-y-2">
      <h1 class="font-body font-bold text-lg text-hls-gold mb-4">后台管理</h1>
      <button
        v-for="t in tabs"
        :key="t.key"
        class="block w-full text-left px-3 py-2 rounded font-body text-sm"
        :class="active === t.key ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="active = t.key"
      >
        {{ t.label }}
      </button>
    </aside>

    <main class="flex-1 px-6 py-8 max-w-5xl">
      <OrdersPanel v-if="active === 'orders'" />
      <UsersPanel v-else-if="active === 'users'" />
      <MyOrdersPanel v-else-if="active === 'my-orders'" />
    </main>
  </div>
</template>

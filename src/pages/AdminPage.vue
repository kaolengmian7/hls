<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useAuthStore } from '../stores/auth'
import UsersPanel from './admin/UsersPanel.vue'
import OrdersPanel from './admin/OrdersPanel.vue'
import MyOrdersPanel from './admin/MyOrdersPanel.vue'
import EnforcersPanel from './admin/EnforcersPanel.vue'
import PayrollPanel from './admin/PayrollPanel.vue'
import TeachingPanel from './admin/TeachingPanel.vue'
import MyTeachingTeacherPanel from './admin/MyTeachingTeacherPanel.vue'
import MyTeachingStudentPanel from './admin/MyTeachingStudentPanel.vue'

type TabKey = 'orders' | 'users' | 'my-orders' | 'enforcers' | 'payroll' | 'teaching' | 'my-teaching-teacher' | 'my-teaching-student'

const auth = useAuthStore()

interface TabDef { key: TabKey; label: string; roles: string[] }
const allTabs: TabDef[] = [
  { key: 'orders', label: '订单管理', roles: ['owner', 'service'] },
  { key: 'users', label: '用户管理', roles: ['owner'] },
  { key: 'enforcers', label: '打手管理', roles: ['owner'] },
  { key: 'payroll', label: '工资管理', roles: ['owner'] },
  { key: 'my-orders', label: '我的订单', roles: ['enforcer'] },
  { key: 'teaching', label: '教学管理', roles: ['owner', 'service'] },
  { key: 'my-teaching-teacher', label: '我的教学', roles: ['enforcer'] },
  { key: 'my-teaching-student', label: '我的教学', roles: ['customer'] },
]

const tabs = computed(() =>
  allTabs.filter((t) => auth.user && t.roles.includes(auth.user.role)),
)
const active = ref<TabKey>('orders')
const drawerOpen = ref(false)

const activeLabel = computed(
  () => tabs.value.find((t) => t.key === active.value)?.label ?? '后台管理',
)

function select(key: TabKey) {
  active.value = key
  drawerOpen.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') drawerOpen.value = false
}

watch(drawerOpen, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
  if (open) window.addEventListener('keydown', onKeydown)
  else window.removeEventListener('keydown', onKeydown)
})

onMounted(() => {
  if (tabs.value.length && !tabs.value.find((t) => t.key === active.value)) {
    active.value = tabs.value[0].key
  }
})

onUnmounted(() => {
  document.body.style.overflow = ''
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="min-h-screen lg:flex">
    <!-- Mobile top bar -->
    <header
      class="lg:hidden sticky top-0 z-30 flex items-center gap-3 border-b-2 border-hls-border/50 bg-hls-bg/95 px-4 py-3 backdrop-blur"
    >
      <button
        class="inline-flex h-9 w-9 items-center justify-center rounded border border-hls-border/50 text-lg leading-none text-hls-gold-dark"
        aria-label="打开菜单"
        @click="drawerOpen = true"
      >
        ☰
      </button>
      <h1 class="font-body font-bold text-base text-hls-gold">{{ activeLabel }}</h1>
    </header>

    <!-- Desktop sidebar -->
    <aside class="hidden lg:block w-48 shrink-0 border-r-2 border-hls-border/50 p-4 space-y-2">
      <h1 class="font-body font-bold text-lg text-hls-gold mb-4">后台管理</h1>
      <button
        v-for="t in tabs"
        :key="t.key"
        class="block w-full text-left px-3 py-2 rounded font-body text-sm"
        :class="active === t.key ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="select(t.key)"
      >
        {{ t.label }}
      </button>
    </aside>

    <!-- Mobile drawer + backdrop -->
    <transition name="fade">
      <div
        v-if="drawerOpen"
        class="lg:hidden fixed inset-0 z-40 bg-black/60"
        @click="drawerOpen = false"
      />
    </transition>
    <transition name="drawer">
      <aside
        v-if="drawerOpen"
        class="lg:hidden fixed inset-y-0 left-0 z-50 w-64 border-r-2 border-hls-border/50 bg-hls-bg p-4 space-y-2 shadow-xl"
      >
        <div class="flex items-center justify-between mb-4">
          <h1 class="font-body font-bold text-lg text-hls-gold">后台管理</h1>
          <button
            class="text-xl leading-none text-hls-text-nav"
            aria-label="关闭菜单"
            @click="drawerOpen = false"
          >
            ✕
          </button>
        </div>
        <button
          v-for="t in tabs"
          :key="t.key"
          class="block w-full text-left px-3 py-2 rounded font-body text-sm"
          :class="active === t.key ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
          @click="select(t.key)"
        >
          {{ t.label }}
        </button>
      </aside>
    </transition>

    <main class="flex-1 w-full min-w-0 px-4 py-5 lg:px-6 lg:py-8 lg:max-w-5xl">
      <OrdersPanel v-if="active === 'orders'" />
      <UsersPanel v-else-if="active === 'users'" />
      <EnforcersPanel v-else-if="active === 'enforcers'" />
      <PayrollPanel v-else-if="active === 'payroll'" />
      <MyOrdersPanel v-else-if="active === 'my-orders'" />
      <TeachingPanel v-else-if="active === 'teaching'" />
      <MyTeachingTeacherPanel v-else-if="active === 'my-teaching-teacher'" />
      <MyTeachingStudentPanel v-else-if="active === 'my-teaching-student'" />
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
.drawer-enter-active, .drawer-leave-active { transition: transform 0.25s ease; }
.drawer-enter-from, .drawer-leave-to { transform: translateX(-100%); }
</style>

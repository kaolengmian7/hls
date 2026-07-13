<script setup lang="ts">
import { ref, onMounted } from 'vue'
import http from '../http'
import ConfirmDialog from '../shared/components/ConfirmDialog.vue'

interface AdminUser {
  id: number
  username: string
  nickname: string
  role: 'owner' | 'service' | 'enforcer' | 'customer'
}

const users = ref<AdminUser[]>([])
const loading = ref(false)
const listError = ref('')

// Create user modal
const showCreate = ref(false)
const createNickname = ref('')
const createRole = ref<AdminUser['role']>('customer')
const createLoading = ref(false)
const createError = ref('')
const createResult = ref<{ username: string; password: string } | null>(null)

// Confirm dialog state
const confirmOpen = ref(false)
const confirmTitle = ref('')
const confirmMessage = ref('')
const confirmAction = ref<(() => Promise<void>) | null>(null)
const confirmError = ref('')

const roleLabels: Record<AdminUser['role'], string> = {
  owner: '店长',
  service: '客服',
  enforcer: '打手',
  customer: '客户',
}

async function loadUsers() {
  loading.value = true
  listError.value = ''
  try {
    const res = await http.get<AdminUser[]>('/admin/users')
    users.value = res.data
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}

onMounted(loadUsers)

function openCreate() {
  createNickname.value = ''
  createRole.value = 'customer'
  createError.value = ''
  createResult.value = null
  showCreate.value = true
}

async function submitCreate() {
  createError.value = ''
  createLoading.value = true
  try {
    const body: { nickname?: string; role: AdminUser['role'] } = { role: createRole.value }
    if (createNickname.value.trim()) body.nickname = createNickname.value.trim()
    const res = await http.post<AdminUser & { password?: string }>('/admin/users', body)
    createResult.value = {
      username: res.data.username,
      password: res.data.password ?? '123456',
    }
    await loadUsers()
  } catch {
    createError.value = '创建失败，请重试'
  } finally {
    createLoading.value = false
  }
}

function askResetPassword(u: AdminUser) {
  confirmTitle.value = '重置密码'
  confirmMessage.value = `将 ${u.nickname || u.username} 的密码重置为 123456？`
  confirmAction.value = async () => {
    await http.put(`/admin/users/${u.id}/password`)
    await loadUsers()
  }
  confirmOpen.value = true
}

function askResetNickname(u: AdminUser) {
  confirmTitle.value = '重置昵称'
  confirmMessage.value = `将 ${u.nickname || u.username} 的昵称重置为默认值？`
  confirmAction.value = async () => {
    await http.put(`/admin/users/${u.id}/nickname`)
    await loadUsers()
  }
  confirmOpen.value = true
}

async function handleConfirm() {
  confirmError.value = ''
  try {
    await confirmAction.value?.()
  } catch {
    confirmError.value = '操作失败，请重试'
  } finally {
    confirmOpen.value = false
  }
}
</script>

<template>
  <div class="min-h-screen px-4 py-8 max-w-3xl mx-auto space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="font-pixel text-2xl tracking-widest text-hls-gold">后台管理</h1>
      <button class="hls-btn py-2 px-5" @click="openCreate">▶ 创建用户</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60 animate-pixel-pulse">加载中…</p>

    <div v-if="!loading && !listError" class="hls-panel overflow-hidden">
      <div class="hls-panel-title">用户列表</div>
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b-2 border-hls-border/50">
            <th class="text-left px-4 py-3 font-pixel text-hls-text-dark tracking-wider font-normal">ID</th>
            <th class="text-left px-4 py-3 font-pixel text-hls-text-dark tracking-wider font-normal">账号</th>
            <th class="text-left px-4 py-3 font-pixel text-hls-text-dark tracking-wider font-normal">昵称</th>
            <th class="text-left px-4 py-3 font-pixel text-hls-text-dark tracking-wider font-normal">角色</th>
            <th class="text-left px-4 py-3 font-pixel text-hls-text-dark tracking-wider font-normal">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="u in users"
            :key="u.id"
            class="border-b border-hls-border/25 last:border-0"
          >
            <td class="px-4 py-3 font-body text-hls-text-muted">{{ u.id }}</td>
            <td class="px-4 py-3 font-body text-hls-text-dark">{{ u.username }}</td>
            <td class="px-4 py-3 font-body text-hls-text-dark">{{ u.nickname }}</td>
            <td class="px-4 py-3"><span class="hls-badge hls-badge-gold">{{ roleLabels[u.role] }}</span></td>
            <td class="px-4 py-3">
              <div class="flex gap-2">
                <button class="hls-btn-secondary text-xs py-1 px-3" @click="askResetPassword(u)">重置密码</button>
                <button class="hls-btn-secondary text-xs py-1 px-3" @click="askResetNickname(u)">重置昵称</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create User Modal -->
    <transition name="fade">
      <div
        v-if="showCreate"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
        @click.self="showCreate = false"
      >
        <div class="hls-panel w-[92%] max-w-sm animate-pixel-slide">
          <div class="hls-panel-title text-center">创建用户</div>
          <div class="p-6 space-y-4">

            <div v-if="createResult" class="space-y-3">
              <p class="font-body text-sm text-hls-text-dark text-center">创建成功！</p>
              <div class="hls-stat space-y-2">
                <p class="font-body text-sm text-hls-text-muted">账号：<span class="font-pixel text-hls-text-dark">{{ createResult.username }}</span></p>
                <p class="font-body text-sm text-hls-text-muted">初始密码：<span class="font-pixel text-hls-gold-dark">{{ createResult.password }}</span></p>
              </div>
              <button class="hls-btn w-full py-2" @click="showCreate = false">关闭</button>
            </div>

            <template v-else>
              <p v-if="createError" class="font-body text-xs text-red-400 text-center">{{ createError }}</p>
              <input v-model="createNickname" type="text" placeholder="昵称（可选）" class="hls-input w-full" />
              <select v-model="createRole" class="hls-input w-full">
                <option value="customer">客户</option>
                <option value="service">客服</option>
                <option value="enforcer">打手</option>
                <option value="owner">店长</option>
              </select>
              <div class="flex gap-3 pt-1">
                <button class="flex-1 hls-btn-secondary py-2" @click="showCreate = false">取消</button>
                <button class="flex-1 hls-btn py-2 disabled:opacity-40" :disabled="createLoading" @click="submitCreate">
                  {{ createLoading ? '创建中…' : '▶ 创建' }}
                </button>
              </div>
            </template>

          </div>
        </div>
      </div>
    </transition>

    <p v-if="confirmError" class="font-body text-sm text-red-400 text-center">{{ confirmError }}</p>

    <ConfirmDialog
      :open="confirmOpen"
      :title="confirmTitle"
      :message="confirmMessage"
      confirm-text="确认"
      cancel-text="取消"
      danger
      @confirm="handleConfirm"
      @cancel="confirmOpen = false"
    />
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

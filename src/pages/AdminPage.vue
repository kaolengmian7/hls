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
      <h1 class="rpg-title text-xl tracking-widest">后台管理</h1>
      <button
        class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 transition-colors"
        @click="openCreate"
      >
        创建用户
      </button>
    </div>

    <p v-if="listError" class="text-red-400 text-sm">{{ listError }}</p>
    <p v-if="loading" class="text-parchment-400/60 text-sm font-serif">加载中…</p>

    <div v-if="!loading && !listError" class="card border border-white/10 rounded-lg overflow-hidden">
      <table class="w-full text-sm font-serif">
        <thead>
          <tr class="border-b border-white/10">
            <th class="text-left px-4 py-3 text-parchment-400/60 font-normal">ID</th>
            <th class="text-left px-4 py-3 text-parchment-400/60 font-normal">账号</th>
            <th class="text-left px-4 py-3 text-parchment-400/60 font-normal">昵称</th>
            <th class="text-left px-4 py-3 text-parchment-400/60 font-normal">角色</th>
            <th class="text-left px-4 py-3 text-parchment-400/60 font-normal">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="u in users"
            :key="u.id"
            class="border-b border-white/5 last:border-0"
          >
            <td class="px-4 py-3 text-parchment-400/60">{{ u.id }}</td>
            <td class="px-4 py-3 text-parchment-200">{{ u.username }}</td>
            <td class="px-4 py-3 text-parchment-200">{{ u.nickname }}</td>
            <td class="px-4 py-3 text-gold-300/80">{{ roleLabels[u.role] }}</td>
            <td class="px-4 py-3">
              <div class="flex gap-2">
                <button
                  class="text-xs text-parchment-300/70 border border-white/10 px-2 py-1 rounded hover:text-gold-300 hover:border-gold-400/40 transition-colors"
                  @click="askResetPassword(u)"
                >
                  重置密码
                </button>
                <button
                  class="text-xs text-parchment-300/70 border border-white/10 px-2 py-1 rounded hover:text-gold-300 hover:border-gold-400/40 transition-colors"
                  @click="askResetNickname(u)"
                >
                  重置昵称
                </button>
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
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
        @click.self="showCreate = false"
      >
        <div class="card rpg-corners w-[92%] max-w-sm p-7 space-y-4 animate-slide-up">
          <h3 class="rpg-title text-lg text-center">创建用户</h3>

          <div v-if="createResult" class="space-y-3">
            <p class="text-parchment-200/90 font-serif text-sm text-center">创建成功！</p>
            <div class="bg-white/5 border border-white/10 rounded p-4 space-y-2 font-serif text-sm">
              <p class="text-parchment-400/60">账号：<span class="text-parchment-200">{{ createResult.username }}</span></p>
              <p class="text-parchment-400/60">初始密码：<span class="text-gold-300">{{ createResult.password }}</span></p>
            </div>
            <button
              class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 transition-colors"
              @click="showCreate = false"
            >
              关闭
            </button>
          </div>

          <template v-else>
            <p v-if="createError" class="text-red-400 text-xs text-center">{{ createError }}</p>

            <div class="space-y-3">
              <input
                v-model="createNickname"
                type="text"
                placeholder="昵称（可选）"
                class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
              />

              <select
                v-model="createRole"
                class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 focus:outline-none focus:border-gold-400/60"
              >
                <option value="customer">客户</option>
                <option value="service">客服</option>
                <option value="enforcer">打手</option>
                <option value="owner">店长</option>
              </select>
            </div>

            <div class="flex gap-3 pt-2">
              <button
                class="flex-1 btn-ghost px-4 py-2 font-serif text-sm"
                @click="showCreate = false"
              >
                取消
              </button>
              <button
                class="flex-1 btn-primary font-serif text-sm disabled:opacity-40"
                :disabled="createLoading"
                @click="submitCreate"
              >
                {{ createLoading ? '创建中…' : '创建' }}
              </button>
            </div>
          </template>
        </div>
      </div>
    </transition>

    <!-- Confirm error feedback -->
    <p v-if="confirmError" class="text-red-400 text-sm text-center">{{ confirmError }}</p>

    <!-- Confirm Dialog -->
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
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

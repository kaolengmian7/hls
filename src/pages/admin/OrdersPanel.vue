<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listOrders, createOrder, removeEnforcer, updateOrderStatus, updateComplaint,
  type CreateOrderPayload,
} from '../../api/orders'
import {
  orderTypeLabels, orderStatusLabels, complaintLabels,
  type OrderView, type OrderType, type OrderStatus, type ComplaintStatus,
} from '../../shared/orderLabels'

const orders = ref<OrderView[]>([])
const loading = ref(false)
const listError = ref('')

async function load() {
  loading.value = true
  listError.value = ''
  try {
    orders.value = await listOrders()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ---- create ----
const showCreate = ref(false)
const form = ref<CreateOrderPayload>({
  title: '', orderType: 'companion', price: 0,
  commissionRate: 20, bonusRate: 3, playerCount: 1, remark: '',
})
const createError = ref('')
const creating = ref(false)

function openCreate() {
  form.value = { title: '', orderType: 'companion', price: 0, commissionRate: 20, bonusRate: 3, playerCount: 1, remark: '' }
  createError.value = ''
  showCreate.value = true
}

async function submitCreate() {
  createError.value = ''
  if (!form.value.title.trim() || form.value.price <= 0) {
    createError.value = '请填写单子名称与价格'
    return
  }
  creating.value = true
  try {
    await createOrder({ ...form.value, title: form.value.title.trim() })
    showCreate.value = false
    await load()
  } catch {
    createError.value = '创建失败，请重试'
  } finally {
    creating.value = false
  }
}

function shareLink(code: string) {
  const url = `${window.location.origin}/order/${code}`
  navigator.clipboard?.writeText(url)
  copiedCode.value = code
  setTimeout(() => { if (copiedCode.value === code) copiedCode.value = '' }, 1500)
}
const copiedCode = ref('')

async function onRemoveEnforcer(code: string, slot: 'a' | 'b') {
  try {
    await removeEnforcer(code, slot)
    await load()
  } catch { listError.value = '操作失败，请重试' }
}

async function onStatus(code: string, status: OrderStatus) {
  try { await updateOrderStatus(code, status); await load() }
  catch { listError.value = '操作失败，请重试' }
}

async function onComplaint(code: string, complaintStatus: ComplaintStatus) {
  try { await updateComplaint(code, complaintStatus); await load() }
  catch { listError.value = '操作失败，请重试' }
}

const orderTypeOptions = Object.entries(orderTypeLabels) as [OrderType, string][]
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">订单管理</h2>
      <button class="hls-btn py-2 px-5" @click="openCreate">▶ 创建订单</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60">加载中…</p>

    <div v-if="!loading" class="hls-panel overflow-x-auto">
      <div class="hls-panel-title">订单列表</div>
      <table class="w-full text-sm whitespace-nowrap">
        <thead>
          <tr class="border-b-2 border-hls-border/50 text-hls-text-dark">
            <th class="text-left px-3 py-2 font-medium">编号</th>
            <th class="text-left px-3 py-2 font-medium">名称</th>
            <th class="text-left px-3 py-2 font-medium">类型</th>
            <th class="text-left px-3 py-2 font-medium">价格</th>
            <th class="text-left px-3 py-2 font-medium">人数</th>
            <th class="text-left px-3 py-2 font-medium">打手A/B</th>
            <th class="text-left px-3 py-2 font-medium">状态</th>
            <th class="text-left px-3 py-2 font-medium">客诉</th>
            <th class="text-left px-3 py-2 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.code" class="border-b border-hls-border/25 last:border-0">
            <td class="px-3 py-2 text-hls-text-muted">{{ o.code }}</td>
            <td class="px-3 py-2 text-hls-text-dark">{{ o.title }}</td>
            <td class="px-3 py-2">{{ orderTypeLabels[o.orderType] }}</td>
            <td class="px-3 py-2">{{ o.price }}</td>
            <td class="px-3 py-2">{{ o.playerCount }}</td>
            <td class="px-3 py-2">
              <div>A：{{ o.enforcerA?.name || '—' }}
                <button v-if="o.enforcerA" class="text-xs text-red-400 ml-1" @click="onRemoveEnforcer(o.code, 'a')">移除</button>
              </div>
              <div v-if="o.playerCount === 2">B：{{ o.enforcerB?.name || '—' }}
                <button v-if="o.enforcerB" class="text-xs text-red-400 ml-1" @click="onRemoveEnforcer(o.code, 'b')">移除</button>
              </div>
            </td>
            <td class="px-3 py-2">
              <select :value="o.status" class="hls-input text-xs py-1"
                @change="onStatus(o.code, ($event.target as HTMLSelectElement).value as OrderStatus)">
                <option v-for="(label, key) in orderStatusLabels" :key="key" :value="key">{{ label }}</option>
              </select>
            </td>
            <td class="px-3 py-2">
              <select :value="o.complaintStatus" class="hls-input text-xs py-1"
                @change="onComplaint(o.code, ($event.target as HTMLSelectElement).value as ComplaintStatus)">
                <option v-for="(label, key) in complaintLabels" :key="key" :value="key">{{ label }}</option>
              </select>
            </td>
            <td class="px-3 py-2">
              <button class="hls-btn-secondary text-xs py-1 px-3" @click="shareLink(o.code)">
                {{ copiedCode === o.code ? '已复制' : '复制链接' }}
              </button>
            </td>
          </tr>
          <tr v-if="!orders.length"><td colspan="9" class="px-3 py-6 text-center text-hls-text-nav/60">暂无订单</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <transition name="fade">
      <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4" @click.self="showCreate = false">
        <div class="hls-panel w-[92%] max-w-md">
          <div class="hls-panel-title text-center">创建订单</div>
          <div class="p-6 space-y-3">
            <p v-if="createError" class="font-body text-xs text-red-400 text-center">{{ createError }}</p>
            <input v-model="form.title" placeholder="单子名称" class="hls-input w-full" />
            <select v-model="form.orderType" class="hls-input w-full">
              <option v-for="[key, label] in orderTypeOptions" :key="key" :value="key">{{ label }}</option>
            </select>
            <div class="flex gap-3">
              <input v-model.number="form.price" type="number" placeholder="价格" class="hls-input w-full" />
              <select v-model.number="form.playerCount" class="hls-input w-full">
                <option :value="1">1 人</option>
                <option :value="2">2 人</option>
              </select>
            </div>
            <div class="flex gap-3">
              <input v-model.number="form.commissionRate" type="number" placeholder="抽成%" class="hls-input w-full" />
              <input v-model.number="form.bonusRate" type="number" placeholder="鸡腿%" class="hls-input w-full" />
            </div>
            <textarea v-model="form.remark" placeholder="备注（可选）" class="hls-input w-full" rows="2" />
            <div class="flex gap-3 pt-1">
              <button class="flex-1 hls-btn-secondary py-2" @click="showCreate = false">取消</button>
              <button class="flex-1 hls-btn py-2 disabled:opacity-40" :disabled="creating" @click="submitCreate">
                {{ creating ? '创建中…' : '▶ 创建' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

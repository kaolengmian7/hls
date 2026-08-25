<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getOrder, claimOrder, settleOrder } from '../api/orders'
import { useAuthStore } from '../stores/auth'
import {
  orderTypeLabels, orderStatusLabels, complaintLabels, type OrderView, type OrderStatus,
  type ComplaintStatus,
} from '../shared/orderLabels'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const code = route.params.code as string
const order = ref<OrderView | null>(null)
const loading = ref(true)
const loadError = ref('')
const actionError = ref('')
const claiming = ref(false)
const settling = ref(false)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    order.value = await getOrder(code)
  } catch {
    loadError.value = '订单不存在或已失效'
  } finally {
    loading.value = false
  }
}
onMounted(load)

const meId = computed(() => auth.user?.id ?? null)
const alreadyClaimed = computed(() =>
  !!order.value &&
  ((order.value.enforcerA?.id === meId.value) || (order.value.enforcerB?.id === meId.value)),
)
const isFull = computed(() => {
  const o = order.value
  if (!o) return false
  return o.playerCount === 1 ? !!o.enforcerA : !!o.enforcerA && !!o.enforcerB
})
const claimable = computed(() =>
  !!order.value && order.value.status === 'dispatching' && !isFull.value && !alreadyClaimed.value &&
  auth.user?.role === 'enforcer',
)
const canSettle = computed(() =>
  !!order.value && order.value.status === 'ongoing' &&
  auth.user?.role === 'enforcer' &&
  ((order.value.enforcerA?.id === meId.value) || (order.value.enforcerB?.id === meId.value)),
)
const fillCount = computed(() => {
  const o = order.value
  if (!o) return 0
  return Number(Boolean(o.enforcerA)) + Number(Boolean(o.enforcerB))
})
const actionLabel = computed(() => {
  if (!auth.token) return '登录并扣单'
  if (alreadyClaimed.value) return '已接单'
  if (isFull.value) return '已招满'
  if (order.value?.status !== 'dispatching') return '不可接单'
  return claiming.value ? '扣单中…' : '▶ 立即扣单'
})
const statusHint = computed(() => {
  if (!auth.token) return '登录后即可尝试扣单，系统会保留当前页面并返回。'
  if (auth.user?.role !== 'enforcer') return '当前账号不是打手角色，无法参与扣单。'
  if (alreadyClaimed.value) return '你已经在本订单中占位，无需重复操作。'
  if (isFull.value) return '当前订单名额已满，可以等待后台释放空位。'
  if (order.value?.status !== 'dispatching') return '该订单已不处于可扣单状态。'
  return '确认信息无误后即可扣单，成功后将自动占用空位。'
})

async function onClaim() {
  if (!auth.token) {
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  actionError.value = ''
  claiming.value = true
  try {
    order.value = await claimOrder(code)
  } catch {
    actionError.value = '扣单失败，可能已被他人接单'
    await load()
  } finally {
    claiming.value = false
  }
}

async function settle() {
  if (!window.confirm('确认结单？结单后将进入工资结算流程。')) return
  settling.value = true
  actionError.value = ''
  try {
    order.value = await settleOrder(code)
  } catch {
    actionError.value = '结单失败，请刷新后重试'
  } finally {
    settling.value = false
  }
}

function formatMoney(value: number) {
  return Number.isInteger(value) ? `${value}` : value.toFixed(2)
}

function getStatusClass(status: OrderStatus) {
  return {
    dispatching: 'bg-sky-100 text-sky-900 border-sky-300',
    ongoing: 'bg-amber-100 text-amber-900 border-amber-300',
    settled: 'bg-emerald-100 text-emerald-900 border-emerald-300',
    refunded: 'bg-rose-100 text-rose-900 border-rose-300',
  }[status]
}

function getComplaintClass(status: ComplaintStatus) {
  return {
    none: 'bg-stone-100 text-stone-800 border-stone-300',
    complained: 'bg-rose-100 text-rose-900 border-rose-300',
    resolved: 'bg-emerald-100 text-emerald-900 border-emerald-300',
  }[status]
}
</script>

<template>
  <div class="min-h-screen bg-hls-bg px-4 py-8 sm:px-6">
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="space-y-2 text-center">
        <p class="font-body text-sm uppercase tracking-[0.3em] text-hls-gold-dark">HLS Order Claim</p>
        <h1 class="font-body text-3xl font-bold text-hls-gold">订单扣单</h1>
        <p class="font-body text-sm text-hls-text-light">确认订单信息、查看当前空位，并直接完成扣单。</p>
      </div>

      <div v-if="loading" class="hls-panel px-6 py-10 text-center">
        <p class="font-body text-base text-hls-text-dark">加载中…</p>
        <p class="mt-2 font-body text-sm text-hls-text-light">正在获取订单详情，请稍候。</p>
      </div>

      <div v-else-if="loadError" class="hls-panel px-6 py-10 text-center">
        <p class="font-body text-lg font-bold text-red-500">订单加载失败</p>
        <p class="mt-2 font-body text-sm text-hls-text-light">{{ loadError }}</p>
      </div>

      <div v-else-if="order" class="grid gap-6 lg:grid-cols-[1.35fr_0.95fr]">
        <section class="hls-panel overflow-hidden">
          <div class="border-b border-hls-border/20 bg-hls-panel-deep/45 px-6 py-5">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">
                {{ orderTypeLabels[order.orderType] }}
              </span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="getStatusClass(order.status)"
              >
                {{ orderStatusLabels[order.status] }}
              </span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="getComplaintClass(order.complaintStatus)"
              >
                {{ complaintLabels[order.complaintStatus] }}
              </span>
            </div>

            <h2 class="mt-4 font-body text-2xl font-bold text-hls-text-dark break-all">{{ order.title }}</h2>
            <p class="mt-2 font-body text-sm text-hls-text-light break-all">订单编号：{{ order.code }}</p>
          </div>

          <div class="p-6 space-y-5">
            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">订单金额</p>
                <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(order.price) }}</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">预估收益 / 人</p>
                <p class="mt-2 font-body text-lg font-bold text-emerald-700">{{ formatMoney(order.earnings) }}</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">接单进度</p>
                <p class="mt-2 font-body text-lg font-bold text-hls-text-dark">{{ fillCount }}/{{ order.playerCount }}</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">派单人</p>
                <p class="mt-2 font-body text-lg font-bold text-hls-text-dark break-all">{{ order.dispatcherName }}</p>
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
                <p class="font-body text-sm font-semibold text-hls-text-dark">订单详情</p>
                <div class="mt-4 space-y-3 font-body text-sm">
                  <div class="flex items-center justify-between gap-4">
                    <span class="text-hls-text-light">抽成比例</span>
                    <span class="font-medium text-hls-text-dark">{{ formatMoney(order.commissionRate) }}%</span>
                  </div>
                  <div class="flex items-center justify-between gap-4">
                    <span class="text-hls-text-light">鸡腿比例</span>
                    <span class="font-medium text-hls-text-dark">{{ formatMoney(order.bonusRate) }}%</span>
                  </div>
                  <div class="flex items-center justify-between gap-4">
                    <span class="text-hls-text-light">打手人数</span>
                    <span class="font-medium text-hls-text-dark">{{ order.playerCount }} 人</span>
                  </div>
                  <div class="flex items-center justify-between gap-4">
                    <span class="text-hls-text-light">当前状态</span>
                    <span class="font-medium text-hls-text-dark">{{ orderStatusLabels[order.status] }}</span>
                  </div>
                </div>
              </div>

              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
                <p class="font-body text-sm font-semibold text-hls-text-dark">订单备注</p>
                <p class="mt-4 font-body text-sm leading-6 text-hls-text-light whitespace-pre-wrap break-words">
                  {{ order.remark || '暂无补充说明，可直接按当前信息扣单。' }}
                </p>
              </div>
            </div>
          </div>
        </section>

        <aside class="space-y-6">
          <section class="hls-panel px-5 py-5">
            <div class="flex items-center justify-between gap-3">
              <div>
                <p class="font-body text-sm font-semibold text-hls-text-dark">接单席位</p>
              </div>
              <div class="rounded-full border border-hls-border/35 bg-hls-border/10 px-3 py-1 font-body text-xs text-hls-text-dark">
                {{ fillCount }}/{{ order.playerCount }}
              </div>
            </div>

            <div class="mt-4 space-y-3">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
                <p class="font-body text-xs text-hls-text-dark/75">打手 A</p>
                <p class="mt-1 font-body text-base font-medium text-hls-text-dark break-all">{{ order.enforcerA?.name || '当前空缺' }}</p>
              </div>
              <div v-if="order.playerCount === 2" class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
                <p class="font-body text-xs text-hls-text-dark/75">打手 B</p>
                <p class="mt-1 font-body text-base font-medium text-hls-text-dark break-all">{{ order.enforcerB?.name || '当前空缺' }}</p>
              </div>
            </div>
          </section>

          <section class="hls-panel px-5 py-5">
            <p class="font-body text-sm font-semibold text-hls-text-dark">操作确认</p>
            <p class="mt-2 font-body text-sm leading-6 text-hls-text-light">{{ statusHint }}</p>

            <p v-if="actionError" class="mt-4 rounded-lg border border-red-300/50 bg-red-50 px-4 py-3 font-body text-sm text-red-500">
              {{ actionError }}
            </p>
            <p v-else-if="alreadyClaimed" class="mt-4 rounded-lg border border-hls-gold/30 bg-hls-gold/10 px-4 py-3 font-body text-sm text-hls-gold-dark">
              你已接此单，可等待后台继续处理。
            </p>

            <button
              class="mt-5 hls-btn w-full py-3 disabled:opacity-40"
              :disabled="claiming || !claimable"
              @click="onClaim"
            >
              {{ actionLabel }}
            </button>

            <button
              v-if="canSettle"
              class="mt-3 hls-btn-secondary w-full py-3"
              :disabled="settling"
              @click="settle"
            >
              {{ settling ? '结单中…' : '✅ 结单' }}
            </button>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>

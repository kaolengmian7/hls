<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { myOrders } from '../../api/orders'
import {
  complaintLabels,
  orderStatusLabels,
  orderTypeLabels,
  type ComplaintStatus,
  type OrderStatus,
  type OrderView,
} from '../../shared/orderLabels'

const orders = ref<OrderView[]>([])
const loading = ref(false)
const listError = ref('')

const summary = computed(() => ({
  total: orders.value.length,
  ongoing: orders.value.filter((order) => order.status === 'ongoing').length,
  settled: orders.value.filter((order) => order.status === 'settled').length,
  complained: orders.value.filter((order) => order.complaintStatus === 'complained').length,
}))

async function load() {
  loading.value = true
  listError.value = ''
  try {
    orders.value = await myOrders()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

function formatMoney(value: number) {
  return Number.isInteger(value) ? `${value}` : value.toFixed(2)
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
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
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">我的订单</h2>
      </div>
      <button class="hls-btn-secondary px-4 py-2.5" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新订单' }}
      </button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div class="hls-panel px-4 py-4 xl:col-span-2">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">已接订单</p>
        <div class="mt-3">
          <p class="font-body text-3xl font-bold text-hls-gold-dark">{{ summary.total }}</p>
        </div>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">进行中</p>
        <p class="mt-3 font-body text-2xl font-bold text-amber-700">{{ summary.ongoing }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">已结单</p>
        <p class="mt-3 font-body text-2xl font-bold text-emerald-700">{{ summary.settled }}</p>
      </div>
      <div class="hls-panel px-4 py-4 md:col-span-2 xl:col-span-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">被投诉</p>
        <p class="mt-3 font-body text-2xl font-bold text-rose-700">{{ summary.complained }}</p>
      </div>
    </div>

    <div v-if="!loading && orders.length" class="space-y-4">
      <article
        v-for="o in orders"
        :key="o.code"
        class="hls-panel overflow-hidden"
      >
        <div class="flex flex-col gap-4 border-b border-hls-border/20 bg-hls-panel-deep/45 px-5 py-4 lg:flex-row lg:items-start lg:justify-between">
          <div class="space-y-3 min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">
                {{ orderTypeLabels[o.orderType] }}
              </span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="getStatusClass(o.status)"
              >
                {{ orderStatusLabels[o.status] }}
              </span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="getComplaintClass(o.complaintStatus)"
              >
                {{ complaintLabels[o.complaintStatus] }}
              </span>
            </div>

            <div>
              <h3 class="font-body text-xl font-bold text-hls-text-dark break-all">{{ o.title }}</h3>
              <p class="mt-1 font-body text-xs text-hls-text-light">
                订单编号：{{ o.code }} · 接单时间：{{ formatDate(o.createdAt) }}
              </p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:min-w-[24rem]">
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">我的收益</p>
              <p class="mt-2 font-body text-lg font-bold text-emerald-700">{{ formatMoney(o.earnings) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">订单金额</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(o.price) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">抽成比例</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark">{{ formatMoney(o.commissionRate) }}%</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">鸡腿比例</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark">{{ formatMoney(o.bonusRate) }}%</p>
            </div>
          </div>
        </div>

        <div class="grid gap-4 px-5 py-5 lg:grid-cols-[1.2fr_1fr]">
          <section class="space-y-4">
            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">派单人</p>
                <p class="mt-2 font-body text-base font-semibold text-hls-text-dark break-all">{{ o.dispatcherName }}</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">需要打手</p>
                <p class="mt-2 font-body text-base font-semibold text-hls-text-dark">{{ o.playerCount }} 人</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">协作模式</p>
                <p class="mt-2 font-body text-base font-semibold text-hls-text-dark">{{ o.playerCount === 2 ? '双人协作' : '单人处理' }}</p>
              </div>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex items-center justify-between gap-3 flex-wrap">
                <p class="font-body text-sm font-semibold text-hls-text-dark">接单人员</p>
              </div>
              <div class="mt-3 grid gap-3 md:grid-cols-2">
                <div class="rounded-lg border border-hls-border/15 bg-black/10 px-4 py-3">
                  <p class="font-body text-xs text-hls-text-dark/75">打手 A</p>
                  <p class="mt-1 font-body text-base font-semibold text-hls-text-dark break-all">{{ o.enforcerA?.name || '空缺' }}</p>
                </div>
                <div v-if="o.playerCount === 2" class="rounded-lg border border-hls-border/15 bg-black/10 px-4 py-3">
                  <p class="font-body text-xs text-hls-text-dark/75">打手 B</p>
                  <p class="mt-1 font-body text-base font-semibold text-hls-text-dark break-all">{{ o.enforcerB?.name || '空缺' }}</p>
                </div>
              </div>
            </div>
          </section>

          <section class="space-y-4">
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">订单备注</p>
              <p class="mt-2 font-body text-sm leading-6 text-hls-text-light whitespace-pre-wrap break-words">
                {{ o.remark || '暂无备注信息' }}
              </p>
            </div>
          </section>
        </div>
      </article>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <div class="mx-auto max-w-sm space-y-2">
        <p class="font-body text-lg font-bold text-hls-text-dark">暂无订单</p>
        <p class="font-body text-sm text-hls-text-light">当前还没有你参与的订单，接单后会在这里展示完整详情。</p>
      </div>
    </div>
  </div>
</template>

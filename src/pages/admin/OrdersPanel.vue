<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import {
  listOrders, createOrder, removeEnforcer, updateOrderStatus, updateComplaint,
  type CreateOrderPayload,
} from '../../api/orders'
import {
  orderTypeLabels, orderStatusLabels, complaintLabels,
  type OrderView, type OrderType, type OrderStatus, type ComplaintStatus,
} from '../../shared/orderLabels'
import ConfirmDialog from '../../shared/components/ConfirmDialog.vue'

const orders = ref<OrderView[]>([])
const loading = ref(false)
const listError = ref('')

const summary = computed(() => ({
  total: orders.value.length,
  dispatching: orders.value.filter((order) => order.status === 'dispatching').length,
  ongoing: orders.value.filter((order) => order.status === 'ongoing').length,
  settled: orders.value.filter((order) => order.status === 'settled').length,
  refunded: orders.value.filter((order) => order.status === 'refunded').length,
  complained: orders.value.filter((order) => order.complaintStatus === 'complained').length,
}))

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

// ---- confirm dialog ----
const confirmOpen = ref(false)
const confirmTitle = ref('')
const confirmMessage = ref('')
const confirmAction = ref<(() => Promise<void>) | null>(null)

async function handleConfirm() {
  if (confirmAction.value) {
    try {
      await confirmAction.value()
    } catch {
      listError.value = '操作失败，请重试'
      await load()
    }
  }
  confirmOpen.value = false
}

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

const copiedCode = ref('')

function shareLink(code: string) {
  const url = `${window.location.origin}/order/${code}`
  navigator.clipboard?.writeText(url).then(() => {
    copiedCode.value = code
    setTimeout(() => { if (copiedCode.value === code) copiedCode.value = '' }, 1500)
  }).catch(() => {
    listError.value = '复制失败，请重试'
  })
}

function askRemoveEnforcer(code: string, slot: 'a' | 'b') {
  confirmTitle.value = '移除打手'
  confirmMessage.value = '确定移除该打手？移除后该位置将重新开放扣单。'
  confirmAction.value = async () => {
    await removeEnforcer(code, slot)
    await load()
  }
  confirmOpen.value = true
}

async function onStatus(code: string, status: OrderStatus) {
  if (status === 'refunded') {
    confirmTitle.value = '退单'
    confirmMessage.value = '确定将订单改为退单？'
    confirmAction.value = async () => {
      await updateOrderStatus(code, status)
      await load()
    }
    confirmOpen.value = true
  } else {
    try {
      await updateOrderStatus(code, status)
      await load()
    } catch {
      listError.value = '操作失败，请重试'
      await load()
    }
  }
}

async function onComplaint(code: string, complaintStatus: ComplaintStatus) {
  try {
    await updateComplaint(code, complaintStatus)
    await load()
  } catch {
    listError.value = '操作失败，请重试'
    await load()
  }
}

const orderTypeOptions = Object.entries(orderTypeLabels) as [OrderType, string][]

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

function getFillCount(order: OrderView) {
  return Number(Boolean(order.enforcerA)) + Number(Boolean(order.enforcerB))
}

function getRatingClass(total: number) {
  if (total >= 8) return 'bg-emerald-100 text-emerald-900 border-emerald-300'
  if (total < 6) return 'bg-rose-100 text-rose-900 border-rose-300'
  return 'bg-stone-100 text-stone-800 border-stone-300'
}

// filled in by later tasks
async function openReviewForm(_o: OrderView) {}
async function openReviewDetail(_code: string) {}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">订单管理</h2>
        <p class="font-body text-sm text-hls-text-light">查看完整订单信息、处理状态变更，并快速管理接单人员。</p>
      </div>
      <button class="hls-btn py-2 px-5 whitespace-nowrap" @click="openCreate">▶ 创建订单</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading" class="grid gap-4 md:grid-cols-2 xl:grid-cols-6">
      <div class="hls-panel px-4 py-4 xl:col-span-2">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">总订单</p>
        <div class="mt-3 flex items-end justify-between">
          <p class="font-body text-3xl font-bold text-hls-gold-dark">{{ summary.total }}</p>
          <p class="font-body text-xs text-hls-text-light">实时统计</p>
        </div>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">派单中</p>
        <p class="mt-3 font-body text-2xl font-bold text-sky-700">{{ summary.dispatching }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">进行中</p>
        <p class="mt-3 font-body text-2xl font-bold text-amber-700">{{ summary.ongoing }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">已结单</p>
        <p class="mt-3 font-body text-2xl font-bold text-emerald-700">{{ summary.settled }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">待处理客诉</p>
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
                订单编号：{{ o.code }} · 创建时间：{{ formatDate(o.createdAt) }}
              </p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:min-w-[24rem]">
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">订单金额</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(o.price) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">人均收益</p>
              <p class="mt-2 font-body text-lg font-bold text-emerald-700">{{ formatMoney(o.earnings) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">接单进度</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark">{{ getFillCount(o) }}/{{ o.playerCount }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">派单人</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark break-all">{{ o.dispatcherName }}</p>
            </div>
          </div>
        </div>

        <div class="grid gap-4 px-5 py-5 lg:grid-cols-[1.2fr_1fr]">
          <section class="space-y-4">
            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">抽成比例</p>
                <p class="mt-2 font-body text-base text-hls-text-dark">{{ formatMoney(o.commissionRate) }}%</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">鸡腿比例</p>
                <p class="mt-2 font-body text-base text-hls-text-dark">{{ formatMoney(o.bonusRate) }}%</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">需要打手</p>
                <p class="mt-2 font-body text-base text-hls-text-dark">{{ o.playerCount }} 人</p>
              </div>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex items-center justify-between gap-3 flex-wrap">
                <p class="font-body text-sm font-semibold text-hls-text-dark">接单人员</p>
                <p class="font-body text-xs text-hls-text-light">可直接移除已占位打手</p>
              </div>
              <div class="mt-3 grid gap-3 md:grid-cols-2">
                <div class="rounded-lg border border-hls-border/15 bg-black/10 px-4 py-3">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <p class="font-body text-xs text-hls-text-dark/75">打手 A</p>
                      <p class="mt-1 font-body text-base text-hls-text-dark break-all">{{ o.enforcerA?.name || '空缺' }}</p>
                    </div>
                    <button v-if="o.enforcerA" class="hls-btn-danger text-xs px-3 py-1.5" @click="askRemoveEnforcer(o.code, 'a')">
                      移除
                    </button>
                  </div>
                </div>
                <div v-if="o.playerCount === 2" class="rounded-lg border border-hls-border/15 bg-black/10 px-4 py-3">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <p class="font-body text-xs text-hls-text-dark/75">打手 B</p>
                      <p class="mt-1 font-body text-base text-hls-text-dark break-all">{{ o.enforcerB?.name || '空缺' }}</p>
                    </div>
                    <button v-if="o.enforcerB" class="hls-btn-danger text-xs px-3 py-1.5" @click="askRemoveEnforcer(o.code, 'b')">
                      移除
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">订单备注</p>
              <p class="mt-2 font-body text-sm leading-6 text-hls-text-light whitespace-pre-wrap break-words">
                {{ o.remark || '暂无备注信息' }}
              </p>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex items-center justify-between gap-3 flex-wrap">
                <p class="font-body text-sm font-semibold text-hls-text-dark">订单评价</p>
                <div class="flex gap-2">
                  <button class="hls-btn-secondary text-xs px-3 py-1.5" @click="openReviewForm(o)">
                    {{ o.review ? '修改评价' : '录入评价' }}
                  </button>
                  <button v-if="o.review" class="hls-btn-secondary text-xs px-3 py-1.5" @click="openReviewDetail(o.code)">
                    评价详情
                  </button>
                </div>
              </div>
              <div v-if="o.review" class="mt-3 flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-3 py-1 font-body text-xs text-hls-text-dark">态度 {{ o.review.attitudeScore }}/5</span>
                <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-3 py-1 font-body text-xs text-hls-text-dark">技术 {{ o.review.technicalScore }}/5</span>
                <span class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs" :class="getRatingClass(o.review.total)">总分 {{ o.review.total }}/10</span>
                <span v-if="o.review.bonusQualified" class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">🍗 鸡腿达标</span>
              </div>
              <p v-else class="mt-3 font-body text-sm text-hls-text-light">暂无评价</p>
            </div>
          </section>

          <section class="space-y-4">
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">订单操作</p>
              <div class="mt-4 space-y-4">
                <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">订单状态</span>
                  <select
                    :value="o.status"
                    class="hls-input mt-2 w-full"
                    @change="onStatus(o.code, ($event.target as HTMLSelectElement).value as OrderStatus)"
                  >
                    <option v-for="(label, key) in orderStatusLabels" :key="key" :value="key">{{ label }}</option>
                  </select>
                </label>

                <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">客诉处理</span>
                  <select
                    :value="o.complaintStatus"
                    class="hls-input mt-2 w-full"
                    @change="onComplaint(o.code, ($event.target as HTMLSelectElement).value as ComplaintStatus)"
                  >
                    <option v-for="(label, key) in complaintLabels" :key="key" :value="key">{{ label }}</option>
                  </select>
                </label>

                <button class="hls-btn-secondary w-full py-2.5" @click="shareLink(o.code)">
                  {{ copiedCode === o.code ? '已复制接单链接' : '复制接单链接' }}
                </button>
              </div>
            </div>
          </section>
        </div>
      </article>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <div class="mx-auto max-w-sm space-y-2">
        <p class="font-body text-lg font-bold text-hls-text-dark">暂无订单</p>
        <p class="font-body text-sm text-hls-text-light">当前还没有可管理的订单，创建后会在这里展示完整详情。</p>
      </div>
    </div>

    <!-- Create Modal -->
    <transition name="fade">
      <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4" @click.self="showCreate = false">
        <div class="hls-panel w-[94%] max-w-2xl">
          <div class="hls-panel-title text-center">创建订单</div>
          <div class="p-6 space-y-5">
            <p v-if="createError" class="font-body text-xs text-red-400 text-center">{{ createError }}</p>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="block md:col-span-2">
                  <span class="font-body text-xs text-hls-text-dark/75">订单标题</span>
                <input v-model="form.title" placeholder="例如：王者荣耀钻石局陪玩" class="hls-input mt-2 w-full" />
              </label>

              <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">订单类型</span>
                <select v-model="form.orderType" class="hls-input mt-2 w-full">
                  <option v-for="[key, label] in orderTypeOptions" :key="key" :value="key">{{ label }}</option>
                </select>
              </label>

              <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">打手人数</span>
                <select v-model.number="form.playerCount" class="hls-input mt-2 w-full">
                  <option :value="1">1 人</option>
                  <option :value="2">2 人</option>
                </select>
              </label>

              <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">订单价格</span>
                <input v-model.number="form.price" type="number" placeholder="请输入价格" class="hls-input mt-2 w-full" />
              </label>

              <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">抽成比例</span>
                <input v-model.number="form.commissionRate" type="number" placeholder="例如 20" class="hls-input mt-2 w-full" />
              </label>

              <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">鸡腿比例</span>
                <input v-model.number="form.bonusRate" type="number" placeholder="例如 3" class="hls-input mt-2 w-full" />
              </label>

              <label class="block md:col-span-2">
                  <span class="font-body text-xs text-hls-text-dark/75">备注</span>
                <textarea v-model="form.remark" placeholder="补充需求、账号段位、时间安排等信息" class="hls-input mt-2 w-full" rows="3" />
              </label>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">预计人均收益</p>
              <p class="mt-2 font-body text-xl font-bold text-emerald-700">
                {{
                  formatMoney(
                    form.playerCount > 0
                      ? ((form.price * (1 - form.commissionRate / 100)) + (form.price * form.bonusRate / 100)) / form.playerCount
                      : 0,
                  )
                }}
              </p>
            </div>

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

    <!-- Confirm Dialog -->
    <ConfirmDialog
      :open="confirmOpen"
      :title="confirmTitle"
      :message="confirmMessage"
      danger
      @confirm="handleConfirm"
      @cancel="() => { confirmOpen = false; load() }"
    />
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

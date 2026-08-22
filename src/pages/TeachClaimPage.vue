<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getTeachingOrder, claimTeachingOrder, type TeachingOrderView } from '../api/teaching'
import { useAuthStore } from '../stores/auth'
import { teachingStatusLabels, getTeachingStatusClass, type TeachingStatus } from '../shared/teachingLabels'
import { formatMoney } from '../shared/format'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const code = route.params.code as string
const order = ref<TeachingOrderView | null>(null)
const loading = ref(true)
const loadError = ref('')
const actionError = ref('')
const claiming = ref(false)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    order.value = await getTeachingOrder(code)
  } catch {
    loadError.value = '教学订单不存在或已失效'
  } finally {
    loading.value = false
  }
}
onMounted(load)

const meId = computed(() => auth.user?.id ?? null)
const alreadyClaimed = computed(() => !!order.value && order.value.teacher?.id === meId.value)
const isTaken = computed(() => !!order.value && !!order.value.teacher && !alreadyClaimed.value)
const claimable = computed(() =>
  !!order.value && order.value.status === 'dispatching' && !order.value.teacher &&
  auth.user?.role === 'enforcer',
)

const actionLabel = computed(() => {
  if (!auth.token) return '登录并接单'
  if (alreadyClaimed.value) return '已接单'
  if (isTaken.value) return '已被接单'
  if (order.value?.status !== 'dispatching') return '不可接单'
  return claiming.value ? '接单中…' : '▶ 立即接单'
})
const statusHint = computed(() => {
  if (!auth.token) return '登录后即可尝试接单，系统会保留当前页面并返回。'
  if (auth.user?.role !== 'enforcer') return '当前账号不是老师（打手）角色，无法接单。'
  if (alreadyClaimed.value) return '你已是本课程的老师，无需重复接单。'
  if (isTaken.value) return '该课程已被其他老师接单。'
  if (order.value?.status !== 'dispatching') return '该课程已不处于可接单状态。'
  return '确认信息无误后即可接单，成功后你将成为本课程的授课老师。'
})

async function onClaim() {
  if (!auth.token) {
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  actionError.value = ''
  claiming.value = true
  try {
    order.value = await claimTeachingOrder(code)
  } catch {
    actionError.value = '接单失败，可能已被他人接单'
    await load()
  } finally {
    claiming.value = false
  }
}

function hoursPercent(o: TeachingOrderView) {
  const total = o.totalMinutes || o.totalHours * 60
  if (total <= 0) return 0
  const remaining = o.remainingMinutes || o.remainingHours * 60
  return Math.max(0, Math.min(100, Math.round((remaining / total) * 100)))
}
</script>

<template>
  <div class="min-h-screen bg-hls-bg px-4 py-8 sm:px-6">
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="space-y-2 text-center">
        <p class="font-body text-sm uppercase tracking-[0.3em] text-hls-gold-dark">HLS Teaching Claim</p>
        <h1 class="font-body text-3xl font-bold text-hls-gold">教学接单</h1>
        <p class="font-body text-sm text-hls-text-light">确认课程信息、查看课时安排，并直接完成接单。</p>
      </div>

      <div v-if="loading" class="hls-panel px-6 py-10 text-center">
        <p class="font-body text-base text-hls-text-dark">加载中…</p>
        <p class="mt-2 font-body text-sm text-hls-text-light">正在获取课程详情，请稍候。</p>
      </div>

      <div v-else-if="loadError" class="hls-panel px-6 py-10 text-center">
        <p class="font-body text-lg font-bold text-red-500">课程加载失败</p>
        <p class="mt-2 font-body text-sm text-hls-text-light">{{ loadError }}</p>
      </div>

      <div v-else-if="order" class="grid gap-6 lg:grid-cols-[1.35fr_0.95fr]">
        <section class="hls-panel overflow-hidden">
          <div class="border-b border-hls-border/20 bg-hls-panel-deep/45 px-6 py-5">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">教学单</span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="getTeachingStatusClass(order.status as TeachingStatus)"
              >
                {{ teachingStatusLabels[order.status as TeachingStatus] }}
              </span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="order.bonusQualified ? 'border-amber-300 bg-amber-100 text-amber-900' : 'border-stone-300 bg-stone-100 text-stone-800'"
              >
                {{ order.bonusQualified ? '🍗 奉金达标' : '奉金未达标' }}
              </span>
            </div>

            <h2 class="mt-4 font-body text-2xl font-bold text-hls-text-dark break-all">{{ order.title }}</h2>
            <p class="mt-2 font-body text-sm text-hls-text-light break-all">课程编号：{{ order.code }} · 学员：{{ order.student.name }}</p>
          </div>

          <div class="p-6 space-y-5">
            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">课程金额</p>
                <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(order.price) }}</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">预估应发收益</p>
                <p class="mt-2 font-body text-lg font-bold text-emerald-700">{{ formatMoney(order.earnings) }}</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">总课时</p>
                <p class="mt-2 font-body text-lg font-bold text-hls-text-dark">{{ formatMoney(order.totalHours) }}h</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">派单人</p>
                <p class="mt-2 font-body text-lg font-bold text-hls-text-dark break-all">{{ order.dispatcherName }}</p>
              </div>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="font-body text-sm font-semibold text-hls-text-dark">课时银行</p>
                <p class="font-body text-xs text-hls-text-light">
                  剩余 <span class="font-bold text-hls-text-dark">{{ formatMoney(order.remainingHours) }}h</span> / 共 {{ formatMoney(order.totalHours) }}h
                </p>
              </div>
              <div class="mt-3 h-3 w-full overflow-hidden rounded-full border border-hls-border/30 bg-black/10">
                <div class="h-full rounded-full bg-gradient-to-r from-emerald-400 to-emerald-600 transition-all" :style="{ width: hoursPercent(order) + '%' }" />
              </div>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">课程详情</p>
              <div class="mt-4 space-y-3 font-body text-sm">
                <div class="flex items-center justify-between gap-4">
                  <span class="text-hls-text-light">分成比例</span>
                  <span class="font-medium text-hls-text-dark">{{ formatMoney(order.commissionRate) }}%</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-hls-text-light">奉金比例</span>
                  <span class="font-medium text-hls-text-dark">{{ formatMoney(order.bonusRate) }}%</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-hls-text-light">当前状态</span>
                  <span class="font-medium text-hls-text-dark">{{ teachingStatusLabels[order.status as TeachingStatus] }}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <aside class="space-y-6">
          <section class="hls-panel px-5 py-5">
            <p class="font-body text-sm font-semibold text-hls-text-dark">授课老师</p>
            <div class="mt-4 rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-xs text-hls-text-dark/75">当前老师</p>
              <p class="mt-1 font-body text-base font-medium text-hls-text-dark break-all">{{ order.teacher?.name || '当前空缺' }}</p>
            </div>
          </section>

          <section class="hls-panel px-5 py-5">
            <p class="font-body text-sm font-semibold text-hls-text-dark">操作确认</p>
            <p class="mt-2 font-body text-sm leading-6 text-hls-text-light">{{ statusHint }}</p>

            <p v-if="actionError" class="mt-4 rounded-lg border border-red-300/50 bg-red-50 px-4 py-3 font-body text-sm text-red-500">
              {{ actionError }}
            </p>
            <p v-else-if="alreadyClaimed" class="mt-4 rounded-lg border border-hls-gold/30 bg-hls-gold/10 px-4 py-3 font-body text-sm text-hls-gold-dark">
              你已接此课程，可前往「我的教学」录入成绩单。
            </p>

            <button
              class="mt-5 hls-btn w-full py-3 disabled:opacity-40"
              :disabled="claiming || !claimable"
              @click="onClaim"
            >
              {{ actionLabel }}
            </button>
          </section>
        </aside>
      </div>
    </div>
  </div>
</template>

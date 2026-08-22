<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../../stores/auth'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listTeachingOrders, createTeachingOrder, topUpTeachingOrder,
  listScorecards, listTimeLogs, listCustomers,
  type TeachingOrderView, type ScorecardView, type TimeLogView, type CustomerOption,
} from '../../api/teaching'
import {
  teachingStatusLabels, timeLogReasonLabels, getTeachingStatusClass,
  getScoreClass, estimateTeachingEarnings, groupScoresBySyllabus,
  type TeachingStatus,
} from '../../shared/teachingLabels'
import { formatMoney, formatDate } from '../../shared/format'
import ConfirmDialog from '../../shared/components/ConfirmDialog.vue'

const auth = useAuthStore()
const isOwner = computed(() => auth.user?.role === 'owner')
const syllabus = useSyllabusStore()

const rows = ref<TeachingOrderView[]>([])
const customers = ref<CustomerOption[]>([])
const loading = ref(false)
const listError = ref('')

const summary = computed(() => ({
  total: rows.value.length,
  dispatching: rows.value.filter((r) => r.status === 'dispatching').length,
  ongoing: rows.value.filter((r) => r.status === 'ongoing').length,
  settled: rows.value.filter((r) => r.status === 'settled').length,
  remainingHours: rows.value.reduce((sum, r) => sum + r.remainingHours, 0),
  earnings: rows.value.reduce((sum, r) => sum + r.earnings, 0),
}))

async function load() {
  loading.value = true
  listError.value = ''
  try {
    rows.value = await listTeachingOrders()
    customers.value = await listCustomers()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

function hoursPercent(o: TeachingOrderView) {
  const total = o.totalMinutes || o.totalHours * 60
  if (total <= 0) return 0
  const remaining = o.remainingMinutes || o.remainingHours * 60
  return Math.max(0, Math.min(100, Math.round((remaining / total) * 100)))
}

// ---- create ----
const showCreate = ref(false)
const creating = ref(false)
const createError = ref('')
const createdCreds = ref<{ username: string; password: string } | null>(null)
const emptyCreate = () => ({
  title: '', mode: 'existing' as 'existing' | 'new', studentId: 0, newStudentNickname: '',
  totalHours: 6, price: 0, commissionRate: 20, bonusRate: 0,
})
const form = ref(emptyCreate())

const previewEarnings = computed(() =>
  estimateTeachingEarnings(form.value.price, form.value.commissionRate, form.value.bonusRate),
)

function openCreate() {
  form.value = emptyCreate()
  createError.value = ''
  createdCreds.value = null
  showCreate.value = true
}

async function submitCreate() {
  createError.value = ''
  if (!form.value.title.trim() || !(form.value.price > 0)) {
    createError.value = '请填写课程标题与价格'
    return
  }
  if (form.value.mode === 'existing' && !form.value.studentId) {
    createError.value = '请选择学员'
    return
  }
  if (form.value.mode === 'new' && !form.value.newStudentNickname.trim()) {
    createError.value = '请填写新学员昵称'
    return
  }
  creating.value = true
  try {
    const payload = {
      title: form.value.title.trim(),
      totalHours: form.value.totalHours,
      price: form.value.price,
      commissionRate: form.value.commissionRate,
      bonusRate: form.value.bonusRate,
      ...(form.value.mode === 'existing'
        ? { studentId: form.value.studentId }
        : { newStudentNickname: form.value.newStudentNickname.trim() }),
    }
    const created = await createTeachingOrder(payload)
    createdCreds.value = created.newStudentCredentials ?? null
    await load()
    // 有新学员凭证时保留弹窗展示凭证，否则直接关闭
    if (!createdCreds.value) showCreate.value = false
  } catch {
    createError.value = '创建失败，请检查输入后重试'
  } finally {
    creating.value = false
  }
}

// ---- copy claim link ----
const copiedLink = ref('')
function copyClaimLink(code: string) {
  const url = `${window.location.origin}/teaching/${code}`
  navigator.clipboard?.writeText(url).then(() => {
    copiedLink.value = code
    setTimeout(() => { if (copiedLink.value === code) copiedLink.value = '' }, 1500)
  }).catch(() => {
    listError.value = '复制失败，请重试'
  })
}

function copyCreds() {
  if (!createdCreds.value) return
  const text = `账号：${createdCreds.value.username}  密码：${createdCreds.value.password}`
  navigator.clipboard?.writeText(text).catch(() => { createError.value = '复制失败，请重试' })
}

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

// ---- top-up ----
const showTopUp = ref(false)
const topUpCode = ref('')
const topUpTitle = ref('')
const topUpBase = ref({ totalHours: 0, price: 0 })
const topUpForm = ref({ addHours: 1, addPrice: 0 })
const topUpError = ref('')

function openTopUp(o: TeachingOrderView) {
  topUpCode.value = o.code
  topUpTitle.value = o.title
  topUpBase.value = { totalHours: o.totalHours, price: o.price }
  topUpForm.value = { addHours: 1, addPrice: 0 }
  topUpError.value = ''
  showTopUp.value = true
}

function submitTopUp() {
  topUpError.value = ''
  if (!(topUpForm.value.addHours > 0)) {
    topUpError.value = '加课时需大于 0'
    return
  }
  const code = topUpCode.value
  const { addHours, addPrice } = topUpForm.value
  const newTotal = topUpBase.value.totalHours + addHours
  const newPrice = topUpBase.value.price + addPrice
  showTopUp.value = false
  confirmTitle.value = '确认续时长'
  confirmMessage.value = `将为该课程增加 ${addHours}h（+${formatMoney(addPrice)} 价格）。续后合计 ${newTotal}h / ${formatMoney(newPrice)}。`
  confirmAction.value = async () => {
    await topUpTeachingOrder(code, addHours, addPrice)
    await load()
  }
  confirmOpen.value = true
}

// ---- scorecards modal ----
const showScorecards = ref(false)
const scorecardTitle = ref('')
const scorecards = ref<ScorecardView[]>([])
const scorecardError = ref('')
const loadingScorecards = ref(false)

async function openScorecards(o: TeachingOrderView) {
  scorecardTitle.value = o.title
  scorecards.value = []
  scorecardError.value = ''
  loadingScorecards.value = true
  showScorecards.value = true
  try {
    await syllabus.ensureLoaded()
    scorecards.value = await listScorecards(o.code)
  } catch {
    scorecardError.value = '加载成绩单失败，请重试'
  } finally {
    loadingScorecards.value = false
  }
}

function scoreGroups(card: ScorecardView) {
  return groupScoresBySyllabus(card.scores, syllabus.categories)
}

// ---- time logs modal ----
const showTimeLogs = ref(false)
const timeLogTitle = ref('')
const timeLogs = ref<TimeLogView[]>([])
const timeLogError = ref('')
const loadingTimeLogs = ref(false)

async function openTimeLogs(o: TeachingOrderView) {
  timeLogTitle.value = o.title
  timeLogs.value = []
  timeLogError.value = ''
  loadingTimeLogs.value = true
  showTimeLogs.value = true
  try {
    timeLogs.value = await listTimeLogs(o.code)
  } catch {
    timeLogError.value = '加载时长台账失败，请重试'
  } finally {
    loadingTimeLogs.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">教学管理</h2>
        <p class="font-body text-sm text-hls-text-light">管理教学课程、课时银行与成绩单，并快速续时长、核对时长台账。</p>
      </div>
      <button class="hls-btn w-full py-2.5 sm:w-auto sm:px-5 whitespace-nowrap" @click="openCreate">▶ 创建教学订单</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading" class="grid gap-3 sm:gap-4 sm:grid-cols-2 xl:grid-cols-6">
      <div class="hls-panel px-4 py-4 sm:col-span-2 xl:col-span-2">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">总课程</p>
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
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">教学中</p>
        <p class="mt-3 font-body text-2xl font-bold text-amber-700">{{ summary.ongoing }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">已结单</p>
        <p class="mt-3 font-body text-2xl font-bold text-emerald-700">{{ summary.settled }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">剩余课时</p>
        <p class="mt-3 font-body text-2xl font-bold text-hls-text-dark">{{ formatMoney(summary.remainingHours) }}h</p>
      </div>
    </div>

    <div v-if="!loading && rows.length" class="space-y-4">
      <article v-for="o in rows" :key="o.code" class="hls-panel overflow-hidden">
        <div class="flex flex-col gap-4 border-b border-hls-border/20 bg-hls-panel-deep/45 px-4 py-4 sm:px-5 lg:flex-row lg:items-start lg:justify-between">
          <div class="space-y-3 min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">教学单</span>
              <span class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs" :class="getTeachingStatusClass(o.status as TeachingStatus)">
                {{ teachingStatusLabels[o.status as TeachingStatus] }}
              </span>
              <span
                class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs"
                :class="o.bonusQualified ? 'border-amber-300 bg-amber-100 text-amber-900' : 'border-stone-300 bg-stone-100 text-stone-800'"
              >
                {{ o.bonusQualified ? '🍗 奉金达标' : '奉金未达标' }}
              </span>
            </div>
            <div>
              <h3 class="font-body text-lg sm:text-xl font-bold text-hls-text-dark break-all">{{ o.title }}</h3>
              <p class="mt-1 font-body text-xs leading-5 text-hls-text-light break-all">
                课程编号：{{ o.code }}
                <span class="hidden sm:inline"> · </span>
                <span class="block sm:inline">学员：{{ o.student.name }}</span>
                <span class="hidden sm:inline"> · </span>
                <span class="block sm:inline">创建时间：{{ formatDate(o.createdAt) }}</span>
              </p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3 lg:min-w-[24rem]">
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">课程金额</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(o.price) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">应发收益</p>
              <p class="mt-2 font-body text-lg font-bold text-emerald-700">{{ formatMoney(o.earnings) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">授课老师</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark break-all">{{ o.teacher?.name ?? '未接单' }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">派单人</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark break-all">{{ o.dispatcherName }}</p>
            </div>
          </div>
        </div>

        <div class="grid gap-4 px-4 py-4 sm:px-5 sm:py-5 lg:grid-cols-[1.2fr_1fr]">
          <section class="space-y-4">
            <!-- 课时银行 -->
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="font-body text-sm font-semibold text-hls-text-dark">课时银行</p>
                <p class="font-body text-xs text-hls-text-light">
                  剩余 <span class="font-bold text-hls-text-dark">{{ formatMoney(o.remainingHours) }}h</span> / 共 {{ formatMoney(o.totalHours) }}h
                </p>
              </div>
              <div class="mt-3 h-3 w-full overflow-hidden rounded-full border border-hls-border/30 bg-black/10">
                <div
                  class="h-full rounded-full bg-gradient-to-r from-emerald-400 to-emerald-600 transition-all"
                  :style="{ width: hoursPercent(o) + '%' }"
                />
              </div>
              <p class="mt-2 font-body text-xs text-hls-text-light">剩余占比 {{ hoursPercent(o) }}%</p>
            </div>

            <div class="grid gap-3 sm:grid-cols-3">
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">分成比例</p>
                <p class="mt-2 font-body text-base text-hls-text-dark">{{ formatMoney(o.commissionRate) }}%</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">奉金比例</p>
                <p class="mt-2 font-body text-base text-hls-text-dark">{{ formatMoney(o.bonusRate) }}%</p>
              </div>
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <p class="font-body text-xs text-hls-text-dark/75">奉金状态</p>
                <p class="mt-2 font-body text-base" :class="o.bonusQualified ? 'text-amber-700' : 'text-hls-text-light'">
                  {{ o.bonusQualified ? '已达标' : '未达标' }}
                </p>
              </div>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <p class="font-body text-sm font-semibold text-hls-text-dark">教学成绩单</p>
                  <p class="mt-1 font-body text-xs text-hls-text-light">按教学大纲分类查看每节课评分</p>
                </div>
                <button class="hls-btn-secondary w-full text-xs px-3 py-2 sm:w-auto sm:py-1.5" @click="openScorecards(o)">查看成绩单</button>
              </div>
            </div>
          </section>

          <section class="space-y-4">
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">教学操作</p>
              <div class="mt-4 space-y-3">
                <button
                  v-if="isOwner"
                  class="hls-btn-secondary w-full py-2.5 disabled:opacity-40"
                  :disabled="o.status === 'settled'"
                  @click="openTopUp(o)"
                >
                  续时长
                </button>
                <button v-if="isOwner" class="hls-btn-secondary w-full py-2.5" @click="openTimeLogs(o)">时长台账</button>
                <button class="hls-btn-secondary w-full py-2.5" @click="copyClaimLink(o.code)">
                  {{ copiedLink === o.code ? '已复制接单链接' : '复制接单链接' }}
                </button>
                <p v-if="o.status === 'dispatching'" class="font-body text-xs text-hls-text-light">尚未接单，将链接发给老师，打开即可接单。</p>
              </div>
            </div>
          </section>
        </div>
      </article>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <div class="mx-auto max-w-sm space-y-2">
        <p class="font-body text-lg font-bold text-hls-text-dark">暂无教学订单</p>
        <p class="font-body text-sm text-hls-text-light">当前还没有教学课程，创建后会在这里展示课时进度与成绩单。</p>
      </div>
    </div>

    <!-- Create Modal -->
    <transition name="fade">
      <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showCreate = false">
        <div class="hls-panel w-full max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">创建教学订单</div>
          <div class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <p v-if="createError" class="font-body text-xs text-red-400 text-center">{{ createError }}</p>

            <div class="grid gap-4 md:grid-cols-2">
              <label class="block md:col-span-2">
                <span class="font-body text-xs text-hls-text-dark/75">课程标题</span>
                <input v-model="form.title" placeholder="例如：三角洲行动 上分特训" class="hls-input mt-2 w-full" />
              </label>

              <div class="md:col-span-2">
                <span class="font-body text-xs text-hls-text-dark/75">学员</span>
                <div class="mt-2 flex gap-4 font-body text-sm text-hls-text-dark">
                  <label class="flex items-center gap-2"><input type="radio" value="existing" v-model="form.mode" /> 选择已有学员</label>
                  <label class="flex items-center gap-2"><input type="radio" value="new" v-model="form.mode" /> 新建学员</label>
                </div>
                <select v-if="form.mode === 'existing'" v-model.number="form.studentId" class="hls-input mt-2 w-full">
                  <option :value="0" disabled>请选择学员</option>
                  <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.nickname }}（{{ c.username }}）</option>
                </select>
                <input v-else v-model="form.newStudentNickname" placeholder="新学员昵称（将自动开通账号）" class="hls-input mt-2 w-full" />
              </div>

              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">总课时（小时）</span>
                <input v-model.number="form.totalHours" type="number" step="0.5" class="hls-input mt-2 w-full" />
              </label>
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">课程价格</span>
                <input v-model.number="form.price" type="number" placeholder="请输入价格" class="hls-input mt-2 w-full" />
              </label>
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">分成比例</span>
                <input v-model.number="form.commissionRate" type="number" placeholder="例如 20" class="hls-input mt-2 w-full" />
              </label>
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">奉金比例</span>
                <input v-model.number="form.bonusRate" type="number" placeholder="例如 3" class="hls-input mt-2 w-full" />
              </label>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">预计应发收益（含奉金）</p>
              <p class="mt-2 font-body text-xl font-bold text-emerald-700">{{ formatMoney(previewEarnings) }}</p>
            </div>

            <!-- 学员开通凭证交接 -->
            <div v-if="createdCreds" class="rounded-lg border-2 border-amber-300 bg-amber-50 px-4 py-4">
              <p class="font-body text-sm font-semibold text-amber-900">🔑 新学员登录凭证（仅此一次，请转交学员）</p>
              <div class="mt-3 grid gap-2 sm:grid-cols-2">
                <div class="rounded border border-amber-200 bg-white/70 px-3 py-2">
                  <p class="font-body text-xs text-hls-text-dark/60">账号</p>
                  <p class="font-body text-sm font-bold text-hls-text-dark break-all">{{ createdCreds.username }}</p>
                </div>
                <div class="rounded border border-amber-200 bg-white/70 px-3 py-2">
                  <p class="font-body text-xs text-hls-text-dark/60">密码</p>
                  <p class="font-body text-sm font-bold text-hls-text-dark break-all">{{ createdCreds.password }}</p>
                </div>
              </div>
              <button class="hls-btn-secondary mt-3 w-full py-2 text-xs" @click="copyCreds">复制账号密码</button>
            </div>

            <div class="flex flex-col-reverse gap-3 pt-1 sm:flex-row">
              <button class="hls-btn-secondary w-full py-2.5 sm:flex-1" @click="showCreate = false">
                {{ createdCreds ? '完成' : '取消' }}
              </button>
              <button v-if="!createdCreds" class="hls-btn w-full py-2.5 disabled:opacity-40 sm:flex-1" :disabled="creating" @click="submitCreate">
                {{ creating ? '创建中…' : '▶ 创建' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- Top-up Modal -->
    <transition name="fade">
      <div v-if="showTopUp" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showTopUp = false">
        <div class="hls-panel w-full max-w-md max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">续时长</div>
          <div class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <p class="font-body text-xs text-hls-text-light break-all">{{ topUpTitle }}</p>
            <p v-if="topUpError" class="font-body text-xs text-red-400 text-center">{{ topUpError }}</p>
            <div class="grid gap-4 grid-cols-2">
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">增加课时（小时）</span>
                <input v-model.number="topUpForm.addHours" type="number" step="0.5" class="hls-input mt-2 w-full" />
              </label>
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">增加价格</span>
                <input v-model.number="topUpForm.addPrice" type="number" class="hls-input mt-2 w-full" />
              </label>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">续后合计</p>
              <p class="mt-2 font-body text-base font-bold text-hls-text-dark">
                {{ formatMoney(topUpBase.totalHours + (topUpForm.addHours || 0)) }}h ·
                {{ formatMoney(topUpBase.price + (topUpForm.addPrice || 0)) }}
              </p>
            </div>
            <div class="flex flex-col-reverse gap-3 pt-1 sm:flex-row">
              <button class="hls-btn-secondary w-full py-2.5 sm:flex-1" @click="showTopUp = false">取消</button>
              <button class="hls-btn w-full py-2.5 sm:flex-1" @click="submitTopUp">确认续时</button>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- Scorecards Modal -->
    <transition name="fade">
      <div v-if="showScorecards" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showScorecards = false">
        <div class="hls-panel w-full max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">教学成绩单</div>
          <div class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <p class="font-body text-xs text-hls-text-light break-all">{{ scorecardTitle }}</p>
            <p v-if="loadingScorecards" class="font-body text-sm text-hls-text-light text-center">加载中…</p>
            <p v-else-if="scorecardError" class="font-body text-sm text-red-400 text-center">{{ scorecardError }}</p>
            <p v-else-if="!scorecards.length" class="font-body text-sm text-hls-text-light text-center">暂无成绩单</p>

            <div v-for="card in scorecards" :key="card.id" class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4 space-y-3">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="font-body text-sm font-semibold text-hls-text-dark">{{ card.teacherName }}</p>
                <p class="font-body text-xs text-hls-text-light">课时 {{ formatMoney(card.durationHours) }}h · {{ formatDate(card.createdAt) }}</p>
              </div>
              <div v-for="group in scoreGroups(card)" :key="group.categoryKey">
                <p class="font-body text-xs font-semibold text-hls-text-dark/75">{{ group.categoryLabel }}</p>
                <div class="mt-2 flex flex-wrap gap-2">
                  <span
                    v-for="item in group.items"
                    :key="item.key"
                    class="inline-flex items-center gap-1 rounded-full border px-3 py-1 font-body text-xs"
                    :class="getScoreClass(item.score)"
                  >
                    {{ item.label }} <span class="font-bold">{{ item.score }}</span>
                  </span>
                </div>
              </div>
              <p v-if="card.comment" class="font-body text-sm leading-6 text-hls-text-light whitespace-pre-wrap break-words">{{ card.comment }}</p>
            </div>

            <button class="hls-btn-secondary py-2 w-full" @click="showScorecards = false">关闭</button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Time Logs Modal -->
    <transition name="fade">
      <div v-if="showTimeLogs" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showTimeLogs = false">
        <div class="hls-panel w-full max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">时长台账</div>
          <div class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <p class="font-body text-xs text-hls-text-light break-all">{{ timeLogTitle }}</p>
            <p v-if="loadingTimeLogs" class="font-body text-sm text-hls-text-light text-center">加载中…</p>
            <p v-else-if="timeLogError" class="font-body text-sm text-red-400 text-center">{{ timeLogError }}</p>
            <p v-else-if="!timeLogs.length" class="font-body text-sm text-hls-text-light text-center">暂无台账记录</p>

            <div v-else class="space-y-2">
              <div v-for="(l, i) in timeLogs" :key="i" class="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-2.5 py-0.5 font-body text-xs text-hls-text-dark">
                      {{ timeLogReasonLabels[l.reason] }}
                    </span>
                    <span class="font-body text-sm font-bold" :class="l.deltaHours >= 0 ? 'text-emerald-700' : 'text-rose-700'">
                      {{ l.deltaHours >= 0 ? '+' : '' }}{{ formatMoney(l.deltaHours) }}h
                    </span>
                    <span v-if="l.priceDelta" class="font-body text-xs text-hls-text-light">价格 {{ l.priceDelta > 0 ? '+' : '' }}{{ formatMoney(l.priceDelta) }}</span>
                  </div>
                  <p v-if="l.note" class="mt-1 font-body text-xs text-hls-text-light break-all">{{ l.note }}</p>
                </div>
                <p class="font-body text-xs text-hls-text-light shrink-0">{{ l.operatorName }}（{{ l.operatorRole }}） · {{ formatDate(l.createdAt) }}</p>
              </div>
            </div>

            <button class="hls-btn-secondary py-2 w-full" @click="showTimeLogs = false">关闭</button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Confirm Dialog -->
    <ConfirmDialog
      :open="confirmOpen"
      :title="confirmTitle"
      :message="confirmMessage"
      confirm-text="确认续时"
      @confirm="handleConfirm"
      @cancel="confirmOpen = false"
    />
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listMyTeaching, claimTeachingOrder, createScorecard, settleTeachingOrder, listScorecards,
  type TeachingOrderView, type ScorecardView,
} from '../../api/teaching'
import {
  teachingStatusLabels, getTeachingStatusClass,
  type TeachingStatus,
} from '../../shared/teachingLabels'
import { formatMoney } from '../../shared/format'
import ConfirmDialog from '../../shared/components/ConfirmDialog.vue'
import TeachingScorecardCard from '../../shared/components/TeachingScorecardCard.vue'

const syllabus = useSyllabusStore()
const mine = ref<TeachingOrderView[]>([])
const loading = ref(false)
const listError = ref('')

const summary = computed(() => ({
  ongoing: mine.value.filter((r) => r.status === 'ongoing').length,
  settled: mine.value.filter((r) => r.status === 'settled').length,
  remainingHours: mine.value.reduce((sum, r) => sum + r.remainingHours, 0),
}))

async function load() {
  loading.value = true
  listError.value = ''
  try {
    await syllabus.ensureLoaded()
    mine.value = await listMyTeaching()
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

// ---- claim by code ----
const showClaim = ref(false)
const claimCode = ref('')
const claimError = ref('')
const claiming = ref(false)

function openClaim() {
  claimCode.value = ''
  claimError.value = ''
  showClaim.value = true
}

async function submitClaim() {
  claimError.value = ''
  if (!claimCode.value.trim()) {
    claimError.value = '请输入教学单编号'
    return
  }
  claiming.value = true
  try {
    await claimTeachingOrder(claimCode.value.trim())
    showClaim.value = false
    await load()
  } catch {
    claimError.value = '接单失败：已被接单或编号不存在'
  } finally {
    claiming.value = false
  }
}

// ---- scorecard modal (author + history) ----
const showScorecard = ref(false)
const active = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])
const cardError = ref('')
const saving = ref(false)
const form = ref<{ durationHours: number; scores: Record<string, number>; comment: string }>({ durationHours: 1, scores: {}, comment: '' })

async function openScorecard(o: TeachingOrderView) {
  active.value = o
  form.value = { durationHours: 1, scores: {}, comment: '' }
  cardError.value = ''
  cards.value = []
  showScorecard.value = true
  try {
    await syllabus.ensureLoaded()
    cards.value = await listScorecards(o.code)
  } catch {
    cardError.value = '加载历史成绩单失败'
  }
}

async function submitCard() {
  if (!active.value) return
  cardError.value = ''
  if (!(form.value.durationHours > 0)) {
    cardError.value = '本节课时长需大于 0'
    return
  }
  saving.value = true
  try {
    const scores = Object.fromEntries(
      Object.entries(form.value.scores).filter(([, v]) => typeof v === 'number' && !Number.isNaN(v)),
    ) as Record<string, number>
    await createScorecard(active.value.code, form.value.durationHours, scores, form.value.comment.trim())
    await load()
    const fresh = mine.value.find((r) => r.code === active.value!.code)
    if (fresh) {
      active.value = fresh
      cards.value = await listScorecards(fresh.code)
    }
    form.value = { durationHours: 1, scores: {}, comment: '' }
  } catch {
    cardError.value = '录入失败：剩余课时可能不足'
  } finally {
    saving.value = false
  }
}

// ---- settle (confirm dialog) ----
const confirmOpen = ref(false)
const confirmMessage = ref('')
const confirmAction = ref<(() => Promise<void>) | null>(null)

function askSettle(o: TeachingOrderView) {
  confirmMessage.value = `确认结单并发起工资结算？该课程剩余课时需为 0（当前 ${formatMoney(o.remainingHours)}h）。`
  confirmAction.value = async () => {
    await settleTeachingOrder(o.code)
    await load()
  }
  confirmOpen.value = true
}

async function handleConfirm() {
  if (confirmAction.value) {
    try {
      await confirmAction.value()
    } catch {
      listError.value = '结单失败：剩余课时未清零或状态不符'
      await load()
    }
  }
  confirmOpen.value = false
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">我的教学</h2>
        <p class="font-body text-sm text-hls-text-light">查看你负责的课程、录入每节课成绩单，并在课时用尽后结单发工资。</p>
      </div>
      <div class="flex gap-2">
        <button class="hls-btn-secondary px-4 py-2.5 text-sm whitespace-nowrap" @click="openClaim">按编号接单</button>
        <button class="hls-btn-secondary px-4 py-2.5 text-sm whitespace-nowrap" :disabled="loading" @click="load">{{ loading ? '刷新中…' : '刷新' }}</button>
      </div>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading" class="grid gap-3 sm:gap-4 grid-cols-1 sm:grid-cols-3">
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

    <div v-if="!loading && mine.length" class="space-y-4">
      <article v-for="o in mine" :key="o.code" class="hls-panel overflow-hidden">
        <div class="flex flex-col gap-4 border-b border-hls-border/20 bg-hls-panel-deep/45 px-4 py-4 sm:px-5 lg:flex-row lg:items-start lg:justify-between">
          <div class="space-y-3 min-w-0">
            <div class="flex flex-wrap items-center gap-2">
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
              </p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3 lg:min-w-[20rem]">
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">课程金额</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(o.price) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">应发收益</p>
              <p class="mt-2 font-body text-lg font-bold text-emerald-700">{{ formatMoney(o.earnings) }}</p>
            </div>
          </div>
        </div>

        <div class="px-4 py-4 sm:px-5 sm:py-5 space-y-4">
          <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <p class="font-body text-sm font-semibold text-hls-text-dark">课时银行</p>
              <p class="font-body text-xs text-hls-text-light">
                剩余 <span class="font-bold text-hls-text-dark">{{ formatMoney(o.remainingHours) }}h</span> / 共 {{ formatMoney(o.totalHours) }}h
              </p>
            </div>
            <div class="mt-3 h-3 w-full overflow-hidden rounded-full border border-hls-border/30 bg-black/10">
              <div class="h-full rounded-full bg-gradient-to-r from-emerald-400 to-emerald-600 transition-all" :style="{ width: hoursPercent(o) + '%' }" />
            </div>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row">
            <button class="hls-btn-secondary w-full py-2.5 sm:flex-1" @click="openScorecard(o)">录成绩单 / 查看</button>
            <button
              v-if="o.status === 'ongoing' && o.remainingMinutes === 0"
              class="hls-btn w-full py-2.5 sm:flex-1"
              @click="askSettle(o)"
            >
              ✅ 结单发工资
            </button>
          </div>
        </div>
      </article>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <div class="mx-auto max-w-sm space-y-2">
        <p class="font-body text-lg font-bold text-hls-text-dark">暂无教学单</p>
        <p class="font-body text-sm text-hls-text-light">可用右上角「按编号接单」自领，或打开管理端发来的接单链接。</p>
      </div>
    </div>

    <!-- Claim Modal -->
    <transition name="fade">
      <div v-if="showClaim" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showClaim = false">
        <div class="hls-panel w-full max-w-md">
          <div class="hls-panel-title text-center">按编号接单</div>
          <div class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <p v-if="claimError" class="font-body text-xs text-red-400 text-center">{{ claimError }}</p>
            <label class="block">
              <span class="font-body text-xs text-hls-text-dark/75">教学单编号</span>
              <input v-model="claimCode" placeholder="输入课程编号自领为老师" class="hls-input mt-2 w-full" @keyup.enter="submitClaim" />
            </label>
            <div class="flex flex-col-reverse gap-3 pt-1 sm:flex-row">
              <button class="hls-btn-secondary w-full py-2.5 sm:flex-1" @click="showClaim = false">取消</button>
              <button class="hls-btn w-full py-2.5 disabled:opacity-40 sm:flex-1" :disabled="claiming" @click="submitClaim">
                {{ claiming ? '接单中…' : '▶ 接单' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- Scorecard Modal -->
    <transition name="fade">
      <div v-if="showScorecard" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showScorecard = false">
        <div class="hls-panel w-full max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">录成绩单</div>
          <div v-if="active" class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="min-w-0">
                  <p class="font-body text-sm font-semibold text-hls-text-dark break-all">{{ active.title }}</p>
                  <p class="mt-1 font-body text-xs text-hls-text-light break-all">学员：{{ active.student.name }} · 编号：{{ active.code }}</p>
                </div>
                <div class="grid grid-cols-2 gap-2 sm:min-w-[13rem]">
                  <div class="rounded-lg border border-hls-border/20 bg-white/75 px-3 py-2 text-center">
                    <p class="font-body text-[11px] uppercase tracking-[0.2em] text-hls-text-dark/70">剩余课时</p>
                    <p class="mt-1 font-body text-lg font-bold text-hls-text-dark">{{ formatMoney(active.remainingHours) }}h</p>
                  </div>
                  <div class="rounded-lg border border-hls-border/20 bg-white/75 px-3 py-2 text-center">
                    <p class="font-body text-[11px] uppercase tracking-[0.2em] text-hls-text-dark/70">历史成绩单</p>
                    <p class="mt-1 font-body text-lg font-bold text-hls-gold-dark">{{ cards.length }}</p>
                  </div>
                </div>
              </div>
            </div>
            <p v-if="cardError" class="font-body text-xs text-red-400 text-center">{{ cardError }}</p>

            <label class="block">
              <span class="font-body text-xs text-hls-text-dark/75">本节课时长（小时）</span>
              <input v-model.number="form.durationHours" type="number" step="0.5" class="hls-input mt-2 w-full" />
            </label>

            <div v-for="cat in syllabus.categories" :key="cat.key" class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-gold-dark">{{ cat.label }}</p>
              <div class="mt-3 grid gap-3 sm:grid-cols-2">
                <div
                  v-for="it in cat.items"
                  :key="it.key"
                  class="rounded-lg border border-hls-border/15 bg-white/75 px-3 py-3"
                >
                  <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                      <p class="font-body text-sm font-semibold text-hls-text-dark">{{ it.label }}</p>
                      <p v-if="it.desc" class="mt-1 font-body text-xs leading-5 text-hls-text-light">{{ it.desc }}</p>
                    </div>
                    <input
                      v-model.number="form.scores[it.key]"
                      type="number"
                      min="0"
                      max="10"
                      placeholder="—"
                      class="hls-input w-20 shrink-0 text-center"
                    />
                  </div>
                </div>
              </div>
            </div>

            <label class="block">
              <span class="font-body text-xs text-hls-text-dark/75">评语（选填）</span>
              <textarea v-model="form.comment" placeholder="本节课表现、待改进点等" class="hls-input mt-2 w-full" rows="3" />
            </label>

            <div class="flex flex-col-reverse gap-3 pt-1 sm:flex-row">
              <button class="hls-btn-secondary w-full py-2.5 sm:flex-1" @click="showScorecard = false">关闭</button>
              <button class="hls-btn w-full py-2.5 disabled:opacity-40 sm:flex-1" :disabled="saving" @click="submitCard">
                {{ saving ? '保存中…' : '保存成绩单' }}
              </button>
            </div>

            <div class="border-t border-hls-border/20 pt-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="font-body text-sm font-semibold text-hls-text-dark">历史成绩单（{{ cards.length }}）</p>
                <p v-if="cards.length" class="font-body text-xs text-hls-text-light">按时间倒序展示，方便连续回看上课表现</p>
              </div>
              <p v-if="!cards.length" class="mt-2 font-body text-sm text-hls-text-light">暂无历史成绩单</p>
              <div class="mt-3 space-y-3">
                <TeachingScorecardCard
                  v-for="card in cards"
                  :key="card.id"
                  :card="card"
                  :categories="syllabus.categories"
                  :show-teacher="false"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- Confirm Dialog -->
    <ConfirmDialog
      :open="confirmOpen"
      title="结单发工资"
      :message="confirmMessage"
      confirm-text="确认结单"
      @confirm="handleConfirm"
      @cancel="confirmOpen = false"
    />
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>

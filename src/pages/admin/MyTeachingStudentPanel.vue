<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listMyTeaching, listScorecards, getTeachingReview, upsertTeachingReview,
  type TeachingOrderView, type ScorecardView, type TeachingReviewView,
} from '../../api/teaching'
import {
  teachingStatusLabels, getTeachingStatusClass,
  teachingBonusQualifies, type TeachingStatus,
} from '../../shared/teachingLabels'
import { formatMoney } from '../../shared/format'
import TeachingScorecardCard from '../../shared/components/TeachingScorecardCard.vue'

const syllabus = useSyllabusStore()
const rows = ref<TeachingOrderView[]>([])
const loading = ref(false)
const listError = ref('')

const summary = computed(() => ({
  total: rows.value.length,
  ongoing: rows.value.filter((r) => r.status === 'ongoing').length,
  remainingHours: rows.value.reduce((sum, r) => sum + r.remainingHours, 0),
}))

async function load() {
  loading.value = true
  listError.value = ''
  try {
    await syllabus.ensureLoaded()
    rows.value = await listMyTeaching()
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

// ---- detail modal ----
const showDetail = ref(false)
const active = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])
const review = ref<TeachingReviewView | null>(null)
const detailError = ref('')
const loadingDetail = ref(false)

const reviewForm = ref({ attitudeScore: 5, technicalScore: 5, comment: '' })
const reviewMsg = ref('')
const reviewError = ref('')
const savingReview = ref(false)

const reviewTotal = computed(() => reviewForm.value.attitudeScore + reviewForm.value.technicalScore)
const reviewQualified = computed(() => teachingBonusQualifies(reviewForm.value.attitudeScore, reviewForm.value.technicalScore))

async function openDetail(o: TeachingOrderView) {
  active.value = o
  cards.value = []
  review.value = null
  reviewForm.value = { attitudeScore: 5, technicalScore: 5, comment: '' }
  reviewMsg.value = ''
  reviewError.value = ''
  detailError.value = ''
  loadingDetail.value = true
  showDetail.value = true
  try {
    await syllabus.ensureLoaded()
    cards.value = await listScorecards(o.code)
    review.value = await getTeachingReview(o.code)
    if (review.value) {
      reviewForm.value = {
        attitudeScore: review.value.attitudeScore,
        technicalScore: review.value.technicalScore,
        comment: review.value.comment,
      }
    }
  } catch {
    detailError.value = '加载课程详情失败'
  } finally {
    loadingDetail.value = false
  }
}

async function submitReview() {
  if (!active.value) return
  reviewMsg.value = ''
  reviewError.value = ''
  savingReview.value = true
  try {
    review.value = await upsertTeachingReview(
      active.value.code,
      reviewForm.value.attitudeScore,
      reviewForm.value.technicalScore,
      reviewForm.value.comment.trim(),
    )
    reviewMsg.value = '评价已提交，感谢你的反馈！'
  } catch {
    reviewError.value = '评价提交失败，请重试'
  } finally {
    savingReview.value = false
  }
}

</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">我的教学</h2>
        <p class="font-body text-sm text-hls-text-light">查看你的课程进度与每节课成绩单，并为老师的教学表现打分。</p>
      </div>
      <button class="hls-btn-secondary px-4 py-2.5 text-sm whitespace-nowrap" :disabled="loading" @click="load">{{ loading ? '刷新中…' : '刷新' }}</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading" class="grid gap-3 sm:gap-4 grid-cols-1 sm:grid-cols-3">
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">课程总数</p>
        <p class="mt-3 font-body text-2xl font-bold text-hls-gold-dark">{{ summary.total }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">进行中</p>
        <p class="mt-3 font-body text-2xl font-bold text-amber-700">{{ summary.ongoing }}</p>
      </div>
      <div class="hls-panel px-4 py-4">
        <p class="font-body text-xs uppercase tracking-[0.3em] text-hls-text-dark/75">剩余课时</p>
        <p class="mt-3 font-body text-2xl font-bold text-hls-text-dark">{{ formatMoney(summary.remainingHours) }}h</p>
      </div>
    </div>

    <div v-if="!loading && rows.length" class="space-y-4">
      <article v-for="o in rows" :key="o.code" class="hls-panel overflow-hidden cursor-pointer" @click="openDetail(o)">
        <div class="flex flex-col gap-4 border-b border-hls-border/20 bg-hls-panel-deep/45 px-4 py-4 sm:px-5 lg:flex-row lg:items-start lg:justify-between">
          <div class="space-y-3 min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs" :class="getTeachingStatusClass(o.status as TeachingStatus)">
                {{ teachingStatusLabels[o.status as TeachingStatus] }}
              </span>
            </div>
            <div>
              <h3 class="font-body text-lg sm:text-xl font-bold text-hls-text-dark break-all">{{ o.title }}</h3>
              <p class="mt-1 font-body text-xs leading-5 text-hls-text-light break-all">授课老师：{{ o.teacher?.name ?? '未接单' }}</p>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-3 lg:min-w-[20rem]">
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">课程金额</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-gold-dark">{{ formatMoney(o.price) }}</p>
            </div>
            <div class="rounded-lg border border-hls-border/20 bg-black/10 px-3 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">剩余课时</p>
              <p class="mt-2 font-body text-lg font-bold text-hls-text-dark">{{ formatMoney(o.remainingHours) }}h</p>
            </div>
          </div>
        </div>

        <div class="px-4 py-4 sm:px-5 sm:py-5">
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
            <p class="mt-2 font-body text-xs text-hls-text-light">点击卡片查看成绩单与评价老师</p>
          </div>
        </div>
      </article>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <div class="mx-auto max-w-sm space-y-2">
        <p class="font-body text-lg font-bold text-hls-text-dark">暂无课程</p>
        <p class="font-body text-sm text-hls-text-light">当你被开通教学课程后，会在这里看到课时进度与成绩单。</p>
      </div>
    </div>

    <!-- Detail Modal -->
    <transition name="fade">
      <div v-if="showDetail" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-3 sm:p-4" @click.self="showDetail = false">
        <div class="hls-panel w-full max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">课程详情</div>
          <div v-if="active" class="space-y-5 p-4 sm:p-6 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <p class="font-body text-sm font-semibold text-hls-text-dark break-all">{{ active.title }}</p>
              <p class="font-body text-xs text-hls-text-light">老师 {{ active.teacher?.name ?? '未接单' }} · 剩余 {{ formatMoney(active.remainingHours) }}h</p>
            </div>
            <p v-if="loadingDetail" class="font-body text-sm text-hls-text-light text-center">加载中…</p>
            <p v-else-if="detailError" class="font-body text-sm text-red-400 text-center">{{ detailError }}</p>

            <template v-else>
              <!-- scorecards -->
              <div>
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="font-body text-sm font-semibold text-hls-text-dark">成绩单（{{ cards.length }}）</p>
                  <p v-if="cards.length" class="font-body text-xs text-hls-text-light">
                    已上课 <span class="font-semibold text-hls-text-dark">{{ cards.length }}</span> 次
                  </p>
                </div>
                <p v-if="!cards.length" class="mt-2 font-body text-sm text-hls-text-light">老师还没有录入成绩单</p>
                <div class="mt-3 space-y-3">
                  <TeachingScorecardCard
                    v-for="card in cards"
                    :key="card.id"
                    :card="card"
                    :categories="syllabus.categories"
                  />
                </div>
              </div>

              <!-- review -->
              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4 space-y-4">
                <p class="font-body text-sm font-semibold text-hls-text-dark">{{ review ? '修改评价' : '评价老师' }}</p>
                <div class="grid gap-4 sm:grid-cols-2">
                  <label class="block">
                    <span class="font-body text-xs text-hls-text-dark/75">态度评分</span>
                    <select v-model.number="reviewForm.attitudeScore" class="hls-input mt-2 w-full">
                      <option v-for="n in [0,1,2,3,4,5]" :key="n" :value="n">{{ n }} 分</option>
                    </select>
                  </label>
                  <label class="block">
                    <span class="font-body text-xs text-hls-text-dark/75">教学评分</span>
                    <select v-model.number="reviewForm.technicalScore" class="hls-input mt-2 w-full">
                      <option v-for="n in [0,1,2,3,4,5]" :key="n" :value="n">{{ n }} 分</option>
                    </select>
                  </label>
                </div>

                <div class="rounded-lg border border-hls-border/20 bg-black/10 px-4 py-3">
                  <p class="font-body text-xs text-hls-text-dark/75">总分</p>
                  <p class="mt-1 font-body text-lg font-bold" :class="reviewQualified ? 'text-emerald-700' : 'text-hls-text-dark'">
                    {{ reviewTotal }}/10
                    <span v-if="reviewQualified" class="font-body text-xs text-amber-700">· 🍗 老师奉金达标</span>
                  </p>
                </div>

                <label class="block">
                  <span class="font-body text-xs text-hls-text-dark/75">评语（选填）</span>
                  <textarea v-model="reviewForm.comment" placeholder="对老师教学的评价与建议" class="hls-input mt-2 w-full" rows="3" />
                </label>

                <p v-if="reviewError" class="font-body text-xs text-red-400">{{ reviewError }}</p>
                <p v-if="reviewMsg" class="font-body text-xs text-emerald-700">{{ reviewMsg }}</p>

                <button class="hls-btn w-full py-2.5 disabled:opacity-40" :disabled="savingReview" @click="submitReview">
                  {{ savingReview ? '提交中…' : (review ? '更新评价' : '提交评价') }}
                </button>
              </div>
            </template>

            <button class="hls-btn-secondary py-2 w-full" @click="showDetail = false">关闭</button>
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

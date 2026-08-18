<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getPublicOrderReview, upsertPublicOrderReview, type PublicReviewPayload } from '../api/orders'
import {
  orderTypeLabels, orderStatusLabels, complaintLabels, reviewFlagLabels,
  type OrderReview, type OrderView, type OrderStatus, type ComplaintStatus, type ReviewFlagKey,
} from '../shared/orderLabels'

const route = useRoute()
const code = route.params.code as string

const order = ref<OrderView | null>(null)
const existingReview = ref<OrderReview | null>(null)
const loading = ref(true)
const loadError = ref('')
const submitting = ref(false)
const submitError = ref('')
const submitSuccess = ref('')

const emptyReview = (): PublicReviewPayload => ({
  attitudeScore: 5,
  technicalScore: 5,
  blameBoss: false,
  late: false,
  tempUnavailable: false,
  slowReply: false,
  noBackpack: false,
  noChannelCard: false,
  noSupplyCard: false,
  noEmotionalValue: false,
  grabContainer: false,
  comment: '',
})

const form = ref<PublicReviewPayload>(emptyReview())
const reviewFlagOptions = Object.entries(reviewFlagLabels) as [ReviewFlagKey, string][]
const reviewTotal = computed(() => form.value.attitudeScore + form.value.technicalScore)
const starLevels = [1, 2, 3, 4, 5] as const

function fillForm(review: OrderReview | null) {
  if (!review) {
    form.value = emptyReview()
    return
  }

  form.value = {
    attitudeScore: review.attitudeScore,
    technicalScore: review.technicalScore,
    blameBoss: review.blameBoss,
    late: review.late,
    tempUnavailable: review.tempUnavailable,
    slowReply: review.slowReply,
    noBackpack: review.noBackpack,
    noChannelCard: review.noChannelCard,
    noSupplyCard: review.noSupplyCard,
    noEmotionalValue: review.noEmotionalValue,
    grabContainer: review.grabContainer,
    comment: review.comment,
  }
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const data = await getPublicOrderReview(code)
    order.value = data.order
    existingReview.value = data.review
    fillForm(data.review)
  } catch {
    loadError.value = '评价链接无效，或订单不存在。'
  } finally {
    loading.value = false
  }
}

async function submitReview() {
  submitError.value = ''
  submitSuccess.value = ''
  submitting.value = true
  try {
    const saved = await upsertPublicOrderReview(code, {
      ...form.value,
      comment: form.value.comment.trim(),
    })
    existingReview.value = saved
    fillForm(saved)
    submitSuccess.value = '评价已提交'
  } catch {
    submitError.value = '提交失败，请稍后重试'
  } finally {
    submitting.value = false
  }
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

function getStarClass(current: number, level: number) {
  return current >= level ? 'text-amber-400' : 'text-hls-text-light/35'
}

onMounted(load)
</script>

<template>
  <div class="min-h-screen bg-hls-bg px-4 py-8 sm:px-6">
    <div class="mx-auto max-w-3xl space-y-6">
      <div class="space-y-2 text-center">
        <p class="font-body text-sm uppercase tracking-[0.3em] text-hls-gold-dark">HLS Review</p>
        <h1 class="font-body text-3xl font-bold text-hls-gold">反馈</h1>
        <p class="font-body text-sm text-hls-text-light">
          花栗鼠俱乐部致力于提供最优质的陪玩服务，感谢板板每一次认真的评价
        </p>
      </div>

      <div v-if="loading" class="hls-panel px-6 py-10 text-center">
        <p class="font-body text-base text-hls-text-dark">加载中…</p>
        <p class="mt-2 font-body text-sm text-hls-text-light">正在获取订单信息，请稍候。</p>
      </div>

      <div v-else-if="loadError" class="hls-panel px-6 py-10 text-center">
        <p class="font-body text-lg font-bold text-red-500">页面加载失败</p>
        <p class="mt-2 font-body text-sm text-hls-text-light">{{ loadError }}</p>
      </div>

      <div v-else-if="order">
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
            <p class="mt-2 font-body text-sm text-hls-text-light break-all">
              订单编号：{{ order.code }} · 创建时间：{{ formatDate(order.createdAt) }}
            </p>
          </div>

          <div class="p-6 space-y-5">
            <div v-if="submitError" class="rounded-lg border border-rose-300 bg-rose-50 px-4 py-3 font-body text-sm text-rose-800">
              {{ submitError }}
            </div>
            <div v-if="submitSuccess" class="rounded-lg border border-emerald-300 bg-emerald-50 px-4 py-3 font-body text-sm text-emerald-800">
              {{ submitSuccess }}
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">态度评分</span>
                <div class="mt-2 rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                  <div class="flex items-center gap-1">
                    <button
                      v-for="n in starLevels"
                      :key="`attitude-${n}`"
                      type="button"
                      class="text-3xl leading-none transition-transform hover:scale-110"
                      :class="getStarClass(form.attitudeScore, n)"
                      @click="form.attitudeScore = n"
                    >
                      ★
                    </button>
                    <span class="ml-3 font-body text-sm text-hls-text-dark">{{ form.attitudeScore }}/5</span>
                  </div>
                </div>
              </label>
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">技术评分</span>
                <div class="mt-2 rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
                  <div class="flex items-center gap-1">
                    <button
                      v-for="n in starLevels"
                      :key="`technical-${n}`"
                      type="button"
                      class="text-3xl leading-none transition-transform hover:scale-110"
                      :class="getStarClass(form.technicalScore, n)"
                      @click="form.technicalScore = n"
                    >
                      ★
                    </button>
                    <span class="ml-3 font-body text-sm text-hls-text-dark">{{ form.technicalScore }}/5</span>
                  </div>
                </div>
              </label>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">总分</p>
              <p class="mt-1 font-body text-lg font-bold" :class="reviewTotal >= 8 ? 'text-emerald-700' : reviewTotal < 6 ? 'text-rose-700' : 'text-hls-text-dark'">
                {{ reviewTotal }}/10
              </p>
              <p class="mt-1 font-body text-xs text-hls-text-light">
                8 分以上会奖励鸡腿给陪陪，低于 6 分属于客诉，店长会化身雷电法王帮你主持公道
              </p>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">扣分项标记</p>
              <div class="mt-3 grid gap-2 sm:grid-cols-2">
                <label v-for="[key, label] in reviewFlagOptions" :key="key" class="flex items-center gap-2 font-body text-sm text-hls-text-dark">
                  <input v-model="form[key]" type="checkbox" class="h-4 w-4" />
                  {{ label }}
                </label>
              </div>
            </div>

            <label class="block">
              <span class="font-body text-xs text-hls-text-dark/75">评语</span>
              <textarea
                v-model="form.comment"
                maxlength="500"
                rows="4"
                placeholder="补充本次服务体验、配合情况或问题说明"
                class="hls-input mt-2 w-full"
              />
            </label>

            <button class="hls-btn w-full py-3 disabled:opacity-40" :disabled="submitting" @click="submitReview">
              {{ submitting ? '提交中…' : existingReview ? '更新评价' : '提交评价' }}
            </button>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

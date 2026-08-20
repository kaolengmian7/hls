<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listMyTeaching, listScorecards, getTeachingReview, upsertTeachingReview,
  type TeachingOrderView, type ScorecardView, type TeachingReviewView,
} from '../../api/teaching'

const syllabus = useSyllabusStore()
const rows = ref<TeachingOrderView[]>([])
const loading = ref(false)
const selected = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])
const review = ref<TeachingReviewView | null>(null)
const reviewForm = ref({ attitudeScore: 5, technicalScore: 5, comment: '' })

function labelOf(key: string): string {
  for (const c of syllabus.categories) for (const it of c.items) if (it.key === key) return it.label
  return key
}

async function load() {
  loading.value = true
  try { await syllabus.ensureLoaded(); rows.value = await listMyTeaching() }
  finally { loading.value = false }
}
onMounted(load)

async function open(o: TeachingOrderView) {
  selected.value = o
  cards.value = await listScorecards(o.code)
  review.value = await getTeachingReview(o.code)
  if (review.value) reviewForm.value = { attitudeScore: review.value.attitudeScore, technicalScore: review.value.technicalScore, comment: review.value.comment }
}

async function submitReview() {
  if (!selected.value) return
  try {
    review.value = await upsertTeachingReview(selected.value.code, reviewForm.value.attitudeScore, reviewForm.value.technicalScore, reviewForm.value.comment)
    window.alert('评价已提交')
  } catch { window.alert('评价失败') }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">我的教学</h2>
      <button class="hls-btn-secondary px-3 py-2 text-sm" :disabled="loading" @click="load">刷新</button>
    </div>

    <article v-for="o in rows" :key="o.code" class="hls-panel p-4 space-y-1 cursor-pointer" @click="open(o)">
      <p class="font-bold text-hls-text-dark">{{ o.title }}</p>
      <p class="text-xs text-hls-text-light">老师 {{ o.teacher?.name ?? '未接单' }} · 剩余 {{ o.remainingHours }}h / 共 {{ o.totalHours }}h</p>
    </article>
    <p v-if="!loading && !rows.length" class="text-sm text-hls-text-light">暂无教学单</p>

    <div v-if="selected" class="hls-panel p-4 space-y-4">
      <div class="flex justify-between">
        <div class="hls-panel-title">{{ selected.title }} · 剩余 {{ selected.remainingHours }}h</div>
        <button class="text-sm text-hls-text-nav" @click="selected = null">关闭 ✕</button>
      </div>

      <div>
        <p class="text-sm font-semibold mb-2">成绩单（{{ cards.length }}）</p>
        <div v-for="c in cards" :key="c.id" class="rounded border border-hls-border/30 p-2 mb-2 text-sm">
          <p class="text-xs text-hls-text-light">{{ c.teacherName }} · {{ c.durationHours }}h · {{ new Date(c.createdAt).toLocaleString('zh-CN') }}</p>
          <p v-if="c.comment" class="text-hls-text-dark">{{ c.comment }}</p>
          <p class="text-xs">{{ Object.entries(c.scores).map(([k, v]) => `${labelOf(k)} ${v}`).join(' · ') }}</p>
        </div>
      </div>

      <div class="rounded border border-hls-border/40 p-3 space-y-2">
        <p class="text-sm font-semibold">评价老师</p>
        <label class="text-xs block">态度分 <input type="number" min="0" max="5" v-model.number="reviewForm.attitudeScore" class="w-16 rounded border px-2 py-1 text-sm" /></label>
        <label class="text-xs block">教学分 <input type="number" min="0" max="5" v-model.number="reviewForm.technicalScore" class="w-16 rounded border px-2 py-1 text-sm" /></label>
        <textarea v-model="reviewForm.comment" placeholder="评语（可选）" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
        <button class="hls-btn-secondary px-4 py-2 text-sm" @click="submitReview">提交评价</button>
      </div>
    </div>
  </div>
</template>

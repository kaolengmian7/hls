<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listMyTeaching, claimTeachingOrder, createScorecard, settleTeachingOrder, listScorecards,
  type TeachingOrderView, type ScorecardView,
} from '../../api/teaching'

const syllabus = useSyllabusStore()
const mine = ref<TeachingOrderView[]>([])
const loading = ref(false)
const selected = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])

const form = ref<{ durationHours: number; scores: Record<string, number>; comment: string }>({ durationHours: 1, scores: {}, comment: '' })
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    await syllabus.ensureLoaded()
    mine.value = await listMyTeaching()
  } catch { /* ignore */ } finally { loading.value = false }
}
onMounted(load)

async function open(o: TeachingOrderView) {
  selected.value = o
  form.value = { durationHours: 1, scores: {}, comment: '' }
  cards.value = await listScorecards(o.code)
}

async function submitCard() {
  if (!selected.value) return
  saving.value = true
  try {
    const scores = Object.fromEntries(
      Object.entries(form.value.scores).filter(([, v]) => typeof v === 'number' && !Number.isNaN(v)),
    ) as Record<string, number>
    await createScorecard(selected.value.code, form.value.durationHours, scores, form.value.comment)
    mine.value = await listMyTeaching()
    const fresh = mine.value.find(r => r.code === selected.value!.code)
    if (fresh) await open(fresh)
  } catch { window.alert('录入失败：课时可能不足') } finally { saving.value = false }
}

async function settle(o: TeachingOrderView) {
  if (!window.confirm('确认结单发工资？需剩余课时为 0。')) return
  try { await settleTeachingOrder(o.code); mine.value = await listMyTeaching(); selected.value = null }
  catch { window.alert('结单失败：剩余课时未清零或状态不符') }
}

async function claimByCode() {
  const code = window.prompt('输入教学单编号自领为老师')
  if (!code) return
  try { await claimTeachingOrder(code.trim()); mine.value = await listMyTeaching() }
  catch { window.alert('自领失败：已被接单或不存在') }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">我的教学</h2>
      <div class="flex gap-2">
        <button class="hls-btn-secondary px-3 py-2 text-sm" @click="claimByCode">按编号接单</button>
        <button class="hls-btn-secondary px-3 py-2 text-sm" :disabled="loading" @click="load">刷新</button>
      </div>
    </div>

    <article v-for="o in mine" :key="o.code" class="hls-panel p-4 space-y-1">
      <div class="flex justify-between">
        <p class="font-bold text-hls-text-dark">{{ o.title }}</p>
        <span class="hls-badge hls-badge-gold">{{ o.status === 'ongoing' ? '教学中' : o.status === 'settled' ? '已结单' : '待接单' }}</span>
      </div>
      <p class="text-xs text-hls-text-light">学员 {{ o.student.name }} · 剩余 {{ o.remainingHours }}h / 共 {{ o.totalHours }}h</p>
      <div class="flex gap-2 pt-1">
        <button class="hls-btn-secondary px-3 py-1.5 text-xs" @click="open(o)">录成绩单 / 查看</button>
        <button v-if="o.status === 'ongoing' && o.remainingMinutes === 0" class="hls-btn-secondary px-3 py-1.5 text-xs" @click="settle(o)">结单发工资</button>
      </div>
    </article>
    <p v-if="!loading && !mine.length" class="text-sm text-hls-text-light">暂无教学单（可用「按编号接单」自领）</p>

    <!-- scorecard editor -->
    <div v-if="selected" class="hls-panel p-4 space-y-4">
      <div class="flex justify-between">
        <div class="hls-panel-title">录成绩单 · {{ selected.title }}（剩余 {{ selected.remainingHours }}h）</div>
        <button class="text-sm text-hls-text-nav" @click="selected = null">关闭 ✕</button>
      </div>
      <label class="text-xs text-hls-text-muted block">本节课时长(小时)
        <input type="number" step="0.5" v-model.number="form.durationHours" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" />
      </label>
      <div v-for="cat in syllabus.categories" :key="cat.key" class="space-y-2">
        <p class="text-sm font-semibold text-hls-gold-dark">{{ cat.label }}</p>
        <div v-for="it in cat.items" :key="it.key" class="flex items-center justify-between gap-2">
          <span class="text-sm" :title="it.desc">{{ it.label }}</span>
          <input type="number" min="0" max="10" placeholder="—" v-model.number="form.scores[it.key]" class="w-16 rounded border border-hls-border/40 px-2 py-1 text-sm" />
        </div>
      </div>
      <textarea v-model="form.comment" placeholder="评语（可选）" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
      <button class="hls-btn-secondary px-4 py-2 text-sm" :disabled="saving" @click="submitCard">{{ saving ? '保存中…' : '保存成绩单' }}</button>

      <div class="pt-2">
        <p class="text-sm font-semibold mb-2">历史成绩单（{{ cards.length }}）</p>
        <div v-for="c in cards" :key="c.id" class="rounded border border-hls-border/30 p-2 mb-2 text-xs">
          <p class="text-hls-text-light">{{ c.durationHours }}h · {{ new Date(c.createdAt).toLocaleString('zh-CN') }}</p>
          <p v-if="c.comment">{{ c.comment }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

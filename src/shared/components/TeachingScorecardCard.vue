<script setup lang="ts">
import { computed } from 'vue'
import type { ScorecardView, SyllabusCategory } from '../../api/teaching'
import { formatDate, formatMoney } from '../format'
import { getScoreClass, groupScoresBySyllabus } from '../teachingLabels'

const props = withDefaults(defineProps<{
  card: ScorecardView
  categories: SyllabusCategory[]
  showTeacher?: boolean
}>(), {
  showTeacher: true,
})

const groups = computed(() => groupScoresBySyllabus(props.card.scores, props.categories))
const scoreValues = computed(() => Object.values(props.card.scores).filter((score) => Number.isFinite(score)))

const scoreCount = computed(() => scoreValues.value.length)
const averageScore = computed(() => {
  if (!scoreCount.value) return 0
  return scoreValues.value.reduce((sum, score) => sum + score, 0) / scoreCount.value
})
const strongCount = computed(() => scoreValues.value.filter((score) => score >= 8).length)
const weakCount = computed(() => scoreValues.value.filter((score) => score < 6).length)

const averageLabel = computed(() => {
  if (!scoreCount.value) return '--'
  const rounded = Math.round(averageScore.value * 10) / 10
  return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(1)
})

const averageToneClass = computed(() => {
  if (averageScore.value >= 8) return 'text-emerald-700'
  if (averageScore.value < 6) return 'text-rose-700'
  return 'text-hls-text-dark'
})

const averageSummary = computed(() => {
  if (!scoreCount.value) return '本节课还没有录入评分项'
  if (averageScore.value >= 8) return '整体表现稳定，优势项比较集中'
  if (averageScore.value < 6) return '有明显待提升项，建议重点复盘'
  return '整体表现中规中矩，还有提升空间'
})
</script>

<template>
  <article class="rounded-xl border border-hls-border/25 bg-gradient-to-br from-white via-hls-bg/80 to-amber-50/70 px-4 py-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.75)] sm:px-5">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <span
            v-if="showTeacher"
            class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-2.5 py-1 font-body font-semibold text-hls-text-dark"
          >
            {{ card.teacherName }}
          </span>
          <span class="inline-flex items-center rounded-full border border-hls-border/20 bg-white/70 px-2.5 py-1 font-body text-hls-text-dark">
            课时 {{ formatMoney(card.durationHours) }}h
          </span>
          <span class="font-body text-hls-text-light">{{ formatDate(card.createdAt) }}</span>
        </div>
        <div class="mt-3 flex flex-wrap items-end gap-x-3 gap-y-1">
          <p class="font-body text-3xl font-bold leading-none" :class="averageToneClass">{{ averageLabel }}</p>
          <p class="pb-0.5 font-body text-sm font-semibold text-hls-text-dark">平均分</p>
          <p class="pb-0.5 font-body text-xs text-hls-text-light">{{ scoreCount }} 项评分</p>
        </div>
        <p class="mt-1 font-body text-sm" :class="averageToneClass">{{ averageSummary }}</p>
      </div>

      <div class="grid grid-cols-3 gap-2 lg:min-w-[15rem]">
        <div class="rounded-lg border border-hls-border/20 bg-white/75 px-3 py-2.5 text-center">
          <p class="font-body text-[11px] uppercase tracking-[0.2em] text-hls-text-dark/70">评分项</p>
          <p class="mt-1 font-body text-lg font-bold text-hls-text-dark">{{ scoreCount }}</p>
        </div>
        <div class="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2.5 text-center">
          <p class="font-body text-[11px] uppercase tracking-[0.2em] text-emerald-800/80">高分项</p>
          <p class="mt-1 font-body text-lg font-bold text-emerald-700">{{ strongCount }}</p>
        </div>
        <div class="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2.5 text-center">
          <p class="font-body text-[11px] uppercase tracking-[0.2em] text-rose-800/80">待提升</p>
          <p class="mt-1 font-body text-lg font-bold text-rose-700">{{ weakCount }}</p>
        </div>
      </div>
    </div>

    <div v-if="groups.length" class="mt-4 grid gap-3 lg:grid-cols-2">
      <section
        v-for="group in groups"
        :key="group.categoryKey"
        class="rounded-lg border border-hls-border/20 bg-white/75 px-3 py-3"
      >
        <p class="font-body text-sm font-semibold text-hls-text-dark">{{ group.categoryLabel }}</p>
        <div class="mt-3 space-y-2">
          <div
            v-for="item in group.items"
            :key="item.key"
            class="flex items-center justify-between gap-3 rounded-lg border border-hls-border/15 bg-black/[0.03] px-3 py-2"
          >
            <span class="font-body text-sm text-hls-text-dark">{{ item.label }}</span>
            <span
              class="inline-flex min-w-[3.75rem] items-center justify-center rounded-full border px-2.5 py-1 font-body text-xs font-bold"
              :class="getScoreClass(item.score)"
            >
              {{ item.score }} 分
            </span>
          </div>
        </div>
      </section>
    </div>

    <section
      v-if="card.comment"
      class="mt-4 rounded-lg border border-hls-border/20 bg-black/[0.04] px-4 py-3"
    >
      <p class="font-body text-xs uppercase tracking-[0.2em] text-hls-text-dark/70">教师评语</p>
      <p class="mt-2 font-body text-sm leading-6 text-hls-text-dark whitespace-pre-wrap break-words">{{ card.comment }}</p>
    </section>
  </article>
</template>

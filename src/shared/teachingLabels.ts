import type { SyllabusCategory, TimeLogView } from '../api/teaching'

export type TeachingStatus = 'dispatching' | 'ongoing' | 'settled'
export type TimeLogReason = TimeLogView['reason']

export const teachingStatusLabels: Record<TeachingStatus, string> = {
  dispatching: '派单中',
  ongoing: '教学中',
  settled: '已结单',
}

export const timeLogReasonLabels: Record<TimeLogReason, string> = {
  initial: '建单初始',
  topup: '续时长',
  lesson: '课程消耗',
}

export function getTeachingStatusClass(status: TeachingStatus) {
  return {
    dispatching: 'bg-sky-100 text-sky-900 border-sky-300',
    ongoing: 'bg-amber-100 text-amber-900 border-amber-300',
    settled: 'bg-emerald-100 text-emerald-900 border-emerald-300',
  }[status]
}

/** Score is on a 0–10 scale; color mirrors the order rating chips. */
export function getScoreClass(score: number) {
  if (score >= 8) return 'bg-emerald-100 text-emerald-900 border-emerald-300'
  if (score < 6) return 'bg-rose-100 text-rose-900 border-rose-300'
  return 'bg-stone-100 text-stone-800 border-stone-300'
}

/**
 * Optimistic per-teacher earnings preview for the create form, mirroring the
 * order panel's preview (assumes the bonus is qualified). Actual settled
 * earnings come from the backend on each order.
 */
export function estimateTeachingEarnings(price: number, commissionRate: number, bonusRate: number) {
  if (!(price > 0)) return 0
  return price * (1 - commissionRate / 100) + (price * bonusRate) / 100
}

export interface GroupedScore {
  categoryKey: string
  categoryLabel: string
  items: { key: string; label: string; score: number }[]
}

/**
 * Groups a scorecard's flat `scores` map under syllabus categories so the UI
 * can render them by section. Keys absent from the syllabus fall under 其他.
 */
export function groupScoresBySyllabus(
  scores: Record<string, number>,
  categories: SyllabusCategory[],
): GroupedScore[] {
  const seen = new Set<string>()
  const groups: GroupedScore[] = []

  for (const category of categories) {
    const items = category.items
      .filter((item) => item.key in scores)
      .map((item) => {
        seen.add(item.key)
        return { key: item.key, label: item.label, score: scores[item.key] }
      })
    if (items.length) {
      groups.push({ categoryKey: category.key, categoryLabel: category.label, items })
    }
  }

  const leftovers = Object.keys(scores)
    .filter((key) => !seen.has(key))
    .map((key) => ({ key, label: key, score: scores[key] }))
  if (leftovers.length) {
    groups.push({ categoryKey: '__other__', categoryLabel: '其他', items: leftovers })
  }

  return groups
}

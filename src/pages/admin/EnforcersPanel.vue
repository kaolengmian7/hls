<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { listEnforcers, updateEnforcerDeposit, updateEnforcerFine, type EnforcerStat } from '../../api/enforcers'

const rows = ref<EnforcerStat[]>([])
const loading = ref(false)
const listError = ref('')

async function load() {
  loading.value = true
  listError.value = ''
  try {
    rows.value = await listEnforcers()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

const totalEnforcers = computed(() => rows.value.length)
const totalDepositOwed = computed(() => rows.value.reduce((sum, row) => sum + row.depositOwed, 0))
const totalPendingFine = computed(() => rows.value.reduce((sum, row) => sum + row.pendingFine, 0))
const totalOrders = computed(() => rows.value.reduce((sum, row) => sum + row.orderCount, 0))
const totalReviewed = computed(() => rows.value.reduce((sum, row) => sum + row.reviewedCount, 0))
const totalGood = computed(() => rows.value.reduce((sum, row) => sum + row.goodCount, 0))
const overallGoodRate = computed(() => {
  if (!totalReviewed.value) return null
  return totalGood.value / totalReviewed.value
})

function money(v: number) {
  return Number.isInteger(v) ? `${v}` : v.toFixed(2)
}
function rate(v: number | null) {
  return v === null ? '—' : `${(v * 100).toFixed(0)}%`
}
function badgeClass(row: EnforcerStat) {
  if (row.pendingFine > 0 && row.depositOwed > 0) return 'hls-badge hls-badge-red'
  if (row.pendingFine > 0) return 'hls-badge hls-badge-amber'
  if (row.depositOwed > 0) return 'hls-badge hls-badge-gold'
  return 'hls-badge hls-badge-teal'
}
function badgeText(row: EnforcerStat) {
  if (row.pendingFine > 0 && row.depositOwed > 0) return '待补押金 + 罚款'
  if (row.pendingFine > 0) return '待处理罚款'
  if (row.depositOwed > 0) return '待补押金'
  return '状态正常'
}
function reviewText(row: EnforcerStat) {
  if (!row.reviewedCount) return '暂无评价'
  return `${row.goodCount} 好评 / ${row.badCount} 差评`
}

async function editDeposit(row: EnforcerStat) {
  const input = window.prompt(`设置「${row.nickname}」的应缴押金（留空=用全局默认300）`, String(row.depositRequired))
  if (input === null) return
  const trimmed = input.trim()
  const value = trimmed === '' ? null : Number(trimmed)
  if (value !== null && (Number.isNaN(value) || value < 0)) {
    window.alert('请输入 ≥0 的数字，或留空清除')
    return
  }
  try {
    await updateEnforcerDeposit(row.userId, value)
    await load()
  } catch {
    window.alert('保存失败')
  }
}

async function editFine(row: EnforcerStat) {
  const input = window.prompt(`设置「${row.nickname}」的待罚款金额`, String(row.pendingFine))
  if (input === null) return
  const value = Number(input.trim())
  if (Number.isNaN(value) || value < 0) {
    window.alert('请输入 ≥0 的数字')
    return
  }
  try {
    await updateEnforcerFine(row.userId, value)
    await load()
  } catch {
    window.alert('保存失败')
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">打手管理</h2>
        <p class="font-body text-sm text-hls-text-light">
          查看押金、评价与罚款状态<span v-if="rows.length"> · 共 {{ rows.length }} 人</span>
        </p>
      </div>
      <button class="hls-btn-secondary w-full px-4 py-2.5 sm:w-auto" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60 animate-pixel-pulse">加载中…</p>

    <template v-if="!loading && rows.length">
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <div class="hls-stat space-y-1">
          <p class="font-body text-xs text-hls-text-muted">打手人数</p>
          <p class="font-body text-2xl font-bold text-hls-text-dark">{{ totalEnforcers }}</p>
        </div>
        <div class="hls-stat space-y-1">
          <p class="font-body text-xs text-hls-text-muted">总接单数</p>
          <p class="font-body text-2xl font-bold text-hls-text-dark">{{ totalOrders }}</p>
        </div>
        <div class="hls-stat space-y-1">
          <p class="font-body text-xs text-hls-text-muted">整体好评率</p>
          <p class="font-body text-2xl font-bold text-emerald-700">{{ rate(overallGoodRate) }}</p>
          <p class="font-body text-xs text-hls-text-light">已评价 {{ totalReviewed }} 单</p>
        </div>
        <div class="hls-stat space-y-1">
          <p class="font-body text-xs text-hls-text-muted">待补押金</p>
          <p class="font-body text-2xl font-bold text-amber-700">{{ money(totalDepositOwed) }}</p>
        </div>
        <div class="hls-stat space-y-1">
          <p class="font-body text-xs text-hls-text-muted">待罚款</p>
          <p class="font-body text-2xl font-bold text-rose-700">{{ money(totalPendingFine) }}</p>
        </div>
      </div>

      <div class="hls-stat">
        <p class="font-body text-sm text-hls-text-dark">
          点击“改押金”可设置个人应缴押金，留空会恢复全局默认值；“录罚款”用于登记当前待扣金额。
        </p>
      </div>

      <div class="space-y-3 lg:hidden">
        <article v-for="row in rows" :key="row.userId" class="hls-panel p-4 space-y-4">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 space-y-1">
              <p class="font-body text-base font-bold text-hls-text-dark break-all">{{ row.nickname }}</p>
              <p class="font-body text-sm text-hls-text-light break-all">@{{ row.username }}</p>
            </div>
            <span :class="badgeClass(row)">{{ badgeText(row) }}</span>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">押金应缴 / 已缴</p>
              <p class="font-body text-sm font-semibold text-hls-text-dark">{{ money(row.depositRequired) }} / {{ money(row.depositPaid) }}</p>
            </div>
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">待补押金 / 待罚款</p>
              <p class="font-body text-sm font-semibold text-hls-text-dark">{{ money(row.depositOwed) }} / {{ money(row.pendingFine) }}</p>
            </div>
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">接单 / 已评价</p>
              <p class="font-body text-sm font-semibold text-hls-text-dark">{{ row.orderCount }} / {{ row.reviewedCount }}</p>
            </div>
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">好评率</p>
              <p class="font-body text-sm font-semibold text-emerald-700">{{ rate(row.goodRate) }}</p>
            </div>
          </div>

          <div class="rounded border border-hls-border/40 bg-hls-bg/35 px-3 py-2">
            <p class="font-body text-xs text-hls-text-muted">评价详情</p>
            <p class="font-body text-sm text-hls-text-dark">{{ reviewText(row) }}</p>
          </div>

          <div class="grid grid-cols-2 gap-2">
            <button class="hls-btn-secondary px-3 py-2 text-xs" @click="editDeposit(row)">改押金</button>
            <button class="hls-btn-secondary px-3 py-2 text-xs" @click="editFine(row)">录罚款</button>
          </div>
        </article>
      </div>

      <div class="hidden lg:block hls-panel overflow-hidden">
        <div class="hls-panel-title">打手列表</div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[68rem] border-collapse font-body text-sm">
            <thead>
              <tr class="border-b-2 border-hls-border/45 text-left text-hls-text-dark">
                <th class="px-4 py-3 font-medium">打手</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">押金</th>
                <th class="px-4 py-3 font-medium">接单</th>
                <th class="px-4 py-3 font-medium">评价</th>
                <th class="px-4 py-3 font-medium">待罚款</th>
                <th class="px-4 py-3 font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in rows"
                :key="row.userId"
                class="border-b border-hls-border/20 align-top transition-colors hover:bg-hls-bg/35"
              >
                <td class="px-4 py-3">
                  <p class="font-semibold text-hls-text-dark">{{ row.nickname }}</p>
                  <p class="mt-1 text-xs text-hls-text-light">@{{ row.username }}</p>
                </td>
                <td class="px-4 py-3">
                  <span :class="badgeClass(row)">{{ badgeText(row) }}</span>
                </td>
                <td class="px-4 py-3">
                  <p class="text-hls-text-dark">应缴 {{ money(row.depositRequired) }}</p>
                  <p class="mt-1 text-emerald-700">已缴 {{ money(row.depositPaid) }}</p>
                  <p class="mt-1 text-amber-700">待缴 {{ money(row.depositOwed) }}</p>
                </td>
                <td class="px-4 py-3">
                  <p class="text-hls-text-dark">{{ row.orderCount }} 单</p>
                  <p class="mt-1 text-xs text-hls-text-light">已评价 {{ row.reviewedCount }} 单</p>
                </td>
                <td class="px-4 py-3">
                  <p class="text-emerald-700">{{ row.goodCount }} 好评</p>
                  <p class="mt-1 text-rose-700">{{ row.badCount }} 差评</p>
                  <p class="mt-1 text-hls-text-dark">好评率 {{ rate(row.goodRate) }}</p>
                </td>
                <td class="px-4 py-3 font-semibold text-rose-700">{{ money(row.pendingFine) }}</td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap gap-2">
                    <button class="hls-btn-secondary px-3 py-1.5 text-xs" @click="editDeposit(row)">改押金</button>
                    <button class="hls-btn-secondary px-3 py-1.5 text-xs" @click="editFine(row)">录罚款</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <p class="font-body text-sm text-hls-text-light">暂无打手</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
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

function money(v: number) {
  return Number.isInteger(v) ? `${v}` : v.toFixed(2)
}
function rate(v: number | null) {
  return v === null ? '—' : `${(v * 100).toFixed(0)}%`
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
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <h2 class="font-body font-bold text-xl text-hls-gold">打手管理</h2>
      <button class="hls-btn-secondary px-4 py-2.5" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading && rows.length" class="overflow-x-auto">
      <table class="w-full min-w-[56rem] border-collapse font-body text-sm">
        <thead>
          <tr class="text-left text-hls-text-dark/75">
            <th class="px-3 py-2">打手</th>
            <th class="px-3 py-2">押金应缴</th>
            <th class="px-3 py-2">已缴押金</th>
            <th class="px-3 py-2">待缴押金</th>
            <th class="px-3 py-2">接单数</th>
            <th class="px-3 py-2">好评</th>
            <th class="px-3 py-2">差评</th>
            <th class="px-3 py-2">好评率</th>
            <th class="px-3 py-2">待罚款</th>
            <th class="px-3 py-2">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.userId" class="border-t border-hls-border/20">
            <td class="px-3 py-2 font-semibold text-hls-text-dark">{{ row.nickname }}</td>
            <td class="px-3 py-2">{{ money(row.depositRequired) }}</td>
            <td class="px-3 py-2 text-emerald-700">{{ money(row.depositPaid) }}</td>
            <td class="px-3 py-2 text-amber-700">{{ money(row.depositOwed) }}</td>
            <td class="px-3 py-2">{{ row.orderCount }}</td>
            <td class="px-3 py-2 text-emerald-700">{{ row.goodCount }}</td>
            <td class="px-3 py-2 text-rose-700">{{ row.badCount }}</td>
            <td class="px-3 py-2">{{ rate(row.goodRate) }}</td>
            <td class="px-3 py-2 text-rose-700">{{ money(row.pendingFine) }}</td>
            <td class="px-3 py-2 space-x-2 whitespace-nowrap">
              <button class="hls-btn-secondary px-2 py-1 text-xs" @click="editDeposit(row)">改押金</button>
              <button class="hls-btn-secondary px-2 py-1 text-xs" @click="editFine(row)">录罚款</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <p class="font-body text-sm text-hls-text-light">暂无打手</p>
    </div>
  </div>
</template>

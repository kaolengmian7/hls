<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listPayroll, payPayroll, type PayrollRow, type PayrollStatus } from '../../api/payroll'

const tab = ref<PayrollStatus>('pending')
const rows = ref<PayrollRow[]>([])
const loading = ref(false)
const listError = ref('')
const payingId = ref<number | null>(null)

async function load() {
  loading.value = true
  listError.value = ''
  try {
    rows.value = await listPayroll(tab.value)
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

function switchTab(t: PayrollStatus) {
  if (tab.value === t) return
  tab.value = t
  load()
}

function money(v: number) {
  return Number.isInteger(v) ? `${v}` : v.toFixed(2)
}
function formatDate(value: string | null) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  }).format(d)
}
function deductionText(row: PayrollRow) {
  const parts = [`应发 ${money(row.gross)}`]
  if (row.depositDeducted > 0) parts.push(`抵缴押金 ${money(row.depositDeducted)}`)
  if (row.fineDeducted > 0) parts.push(`扣罚款 ${money(row.fineDeducted)}`)
  parts.push(`实发 ${money(row.netPaid)}`)
  return parts.join(' · ')
}

async function pay(row: PayrollRow) {
  if (!window.confirm(`确认给「${row.enforcerName}」发放订单 ${row.orderCode} 的工资？\n${deductionText(row)}`)) return
  payingId.value = row.id
  try {
    await payPayroll(row.id)
    await load()
  } catch {
    window.alert('发放失败（可能已发放）')
  } finally {
    payingId.value = null
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <h2 class="font-body font-bold text-xl text-hls-gold">工资管理</h2>
      <button class="hls-btn-secondary px-4 py-2.5" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <div class="flex gap-2">
      <button
        class="px-3 py-1.5 rounded font-body text-sm"
        :class="tab === 'pending' ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('pending')"
      >待发放</button>
      <button
        class="px-3 py-1.5 rounded font-body text-sm"
        :class="tab === 'paid' ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('paid')"
      >已发放</button>
    </div>

    <p v-if="tab === 'pending'" class="font-body text-xs text-hls-text-light">
      提示：待发放的扣减为预测值，按当前欠缴额计算；同一打手多单时以逐单发放为准。
    </p>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading && rows.length" class="overflow-x-auto">
      <table class="w-full min-w-[60rem] border-collapse font-body text-sm">
        <thead>
          <tr class="text-left text-hls-text-dark/75">
            <th class="px-3 py-2">订单 / 打手</th>
            <th class="px-3 py-2">应发</th>
            <th class="px-3 py-2">抵押金</th>
            <th class="px-3 py-2">扣罚款</th>
            <th class="px-3 py-2">实发</th>
            <th class="px-3 py-2">扣减说明</th>
            <th class="px-3 py-2">{{ tab === 'pending' ? '操作' : '发放时间' }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id" class="border-t border-hls-border/20 align-top">
            <td class="px-3 py-2">
              <p class="font-semibold text-hls-text-dark">{{ row.orderTitle }}</p>
              <p class="text-xs text-hls-text-light">{{ row.orderCode }} · {{ row.enforcerName }}</p>
            </td>
            <td class="px-3 py-2 text-hls-gold-dark">{{ money(row.gross) }}</td>
            <td class="px-3 py-2 text-amber-700">{{ money(row.depositDeducted) }}</td>
            <td class="px-3 py-2 text-rose-700">{{ money(row.fineDeducted) }}</td>
            <td class="px-3 py-2 font-bold text-emerald-700">{{ money(row.netPaid) }}</td>
            <td class="px-3 py-2 text-xs text-hls-text-light max-w-[16rem]">{{ deductionText(row) }}</td>
            <td class="px-3 py-2 whitespace-nowrap">
              <button
                v-if="tab === 'pending'"
                class="hls-btn-secondary px-3 py-1 text-xs"
                :disabled="payingId === row.id"
                @click="pay(row)"
              >{{ payingId === row.id ? '发放中…' : '发工资' }}</button>
              <span v-else class="text-xs text-hls-text-light">{{ formatDate(row.paidAt) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <p class="font-body text-sm text-hls-text-light">
        {{ tab === 'pending' ? '暂无待发放工资（订单结单满3天后出现）' : '暂无已发放记录' }}
      </p>
    </div>
  </div>
</template>

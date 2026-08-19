<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
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

const totalRows = computed(() => rows.value.length)
const totalGross = computed(() => rows.value.reduce((sum, row) => sum + row.gross, 0))
const totalDepositDeducted = computed(() => rows.value.reduce((sum, row) => sum + row.depositDeducted, 0))
const totalFineDeducted = computed(() => rows.value.reduce((sum, row) => sum + row.fineDeducted, 0))
const totalNet = computed(() => rows.value.reduce((sum, row) => sum + row.netPaid, 0))

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
function tabLabel(value: PayrollStatus) {
  if (value === 'frozen') return '冻结中'
  return value === 'pending' ? '待发放' : '已发放'
}
function badgeClass(value: PayrollStatus) {
  if (value === 'frozen') return 'hls-badge hls-badge-gray'
  return value === 'pending' ? 'hls-badge hls-badge-gold' : 'hls-badge hls-badge-teal'
}
function netLabel(value: PayrollStatus) {
  return value === 'paid' ? '已实发' : '预计实发'
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
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="space-y-1">
        <h2 class="font-body font-bold text-xl text-hls-gold">工资管理</h2>
        <p class="font-body text-sm text-hls-text-light">
          汇总工资、扣押金与罚款<span v-if="rows.length"> · 当前 {{ tabLabel(tab) }} {{ rows.length }} 条</span>
        </p>
      </div>
      <button class="hls-btn-secondary w-full px-4 py-2.5 sm:w-auto" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <div class="flex gap-2 rounded border border-hls-border/40 bg-white/60 p-1 w-full sm:w-fit">
      <button
        class="flex-1 rounded px-4 py-2 font-body text-sm sm:flex-none"
        :class="tab === 'frozen' ? 'bg-hls-gold/20 text-hls-gold-dark font-semibold shadow-sm' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('frozen')"
      >冻结中</button>
      <button
        class="flex-1 rounded px-4 py-2 font-body text-sm sm:flex-none"
        :class="tab === 'pending' ? 'bg-hls-gold/20 text-hls-gold-dark font-semibold shadow-sm' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('pending')"
      >待发放</button>
      <button
        class="flex-1 rounded px-4 py-2 font-body text-sm sm:flex-none"
        :class="tab === 'paid' ? 'bg-hls-gold/20 text-hls-gold-dark font-semibold shadow-sm' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('paid')"
      >已发放</button>
    </div>

    <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
      <div class="hls-stat space-y-1">
        <p class="font-body text-xs text-hls-text-muted">{{ tabLabel(tab) }}条数</p>
        <p class="font-body text-2xl font-bold text-hls-text-dark">{{ totalRows }}</p>
      </div>
      <div class="hls-stat space-y-1">
        <p class="font-body text-xs text-hls-text-muted">应发合计</p>
        <p class="font-body text-2xl font-bold text-hls-gold-dark">{{ money(totalGross) }}</p>
      </div>
      <div class="hls-stat space-y-1">
        <p class="font-body text-xs text-hls-text-muted">抵押金合计</p>
        <p class="font-body text-2xl font-bold text-amber-700">{{ money(totalDepositDeducted) }}</p>
      </div>
      <div class="hls-stat space-y-1">
        <p class="font-body text-xs text-hls-text-muted">罚款扣减合计</p>
        <p class="font-body text-2xl font-bold text-rose-700">{{ money(totalFineDeducted) }}</p>
      </div>
      <div class="hls-stat space-y-1">
        <p class="font-body text-xs text-hls-text-muted">{{ netLabel(tab) }}合计</p>
        <p class="font-body text-2xl font-bold text-emerald-700">{{ money(totalNet) }}</p>
      </div>
    </div>

    <div v-if="tab === 'pending'" class="hls-stat">
      <p class="font-body text-sm text-hls-text-dark">
        提示：待发放页展示的是按当前欠缴情形计算的预测扣减；同一打手存在多单时，最终金额以逐单发放顺序为准。
      </p>
    </div>
    <div v-else-if="tab === 'frozen'" class="hls-stat">
      <p class="font-body text-sm text-hls-text-dark">
        提示：冻结中的工资仍在结单冷静期，解冻后自动转入「待发放」方可发放；此处金额为按当前欠缴情形的预测值。
      </p>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60 animate-pixel-pulse">加载中…</p>

    <template v-if="!loading && rows.length">
      <div class="space-y-3 lg:hidden">
        <article v-for="row in rows" :key="row.id" class="hls-panel p-4 space-y-4">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 space-y-1">
              <p class="font-body text-base font-bold text-hls-text-dark break-all">{{ row.orderTitle }}</p>
              <p class="font-body text-sm text-hls-text-light break-all">{{ row.orderCode }} · {{ row.enforcerName }}</p>
            </div>
            <span :class="badgeClass(row.status)">{{ tabLabel(row.status) }}</span>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">应发</p>
              <p class="font-body text-sm font-semibold text-hls-gold-dark">{{ money(row.gross) }}</p>
            </div>
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">{{ netLabel(row.status) }}</p>
              <p class="font-body text-sm font-semibold text-emerald-700">{{ money(row.netPaid) }}</p>
            </div>
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">抵押金</p>
              <p class="font-body text-sm font-semibold text-amber-700">{{ money(row.depositDeducted) }}</p>
            </div>
            <div class="hls-stat">
              <p class="font-body text-xs text-hls-text-muted">扣罚款</p>
              <p class="font-body text-sm font-semibold text-rose-700">{{ money(row.fineDeducted) }}</p>
            </div>
          </div>

          <div class="rounded border border-hls-border/40 bg-hls-bg/35 px-3 py-2 space-y-1">
            <p class="font-body text-xs text-hls-text-muted">扣减说明</p>
            <p class="font-body text-sm text-hls-text-dark">{{ deductionText(row) }}</p>
            <p class="font-body text-xs text-hls-text-light">
              {{ row.status === 'frozen' ? `解冻时间 ${formatDate(row.unlockAt)}` : row.status === 'pending' ? `生成时间 ${formatDate(row.createdAt)}` : `发放时间 ${formatDate(row.paidAt)}` }}
            </p>
          </div>

          <button
            v-if="tab === 'pending'"
            class="hls-btn-secondary w-full px-3 py-2 text-sm"
            :disabled="payingId === row.id"
            @click="pay(row)"
          >{{ payingId === row.id ? '发放中…' : '发工资' }}</button>
        </article>
      </div>

      <div class="hidden lg:block hls-panel overflow-hidden">
        <div class="hls-panel-title">{{ tabLabel(tab) }}工资列表</div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[72rem] border-collapse font-body text-sm">
            <thead>
              <tr class="border-b-2 border-hls-border/45 text-left text-hls-text-dark">
                <th class="px-4 py-3 font-medium">订单 / 打手</th>
                <th class="px-4 py-3 font-medium">状态</th>
                <th class="px-4 py-3 font-medium">金额</th>
                <th class="px-4 py-3 font-medium">扣减说明</th>
                <th class="px-4 py-3 font-medium">{{ tab === 'pending' ? '生成 / 操作' : tab === 'frozen' ? '解冻时间' : '发放时间' }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in rows"
                :key="row.id"
                class="border-b border-hls-border/20 align-top transition-colors hover:bg-hls-bg/35"
              >
                <td class="px-4 py-3">
                  <p class="font-semibold text-hls-text-dark">{{ row.orderTitle }}</p>
                  <p class="mt-1 text-xs text-hls-text-light">{{ row.orderCode }} · {{ row.enforcerName }}</p>
                </td>
                <td class="px-4 py-3">
                  <span :class="badgeClass(row.status)">{{ tabLabel(row.status) }}</span>
                </td>
                <td class="px-4 py-3">
                  <p class="text-hls-gold-dark">应发 {{ money(row.gross) }}</p>
                  <p class="mt-1 text-amber-700">抵押金 {{ money(row.depositDeducted) }}</p>
                  <p class="mt-1 text-rose-700">罚款 {{ money(row.fineDeducted) }}</p>
                  <p class="mt-1 font-semibold text-emerald-700">{{ netLabel(row.status) }} {{ money(row.netPaid) }}</p>
                </td>
                <td class="px-4 py-3 max-w-[20rem]">
                  <p class="text-sm text-hls-text-dark">{{ deductionText(row) }}</p>
                </td>
                <td class="px-4 py-3">
                  <div v-if="tab === 'pending'" class="space-y-2">
                    <p class="text-xs text-hls-text-light">生成时间 {{ formatDate(row.createdAt) }}</p>
                    <button
                      class="hls-btn-secondary px-3 py-1.5 text-xs"
                      :disabled="payingId === row.id"
                      @click="pay(row)"
                    >{{ payingId === row.id ? '发放中…' : '发工资' }}</button>
                  </div>
                  <span v-else-if="tab === 'frozen'" class="text-xs text-hls-text-light">{{ formatDate(row.unlockAt) }}</span>
                  <span v-else class="text-xs text-hls-text-light">{{ formatDate(row.paidAt) }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <p class="font-body text-sm text-hls-text-light">
        {{ tab === 'pending' ? '暂无待发放工资（订单结单满3天后出现）' : tab === 'frozen' ? '暂无冻结中的工资' : '暂无已发放记录' }}
      </p>
    </div>
  </div>
</template>

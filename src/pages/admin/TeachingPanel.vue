<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '../../stores/auth'
import {
  listTeachingOrders, createTeachingOrder, topUpTeachingOrder,
  listScorecards, listTimeLogs, listCustomers,
  type TeachingOrderView, type ScorecardView, type TimeLogView, type CustomerOption,
} from '../../api/teaching'

const auth = useAuthStore()
const isOwner = computed(() => auth.user?.role === 'owner')

const rows = ref<TeachingOrderView[]>([])
const customers = ref<CustomerOption[]>([])
const loading = ref(false)
const err = ref('')

// create form
const form = ref({ title: '', studentId: 0, newStudentNickname: '', totalHours: 6, price: 0, commissionRate: 20, bonusRate: 0, mode: 'existing' as 'existing' | 'new' })
const creating = ref(false)
const createdCreds = ref<{ username: string; password: string } | null>(null)

// detail drawer
const selected = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])
const logs = ref<TimeLogView[]>([])
const topUp = ref({ addHours: 1, addPrice: 0 })

async function load() {
  loading.value = true; err.value = ''
  try {
    rows.value = await listTeachingOrders()
    customers.value = await listCustomers()
  } catch { err.value = '加载失败' } finally { loading.value = false }
}
onMounted(load)

async function submitCreate() {
  creating.value = true; createdCreds.value = null
  try {
    const payload = {
      title: form.value.title, totalHours: form.value.totalHours, price: form.value.price,
      commissionRate: form.value.commissionRate, bonusRate: form.value.bonusRate,
      ...(form.value.mode === 'existing' ? { studentId: form.value.studentId } : { newStudentNickname: form.value.newStudentNickname }),
    }
    const created = await createTeachingOrder(payload)
    if (created.newStudentCredentials) createdCreds.value = created.newStudentCredentials
    form.value.title = ''; form.value.newStudentNickname = ''; form.value.price = 0
    await load()
  } catch { window.alert('建单失败，请检查输入') } finally { creating.value = false }
}

async function openDetail(o: TeachingOrderView) {
  selected.value = o
  cards.value = await listScorecards(o.code)
  logs.value = isOwner.value ? await listTimeLogs(o.code) : []
}

async function doTopUp() {
  if (!selected.value) return
  try {
    await topUpTeachingOrder(selected.value.code, topUp.value.addHours, topUp.value.addPrice)
    await load()
    const fresh = rows.value.find(r => r.code === selected.value!.code)
    if (fresh) await openDetail(fresh)
  } catch { window.alert('续时长失败') }
}

function statusLabel(s: string) { return s === 'dispatching' ? '待接单' : s === 'ongoing' ? '教学中' : '已结单' }
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">教学管理</h2>
      <button class="hls-btn-secondary px-4 py-2" :disabled="loading" @click="load">{{ loading ? '刷新中…' : '刷新' }}</button>
    </div>
    <p v-if="err" class="text-sm text-red-400">{{ err }}</p>

    <!-- create form -->
    <div class="hls-panel p-4 space-y-3">
      <div class="hls-panel-title">新建教学单</div>
      <input v-model="form.title" placeholder="标题" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
      <div class="flex gap-2 text-sm">
        <label><input type="radio" value="existing" v-model="form.mode" /> 选已有学员</label>
        <label><input type="radio" value="new" v-model="form.mode" /> 新建学员</label>
      </div>
      <select v-if="form.mode === 'existing'" v-model.number="form.studentId" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm">
        <option :value="0" disabled>选择学员</option>
        <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.nickname }}（{{ c.username }}）</option>
      </select>
      <input v-else v-model="form.newStudentNickname" placeholder="新学员昵称" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <label class="text-xs text-hls-text-muted">总课时(小时)<input type="number" step="0.5" v-model.number="form.totalHours" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
        <label class="text-xs text-hls-text-muted">价格<input type="number" v-model.number="form.price" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
        <label class="text-xs text-hls-text-muted">分成率%<input type="number" v-model.number="form.commissionRate" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
        <label class="text-xs text-hls-text-muted">奉金率%<input type="number" v-model.number="form.bonusRate" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
      </div>
      <button class="hls-btn-secondary px-4 py-2 text-sm" :disabled="creating" @click="submitCreate">{{ creating ? '创建中…' : '创建' }}</button>
      <p v-if="createdCreds" class="text-sm text-emerald-700">新学员登录：账号 {{ createdCreds.username }} / 密码 {{ createdCreds.password }}（请转交学员）</p>
    </div>

    <!-- list -->
    <div class="space-y-2">
      <article v-for="o in rows" :key="o.code" class="hls-panel p-4 space-y-1 cursor-pointer" @click="openDetail(o)">
        <div class="flex justify-between">
          <p class="font-bold text-hls-text-dark">{{ o.title }}</p>
          <span class="hls-badge hls-badge-gold">{{ statusLabel(o.status) }}</span>
        </div>
        <p class="text-xs text-hls-text-light">{{ o.code }} · 学员 {{ o.student.name }} · 老师 {{ o.teacher?.name ?? '未接单' }}</p>
        <p class="text-xs text-hls-text-light">剩余 {{ o.remainingHours }}h / 共 {{ o.totalHours }}h · 应发 {{ o.earnings }}</p>
      </article>
      <p v-if="!loading && !rows.length" class="text-sm text-hls-text-light">暂无教学单</p>
    </div>

    <!-- detail -->
    <div v-if="selected" class="hls-panel p-4 space-y-4">
      <div class="flex justify-between">
        <div class="hls-panel-title">教学单详情 · {{ selected.title }}</div>
        <button class="text-sm text-hls-text-nav" @click="selected = null">关闭 ✕</button>
      </div>
      <div v-if="isOwner" class="rounded border border-hls-border/40 p-3 space-y-2">
        <p class="text-sm font-semibold">续时长</p>
        <div class="flex gap-2 items-end">
          <label class="text-xs">加课时(小时)<input type="number" step="0.5" v-model.number="topUp.addHours" class="w-full rounded border px-2 py-1 text-sm" /></label>
          <label class="text-xs">加价格<input type="number" v-model.number="topUp.addPrice" class="w-full rounded border px-2 py-1 text-sm" /></label>
          <button class="hls-btn-secondary px-3 py-1.5 text-sm" @click="doTopUp">续时</button>
        </div>
      </div>
      <div>
        <p class="text-sm font-semibold mb-2">成绩单（{{ cards.length }}）</p>
        <div v-for="c in cards" :key="c.id" class="rounded border border-hls-border/30 p-2 mb-2 text-sm">
          <p class="text-xs text-hls-text-light">{{ c.teacherName }} · {{ c.durationHours }}h · {{ new Date(c.createdAt).toLocaleString('zh-CN') }}</p>
          <p v-if="c.comment" class="text-hls-text-dark">{{ c.comment }}</p>
          <p class="text-xs">{{ Object.entries(c.scores).map(([k, v]) => `${k}:${v}`).join('  ') }}</p>
        </div>
      </div>
      <div v-if="isOwner && logs.length">
        <p class="text-sm font-semibold mb-2">时长台账</p>
        <p v-for="(l, i) in logs" :key="i" class="text-xs text-hls-text-light">
          {{ new Date(l.createdAt).toLocaleString('zh-CN') }} · {{ l.reason }} · {{ l.deltaHours }}h · {{ l.operatorName }} · {{ l.note }}
        </p>
      </div>
    </div>
  </div>
</template>

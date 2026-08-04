<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getOrder, claimOrder } from '../api/orders'
import { useAuthStore } from '../stores/auth'
import {
  orderTypeLabels, orderStatusLabels, type OrderView,
} from '../shared/orderLabels'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const code = route.params.code as string
const order = ref<OrderView | null>(null)
const loading = ref(true)
const loadError = ref('')
const actionError = ref('')
const claiming = ref(false)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    order.value = await getOrder(code)
  } catch {
    loadError.value = '订单不存在或已失效'
  } finally {
    loading.value = false
  }
}
onMounted(load)

const meId = computed(() => auth.user?.id ?? null)
const alreadyClaimed = computed(() =>
  !!order.value &&
  ((order.value.enforcerA?.id === meId.value) || (order.value.enforcerB?.id === meId.value)),
)
const isFull = computed(() => {
  const o = order.value
  if (!o) return false
  return o.playerCount === 1 ? !!o.enforcerA : !!o.enforcerA && !!o.enforcerB
})
const claimable = computed(() =>
  !!order.value && order.value.status === 'dispatching' && !isFull.value && !alreadyClaimed.value,
)

async function onClaim() {
  if (!auth.token) {
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  actionError.value = ''
  claiming.value = true
  try {
    order.value = await claimOrder(code)
  } catch {
    actionError.value = '扣单失败，可能已被他人接单'
    await load()
  } finally {
    claiming.value = false
  }
}
</script>

<template>
  <div class="min-h-screen px-4 py-8 max-w-md mx-auto">
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60">加载中…</p>
    <p v-else-if="loadError" class="font-body text-sm text-red-400">{{ loadError }}</p>

    <div v-else-if="order" class="hls-panel">
      <div class="hls-panel-title text-center">{{ order.title }}</div>
      <div class="p-6 space-y-3 font-body text-sm">
        <div class="flex justify-between"><span class="text-hls-text-muted">订单编号</span><span>{{ order.code }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">类型</span><span>{{ orderTypeLabels[order.orderType] }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">价格</span><span>{{ order.price }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">抽成</span><span>{{ order.commissionRate }}%</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">鸡腿</span><span>{{ order.bonusRate }}%</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">打手人数</span><span>{{ order.playerCount }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">派单人</span><span>{{ order.dispatcherName }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">备注</span><span>{{ order.remark || '—' }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">状态</span><span>{{ orderStatusLabels[order.status] }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">打手A</span><span>{{ order.enforcerA?.name || '空缺' }}</span></div>
        <div v-if="order.playerCount === 2" class="flex justify-between"><span class="text-hls-text-muted">打手B</span><span>{{ order.enforcerB?.name || '空缺' }}</span></div>

        <div class="hls-stat text-center py-3">
          <p class="text-hls-text-muted text-xs">预估收益（每人）</p>
          <p class="text-hls-gold-dark font-bold text-lg">{{ order.earnings }}</p>
        </div>

        <p v-if="actionError" class="text-xs text-red-400 text-center">{{ actionError }}</p>
        <p v-if="alreadyClaimed" class="text-xs text-hls-gold-dark text-center">你已接此单</p>

        <button
          class="hls-btn w-full py-2 disabled:opacity-40"
          :disabled="claiming || !claimable"
          @click="onClaim"
        >
          <template v-if="!auth.token">登录并扣单</template>
          <template v-else-if="alreadyClaimed">已接单</template>
          <template v-else-if="isFull">已招满</template>
          <template v-else-if="order.status !== 'dispatching'">不可接单</template>
          <template v-else>{{ claiming ? '扣单中…' : '▶ 扣单' }}</template>
        </button>
      </div>
    </div>
  </div>
</template>
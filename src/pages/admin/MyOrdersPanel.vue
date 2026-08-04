<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { myOrders } from '../../api/orders'
import { orderTypeLabels, type OrderView } from '../../shared/orderLabels'

const orders = ref<OrderView[]>([])
const loading = ref(false)
const listError = ref('')

async function load() {
  loading.value = true
  listError.value = ''
  try {
    orders.value = await myOrders()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <h2 class="font-body font-bold text-xl text-hls-gold">我的订单</h2>
    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60">加载中…</p>

    <div v-if="!loading" class="hls-panel overflow-x-auto">
      <div class="hls-panel-title">订单列表</div>
      <table class="w-full text-sm whitespace-nowrap">
        <thead>
          <tr class="border-b-2 border-hls-border/50 text-hls-text-dark">
            <th class="text-left px-3 py-2 font-medium">订单编号</th>
            <th class="text-left px-3 py-2 font-medium">类型</th>
            <th class="text-left px-3 py-2 font-medium">单子名称</th>
            <th class="text-left px-3 py-2 font-medium">收益</th>
            <th class="text-left px-3 py-2 font-medium">抽成</th>
            <th class="text-left px-3 py-2 font-medium">派单人</th>
            <th class="text-left px-3 py-2 font-medium">备注</th>
            <th class="text-left px-3 py-2 font-medium">鸡腿</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.code" class="border-b border-hls-border/25 last:border-0">
            <td class="px-3 py-2 text-hls-text-muted">{{ o.code }}</td>
            <td class="px-3 py-2">{{ orderTypeLabels[o.orderType] }}</td>
            <td class="px-3 py-2 text-hls-text-dark">{{ o.title }}</td>
            <td class="px-3 py-2 text-hls-gold-dark font-medium">{{ o.earnings }}</td>
            <td class="px-3 py-2">{{ o.commissionRate }}%</td>
            <td class="px-3 py-2">{{ o.dispatcherName }}</td>
            <td class="px-3 py-2 text-hls-text-muted">{{ o.remark || '—' }}</td>
            <td class="px-3 py-2">{{ o.bonusRate }}%</td>
          </tr>
          <tr v-if="!orders.length"><td colspan="8" class="px-3 py-6 text-center text-hls-text-nav/60">暂无订单</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

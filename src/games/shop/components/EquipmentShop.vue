<script setup lang="ts">
import { computed } from 'vue';
import { EQUIPMENT_LIST, type Equipment } from '../game/equipment';
import { useRoomStore } from '../stores/room';

const roomStore = useRoomStore();

const myCoin = computed(() => roomStore.me?.coin ?? 0);

function canAfford(eq: Equipment): boolean {
  return myCoin.value >= eq.price;
}

function startBuy(eq: Equipment): void {
  if (!canAfford(eq)) return;
  roomStore.sendIntent('BUY_EQUIPMENT', { equipmentId: eq.id });
}
</script>

<template>
  <div class="hls-panel-deep p-6">
    <div class="flex items-center justify-between mb-3 gap-2">
      <h3 class="font-body font-bold text-hls-gold text-[22px]">
        ⚔ 护航补给架
      </h3>
      <span class="font-body text-sm uppercase text-hls-gold">
        当前鼠币 <span class="coin-tag ml-1">💰 {{ myCoin }}</span>
      </span>
    </div>
    <div class="hls-divider mb-5" />

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div
        v-for="eq in EQUIPMENT_LIST"
        :key="eq.id"
        class="hls-menu-row flex flex-col gap-2 transition-all"
      >
        <div class="flex items-center gap-3">
          <div class="text-3xl filter drop-shadow">{{ eq.icon }}</div>
          <div class="flex-1 min-w-0">
            <div class="font-body font-bold text-hls-text-light truncate text-lg">{{ eq.name }}</div>
          </div>
        </div>

        <div class="flex items-center justify-end mt-1 pt-2 border-t border-hls-border/30">
          <button
            class="hls-btn py-1.5 px-3 text-[13px]"
            :disabled="!canAfford(eq)"
            @click="startBuy(eq)"
          >
            加入背包
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

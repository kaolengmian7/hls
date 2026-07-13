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

function bundledNames(eq: Equipment): string {
  return eq.bundledLabels?.join('、') ?? '';
}
</script>

<template>
  <div class="sdv-panel-deep p-6">
    <div class="flex items-center justify-between mb-3 gap-2">
      <h3 class="font-pixel text-sdv-gold text-xl">
        ⚔ 护航补给架
      </h3>
      <span class="font-pixel text-xs tracking-[0.2em] uppercase text-sdv-gold">
        当前鼠币 <span class="coin-tag ml-1">💰 {{ myCoin }}</span>
      </span>
    </div>
    <div class="sdv-divider mb-5" />

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div
        v-for="eq in EQUIPMENT_LIST"
        :key="eq.id"
        class="sdv-menu-row flex flex-col gap-2 transition-all"
      >
        <div class="flex items-center gap-3">
          <div class="text-3xl filter drop-shadow">{{ eq.icon }}</div>
          <div class="flex-1 min-w-0">
            <div class="font-pixel font-bold text-sdv-text-light truncate text-base">{{ eq.name }}</div>
            <div class="font-body text-xs text-sdv-text-light/85">{{ eq.description }}</div>
          </div>
        </div>

        <div v-if="eq.bundledLabels?.length" class="text-[11px] font-body text-sdv-text-muted">
          套装包含：<span class="text-sdv-gold">{{ bundledNames(eq) }}</span>
        </div>

        <div class="flex items-center justify-between mt-1 pt-2 border-t border-sdv-border/30">
          <span class="coin-tag text-base">
            <span>💰</span> {{ eq.price }}
          </span>
          <button
            class="sdv-btn py-1.5 px-3 text-[11px]"
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


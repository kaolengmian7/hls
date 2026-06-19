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
  <div class="card rpg-corners p-6">
    <div class="flex items-center justify-between mb-3 gap-2">
      <h3 class="rpg-title text-xl">
        <span class="rune-mark mr-1">⚔</span>护航补给架
      </h3>
      <span class="font-serif text-xs tracking-[0.2em] uppercase text-gold-300">
        当前鼠币 <span class="coin-tag ml-1">💰 {{ myCoin }}</span>
      </span>
    </div>
    <div class="rpg-divider mb-5" />

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div
        v-for="eq in EQUIPMENT_LIST"
        :key="eq.id"
        class="armory-card rounded-md p-4 flex flex-col gap-2 transition-all"
      >
        <div class="flex items-center gap-3">
          <div class="text-3xl filter drop-shadow">{{ eq.icon }}</div>
          <div class="flex-1 min-w-0">
            <div class="font-serif font-bold text-parchment-50 truncate text-base">{{ eq.name }}</div>
            <div class="font-body text-xs text-parchment-300/85">{{ eq.description }}</div>
          </div>
        </div>

        <div v-if="eq.bundledLabels?.length" class="text-[11px] font-body text-parchment-400/80">
          套装包含：<span class="text-gold-200">{{ bundledNames(eq) }}</span>
        </div>

        <div class="flex items-center justify-between mt-1 pt-2 border-t border-gold-500/15">
          <span class="coin-tag text-base">
            <span>💰</span> {{ eq.price }}
          </span>
          <button
            class="btn-escort py-1.5 px-3 text-[11px]"
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

<style scoped>
.armory-card {
  background:
    linear-gradient(180deg, rgba(212, 175, 55, 0.06), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #1a0f0a, #120804);
  border: 1px solid rgba(212, 175, 55, 0.25);
  box-shadow: inset 0 1px 0 rgba(244, 230, 194, 0.06);
}
.armory-card:hover {
  border-color: rgba(232, 200, 124, 0.7);
  transform: translateY(-1px);
  box-shadow:
    inset 0 1px 0 rgba(244, 230, 194, 0.1),
    0 8px 22px -8px rgba(0, 0, 0, 0.6),
    0 0 0 1px rgba(212, 175, 55, 0.25);
}
</style>

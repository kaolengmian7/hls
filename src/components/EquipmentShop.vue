<script setup lang="ts">
import { computed, ref } from 'vue';
import {
  EQUIPMENT_LIST,
  getWeapon,
  needsPick,
  pickableWeapons,
  type Equipment,
} from '@/game/equipment';
import { useRoomStore } from '@/stores/room';

const roomStore = useRoomStore();

const pickingFor = ref<Equipment | null>(null);

const myCoin = computed(() => roomStore.me?.coin ?? 0);

function canAfford(eq: Equipment): boolean {
  return myCoin.value >= eq.price;
}

function startBuy(eq: Equipment): void {
  if (!canAfford(eq)) return;
  if (needsPick(eq)) {
    pickingFor.value = eq;
    return;
  }
  roomStore.sendIntent('BUY_EQUIPMENT', { equipmentId: eq.id });
}

function confirmPick(weaponId: string): void {
  const eq = pickingFor.value;
  if (!eq) return;
  roomStore.sendIntent('BUY_EQUIPMENT', { equipmentId: eq.id, pick: weaponId });
  pickingFor.value = null;
}

function cancelPick(): void {
  pickingFor.value = null;
}

function bundledNames(eq: Equipment): string {
  if (!eq.bundledIds?.length) return '';
  return eq.bundledIds
    .map((id) => getWeapon(id)?.name ?? id)
    .join('、');
}
</script>

<template>
  <div class="card rpg-corners p-6">
    <div class="flex items-center justify-between mb-3 gap-2">
      <h3 class="rpg-title text-xl">
        <span class="rune-mark mr-1">⚔</span>守誓者 · 秘银武库
      </h3>
      <span class="font-serif text-xs tracking-[0.2em] uppercase text-gold-300">
        银币囊 <span class="coin-tag ml-1">💰 {{ myCoin }}</span>
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

        <div v-if="eq.bundledIds?.length" class="text-[11px] font-body text-parchment-400/80">
          套装含：<span class="text-gold-200">{{ bundledNames(eq) }}</span>
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
            {{ needsPick(eq) ? '择器入手' : '请入囊中' }}
          </button>
        </div>
      </div>
    </div>

    <transition name="fade">
      <div
        v-if="pickingFor"
        class="fixed inset-0 z-30 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
        @click.self="cancelPick"
      >
        <div class="card rpg-corners w-full max-w-lg p-5 animate-slide-up">
          <div class="flex items-center justify-between mb-2 gap-2">
            <h4 class="rpg-title text-lg">
              <span class="rune-mark mr-1">⚔</span>择器 · {{ pickingFor.name }}
              <span class="coin-tag ml-2 text-sm">💰 {{ pickingFor.price }}</span>
            </h4>
            <button class="btn-ghost py-1 px-2 text-[11px]" @click="cancelPick">作罢</button>
          </div>
          <div class="rpg-divider mb-3" />
          <div v-if="pickingFor.bundledIds?.length" class="font-body text-xs text-parchment-300/85 mb-3">
            套装已附：<span class="text-gold-200">{{ bundledNames(pickingFor) }}</span>
          </div>
          <div class="font-serif text-[11px] tracking-[0.32em] uppercase text-gold-300/80 mb-2">
            ✦ 自瓦兰诸器中择一趁手 ✦
          </div>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 max-h-[60vh] overflow-y-auto pr-1">
            <button
              v-for="w in pickableWeapons(pickingFor)"
              :key="w.id"
              class="weapon-card rounded-md px-3 py-2 text-left transition-all
                focus:outline-none focus:ring-1 focus:ring-gold-400/60"
              @click="confirmPick(w.id)"
            >
              <div class="flex items-center gap-2">
                <span class="text-xl">{{ w.icon }}</span>
                <div class="min-w-0">
                  <div class="font-serif text-sm font-semibold text-parchment-50 truncate">{{ w.name }}</div>
                  <div class="font-body text-[11px] text-parchment-400/80">{{ w.category }}</div>
                </div>
              </div>
            </button>
          </div>
        </div>
      </div>
    </transition>
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
.weapon-card {
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.3), rgba(0, 0, 0, 0)), #15100b;
  border: 1px solid rgba(212, 175, 55, 0.22);
}
.weapon-card:hover {
  border-color: rgba(232, 200, 124, 0.65);
  background: linear-gradient(180deg, rgba(212, 175, 55, 0.06), rgba(0, 0, 0, 0)), #1c140d;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

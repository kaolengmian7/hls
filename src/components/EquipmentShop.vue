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
  <div class="card p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-slate-50">装备商店</h3>
      <span class="text-xs text-slate-400">余额 💰 {{ myCoin }}</span>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div
        v-for="eq in EQUIPMENT_LIST"
        :key="eq.id"
        class="rounded-xl border border-white/10 bg-slate-900/50 p-4 flex flex-col gap-2 hover:border-blue-400/40 transition-colors"
      >
        <div class="flex items-center gap-3">
          <div class="text-2xl">{{ eq.icon }}</div>
          <div class="flex-1 min-w-0">
            <div class="font-semibold text-slate-100 truncate">{{ eq.name }}</div>
            <div class="text-xs text-slate-400">{{ eq.description }}</div>
          </div>
        </div>

        <div v-if="eq.bundledIds?.length" class="text-[11px] text-slate-500">
          自带：<span class="text-slate-300">{{ bundledNames(eq) }}</span>
        </div>

        <div class="flex items-center justify-between mt-1">
          <span class="text-amber-400 font-semibold flex items-center gap-1">
            <span>💰</span> {{ eq.price }}
          </span>
          <button
            class="btn-escort py-1.5 text-xs"
            :disabled="!canAfford(eq)"
            @click="startBuy(eq)"
          >
            {{ needsPick(eq) ? '选枪购买' : '购买' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 二级选枪面板（modal） -->
    <transition name="fade">
      <div
        v-if="pickingFor"
        class="fixed inset-0 z-30 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
        @click.self="cancelPick"
      >
        <div class="card w-full max-w-lg p-5">
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-base font-semibold text-slate-50">
              选枪 · {{ pickingFor.name }}
              <span class="text-amber-400 ml-2 text-sm font-bold">💰 {{ pickingFor.price }}</span>
            </h4>
            <button class="btn-ghost py-1 text-xs" @click="cancelPick">取消</button>
          </div>
          <div v-if="pickingFor.bundledIds?.length" class="text-xs text-slate-400 mb-3">
            组合中自带：<span class="text-slate-200">{{ bundledNames(pickingFor) }}</span>
          </div>
          <div class="text-xs text-slate-500 mb-2">请选择一把枪：</div>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 max-h-[60vh] overflow-y-auto pr-1">
            <button
              v-for="w in pickableWeapons(pickingFor)"
              :key="w.id"
              class="rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-left
                hover:border-blue-400/40 hover:bg-slate-800 transition-colors
                focus:outline-none focus:ring-2 focus:ring-blue-400/40"
              @click="confirmPick(w.id)"
            >
              <div class="flex items-center gap-2">
                <span class="text-lg">{{ w.icon }}</span>
                <div class="min-w-0">
                  <div class="text-sm font-semibold text-slate-100 truncate">{{ w.name }}</div>
                  <div class="text-[11px] text-slate-500">{{ w.category }}</div>
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
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

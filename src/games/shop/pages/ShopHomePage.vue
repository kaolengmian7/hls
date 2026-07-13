<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { isValidRoomCode } from '@shared/utils/id';
import { usePlayerStore } from '@shared/stores/player';
import { useAuthStore } from '@/stores/auth';
import { useRoomStore } from '../stores/room';
import {
  createRoom,
  queryRoom,
  ApiError,
  DEFAULT_GUARANTEED_AMOUNT,
  MAX_GUARANTEED_AMOUNT,
  MIN_GUARANTEED_AMOUNT,
  isValidGuaranteedAmount,
} from '../net/api';
import AppHeader from '@shared/components/AppHeader.vue';
import ToastContainer from '../components/ToastContainer.vue';

const router = useRouter();
const route = useRoute();
const playerStore = usePlayerStore();
const authStore = useAuthStore();
const roomStore = useRoomStore();

const codeInput = ref<string>('');
const guaranteeInput = ref<string>(String(DEFAULT_GUARANTEED_AMOUNT));
const creating = ref(false);
const joining = ref(false);

const guaranteeNumber = computed<number | null>(() => {
  const trimmed = guaranteeInput.value.trim();
  if (!trimmed) return null;
  if (!/^\d+$/.test(trimmed)) return null;
  const n = Number(trimmed);
  return isValidGuaranteedAmount(n) ? n : null;
});
const guaranteeValid = computed(() => guaranteeNumber.value !== null);

onMounted(() => {
  playerStore.ensureIdentity();
  const queryCode = (route.query.code as string | undefined) ?? '';
  if (queryCode && isValidRoomCode(queryCode)) {
    codeInput.value = queryCode;
  }
});

function onCodeInput(e: Event): void {
  const target = e.target as HTMLInputElement;
  codeInput.value = target.value.replace(/\D/g, '').slice(0, 6);
}

function onGuaranteeInput(e: Event): void {
  const target = e.target as HTMLInputElement;
  guaranteeInput.value = target.value.replace(/\D/g, '').slice(0, 6);
}

const nickname = computed(() => authStore.user?.nickname ?? '');

async function handleCreate(): Promise<void> {
  if (creating.value) return;
  if (!guaranteeValid.value) {
    roomStore.pushToast(
      'warn',
      `商店保底请填 ${MIN_GUARANTEED_AMOUNT}~${MAX_GUARANTEED_AMOUNT} 之间的整数`,
    );
    return;
  }
  creating.value = true;
  try {
    const resp = await createRoom({
      nickname: nickname.value,
      guaranteedAmount: guaranteeNumber.value as number,
    });
    playerStore.rememberRoomCode(resp.code);

    await roomStore.connect({
      code: resp.code,
      nickname: nickname.value,
    });
    router.push({ name: 'shop-room', params: { code: resp.code } });
  } catch (err) {
    const message =
      err instanceof ApiError ? err.message : (err as Error).message;
    roomStore.pushToast('error', `创建商店失败：${message}`);
  } finally {
    creating.value = false;
  }
}

async function handleJoin(): Promise<void> {
  if (joining.value) return;
  if (!isValidRoomCode(codeInput.value)) {
    roomStore.pushToast('warn', '请输入 6 位商店口令');
    return;
  }
  joining.value = true;
  try {
    const summary = await queryRoom(codeInput.value);
    if (!summary.exists) {
      roomStore.pushToast('error', '这家商店不存在，或已经关闭');
      return;
    }

    // Leave current room (if any) before joining new one.
    if (roomStore.status === 'connected') {
      roomStore.leaveRoom();
    }

    await roomStore.connect({
      code: codeInput.value,
      nickname: nickname.value,
    });
    playerStore.rememberRoomCode(codeInput.value);
    const dest = roomStore.snapshot?.status === 'PLAYING' ? 'shop-play' : 'shop-room';
    router.push({ name: dest, params: { code: codeInput.value } });
  } catch (err) {
    const reason = (err as Error).message;
    if (reason === 'FULL') roomStore.pushToast('warn', '店内 3 个位置已满');
    else if (reason === 'PLAYING') roomStore.pushToast('warn', '商店已经开张，暂时不能中途加入');
    else if (reason === 'DUP') roomStore.pushToast('warn', '您已经在这家商店里了');
    else if (reason === 'NOT_FOUND') roomStore.pushToast('error', '这家商店不存在，或已经关闭');
    else if (reason === 'TIMEOUT') roomStore.pushToast('error', '加入失败，请稍后重试');
    else roomStore.pushToast('error', `进入商店失败：${reason}`);
  } finally {
    joining.value = false;
  }
}
</script>

<template>
  <div class="flex-1 flex flex-col">
    <AppHeader />
    <ToastContainer />

    <header class="px-6 pt-12 pb-4 text-center select-none">
      <div class="font-pixel text-sm mb-3 text-sdv-gold/80 tracking-widest animate-pixel-pulse">
        ✦ 花栗鼠俱乐部 · 原创玩法 ✦
      </div>
      <h1 class="font-pixel text-4xl sm:text-6xl font-bold leading-tight tracking-[0.2em] text-sdv-gold">
        花栗鼠的神秘商店
      </h1>
      <div class="mt-4 mx-auto max-w-xs sdv-divider" />
    </header>

    <main class="flex-1 flex items-center justify-center px-4 py-6">
      <div class="sdv-panel w-full max-w-md animate-pixel-slide">
        <div class="sdv-panel-title text-center">开店 / 入店</div>
        <div class="p-7 sm:p-9 space-y-5">

          <div>
            <label class="font-pixel text-sm text-sdv-text-dark tracking-wider block mb-1">
              商店保底
            </label>
            <div class="relative">
              <input
                :value="guaranteeInput"
                class="sdv-input w-full pr-10 text-right"
                :class="{ 'ring-2 ring-red-400': !guaranteeValid }"
                inputmode="numeric"
                maxlength="6"
                :placeholder="String(DEFAULT_GUARANTEED_AMOUNT)"
                @input="onGuaranteeInput"
              />
              <span class="pointer-events-none absolute inset-y-0 right-3 flex items-center font-pixel text-sdv-text-dark">W</span>
            </div>
            <p class="font-body text-xs mt-1.5" :class="guaranteeValid ? 'text-sdv-text-muted' : 'text-red-400'">
              <template v-if="guaranteeValid">
                当前保底：<span class="font-pixel text-sdv-gold-dark">{{ guaranteeNumber }}W</span>
                · 可填 {{ MIN_GUARANTEED_AMOUNT }} ~ {{ MAX_GUARANTEED_AMOUNT }}
              </template>
              <template v-else>
                保底请填 {{ MIN_GUARANTEED_AMOUNT }} ~ {{ MAX_GUARANTEED_AMOUNT }} 之间的整数。
              </template>
            </p>
          </div>

          <button class="sdv-btn w-full py-3" :disabled="creating || !guaranteeValid" @click="handleCreate">
            <span v-if="creating">商店布置中…</span>
            <span v-else>⚜ 开一家神秘商店</span>
          </button>

          <div class="sdv-divider-label"><span>OR</span></div>

          <div>
            <label class="font-pixel text-sm text-sdv-text-dark tracking-wider block mb-1">
              商店口令（六位）
            </label>
            <input
              :value="codeInput"
              class="sdv-input w-full tracking-[.55em] text-center text-2xl"
              placeholder="✦ ✦ ✦ ✦ ✦ ✦"
              inputmode="numeric"
              maxlength="6"
              @input="onCodeInput"
              @keyup.enter="handleJoin"
            />
          </div>

          <button class="sdv-btn-secondary w-full py-3" :disabled="joining" @click="handleJoin">
            <span v-if="joining">正在入店…</span>
            <span v-else>✶ 进入已有商店</span>
          </button>

        </div>
      </div>
    </main>

    <footer class="text-center font-pixel text-sm tracking-[0.4em] uppercase text-sdv-text-nav/40 py-5">
      ✥ 微信服务号 · 花栗鼠俱乐部 ✥
    </footer>
  </div>
</template>

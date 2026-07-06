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
    if (err instanceof ApiError && err.status === 409) {
      const body = err.body as { activeRoomCode?: string } | null;
      const ac = body?.activeRoomCode;
      if (ac) {
        roomStore.pushToast('warn', `您已在商店 ${ac} 中，请先离开当前商店`);
        codeInput.value = ac;
      } else {
        roomStore.pushToast('error', '您已在另一个商店中');
      }
      return;
    }
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

    if (summary.activeRoomCode && summary.activeRoomCode !== codeInput.value) {
      roomStore.pushToast('warn', `您已在商店 ${summary.activeRoomCode} 中，请先离开当前商店`);
      return;
    }

    const isMember = summary.isMember === true;
    if (!isMember) {
      if (summary.status === 'PLAYING') {
        roomStore.pushToast('warn', '商店已经开张，暂时不能中途加入');
        return;
      }
      if ((summary.playerCount ?? 0) >= 3) {
        roomStore.pushToast('warn', '店内 3 个位置已满');
        return;
      }
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
      <div class="rpg-subtitle text-[10px] sm:text-xs mb-3 animate-flicker">
        ✦ 花栗鼠俱乐部 · 原创玩法 ✦
      </div>
      <h1 class="rpg-title text-4xl sm:text-6xl font-bold leading-tight">
        花栗鼠的神秘商店
      </h1>
      <div class="mt-4 mx-auto max-w-xs rpg-divider" />
    </header>

    <main class="flex-1 flex items-center justify-center px-4 py-6">
      <div class="card rpg-corners card-ornate w-full max-w-md p-7 sm:p-9 animate-slide-up">
        <div class="text-center mb-6">
          <div class="rune-mark text-2xl">⚜</div>
        </div>

        <div class="mb-7">
          <label class="font-serif text-xs tracking-[0.32em] uppercase text-gold-300 mb-2 block">
            <span class="rune-mark mr-1">❖</span>商店保底
          </label>
          <div class="relative">
            <input
              :value="guaranteeInput"
              class="input pr-10 text-right font-serif text-lg"
              :class="{ 'border-oxblood-300/80 ring-1 ring-oxblood-300/40': !guaranteeValid }"
              inputmode="numeric"
              maxlength="6"
              :placeholder="String(DEFAULT_GUARANTEED_AMOUNT)"
              @input="onGuaranteeInput"
            />
            <span
              class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-gold-300 text-sm font-bold tracking-widest"
            >W</span>
          </div>
          <p class="text-[11px] mt-2 font-body" :class="guaranteeValid ? 'text-parchment-300/80' : 'text-oxblood-300'">
            <template v-if="guaranteeValid">
              当前保底：<span class="text-gold-300 font-bold">{{ guaranteeNumber }}W</span>
              · 可填 {{ MIN_GUARANTEED_AMOUNT }} ~ {{ MAX_GUARANTEED_AMOUNT }}
            </template>
            <template v-else>
              保底请填 {{ MIN_GUARANTEED_AMOUNT }} ~ {{ MAX_GUARANTEED_AMOUNT }} 之间的整数。
            </template>
          </p>
        </div>

        <button
          class="btn-primary w-full py-3 text-base"
          :disabled="creating || !guaranteeValid"
          @click="handleCreate"
        >
          <span v-if="creating">商店布置中…</span>
          <span v-else>⚜ 开一家神秘商店</span>
        </button>

        <div class="my-7 rpg-divider-text">
          <span>OR</span>
        </div>

        <div class="mb-4">
          <label class="font-serif text-xs tracking-[0.32em] uppercase text-gold-300 mb-2 block">
            <span class="rune-mark mr-1">❖</span>商店口令 · 六位
          </label>
          <input
            :value="codeInput"
            class="input tracking-[.55em] text-center text-2xl font-serif"
            placeholder="✦ ✦ ✦ ✦ ✦ ✦"
            inputmode="numeric"
            maxlength="6"
            @input="onCodeInput"
            @keyup.enter="handleJoin"
          />
        </div>

        <button
          class="btn-secondary w-full py-3 text-base"
          :disabled="joining"
          @click="handleJoin"
        >
          <span v-if="joining">正在入店…</span>
          <span v-else>✶ 进入已有商店</span>
        </button>
      </div>
    </main>

    <footer class="text-center font-serif text-[11px] tracking-[0.4em] uppercase text-parchment-400/60 py-5">
      <span class="rune-mark mr-2">✥</span>微信服务号 · 花栗鼠俱乐部<span class="rune-mark ml-2">✥</span>
    </footer>
  </div>
</template>

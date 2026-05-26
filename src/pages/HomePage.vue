<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { isValidRoomCode } from '@/utils/id';
import { usePlayerStore } from '@/stores/player';
import { useRoomStore } from '@/stores/room';
import {
  createRoom,
  queryRoom,
  ApiError,
  DEFAULT_GUARANTEED_AMOUNT,
  MAX_GUARANTEED_AMOUNT,
  MIN_GUARANTEED_AMOUNT,
  isValidGuaranteedAmount,
} from '@/net/api';
import ToastContainer from '@/components/ToastContainer.vue';

const router = useRouter();
const route = useRoute();
const playerStore = usePlayerStore();
const roomStore = useRoomStore();

const codeInput = ref<string>('');
const nicknameInput = ref<string>('');
const guaranteeInput = ref<string>(String(DEFAULT_GUARANTEED_AMOUNT));
const creating = ref(false);
const joining = ref(false);

/**
 * 保底金额本地校验：仅允许整数，落在 [MIN, MAX]。
 * 空字符串视为非法（要求显式填写或保留默认值）。
 */
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
  nicknameInput.value = playerStore.nickname;
  const queryCode = (route.query.code as string | undefined) ?? '';
  if (queryCode && isValidRoomCode(queryCode)) {
    codeInput.value = queryCode;
  }
});

function syncNickname(): void {
  if (nicknameInput.value.trim()) {
    playerStore.setNickname(nicknameInput.value);
  }
}

function onCodeInput(e: Event): void {
  const target = e.target as HTMLInputElement;
  codeInput.value = target.value.replace(/\D/g, '').slice(0, 6);
}

function onGuaranteeInput(e: Event): void {
  const target = e.target as HTMLInputElement;
  // 仅保留数字，最长 6 位（MAX_GUARANTEED_AMOUNT 是 999999）。
  guaranteeInput.value = target.value.replace(/\D/g, '').slice(0, 6);
}

async function handleCreate(): Promise<void> {
  if (creating.value) return;
  syncNickname();
  if (!playerStore.nickname.trim()) {
    roomStore.pushToast('warn', '请先填写昵称');
    return;
  }
  if (!guaranteeValid.value) {
    roomStore.pushToast(
      'warn',
      `请输入 ${MIN_GUARANTEED_AMOUNT}~${MAX_GUARANTEED_AMOUNT} 之间的整数（保底金额）`,
    );
    return;
  }
  creating.value = true;
  try {
    const resp = await createRoom({
      nickname: playerStore.nickname,
      guaranteedAmount: guaranteeNumber.value as number,
    });
    playerStore.setPlayerId(resp.playerId);
    playerStore.setSessionToken(resp.code, resp.sessionToken);
    playerStore.rememberRoomCode(resp.code);

    await roomStore.connect({
      code: resp.code,
      playerId: resp.playerId,
      nickname: playerStore.nickname,
      sessionToken: resp.sessionToken,
    });
    router.push({ name: 'room', params: { code: resp.code } });
  } catch (err) {
    const message =
      err instanceof ApiError ? err.message : (err as Error).message;
    roomStore.pushToast('error', `创建失败：${message}`);
  } finally {
    creating.value = false;
  }
}

async function handleJoin(): Promise<void> {
  if (joining.value) return;
  syncNickname();
  if (!playerStore.nickname.trim()) {
    roomStore.pushToast('warn', '请先填写昵称');
    return;
  }
  if (!isValidRoomCode(codeInput.value)) {
    roomStore.pushToast('warn', '请输入 6 位房间号');
    return;
  }
  joining.value = true;
  try {
    const summary = await queryRoom(codeInput.value);
    if (!summary.exists) {
      roomStore.pushToast('error', '房间不存在或已关闭');
      return;
    }
    if (summary.status === 'PLAYING') {
      roomStore.pushToast('warn', '游戏已开始，无法加入');
      return;
    }
    if ((summary.playerCount ?? 0) >= 3) {
      roomStore.pushToast('warn', '房间已满');
      return;
    }

    await roomStore.connect({
      code: codeInput.value,
      playerId: playerStore.playerId,
      nickname: playerStore.nickname,
      sessionToken: playerStore.getSessionToken(codeInput.value),
    });
    playerStore.rememberRoomCode(codeInput.value);
    router.push({ name: 'room', params: { code: codeInput.value } });
  } catch (err) {
    const reason = (err as Error).message;
    if (reason === 'FULL') roomStore.pushToast('warn', '房间已满');
    else if (reason === 'PLAYING') roomStore.pushToast('warn', '游戏已开始，无法加入');
    else if (reason === 'DUP') roomStore.pushToast('warn', '该账号已在房间内');
    else if (reason === 'NOT_FOUND') roomStore.pushToast('error', '房间不存在或已关闭');
    else if (reason === 'TIMEOUT') roomStore.pushToast('error', '连接超时，请稍后再试');
    else roomStore.pushToast('error', `加入失败：${reason}`);
  } finally {
    joining.value = false;
  }
}
</script>

<template>
  <div class="flex-1 flex flex-col">
    <ToastContainer />

    <header class="px-6 pt-10 pb-4 text-center">
      <h1 class="text-4xl sm:text-5xl font-extrabold text-white tracking-tight">
        花栗鼠的神秘商店
      </h1>
    </header>

    <main class="flex-1 flex items-center justify-center px-4 py-6">
      <div class="card w-full max-w-md p-6 sm:p-8 animate-slide-up">
        <div class="mb-4">
          <label class="text-xs text-slate-400 mb-1 block">你的昵称</label>
          <input
            v-model="nicknameInput"
            class="input"
            maxlength="16"
            placeholder="给自己起个名字"
            @blur="syncNickname"
          />
        </div>

        <div class="mb-6">
          <label class="text-xs text-slate-400 mb-1 block">
            保底金额
            <span class="text-slate-500">（仅展示，不影响金币结算）</span>
          </label>
          <div class="relative">
            <input
              :value="guaranteeInput"
              class="input pr-10 text-right font-mono"
              :class="{ 'ring-1 ring-rose-500/60': !guaranteeValid }"
              inputmode="numeric"
              maxlength="6"
              :placeholder="String(DEFAULT_GUARANTEED_AMOUNT)"
              @input="onGuaranteeInput"
            />
            <span
              class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-slate-400 text-sm font-semibold"
            >W</span>
          </div>
          <p class="text-[11px] mt-1" :class="guaranteeValid ? 'text-slate-500' : 'text-rose-400'">
            <template v-if="guaranteeValid">
              当前保底：{{ guaranteeNumber }}W ·
              范围 {{ MIN_GUARANTEED_AMOUNT }}~{{ MAX_GUARANTEED_AMOUNT }}
            </template>
            <template v-else>
              请输入 {{ MIN_GUARANTEED_AMOUNT }}~{{ MAX_GUARANTEED_AMOUNT }} 之间的整数
            </template>
          </p>
        </div>

        <button
          class="btn-primary w-full py-3 text-base"
          :disabled="creating || !guaranteeValid"
          @click="handleCreate"
        >
          <span v-if="creating">创建中...</span>
          <span v-else>创建房间</span>
        </button>

        <div class="my-6 flex items-center gap-3 text-xs text-slate-500">
          <div class="flex-1 h-px bg-white/10" />
          <span>或</span>
          <div class="flex-1 h-px bg-white/10" />
        </div>

        <div class="mb-3">
          <label class="text-xs text-slate-400 mb-1 block">输入 6 位房间号</label>
          <input
            :value="codeInput"
            class="input tracking-[.5em] text-center text-lg font-mono"
            placeholder="------"
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
          <span v-if="joining">连接中...</span>
          <span v-else>加入房间</span>
        </button>
      </div>
    </main>

    <footer class="text-center text-xs text-slate-500 py-4">
      <p>花栗鼠俱乐部.三角洲行动</p>
    </footer>
  </div>
</template>

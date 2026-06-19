<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useRoomStore } from '../stores/room';
import { NICKNAME_PRESETS, composeNickname, usePlayerStore } from '@shared/stores/player';
import PlayerSlot from '../components/PlayerSlot.vue';
import ToastContainer from '../components/ToastContainer.vue';
import ConfirmDialog from '@shared/components/ConfirmDialog.vue';
import { MAX_PLAYERS, summarizeRoster } from '../game/rules';

const router = useRouter();
const route = useRoute();
const roomStore = useRoomStore();
const playerStore = usePlayerStore();

const showLeaveConfirm = ref(false);
const showHostClosedDialog = ref(false);
const hostClosedReason = ref('');
const restoring = ref(false);
const selectedPreset = ref('');

const code = computed(() => roomStore.code || (route.params.code as string));
const players = computed(() => roomStore.players);
const me = computed(() => roomStore.me);
const isHost = computed(() => roomStore.isHost);

const slots = computed(() => {
  const list = [...players.value];
  while (list.length < MAX_PLAYERS) list.push(null as never);
  return list.slice(0, MAX_PLAYERS);
});

const startReady = computed(() => {
  const r = summarizeRoster(players.value);
  return r;
});

const startHint = computed(() => {
  if (startReady.value.ok) return '三人已到齐，神秘商店随时可以开启';
  switch (startReady.value.reason) {
    case 'NOT_FULL':
      return '还有空位';
    case 'ROLE_INCOMPLETE':
      return '角色未齐（须 1 个板板 + 2 个护航）';
    case 'NOT_ALL_READY':
      return '还有人没有准备好';
    default:
      return '当前还不能开张';
  }
});

/** 名牌拼接所用的「自定义昵称」基底（不含名牌前缀）。 */
const baseNickname = computed(() => playerStore.nicknameCustom || me.value?.nickname || '');
/** 选中名牌后拼出的最终昵称预览（名牌选填）。 */
const nicknamePreview = computed(() => composeNickname(selectedPreset.value, baseNickname.value));

onMounted(async () => {
  playerStore.ensureIdentity();
  selectedPreset.value = playerStore.nicknamePreset;
  if (roomStore.status === 'connected') return;

  // 刷新或直接打开链接：尝试用本地 sessionToken 透明重连。
  const targetCode = (route.params.code as string) || roomStore.code;
  restoring.value = true;
  try {
    const ok = await roomStore.tryRestoreSession(targetCode);
    if (!ok) {
      router.replace({ name: 'home', query: { code: targetCode } });
    }
  } finally {
    restoring.value = false;
  }
});

watch(
  () => roomStore.status,
  (status) => {
    // 恢复连接阶段失败不弹"房间已关闭"对话框，由 onMounted 静默跳回首页。
    if (status === 'closed' && !restoring.value) {
      hostClosedReason.value = roomStore.closedReason || 'CLOSED';
      showHostClosedDialog.value = true;
      setTimeout(() => {
        if (showHostClosedDialog.value) {
          showHostClosedDialog.value = false;
          router.replace({ name: 'home' });
        }
      }, 3000);
    }
  },
);

watch(
  () => roomStore.snapshot?.status,
  (status) => {
    if (status === 'PLAYING') {
      router.replace({ name: 'shop-play', params: { code: code.value } });
    }
  },
);

async function copyCode(): Promise<void> {
  try {
    await navigator.clipboard.writeText(code.value);
    roomStore.pushToast('success', '商店口令已复制');
  } catch {
    roomStore.pushToast('warn', '誊抄失败');
  }
}

async function copyLink(): Promise<void> {
  try {
    const url = `${window.location.origin}/?code=${code.value}`;
    await navigator.clipboard.writeText(url);
    roomStore.pushToast('success', '邀请链接已复制');
  } catch {
    roomStore.pushToast('warn', '邀请链接复制失败');
  }
}

function changeRole(role: 'BOSS' | 'ESCORT'): void {
  roomStore.sendIntent('CHANGE_ROLE', { role });
}

/**
 * 选填名牌：点击某个名牌将其拼到昵称前；再次点击同一个则取消。
 * 仅更新展示昵称，不影响角色 / 准备状态；空昵称基底时忽略（首页应已填昵称）。
 */
function applyNameplate(preset: string): void {
  const next = selectedPreset.value === preset ? '' : preset;
  selectedPreset.value = next;
  playerStore.setNicknamePreset(next);
  const finalNickname = composeNickname(next, baseNickname.value);
  if (!finalNickname) return;
  roomStore.sendIntent('CHANGE_NICKNAME', { nickname: finalNickname });
}

function clearNameplate(): void {
  if (!selectedPreset.value) return;
  applyNameplate(selectedPreset.value);
}

function toggleReady(): void {
  if (!me.value) return;
  if (!me.value.role) {
    roomStore.pushToast('warn', '请先选择角色');
    return;
  }
  roomStore.sendIntent('READY', { ready: !me.value.ready });
}

function startGame(): void {
  if (!isHost.value) return;
  if (!startReady.value.ok) {
    roomStore.pushToast('warn', `暂时不能开张：${startHint.value}`);
    return;
  }
  roomStore.sendIntent('START_GAME', {});
}

function confirmLeave(): void {
  showLeaveConfirm.value = false;
  roomStore.leaveRoom();
  router.replace({ name: 'home' });
}

function dismissHostClosed(): void {
  showHostClosedDialog.value = false;
  router.replace({ name: 'home' });
}
</script>

<template>
  <div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center">
    <ToastContainer />
    <div class="card rpg-corners p-10 text-center max-w-sm">
      <div class="rune-mark text-3xl animate-flicker">✶</div>
      <div class="rpg-title text-xl mt-2">重返神秘商店</div>
      <div class="font-body text-parchment-300/80 mt-2">正在根据商店口令返回花栗鼠的神秘商店…</div>
      <div class="mt-4 font-serif text-xs tracking-[0.32em] uppercase text-gold-300/70">
        口令 · {{ code }}
      </div>
    </div>
  </div>

  <div v-else class="flex-1 flex flex-col">
    <ToastContainer />

    <header
      class="px-4 sm:px-8 py-4 flex items-center justify-between gap-3 flex-wrap border-b border-gold-500/20 bg-ink-700/60 backdrop-blur"
    >
      <div class="flex items-center gap-3 flex-wrap">
        <div class="leading-tight">
          <div class="font-serif text-[10px] tracking-[0.32em] uppercase text-gold-300/70">
            <span class="rune-mark mr-1">⚜</span>商店口令
          </div>
          <div class="rpg-title text-2xl tracking-[.4em] font-display">{{ code }}</div>
        </div>
        <button class="btn-ghost" @click="copyCode">复制口令</button>
        <button class="btn-ghost" @click="copyLink">复制邀请链接</button>
      </div>
      <div class="flex items-center gap-3">
        <div
          class="rounded-md px-3 py-1.5 leading-tight text-right border border-gold-500/40 bg-ink-700/60"
          :title="`商店保底`"
        >
          <div class="font-serif text-[10px] uppercase tracking-[0.3em] text-gold-300/80">
            商店保底
          </div>
          <div class="text-base font-bold font-serif text-gold-200">
            {{ roomStore.guaranteedAmount }}<span class="text-gold-300/80 ml-0.5">W</span>
          </div>
        </div>
        <div class="badge">
          <span class="rune-mark mr-1">🐿</span>{{ players.length }} / {{ MAX_PLAYERS }}
        </div>
        <button class="btn-ghost text-oxblood-300 hover:!text-oxblood-300" @click="showLeaveConfirm = true">
          离开商店
        </button>
      </div>
    </header>

    <main class="flex-1 px-4 sm:px-8 py-8 max-w-4xl mx-auto w-full">
      <div class="text-center mb-6">
        <h2 class="rpg-title text-2xl sm:text-3xl">角色分配</h2>
        <div class="mx-auto max-w-md mt-2 rpg-divider" />
        <p class="mt-2 font-body text-parchment-300/80 text-sm">
          本局固定 3 个角色：1 个板板，2 个护航。
        </p>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <PlayerSlot
          v-for="(p, idx) in slots"
          :key="p?.id ?? `slot-${idx}`"
          :player="p"
          :is-self="p?.id === roomStore.selfPlayerId"
        />
      </div>

      <div class="card rpg-corners mt-7 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="rpg-title text-lg sm:text-xl">
            <span class="rune-mark mr-1">⚜</span>选择角色
          </h3>
          <span class="font-body text-xs text-parchment-300/70">
            准备后将锁定角色，直到本局开始。
          </span>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <button
            class="role-card"
            :class="{
              'is-active-escort': me?.role === 'ESCORT',
              'opacity-60 cursor-not-allowed': me?.ready,
            }"
            :disabled="me?.ready"
            @click="changeRole('ESCORT')"
          >
            <div class="flex items-center gap-2">
              <span class="text-2xl">🛡️</span>
              <div class="font-display tracking-[0.2em] text-sapphire-300 text-xl uppercase">
                护航
              </div>
            </div>
            <div class="rpg-divider mt-2 mb-2 opacity-70" />
            <p class="font-body text-xs text-parchment-300/85 leading-relaxed">
              名额 2 · 起始 10 金 · 花费鼠币起装备保护板板
            </p>
          </button>
          <button
            class="role-card"
            :class="{
              'is-active-boss': me?.role === 'BOSS',
              'opacity-60 cursor-not-allowed': me?.ready,
            }"
            :disabled="me?.ready"
            @click="changeRole('BOSS')"
          >
            <div class="flex items-center gap-2">
              <span class="text-2xl">🧙</span>
              <div class="font-display tracking-[0.2em] text-ember-400 text-xl uppercase">
                板板
              </div>
            </div>
            <div class="rpg-divider mt-2 mb-2 opacity-70" />
            <p class="font-body text-xs text-parchment-300/85 leading-relaxed">
              名额 1 · 起始 0 金 · 带队撤离 · 从神秘商店中开启丰厚奖励
            </p>
          </button>
        </div>
        <p class="font-body text-[12px] text-parchment-400/80 mt-4 text-center">
          若切换角色，需要重新点击准备。
        </p>
      </div>

      <div class="card rpg-corners mt-5 p-6">
        <div class="flex items-center justify-between mb-3">
          <h3 class="rpg-title text-lg sm:text-xl">
            <span class="rune-mark mr-1">❖</span>名牌 · 选填
          </h3>
          <span class="font-body text-xs text-parchment-300/70">
            挑一个名牌拼到昵称前，或保持纯昵称。
          </span>
        </div>
        <div class="mb-3 rounded-md border border-gold-500/20 bg-ink-700/40 px-3 py-2">
          <div class="text-[11px] font-serif tracking-[0.2em] text-parchment-300/75">
            当前昵称预览
          </div>
          <div class="mt-1 font-serif text-base text-parchment-50 break-all">
            {{ nicknamePreview || '（未设置昵称）' }}
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="btn-ghost !normal-case !tracking-[0.08em] !px-2.5 !py-1.5 text-[11px]"
            :class="!selectedPreset ? '!border-gold-400/70 !text-gold-200 !bg-gold-500/10' : ''"
            @click="clearNameplate"
          >
            不使用名牌
          </button>
          <button
            v-for="preset in NICKNAME_PRESETS"
            :key="preset"
            type="button"
            class="btn-ghost !normal-case !tracking-[0.08em] !px-2.5 !py-1.5 text-[11px]"
            :class="selectedPreset === preset ? '!border-gold-400/70 !text-gold-200 !bg-gold-500/10' : ''"
            @click="applyNameplate(preset)"
          >
            {{ preset }}
          </button>
        </div>
      </div>

      <div class="card rpg-corners mt-5 p-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div class="font-serif text-sm flex items-center gap-2">
            <template v-if="me?.ready">
              <span class="rune-mark text-forest-300 text-lg animate-pulse_soft">✓</span>
              <span class="text-forest-300 font-bold tracking-widest uppercase">已经准备</span>
            </template>
            <template v-else>
              <span class="rune-mark text-gold-300 text-lg animate-flicker">✦</span>
              <span class="text-gold-300 font-bold tracking-widest uppercase">尚未准备</span>
            </template>
          </div>
          <div class="font-body text-xs text-parchment-300/80 mt-1">{{ startHint }}</div>
        </div>
        <div class="flex flex-wrap gap-3 items-center">
          <button
            class="btn"
            :class="me?.ready ? 'btn-danger' : 'btn-primary'"
            @click="toggleReady"
          >
            {{ me?.ready ? '取消准备' : '点击准备' }}
          </button>
          <button
            v-if="isHost"
            class="btn-boss"
            :disabled="!startReady.ok"
            :title="startHint"
            @click="startGame"
          >
            ⚜ 开张营业
          </button>
        </div>
      </div>
    </main>

    <ConfirmDialog
      :open="showLeaveConfirm"
      :title="isHost ? '关闭商店？' : '离开商店？'"
      :message="
        isHost
          ? '您是房主，离开后这家神秘商店会立刻关闭。'
          : '离开后仍可凭商店口令再次进入。'
      "
      :confirm-text="isHost ? '关闭' : '离开'"
      danger
      @confirm="confirmLeave"
      @cancel="showLeaveConfirm = false"
    />

    <ConfirmDialog
      :open="showHostClosedDialog"
      title="商店已关闭"
      :message="`商店已关闭（${hostClosedReason || 'CLOSED'}），3 秒后返回首页。`"
      confirm-text="返回首页"
      cancel-text="稍候"
      @confirm="dismissHostClosed"
      @cancel="dismissHostClosed"
    />
  </div>
</template>

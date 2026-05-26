<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useRoomStore } from '@/stores/room';
import { usePlayerStore } from '@/stores/player';
import PlayerSlot from '@/components/PlayerSlot.vue';
import ToastContainer from '@/components/ToastContainer.vue';
import ConfirmDialog from '@/components/ConfirmDialog.vue';
import { MAX_PLAYERS, summarizeRoster } from '@/game/rules';

const router = useRouter();
const route = useRoute();
const roomStore = useRoomStore();
const playerStore = usePlayerStore();

const showLeaveConfirm = ref(false);
const showHostClosedDialog = ref(false);
const hostClosedReason = ref('');
const restoring = ref(false);

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
  if (startReady.value.ok) return '所有条件已就绪';
  switch (startReady.value.reason) {
    case 'NOT_FULL':
      return '等待玩家加入';
    case 'ROLE_INCOMPLETE':
      return '角色未分配完整 (1 老板 + 2 护航)';
    case 'NOT_ALL_READY':
      return '等待玩家准备';
    default:
      return '条件未满足';
  }
});

onMounted(async () => {
  playerStore.ensureIdentity();
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
      router.replace({ name: 'play', params: { code: code.value } });
    }
  },
);

async function copyCode(): Promise<void> {
  try {
    await navigator.clipboard.writeText(code.value);
    roomStore.pushToast('success', '房间号已复制');
  } catch {
    roomStore.pushToast('warn', '复制失败');
  }
}

async function copyLink(): Promise<void> {
  try {
    const url = `${window.location.origin}/?code=${code.value}`;
    await navigator.clipboard.writeText(url);
    roomStore.pushToast('success', '邀请链接已复制');
  } catch {
    roomStore.pushToast('warn', '复制失败');
  }
}

function changeRole(role: 'BOSS' | 'ESCORT'): void {
  roomStore.sendIntent('CHANGE_ROLE', { role });
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
    roomStore.pushToast('warn', `无法开始：${startHint.value}`);
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
    <div class="card p-8 text-center text-slate-300">
      <div class="text-base font-semibold mb-2">正在恢复房间连接...</div>
      <div class="text-xs text-slate-500">房间号 {{ code }}</div>
    </div>
  </div>

  <div v-else class="flex-1 flex flex-col">
    <ToastContainer />

    <header class="px-4 sm:px-8 py-4 flex items-center justify-between border-b border-white/5">
      <div class="flex items-center gap-3">
        <div>
          <div class="text-xs text-slate-400">房间号</div>
          <div class="text-2xl font-bold tracking-[.4em] font-mono">{{ code }}</div>
        </div>
        <button class="btn-ghost py-1.5 px-3 text-xs" @click="copyCode">复制 Code</button>
        <button class="btn-ghost py-1.5 px-3 text-xs" @click="copyLink">复制邀请链接</button>
      </div>
      <div class="flex items-center gap-3">
        <div
          class="rounded-xl border border-amber-400/20 bg-amber-500/10 px-3 py-1.5 leading-tight text-right"
          :title="`房间保底金额（仅展示，不影响金币结算）`"
        >
          <div class="text-[10px] uppercase tracking-wider text-amber-300/80">保底</div>
          <div class="text-sm font-semibold text-amber-200 font-mono">
            {{ roomStore.guaranteedAmount }}<span class="text-amber-300/70 ml-0.5">W</span>
          </div>
        </div>
        <div class="badge bg-slate-700/50 text-slate-300">
          {{ players.length }} / {{ MAX_PLAYERS }}
        </div>
        <button class="btn-ghost text-rose-400" @click="showLeaveConfirm = true">退出房间</button>
      </div>
    </header>

    <main class="flex-1 px-4 sm:px-8 py-8 max-w-4xl mx-auto w-full">
      <h2 class="text-xl font-semibold text-slate-100 mb-4">选角与准备</h2>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <PlayerSlot
          v-for="(p, idx) in slots"
          :key="p?.id ?? `slot-${idx}`"
          :player="p"
          :is-self="p?.id === roomStore.selfPlayerId"
        />
      </div>

      <div class="card mt-6 p-6">
        <h3 class="text-base font-semibold text-slate-100 mb-3">选择阵营</h3>
        <div class="grid grid-cols-2 gap-4">
          <button
            class="rounded-2xl border border-white/10 bg-slate-900/60 p-4 text-left hover:border-blue-400/40 transition-colors"
            :class="{
              'ring-2 ring-blue-400/60 border-blue-400/60': me?.role === 'ESCORT',
              'opacity-60': me?.ready,
            }"
            :disabled="me?.ready"
            @click="changeRole('ESCORT')"
          >
            <div class="text-blue-300 font-bold text-lg">护航</div>
            <p class="text-xs text-slate-400 mt-1">名额 2 · 初始 10 币 · 装备</p>
          </button>
          <button
            class="rounded-2xl border border-white/10 bg-slate-900/60 p-4 text-left hover:border-orange-400/40 transition-colors"
            :class="{
              'ring-2 ring-orange-400/60 border-orange-400/60': me?.role === 'BOSS',
              'opacity-60': me?.ready,
            }"
            :disabled="me?.ready"
            @click="changeRole('BOSS')"
          >
            <div class="text-orange-300 font-bold text-lg">老板</div>
            <p class="text-xs text-slate-400 mt-1">名额 1 · 初始 0 币 · 抽奖</p>
          </button>
        </div>
        <p class="text-xs text-slate-500 mt-3">
          切换角色会被服务端自动取消「准备」状态。
        </p>
      </div>

      <div class="card mt-4 p-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div class="text-sm text-slate-300">
          <div>当前状态：{{ me?.ready ? '✅ 已准备' : '⏳ 未准备' }}</div>
          <div class="text-xs text-slate-500 mt-1">{{ startHint }}</div>
        </div>
        <div class="flex flex-wrap gap-3 items-center">
          <button
            class="btn"
            :class="me?.ready ? 'bg-amber-500/80 text-white hover:bg-amber-500' : 'btn-primary'"
            @click="toggleReady"
          >
            {{ me?.ready ? '取消准备' : '准备' }}
          </button>
          <button
            v-if="isHost"
            class="btn-primary"
            :disabled="!startReady.ok"
            :title="startHint"
            @click="startGame"
          >
            开始游戏
          </button>
        </div>
      </div>
    </main>

    <ConfirmDialog
      :open="showLeaveConfirm"
      :title="isHost ? '解散房间？' : '退出房间？'"
      :message="
        isHost
          ? '你是房主，退出后房间会立即销毁，所有玩家将被踢回首页。'
          : '退出后将断开当前连接，可凭房间号重新加入。'
      "
      :confirm-text="isHost ? '解散' : '退出'"
      danger
      @confirm="confirmLeave"
      @cancel="showLeaveConfirm = false"
    />

    <ConfirmDialog
      :open="showHostClosedDialog"
      title="连接已断开"
      :message="`房间已关闭（${hostClosedReason || 'CLOSED'}），3 秒后自动返回首页。`"
      confirm-text="返回首页"
      cancel-text="稍候"
      @confirm="dismissHostClosed"
      @cancel="dismissHostClosed"
    />
  </div>
</template>

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
  if (startReady.value.ok) return '万事俱备，可开张迎客';
  switch (startReady.value.reason) {
    case 'NOT_FULL':
      return '尚有空位，等待入伙';
    case 'ROLE_INCOMPLETE':
      return '分位未齐（须 1 位老板 + 2 位护航）';
    case 'NOT_ALL_READY':
      return '尚有同行未签契约';
    default:
      return '尚未具备开张条件';
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
    roomStore.pushToast('success', '分号暗号已抄录');
  } catch {
    roomStore.pushToast('warn', '抄录失败');
  }
}

async function copyLink(): Promise<void> {
  try {
    const url = `${window.location.origin}/?code=${code.value}`;
    await navigator.clipboard.writeText(url);
    roomStore.pushToast('success', '邀请帖已抄录');
  } catch {
    roomStore.pushToast('warn', '抄录失败');
  }
}

function changeRole(role: 'BOSS' | 'ESCORT'): void {
  roomStore.sendIntent('CHANGE_ROLE', { role });
}

function toggleReady(): void {
  if (!me.value) return;
  if (!me.value.role) {
    roomStore.pushToast('warn', '请先认领分位');
    return;
  }
  roomStore.sendIntent('READY', { ready: !me.value.ready });
}

function startGame(): void {
  if (!isHost.value) return;
  if (!startReady.value.ok) {
    roomStore.pushToast('warn', `尚未开张：${startHint.value}`);
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
      <div class="rpg-title text-xl mt-2">敲门归店</div>
      <div class="font-body text-parchment-300/80 mt-2">正在重新对接花栗鼠分号…</div>
      <div class="mt-4 font-serif text-xs tracking-[0.32em] uppercase text-gold-300/70">
        暗号 · {{ code }}
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
            <span class="rune-mark mr-1">⚜</span>分号暗号
          </div>
          <div class="rpg-title text-2xl tracking-[.4em] font-display">{{ code }}</div>
        </div>
        <button class="btn-ghost" @click="copyCode">抄录暗号</button>
        <button class="btn-ghost" @click="copyLink">分发邀请帖</button>
      </div>
      <div class="flex items-center gap-3">
        <div
          class="rounded-md px-3 py-1.5 leading-tight text-right border border-gold-500/40 bg-ink-700/60"
          :title="`本店保底货量（仅供登记，不影响金币结算）`"
        >
          <div class="font-serif text-[10px] uppercase tracking-[0.3em] text-gold-300/80">
            保底货量
          </div>
          <div class="text-base font-bold font-serif text-gold-200">
            {{ roomStore.guaranteedAmount }}<span class="text-gold-300/80 ml-0.5">W</span>
          </div>
        </div>
        <div class="badge">
          <span class="rune-mark mr-1">🐿</span>{{ players.length }} / {{ MAX_PLAYERS }}
        </div>
        <button class="btn-ghost text-oxblood-300 hover:!text-oxblood-300" @click="showLeaveConfirm = true">
          告别本店
        </button>
      </div>
    </header>

    <main class="flex-1 px-4 sm:px-8 py-8 max-w-4xl mx-auto w-full">
      <div class="text-center mb-6">
        <h2 class="rpg-title text-2xl sm:text-3xl">店内议事 · 分位与契约</h2>
        <div class="mx-auto max-w-md mt-2 rpg-divider" />
        <p class="mt-2 font-body text-parchment-300/80 text-sm">
          一鼠坐镇神秘货柜，二鼠押镖出仓护航。
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
            <span class="rune-mark mr-1">⚜</span>认领分位
          </h3>
          <span class="font-body text-xs text-parchment-300/70">
            认下即立约，契约下不得擅改。
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
              名额 2 · 起始 10 金 · 自军械架挑选兵器
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
              <span class="text-2xl">👑</span>
              <div class="font-display tracking-[0.2em] text-ember-400 text-xl uppercase">
                老板
              </div>
            </div>
            <div class="rpg-divider mt-2 mb-2 opacity-70" />
            <p class="font-body text-xs text-parchment-300/85 leading-relaxed">
              名额 1 · 起始 0 金 · 执掌神秘货柜与开箱
            </p>
          </button>
        </div>
        <p class="font-body text-[12px] text-parchment-400/80 mt-4 text-center">
          一旦改换分位，原契约即作废（先前的「已立约」状态会被取消）。
        </p>
      </div>

      <div class="card rpg-corners mt-5 p-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div class="font-serif text-sm flex items-center gap-2">
            <template v-if="me?.ready">
              <span class="rune-mark text-forest-300 text-lg animate-pulse_soft">✓</span>
              <span class="text-forest-300 font-bold tracking-widest uppercase">契约已立</span>
            </template>
            <template v-else>
              <span class="rune-mark text-gold-300 text-lg animate-flicker">✦</span>
              <span class="text-gold-300 font-bold tracking-widest uppercase">尚未签约</span>
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
            {{ me?.ready ? '撤回契约' : '签下契约' }}
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
      :title="isHost ? '关张本店？' : '告别本店？'"
      :message="
        isHost
          ? '您是掌柜，离去之后本店随即关张，同行伙计将各奔前程。'
          : '告别之后契约暂歇，凭分号暗号仍可再度回店。'
      "
      :confirm-text="isHost ? '关张' : '告别'"
      danger
      @confirm="confirmLeave"
      @cancel="showLeaveConfirm = false"
    />

    <ConfirmDialog
      :open="showHostClosedDialog"
      title="本店已歇业"
      :message="`本店已歇业（${hostClosedReason || 'CLOSED'}），3 秒后自动返回门前。`"
      confirm-text="返回门前"
      cancel-text="稍候"
      @confirm="dismissHostClosed"
      @cancel="dismissHostClosed"
    />
  </div>
</template>

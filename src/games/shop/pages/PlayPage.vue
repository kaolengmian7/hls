<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useRoomStore } from '../stores/room';
import { ROLE_LABEL } from '../game/rules';
import Avatar from '@shared/components/Avatar.vue';
import LotteryPanel from '../components/LotteryPanel.vue';
import EquipmentShop from '../components/EquipmentShop.vue';
import FeedStream from '../components/FeedStream.vue';
import LotteryRecordModal from '../components/LotteryRecordModal.vue';
import SettlementModal from '../components/SettlementModal.vue';
import SettlementRecordModal from '../components/SettlementRecordModal.vue';
import ConfirmDialog from '@shared/components/ConfirmDialog.vue';
import ToastContainer from '../components/ToastContainer.vue';
import { usePlayerStore } from '@shared/stores/player';

const router = useRouter();
const route = useRoute();
const roomStore = useRoomStore();
const playerStore = usePlayerStore();

const showLottery = ref(false);
const showSettlement = ref(false);
const showSettlementHistory = ref(false);
const showLeaveConfirm = ref(false);
const showClosedDialog = ref(false);
const closedReason = ref('');
const restoring = ref(false);

/** 房主结算按钮：仅在 PLAYING 状态对房主显示。 */
const canSettle = computed(
  () => roomStore.isHost && roomStore.roomStatus === 'PLAYING',
);

const code = computed(() => roomStore.code || (route.params.code as string));
const me = computed(() => roomStore.me);

/**
 * 同公会在场名单：本人置顶，其余按服务端顺序。
 * 用于左侧"公会伙伴"栏统一展示三人账目。
 */
const roster = computed(() => {
  const players = roomStore.players;
  const self = players.find((p) => p.id === roomStore.selfPlayerId);
  const others = players.filter((p) => p.id !== roomStore.selfPlayerId);
  return self ? [self, ...others] : players;
});

onMounted(async () => {
  playerStore.ensureIdentity();
  roomStore.loadLotteryPool();
  const targetCode = (route.params.code as string) || roomStore.code;

  if (roomStore.status !== 'connected') {
    restoring.value = true;
    try {
      const ok = await roomStore.tryRestoreSession(targetCode);
      if (!ok) {
        router.replace({ name: 'home', query: { code: targetCode } });
        return;
      }
    } finally {
      restoring.value = false;
    }
  }

  // 已连接（或刚恢复连接），根据服务端权威状态决定停留还是回到准备页。
  if (roomStore.snapshot?.status !== 'PLAYING') {
    router.replace({ name: 'shop-room', params: { code: targetCode } });
  }
});

watch(
  () => roomStore.status,
  (status) => {
    // 恢复连接阶段失败不弹"连接已断开"对话框，由 onMounted 静默跳回首页。
    if (status === 'closed' && !restoring.value) {
      closedReason.value = roomStore.closedReason || 'CLOSED';
      showClosedDialog.value = true;
      setTimeout(() => {
        if (showClosedDialog.value) {
          showClosedDialog.value = false;
          router.replace({ name: 'home' });
        }
      }, 3000);
    }
  },
);

function handleCloseGame(): void {
  roomStore.sendIntent('CLOSE_GAME', {})
}

function confirmLeave(): void {
  showLeaveConfirm.value = false;
  roomStore.leaveRoom();
  router.replace({ name: 'home' });
}

function dismissClosed(): void {
  showClosedDialog.value = false;
  router.replace({ name: 'home' });
}
</script>

<template>
  <div class="flex-1 flex flex-col">
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

      <header class="px-4 sm:px-8 py-3 sticky top-0 z-10 backdrop-blur bg-ink-700/80 border-b border-gold-500/25">
        <div class="flex items-center justify-between gap-3 flex-wrap">
          <div class="flex items-center gap-3 min-w-0">
            <div class="rune-mark text-2xl animate-flicker">⚜</div>
            <div class="min-w-0">
              <div class="rpg-title text-base sm:text-lg leading-tight">花栗鼠的神秘商店</div>
              <div class="font-serif text-[10px] tracking-[0.3em] uppercase text-gold-300/70 mt-0.5">
                口令 · {{ code }}
              </div>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button class="btn-ghost text-oxblood-300 hover:!text-oxblood-300 text-[11px]" @click="showLeaveConfirm = true">
              离开
            </button>
          </div>
        </div>
      </header>

      <main class="flex-1 px-4 sm:px-8 py-6 grid grid-cols-1 lg:grid-cols-[220px_1fr_260px] gap-6 max-w-7xl mx-auto w-full">
        <aside class="order-2 lg:order-1">
          <div class="card rpg-corners p-4 lg:sticky lg:top-24 flex flex-col gap-4">

            <!-- 本局数据 -->
            <div>
              <div class="font-serif text-[10px] tracking-[0.2em] uppercase text-gold-300/60 mb-2">
                <span class="rune-mark mr-1">⚜</span>本局数据
              </div>
              <div class="grid grid-cols-2 gap-2">
                <div
                  class="rounded-md px-2.5 py-2 text-center border border-gold-500/40 bg-ink-700/60"
                  title="商店保底"
                >
                  <div class="font-serif text-[10px] uppercase tracking-[0.2em] text-gold-300/80">保底</div>
                  <div class="text-base font-bold font-serif text-gold-200 mt-0.5">
                    {{ roomStore.guaranteedAmount }}<span class="text-gold-300/80 ml-0.5 text-[11px]">W</span>
                  </div>
                </div>
                <div
                  class="rounded-md px-2.5 py-2 text-center border border-forest-400/40 bg-ink-700/60"
                  title="历次撤离成功累计扣减额"
                >
                  <div class="font-serif text-[10px] uppercase tracking-[0.2em] text-forest-300/70">得吃</div>
                  <div class="text-base font-bold font-serif text-forest-300 mt-0.5">
                    {{ roomStore.evacuatedTotal }}<span class="text-forest-300/80 ml-0.5 text-[11px]">W</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 操作 -->
            <div>
              <div class="font-serif text-[10px] tracking-[0.2em] uppercase text-gold-300/60 mb-2">
                🗂 操作
              </div>
              <div class="flex flex-wrap gap-2">
                <button
                  v-if="canSettle"
                  class="btn-boss py-1.5 px-3 text-[11px]"
                  @click="showSettlement = true"
                >
                  ⚖ 本轮结算
                </button>
                <button
                  v-if="canSettle"
                  class="btn-danger py-1.5 px-3 text-[11px]"
                  @click="handleCloseGame"
                >
                  结单
                </button>
                <button class="btn-ghost py-1.5 px-3 text-[11px]" @click="showSettlementHistory = true">结算记录</button>
                <button class="btn-ghost py-1.5 px-3 text-[11px]" @click="showLottery = true">抽奖记录</button>
              </div>
            </div>

            <div class="rpg-divider" />

            <!-- 店内成员 -->
            <div>
              <div class="font-serif text-[10px] tracking-[0.2em] uppercase text-gold-300/60 mb-2">
                👥 店内成员
              </div>
              <div class="grid grid-cols-3 gap-2 lg:grid-cols-1">
                <div
                  v-for="p in roster"
                  :key="p.id"
                  class="rounded-md p-2 border transition-colors flex flex-col lg:flex-row items-center gap-1.5 lg:gap-3 lg:p-2.5"
                  :class="
                    p.id === roomStore.selfPlayerId
                      ? 'border-gold-400/70 bg-gold-500/10'
                      : 'border-gold-500/25 bg-ink-700/60'
                  "
                >
                  <div class="relative shrink-0">
                    <Avatar :seed="p.id" :name="p.nickname" size="md" />
                    <span
                      v-if="!p.connected"
                      class="absolute -bottom-1 -right-1 w-3.5 h-3.5 rounded-full bg-oxblood-500 border border-ink-700"
                      title="失联中"
                    />
                  </div>
                  <div class="flex-1 min-w-0 w-full">
                    <div class="font-serif font-semibold text-parchment-50 flex flex-wrap items-center gap-1 justify-center lg:justify-start">
                      <span v-if="p.isHost" class="text-gold-300 shrink-0" title="房主">👑</span>
                      <span class="flex-1 min-w-0 truncate text-xs lg:text-sm lg:break-all lg:whitespace-normal">{{ p.nickname }}</span>
                      <span
                        v-if="p.id === roomStore.selfPlayerId"
                        class="badge !text-gold-200 !border-gold-400/60 text-[10px] shrink-0"
                      >
                        我
                      </span>
                    </div>
                    <div class="flex items-center gap-1.5 mt-1 justify-center lg:justify-start">
                      <span
                        class="badge text-[10px]"
                        :class="
                          p.role === 'BOSS'
                            ? '!border-ember-400/60 !text-ember-300'
                            : '!border-sapphire-300/60 !text-sapphire-300'
                        "
                      >
                        {{ p.role ? ROLE_LABEL[p.role] : '未择阶' }}
                      </span>
                    </div>
                    <div class="coin-tag text-xs mt-1 lg:text-sm">
                      <span>💰</span> {{ p.coin }}
                    </div>
                  </div>
                </div>
              </div>
            </div>

          </div>
        </aside>

        <section class="order-1 lg:order-2 space-y-5">
          <template v-if="me?.role === 'BOSS'">
            <LotteryPanel />
          </template>

          <template v-else-if="me?.role === 'ESCORT'">
            <EquipmentShop />
          </template>

          <template v-else>
            <div class="card rpg-corners p-10 text-center">
              <div class="rune-mark text-3xl animate-flicker">⚜</div>
              <div class="font-serif text-parchment-300 mt-2">
                正在确认您的店内角色…
              </div>
            </div>
          </template>
        </section>

        <aside class="order-3">
          <FeedStream />
        </aside>
      </main>

      <LotteryRecordModal :open="showLottery" @close="showLottery = false" />
      <SettlementModal :open="showSettlement" @close="showSettlement = false" />
      <SettlementRecordModal
        :open="showSettlementHistory"
        @close="showSettlementHistory = false"
      />

      <ConfirmDialog
        :open="showLeaveConfirm"
        :title="roomStore.isHost ? '关闭商店？' : '中途离开？'"
        :message="
          roomStore.isHost
            ? '您是房主，离开后这家神秘商店会立刻关闭。'
            : '中途离开后，本局记录不会继续跟随您，确认离开吗？'
        "
        :confirm-text="roomStore.isHost ? '关闭' : '离开'"
        danger
        @confirm="confirmLeave"
        @cancel="showLeaveConfirm = false"
      />

      <ConfirmDialog
        :open="showClosedDialog"
        :title="closedReason === 'CLOSED' ? '神秘商店已结单' : '商店已关闭'"
        :message="closedReason === 'CLOSED'
          ? '神秘商店已结单，3 秒后返回主页。'
          : `商店已关闭（${closedReason || 'CLOSED'}），3 秒后返回首页。`"
        confirm-text="返回主页"
        cancel-text="稍候"
        @confirm="dismissClosed"
        @cancel="dismissClosed"
      />
    </div>
  </div>
</template>

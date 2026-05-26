<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useRoomStore } from '@/stores/room';
import { ROLE_LABEL } from '@/game/rules';
import Avatar from '@/components/Avatar.vue';
import LotteryPanel from '@/components/LotteryPanel.vue';
import EquipmentShop from '@/components/EquipmentShop.vue';
import FeedStream from '@/components/FeedStream.vue';
import LotteryRecordModal from '@/components/LotteryRecordModal.vue';
import SettlementModal from '@/components/SettlementModal.vue';
import SettlementRecordModal from '@/components/SettlementRecordModal.vue';
import ConfirmDialog from '@/components/ConfirmDialog.vue';
import ToastContainer from '@/components/ToastContainer.vue';
import { usePlayerStore } from '@/stores/player';

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
const teammates = computed(() =>
  roomStore.players.filter((p) => p.id !== roomStore.selfPlayerId),
);

onMounted(async () => {
  playerStore.ensureIdentity();
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
    router.replace({ name: 'room', params: { code: targetCode } });
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
  <div v-if="restoring || !roomStore.snapshot" class="flex-1 flex items-center justify-center min-h-screen">
    <ToastContainer />
    <div class="card p-8 text-center text-slate-300">
      <div class="text-base font-semibold mb-2">正在恢复对局连接...</div>
      <div class="text-xs text-slate-500">房间号 {{ code }}</div>
    </div>
  </div>

  <div v-else class="flex-1 flex flex-col min-h-screen">
    <ToastContainer />

    <header class="px-4 sm:px-8 py-3 border-b border-white/5 sticky top-0 backdrop-blur bg-slate-950/70 z-10">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-4 min-w-0">
          <div class="flex items-center gap-2">
            <Avatar v-if="me" :seed="me.id" :name="me.nickname" size="md" />
            <div v-if="me" class="min-w-0">
              <div class="font-semibold text-slate-100 truncate flex items-center gap-1">
                <span v-if="me.isHost" class="text-amber-400">👑</span>
                {{ me.nickname }}
                <span
                  class="badge text-[10px]"
                  :class="
                    me.role === 'BOSS'
                      ? 'bg-orange-500/20 text-orange-300'
                      : 'bg-blue-500/20 text-blue-300'
                  "
                >
                  {{ me.role ? ROLE_LABEL[me.role] : '' }}
                </span>
              </div>
              <div class="text-amber-400 text-sm font-bold flex items-center gap-1">
                💰 {{ me.coin }}
              </div>
            </div>
          </div>

          <div class="hidden sm:flex items-center gap-2">
            <div
              v-for="t in teammates"
              :key="t.id"
              class="rounded-xl border border-white/10 bg-slate-900/40 px-3 py-1.5 flex items-center gap-2"
            >
              <Avatar :seed="t.id" :name="t.nickname" size="sm" />
              <div class="text-xs">
                <div class="text-slate-200 truncate max-w-[80px]">{{ t.nickname }}</div>
                <div class="text-slate-400 flex items-center gap-1">
                  <span
                    class="badge text-[10px]"
                    :class="
                      t.role === 'BOSS'
                        ? 'bg-orange-500/20 text-orange-300'
                        : 'bg-blue-500/20 text-blue-300'
                    "
                  >
                    {{ t.role ? ROLE_LABEL[t.role] : '' }}
                  </span>
                  <span class="text-amber-400">💰{{ t.coin }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <div
            class="rounded-xl border border-amber-400/20 bg-amber-500/10 px-2.5 py-1 leading-tight text-right"
            :title="`房间保底金额（仅展示，不影响金币结算）`"
          >
            <div class="text-[10px] uppercase tracking-wider text-amber-300/80">保底</div>
            <div class="text-sm font-semibold text-amber-200 font-mono">
              {{ roomStore.guaranteedAmount }}<span class="text-amber-300/70 ml-0.5">W</span>
            </div>
          </div>
          <button
            v-if="canSettle"
            class="btn-primary py-1.5 px-3 text-xs"
            @click="showSettlement = true"
          >
            本局结算
          </button>
          <button class="btn-ghost py-1.5 text-xs" @click="showSettlementHistory = true">
            结算记录
          </button>
          <button class="btn-ghost py-1.5 text-xs" @click="showLottery = true">抽奖记录</button>
          <button class="btn-ghost text-rose-400 py-1.5 text-xs" @click="showLeaveConfirm = true">
            退出
          </button>
        </div>
      </div>
    </header>

    <main class="flex-1 px-4 sm:px-8 py-6 grid grid-cols-1 lg:grid-cols-[1fr_360px] gap-6 max-w-6xl mx-auto w-full">
      <section class="space-y-5">
        <template v-if="me?.role === 'BOSS'">
          <LotteryPanel />
          <div class="card p-5 text-sm text-slate-400">
            <h3 class="text-base font-semibold text-slate-100 mb-2">老板的策略提示</h3>
            <ul class="list-disc pl-4 space-y-1">
              <li>初始 0 币，等待后续机制（时间奖励 / 任务结算）注入收益。</li>
              <li>抽奖结果由服务端裁决（crypto/rand），可在「抽奖记录」中查看全员历史。</li>
              <li>稀有度越高的物品在公屏播报中越亮眼，可作为博弈筹码。</li>
            </ul>
          </div>
        </template>

        <template v-else-if="me?.role === 'ESCORT'">
          <EquipmentShop />
        </template>

        <template v-else>
          <div class="card p-8 text-center text-slate-400">
            正在等待角色信息同步...
          </div>
        </template>
      </section>

      <aside>
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
      :title="roomStore.isHost ? '解散对局？' : '退出对局？'"
      :message="
        roomStore.isHost
          ? '你是房主，退出后房间立即销毁，所有玩家将回到首页。'
          : '当前对局进度不会保留，确认要退出吗？'
      "
      :confirm-text="roomStore.isHost ? '解散' : '退出'"
      danger
      @confirm="confirmLeave"
      @cancel="showLeaveConfirm = false"
    />

    <ConfirmDialog
      :open="showClosedDialog"
      title="连接已断开"
      :message="`房间已关闭（${closedReason || 'CLOSED'}），3 秒后自动返回首页。`"
      confirm-text="返回首页"
      cancel-text="稍候"
      @confirm="dismissClosed"
      @cancel="dismissClosed"
    />
  </div>
</template>

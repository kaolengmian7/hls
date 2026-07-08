/**
 * 房间 Store：薄客户端。
 *
 * - 维护服务端下发的最新 RoomSnapshot 与本地连接状态。
 * - 暴露 sendIntent 给 UI；不再做任何业务裁决。
 * - 维护 toast 队列、最近一次拒绝原因。
 */

import { defineStore } from 'pinia';
import { computed, ref, shallowRef } from 'vue';
import { GameClient } from '../net/client';
import type {
  EventMap,
  EventType,
  IntentMap,
  IntentType,
  LotteryItem,
  LotteryRecord,
  Player,
  RoomSnapshot,
} from '../net/protocol';
import { fetchLotteryPool } from '../net/api';
import { useAuthStore } from '@/stores/auth';

export type Toast = {
  id: string;
  type: 'info' | 'success' | 'error' | 'warn';
  message: string;
};

export type ConnStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed';

let toastSeq = 0;

export const useRoomStore = defineStore('room', () => {
  const snapshot = shallowRef<RoomSnapshot | null>(null);
  const lotteryPool = ref<LotteryItem[]>([]);
  const latestLotteryReveal = shallowRef<LotteryRecord[]>([]);
  /**
   * 老板本人抽奖时被"延迟播报"的 feed 事件 id 集合。
   *
   * 抽奖框卷轴动画需约 3s 才揭晓结果，而小报童（feed）来自即时 STATE_SNAPSHOT，
   * 若不处理，老板会在卷轴停稳前就从右侧小报童看到自己的结果（剧透）。
   * 这里把"自己刚抽出"的 LOTTERY feed 暂时藏起，等动画揭晓后由 LotteryPanel
   * 调用 releaseSelfLotteryReveal 放出。仅影响老板本机展示，不改变服务端数据，
   * 其他玩家照常实时看到播报。
   */
  const heldLotteryFeedIds = ref<Set<string>>(new Set<string>());
  // 标记"刚收到自己的 LOTTERY_DRAWN，正等待下一个 STATE_SNAPSHOT 去识别并暂存对应 feed"。
  let awaitingSelfLotteryReveal = false;
  const selfPlayerId = ref<string>('');
  const code = ref<string>('');
  const status = ref<ConnStatus>('idle');
  const closedReason = ref<string>('');
  const gameClosed = ref(false);
  const toasts = ref<Toast[]>([]);

  let client: GameClient | null = null;

  /* ------------------------------ Toast 工具 ------------------------------ */

  function pushToast(type: Toast['type'], message: string, duration = 2400): void {
    toastSeq += 1;
    const t: Toast = { id: String(toastSeq), type, message };
    toasts.value = [...toasts.value, t];
    setTimeout(() => {
      toasts.value = toasts.value.filter((x) => x.id !== t.id);
    }, duration);
  }

  /* ------------------------------ 连接控制 ------------------------------ */

  function connect(input: {
    code: string;
    nickname: string;
    avatar?: string;
  }): Promise<void> {
    teardownClient();
    code.value = input.code;
    const auth = useAuthStore();
    selfPlayerId.value = String(auth.user?.id ?? '');
    status.value = 'connecting';
    closedReason.value = '';

    return new Promise<void>((resolve, reject) => {
      let settled = false;
      client = new GameClient({
        onOpen: () => {
          // WS 物理打开；JOIN_ACCEPTED 才算业务层重连完成，故此处不切 connected。
        },
        onAccepted: (payload) => {
          snapshot.value = payload.snapshot;
          latestLotteryReveal.value = [];
          awaitingSelfLotteryReveal = false;
          heldLotteryFeedIds.value = new Set<string>();
          selfPlayerId.value = payload.selfPlayerId;
          const wasReconnecting = status.value === 'reconnecting';
          status.value = 'connected';
          closedReason.value = '';
          if (wasReconnecting) {
            pushToast('success', '已重新回到神秘商店');
          }
          if (!settled) {
            settled = true;
            resolve();
          }
        },
        onRejected: (reason, message) => {
          status.value = 'closed';
          closedReason.value = message ?? reason;
          if (!settled) {
            settled = true;
            reject(new Error(reason));
          }
        },
        onEvent: (type, payload, reqId) => {
          handleEvent(type, payload, reqId);
        },
        onReconnecting: () => {
          // 底层会自动重连：切到"重连中"，UI 灰显但保留快照。
          if (status.value === 'connected' || status.value === 'connecting') {
            status.value = 'reconnecting';
          }
        },
        onClosed: (reason) => {
          status.value = 'closed';
          closedReason.value = reason;
          if (!settled) {
            settled = true;
            reject(new Error(reason));
          }
        },
      });
      client.connect(input);
    });
  }

  function sendIntent<T extends IntentType>(type: T, payload: IntentMap[T]): string {
    if (!client) return '';
    return client.send(type, payload);
  }

  async function loadLotteryPool(): Promise<void> {
    if (lotteryPool.value.length > 0) return;
    try {
      lotteryPool.value = await fetchLotteryPool();
    } catch {
      // 装饰用，失败静默；LotteryPanel 回退用历史记录
    }
  }

  /**
   * 利用已登录用户身份尝试恢复对房间的连接。
   *
   * 触发场景：用户在 /room/:code 或 /room/:code/play 路径下刷新页面，
   * Pinia 状态被清空，服务端通过 rm.HasPlayer(playerID) 判断是否为重连。
   *
   * 行为：
   * - 已处于 connected/connecting/reconnecting：直接返回 true，避免重复建链。
   * - 未登录（auth.user 为空）：返回 false（调用方应跳回首页）。
   * - 否则按正常连接流程建链；成功 resolve true，失败（JOIN_REJECTED / 网络终态）resolve false。
   */
  async function tryRestoreSession(targetCode: string): Promise<boolean> {
    if (status.value === 'connected' || status.value === 'connecting' || status.value === 'reconnecting') {
      return true;
    }
    const auth = useAuthStore();
    if (!auth.user) {
      return false;
    }
    try {
      await connect({
        code: targetCode,
        nickname: auth.user.nickname,
      });
      return true;
    } catch {
      return false;
    }
  }

  function leaveRoom(): void {
    teardownClient(); // destroy() inside sends LEAVE automatically
    snapshot.value = null;
    latestLotteryReveal.value = [];
    awaitingSelfLotteryReveal = false;
    heldLotteryFeedIds.value = new Set<string>();
    code.value = '';
    selfPlayerId.value = '';
    status.value = 'idle';
    closedReason.value = '';
    gameClosed.value = false;
  }

  function teardownClient(): void {
    if (client) {
      client.destroy('STORE_TEARDOWN');
      client = null;
    }
  }

  /* ------------------------------ 服务端事件 ------------------------------ */

  function handleEvent<T extends EventType>(
    type: T,
    payload: EventMap[T],
    _reqId?: string,
  ): void {
    switch (type) {
      case 'STATE_SNAPSHOT':
      case 'GAME_STARTED': {
        const p = payload as EventMap['STATE_SNAPSHOT'] | EventMap['GAME_STARTED'];
        // 紧跟自己 LOTTERY_DRAWN 之后到达的快照：把本次新增的、属于自己的
        // LOTTERY feed 条目暂存起来，待抽奖框动画揭晓后再放出（见 heldLotteryFeedIds）。
        if (awaitingSelfLotteryReveal) {
          const prevIds = new Set((snapshot.value?.feed ?? []).map((f) => f.id));
          const held = new Set(heldLotteryFeedIds.value);
          for (const f of p.snapshot.feed) {
            if (f.kind === 'LOTTERY' && f.payload?.playerId === selfPlayerId.value && !prevIds.has(f.id)) {
              held.add(f.id);
            }
          }
          heldLotteryFeedIds.value = held;
          awaitingSelfLotteryReveal = false;
        }
        snapshot.value = p.snapshot;
        break;
      }
      case 'ACTION_REJECTED': {
        const p = payload as EventMap['ACTION_REJECTED'];
        pushToast('warn', p.message ?? `操作未通过：${p.reason}`);
        break;
      }
      case 'HOST_CHANGED': {
        const p = payload as EventMap['HOST_CHANGED'];
        if (snapshot.value) {
          const next = snapshot.value.players.map((pl) => ({
            ...pl,
            isHost: pl.id === p.newHostId,
          }));
          snapshot.value = { ...snapshot.value, hostId: p.newHostId, players: next };
        }
        pushToast('info', '房主已经变更');
        break;
      }
      case 'LOTTERY_DRAWN': {
        const p = payload as EventMap['LOTTERY_DRAWN'];
        const isSelf = p.records.length > 0 && p.records[0]!.playerId === selfPlayerId.value;
        latestLotteryReveal.value = isSelf ? p.records : [];
        // 自己的抽奖：随后到达的快照里对应 feed 需延迟到动画揭晓后再播报。
        if (isSelf) awaitingSelfLotteryReveal = true;
        break;
      }
      // 其余 PLAYER_JOINED / PLAYER_LEFT / ROLE_CHANGED / READY_CHANGED /
      // COIN_CHANGED / EQUIPMENT_PURCHASED 都由 STATE_SNAPSHOT 覆盖；
      // 这里不做增量合并，依赖服务端紧随其后的 STATE_SNAPSHOT 对账。
      case 'GAME_CLOSED':
        gameClosed.value = true;
        break;
      default:
        break;
    }
  }

  /* ------------------------------ 计算属性 ------------------------------ */

  const players = computed<Player[]>(() => snapshot.value?.players ?? []);
  const me = computed<Player | null>(() => {
    return players.value.find((p) => p.id === selfPlayerId.value) ?? null;
  });
  const isHost = computed(() => me.value?.isHost ?? false);
  const lotteryRecords = computed(() => snapshot.value?.lotteryRecords ?? []);
  const settlementRecords = computed(() => snapshot.value?.settlementRecords ?? []);
  const feed = computed(() => {
    const all = snapshot.value?.feed ?? [];
    if (heldLotteryFeedIds.value.size === 0) return all;
    return all.filter((f) => !heldLotteryFeedIds.value.has(f.id));
  });

  /**
   * 老板抽奖框动画揭晓后调用：放出此前被暂存的自抽 LOTTERY feed，使其出现在小报童。
   * 由 LotteryPanel 在 phase 进入 revealed 或组件卸载时触发。
   */
  function releaseSelfLotteryReveal(): void {
    awaitingSelfLotteryReveal = false;
    if (heldLotteryFeedIds.value.size > 0) {
      heldLotteryFeedIds.value = new Set<string>();
    }
  }
  const roomStatus = computed(() => snapshot.value?.status ?? 'WAITING');
  /**
   * 房间保底金额（单位 W）。来自服务端权威快照，3 名玩家看到的值完全一致。
   *
   * 注意：与 player.coin 完全独立，模板中不可与金币合并展示。
   */
  const guaranteedAmount = computed(() => snapshot.value?.guaranteedAmount ?? 0);

  /**
   * 累计"得吃"金额（单位 W）：历次"撤离成功"（EVACUATED）结算时房主输入的扣减额之和。
   *
   * 取自服务端权威快照里的 settlementRecords，前端仅做求和聚合，
   * 不做其它二次计算，保证 3 名玩家看到的累计值完全一致。
   */
  const evacuatedTotal = computed(() =>
    (snapshot.value?.settlementRecords ?? []).reduce(
      (sum, r) => (r.outcome === 'EVACUATED' ? sum + r.guaranteeDelta : sum),
      0,
    ),
  );

  return {
    snapshot,
    selfPlayerId,
    code,
    status,
    closedReason,
    gameClosed,
    toasts,
    players,
    me,
    isHost,
    lotteryRecords,
    latestLotteryReveal,
    settlementRecords,
    feed,
    roomStatus,
    guaranteedAmount,
    evacuatedTotal,
    lotteryPool,
    connect,
    sendIntent,
    leaveRoom,
    pushToast,
    tryRestoreSession,
    releaseSelfLotteryReveal,
    loadLotteryPool,
  };
});

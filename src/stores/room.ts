/**
 * 房间 Store：薄客户端。
 *
 * - 维护服务端下发的最新 RoomSnapshot 与本地连接状态。
 * - 暴露 sendIntent 给 UI；不再做任何业务裁决。
 * - 维护 toast 队列、最近一次拒绝原因。
 */

import { defineStore } from 'pinia';
import { computed, ref, shallowRef } from 'vue';
import { GameClient } from '@/net/client';
import type {
  EventMap,
  EventType,
  IntentMap,
  IntentType,
  Player,
  RoomSnapshot,
} from '@/net/protocol';
import { usePlayerStore } from './player';

export type Toast = {
  id: string;
  type: 'info' | 'success' | 'error' | 'warn';
  message: string;
};

export type ConnStatus = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed';

let toastSeq = 0;

export const useRoomStore = defineStore('room', () => {
  const snapshot = shallowRef<RoomSnapshot | null>(null);
  const selfPlayerId = ref<string>('');
  const code = ref<string>('');
  const status = ref<ConnStatus>('idle');
  const closedReason = ref<string>('');
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
    playerId: string;
    nickname: string;
    avatar?: string;
    sessionToken?: string;
  }): Promise<void> {
    teardownClient();
    code.value = input.code;
    selfPlayerId.value = input.playerId;
    status.value = 'connecting';
    closedReason.value = '';

    const playerStore = usePlayerStore();

    return new Promise<void>((resolve, reject) => {
      let settled = false;
      client = new GameClient({
        onOpen: () => {
          // WS 物理打开；JOIN_ACCEPTED 才算业务层重连完成，故此处不切 connected。
        },
        onAccepted: (payload) => {
          snapshot.value = payload.snapshot;
          selfPlayerId.value = payload.selfPlayerId;
          const wasReconnecting = status.value === 'reconnecting';
          status.value = 'connected';
          closedReason.value = '';
          playerStore.setSessionToken(input.code, payload.sessionToken);
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

  /**
   * 利用本地持久化的 sessionToken / playerId 尝试恢复对房间的连接。
   *
   * 触发场景：用户在 /room/:code 或 /room/:code/play 路径下刷新页面，
   * Pinia 状态被清空，但 localStorage 中仍持有最近一次签发的 sessionToken。
   *
   * 行为：
   * - 已处于 connected/connecting/reconnecting：直接返回 true，避免重复建链。
   * - 无 playerId 或目标 code 无对应 token：返回 false（调用方应跳回首页）。
   * - 否则按 reconnect 流程建链；成功 resolve true，失败（JOIN_REJECTED / 网络终态）resolve false。
   */
  async function tryRestoreSession(targetCode: string): Promise<boolean> {
    if (status.value === 'connected' || status.value === 'connecting' || status.value === 'reconnecting') {
      return true;
    }
    const playerStore = usePlayerStore();
    playerStore.ensureIdentity();
    const playerId = playerStore.playerId;
    const token = playerStore.getSessionToken(targetCode);
    if (!playerId || !token) {
      return false;
    }
    try {
      await connect({
        code: targetCode,
        playerId,
        nickname: playerStore.nickname,
        sessionToken: token,
      });
      return true;
    } catch {
      // 清理失效的 session，避免下一次刷新仍卡在重连失败。
      playerStore.clearSession();
      return false;
    }
  }

  function leaveRoom(): void {
    const playerStore = usePlayerStore();
    if (snapshot.value && me.value?.isHost) {
      // 房主：通知服务端解散
      client?.send('DISMISS_ROOM', {});
    } else {
      client?.send('LEAVE', {});
    }
    teardownClient();
    playerStore.clearSession();
    snapshot.value = null;
    code.value = '';
    selfPlayerId.value = '';
    status.value = 'idle';
    closedReason.value = '';
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
      // 其余 PLAYER_JOINED / PLAYER_LEFT / ROLE_CHANGED / READY_CHANGED /
      // COIN_CHANGED / EQUIPMENT_PURCHASED / LOTTERY_DRAWN 都由 STATE_SNAPSHOT 覆盖；
      // 这里不做增量合并，依赖服务端紧随其后的 STATE_SNAPSHOT 对账。
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
  const feed = computed(() => snapshot.value?.feed ?? []);
  const roomStatus = computed(() => snapshot.value?.status ?? 'WAITING');
  /**
   * 房间保底金额（单位 W）。来自服务端权威快照，3 名玩家看到的值完全一致。
   *
   * 注意：与 player.coin 完全独立，模板中不可与金币合并展示。
   */
  const guaranteedAmount = computed(() => snapshot.value?.guaranteedAmount ?? 0);

  return {
    snapshot,
    selfPlayerId,
    code,
    status,
    closedReason,
    toasts,
    players,
    me,
    isHost,
    lotteryRecords,
    settlementRecords,
    feed,
    roomStatus,
    guaranteedAmount,
    connect,
    sendIntent,
    leaveRoom,
    pushToast,
    tryRestoreSession,
  };
});

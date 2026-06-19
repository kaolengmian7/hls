/**
 * GameClient：游戏对局期间唯一的网络客户端。
 *
 * 职责：
 *   - 建立 WebSocket（包含自动重连）
 *   - 发送 Intent，自动注入 reqId / ts
 *   - 接收 Event 后回调上层
 *   - 在重连成功后服务端会主动下发 STATE_SNAPSHOT 对账
 */

import { ReconnectingWebSocket } from './ws';
import { wsBase } from './config';
import {
  buildEnvelope,
  type EventEnvelope,
  type EventMap,
  type EventType,
  type IntentMap,
  type IntentType,
} from './protocol';
import { shortId } from '@shared/utils/id';

export type ClientCallbacks = {
  /** 收到 JOIN_ACCEPTED 时触发；snapshot 已就绪。 */
  onAccepted: (snapshot: EventMap['JOIN_ACCEPTED']) => void;
  /** 收到 JOIN_REJECTED 或 KICK 时触发；之后连接已关闭。 */
  onRejected: (reason: string, message?: string) => void;
  /** 所有 server → client 事件（除 JOIN_ACCEPTED/JOIN_REJECTED 外）的回调。 */
  onEvent: <T extends EventType>(type: T, payload: EventMap[T], reqId?: string) => void;
  /** WebSocket 完全关闭（已不再重连）时触发。 */
  onClosed: (reason: string) => void;
  /** 网络打开/重连成功时触发，用于上层 UI 状态展示。 */
  onOpen?: () => void;
  /** 连接断开但底层会自动重连时触发；用于切到"重连中" UI。 */
  onReconnecting?: (reason: string) => void;
};

export type ConnectInput = {
  code: string;
  playerId: string;
  /** 已存在时（来自 createRoom 或上次会话）直接走 reconnect 流程，无需再发 JOIN。 */
  sessionToken?: string;
  /** 仅首次 JOIN 时使用。 */
  nickname: string;
  avatar?: string;
};

const JOIN_ACK_TIMEOUT_MS = 8_000;

export class GameClient {
  private ws: ReconnectingWebSocket | null = null;
  private callbacks: ClientCallbacks;
  private input: ConnectInput | null = null;
  private destroyed = false;
  private joinTimer: ReturnType<typeof setTimeout> | null = null;
  private currentSessionToken = '';
  private hasJoined = false;

  constructor(callbacks: ClientCallbacks) {
    this.callbacks = callbacks;
  }

  /** 启动连接：建立 WebSocket，等待 JOIN_ACCEPTED。 */
  connect(input: ConnectInput): void {
    this.input = input;
    this.currentSessionToken = input.sessionToken ?? '';

    const params = new URLSearchParams({ playerId: input.playerId });
    if (this.currentSessionToken) {
      params.set('token', this.currentSessionToken);
    }
    const url = `${wsBase()}/ws/rooms/${input.code}?${params.toString()}`;

    this.ws = new ReconnectingWebSocket(
      { url },
      {
        onOpen: () => this.handleOpen(),
        onMessage: (env) => this.dispatch(env as EventEnvelope),
        onClose: (reason, isFinal) => this.handleClose(reason, isFinal),
        onError: () => {},
      },
    );
    this.ws.connect();
    this.armJoinTimer();
  }

  /** 主动断开。 */
  destroy(reason = 'CLIENT_DESTROYED'): void {
    if (this.destroyed) return;
    this.destroyed = true;
    if (this.joinTimer) {
      clearTimeout(this.joinTimer);
      this.joinTimer = null;
    }
    // 通知服务端主动离开（如果连接还在）
    try {
      this.send('LEAVE', {});
    } catch {
      // ignore
    }
    this.ws?.close(reason);
    this.ws = null;
  }

  /** 发送一个 Intent 并返回 reqId（便于上层做对账）。 */
  send<T extends IntentType>(type: T, payload: IntentMap[T]): string {
    const reqId = shortId();
    if (!this.ws) return reqId;
    const env = buildEnvelope(type, payload, reqId);
    this.ws.send(env);
    return reqId;
  }

  /** 当前 sessionToken；用于持久化以便页面刷新后断线重连。 */
  get sessionToken(): string {
    return this.currentSessionToken;
  }

  /* ------------------------------ 内部 ------------------------------ */

  private handleOpen(): void {
    this.callbacks.onOpen?.();
    if (this.hasJoined || !this.input) return;
    if (this.currentSessionToken) {
      // reconnect：服务端校验 token 后直接发 JOIN_ACCEPTED + STATE_SNAPSHOT，无需 JOIN。
      return;
    }
    this.send('JOIN', { nickname: this.input.nickname, avatar: this.input.avatar });
  }

  private handleClose(reason: string, isFinal: boolean): void {
    if (isFinal) {
      if (this.joinTimer) {
        clearTimeout(this.joinTimer);
        this.joinTimer = null;
      }
      this.callbacks.onClosed(reason);
      return;
    }
    // 非终态：底层会自动重连，让上层切到"重连中"展示。
    this.callbacks.onReconnecting?.(reason);
  }

  private armJoinTimer(): void {
    this.joinTimer = setTimeout(() => {
      this.joinTimer = null;
      if (!this.hasJoined && !this.destroyed) {
        this.callbacks.onRejected('TIMEOUT', '加入超时');
        this.destroy('JOIN_TIMEOUT');
      }
    }, JOIN_ACK_TIMEOUT_MS);
  }

  private dispatch(env: EventEnvelope): void {
    if (env.type === 'JOIN_ACCEPTED') {
      if (this.joinTimer) {
        clearTimeout(this.joinTimer);
        this.joinTimer = null;
      }
      const payload = env.payload as EventMap['JOIN_ACCEPTED'];
      this.hasJoined = true;
      this.currentSessionToken = payload.sessionToken || this.currentSessionToken;
      this.callbacks.onAccepted(payload);
      return;
    }
    if (env.type === 'JOIN_REJECTED') {
      const payload = env.payload as EventMap['JOIN_REJECTED'];
      this.callbacks.onRejected(payload.reason, payload.message);
      this.destroy('JOIN_REJECTED');
      return;
    }
    if (env.type === 'KICK') {
      const payload = env.payload as EventMap['KICK'];
      this.callbacks.onRejected(payload.reason, '已被服务端断开');
      this.destroy('KICK');
      return;
    }
    this.callbacks.onEvent(
      env.type as EventType,
      env.payload as EventMap[EventType],
      env.reqId,
    );
  }
}

/**
 * 浏览器 WebSocket 的薄封装：
 *   - 自动重连（指数退避；可见性触发可立即重连）
 *   - 心跳 PING / 服务端 PONG / 静默检测
 *   - 页面隐藏时完全停心跳；回到前台立刻探活，必要时立即重连
 *   - JSON 文本帧编解码
 *
 * 设计意图：业务层只关心
 *   - send(envelope) 发消息
 *   - onMessage(env)/onOpen()/onClose(reason)
 * 内部对网络抖动 / 设备休眠做尽量透明的处理。
 */

import type { Envelope } from './protocol';

export type WsBuilderInput = {
  url: string;
  protocols?: string | string[];
};

export type WsCallbacks = {
  onOpen?: () => void;
  onMessage?: (env: Envelope) => void;
  /** 收到 close 帧或重连失败终结后触发；reason 用于上层 toast。
   *  isFinal=false 表示底层将自动重连，业务侧可切到"重连中"展示。 */
  onClose?: (reason: string, isFinal: boolean) => void;
  /** 仅诊断；非致命错误。 */
  onError?: (err: Event | Error) => void;
};

const HEARTBEAT_INTERVAL = 30_000;
/** 仅在页面可见时生效的静默超时；隐藏期间完全不计算。 */
const HEARTBEAT_TIMEOUT = 90_000;
/** 重连次数上限放宽以匹配"小时级使用 + 频繁切后台"的场景。 */
const MAX_RECONNECT = 50;
const BACKOFF_BASE_MS = 600;
const BACKOFF_CAP_MS = 30_000;
/** 用于 close() 的内部 reconnect 信号，不会出现在 onclose 的 code 上。 */
const SELF_CLOSE_CODE_RECONNECT = 4900;

export class ReconnectingWebSocket {
  private url: string;
  private protocols?: string | string[];
  private callbacks: WsCallbacks;

  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private heartbeatDeadlineTimer: ReturnType<typeof setTimeout> | null = null;
  private destroyed = false;
  private hasOpened = false;
  private lastPongAt = 0;
  /** 当前 close() 是否由本地心跳/可见性主动触发；用于在 onclose 中绕过"4xxx 当作终态"规则。 */
  private selfClosingForReconnect = false;
  /** 已注册到 document 上的 visibilitychange 回调，便于 destroy 时清理。 */
  private visibilityHandler: (() => void) | null = null;

  constructor(input: WsBuilderInput, callbacks: WsCallbacks) {
    this.url = input.url;
    this.protocols = input.protocols;
    this.callbacks = callbacks;
    this.bindVisibility();
  }

  /** 主动建立连接（仅首次调用；后续重连自动）。 */
  connect(): void {
    if (this.destroyed) return;
    this.openInternal();
  }

  /** 主动断开并阻止后续自动重连。 */
  close(reason = 'CLIENT_CLOSE'): void {
    this.destroyed = true;
    this.clearTimers();
    this.unbindVisibility();
    if (this.ws) {
      try {
        this.ws.close(1000, reason);
      } catch {
        // ignore
      }
      this.ws = null;
    }
  }

  /** 发送一个完整的 Envelope，不抛错；底层未连接时静默丢弃，业务侧做乐观重发。 */
  send(env: Envelope): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return false;
    try {
      this.ws.send(JSON.stringify(env));
      return true;
    } catch {
      return false;
    }
  }

  /* ------------------------------ 内部 ------------------------------ */

  private openInternal(): void {
    try {
      this.ws = new WebSocket(this.url, this.protocols);
    } catch (err) {
      this.handleError(err as Error);
      this.scheduleReconnect();
      return;
    }
    this.bindEvents(this.ws);
  }

  private bindEvents(ws: WebSocket): void {
    ws.onopen = () => {
      this.hasOpened = true;
      this.reconnectAttempts = 0;
      this.lastPongAt = Date.now();
      this.startHeartbeat();
      this.callbacks.onOpen?.();
    };
    ws.onmessage = (ev) => {
      this.lastPongAt = Date.now();
      let env: Envelope | null = null;
      try {
        env = JSON.parse(typeof ev.data === 'string' ? ev.data : '') as Envelope;
      } catch {
        return;
      }
      if (!env || typeof env.type !== 'string') return;
      this.callbacks.onMessage?.(env);
    };
    ws.onerror = (ev) => {
      this.callbacks.onError?.(ev);
    };
    ws.onclose = (ev) => {
      this.clearHeartbeat();
      this.ws = null;
      const reason = ev.reason || `code_${ev.code}`;
      if (this.destroyed) {
        this.callbacks.onClose?.(reason, true);
        return;
      }
      // 本地主动关掉用于重连：跳过"4xxx 即终态"判定，立即排程重连。
      if (this.selfClosingForReconnect) {
        this.selfClosingForReconnect = false;
        this.scheduleReconnect(true);
        this.callbacks.onClose?.(reason, false);
        return;
      }
      // 服务端主动以 1000/4xxx 关闭时通常代表 KICK / 拒入，不再重连。
      if (ev.code === 1000 || (ev.code >= 4000 && ev.code < 5000)) {
        this.destroyed = true;
        this.callbacks.onClose?.(reason, true);
        return;
      }
      if (!this.hasOpened || this.reconnectAttempts >= MAX_RECONNECT) {
        this.destroyed = true;
        this.callbacks.onClose?.(reason, true);
        return;
      }
      this.scheduleReconnect();
      this.callbacks.onClose?.(reason, false);
    };
  }

  /** 排程重连；immediate=true 时跳过退避立刻尝试。 */
  private scheduleReconnect(immediate = false): void {
    if (this.destroyed) return;
    if (this.reconnectAttempts >= MAX_RECONNECT) {
      this.destroyed = true;
      this.callbacks.onClose?.('MAX_RECONNECT', true);
      return;
    }
    const delay = immediate
      ? 0
      : Math.min(
          BACKOFF_CAP_MS,
          BACKOFF_BASE_MS * Math.pow(2, this.reconnectAttempts),
        );
    this.reconnectAttempts += 1;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.openInternal();
    }, delay);
  }

  private startHeartbeat(): void {
    this.clearHeartbeat();
    if (isDocumentHidden()) return; // 隐藏期间完全不跑心跳
    this.heartbeatTimer = setInterval(() => {
      this.heartbeatTick();
    }, HEARTBEAT_INTERVAL);
  }

  private heartbeatTick(): void {
    if (this.destroyed) return;
    if (isDocumentHidden()) return; // 双保险：隐藏期间不发不判
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.send({ type: 'PING', payload: {}, ts: Date.now() });
    if (Date.now() - this.lastPongAt > HEARTBEAT_TIMEOUT) {
      this.forceReconnect('HEARTBEAT_TIMEOUT');
    }
  }

  /** 主动断开当前连接并立即触发重连（不当作终态）。 */
  private forceReconnect(reason: string): void {
    if (this.destroyed) return;
    if (!this.ws) {
      this.scheduleReconnect(true);
      return;
    }
    this.selfClosingForReconnect = true;
    try {
      this.ws.close(SELF_CLOSE_CODE_RECONNECT, reason);
    } catch {
      // ignore
    }
  }

  /* ------------------------------ 可见性 ------------------------------ */

  private bindVisibility(): void {
    if (typeof document === 'undefined') return;
    this.visibilityHandler = () => this.handleVisibilityChange();
    document.addEventListener('visibilitychange', this.visibilityHandler);
  }

  private unbindVisibility(): void {
    if (typeof document === 'undefined' || !this.visibilityHandler) return;
    document.removeEventListener('visibilitychange', this.visibilityHandler);
    this.visibilityHandler = null;
  }

  private handleVisibilityChange(): void {
    if (this.destroyed) return;
    if (isDocumentHidden()) {
      // 完全停心跳，但保留连接 / 重连排程不动。
      this.clearHeartbeat();
      return;
    }
    // 回到前台：立刻探活；不行就立刻重连。
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.lastPongAt = Date.now();
      this.startHeartbeat();
      // 触发一次主动 PING，让服务端知道我们回来了；
      // 服务端任意回包都会刷新 lastPongAt，从而对账完成。
      this.send({ type: 'PING', payload: {}, ts: Date.now() });
      return;
    }
    if (this.ws && this.ws.readyState === WebSocket.CONNECTING) {
      // 已经在握手，等它走完。
      return;
    }
    // ws 已不在或 CLOSING：取消退避立刻重连。
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (!this.ws) {
      this.openInternal();
    }
  }

  private clearHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    if (this.heartbeatDeadlineTimer) {
      clearTimeout(this.heartbeatDeadlineTimer);
      this.heartbeatDeadlineTimer = null;
    }
  }

  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.clearHeartbeat();
  }

  private handleError(err: Error): void {
    this.callbacks.onError?.(err);
  }
}

function isDocumentHidden(): boolean {
  return typeof document !== 'undefined' && document.hidden === true;
}

/**
 * REST API 封装：负责创建房间 / 预查房间。
 *
 * 与后端 server/internal/handler/http.go 中对应：
 *   - POST /api/rooms     创建房间
 *   - GET  /api/rooms/:c  查询房间状态
 *   - GET  /api/healthz   健康检查
 */

import { apiBase } from './config';
import type { LotteryItem, RoomStatus } from './protocol';

export type CreateRoomResponse = {
  code: string;
  playerId: string;
  sessionToken: string;
  guaranteedAmount: number;
};

/**
 * 保底金额（单位 W）默认值，与后端 game.DefaultGuaranteedAmount 对齐。
 * 客户端 UI 默认填入此值；如需自定义，需经本地校验后再发起请求。
 */
export const DEFAULT_GUARANTEED_AMOUNT = 3188;
export const MIN_GUARANTEED_AMOUNT = 1;
export const MAX_GUARANTEED_AMOUNT = 999_999;

export function isValidGuaranteedAmount(n: number): boolean {
  return Number.isInteger(n) && n >= MIN_GUARANTEED_AMOUNT && n <= MAX_GUARANTEED_AMOUNT;
}

export type RoomSummary = {
  exists: boolean;
  status?: RoomStatus;
  playerCount?: number;
  /** 携带 playerId 查询时回填：该玩家当前是否仍在房间内（可凭口令回归恢复身份/鼠币）。 */
  isMember?: boolean;
};

export class ApiError extends Error {
  status: number;
  body?: unknown;
  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.status = status;
    this.body = body;
    this.name = 'ApiError';
  }
}

async function request<T>(
  method: 'GET' | 'POST',
  path: string,
  body?: unknown,
): Promise<T> {
  const url = apiBase() + path;
  const init: RequestInit = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (body !== undefined) init.body = JSON.stringify(body);

  let resp: Response;
  try {
    resp = await fetch(url, init);
  } catch (err) {
    throw new ApiError(0, `network: ${(err as Error).message}`);
  }
  const text = await resp.text();
  let parsed: unknown = null;
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = text;
    }
  }
  if (!resp.ok) {
    const message =
      (parsed && typeof parsed === 'object' && 'message' in parsed
        ? String((parsed as { message: unknown }).message)
        : undefined) ?? `HTTP ${resp.status}`;
    throw new ApiError(resp.status, message, parsed);
  }
  return parsed as T;
}

export function createRoom(payload: {
  nickname: string;
  avatar?: string;
  /** 保底金额（单位 W），不传则由服务端落到 DEFAULT_GUARANTEED_AMOUNT。 */
  guaranteedAmount?: number;
}): Promise<CreateRoomResponse> {
  return request<CreateRoomResponse>('POST', '/api/rooms', payload);
}

export async function queryRoom(code: string, playerId?: string): Promise<RoomSummary> {
  const suffix = playerId ? `?playerId=${encodeURIComponent(playerId)}` : '';
  try {
    return await request<RoomSummary>('GET', `/api/rooms/${code}${suffix}`);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      return { exists: false };
    }
    throw err;
  }
}

export function fetchLotteryPool(): Promise<LotteryItem[]> {
  return request<LotteryItem[]>('GET', '/api/lottery/pool');
}

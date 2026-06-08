/**
 * 与 Go 后端 internal/protocol 一一对齐的 TypeScript 类型定义。
 *
 * 所有消息均以 Envelope 形式在 WebSocket 上传输：
 *   { type: 'JOIN', payload: {...}, reqId?: '...', ts: 1700000000000 }
 */

/* ------------------------------ 基础类型 ------------------------------ */

export type Role = 'BOSS' | 'ESCORT';
export type RoomStatus = 'WAITING' | 'PLAYING' | 'CLOSED';

export type Player = {
  id: string;
  nickname: string;
  avatar?: string;
  role: Role | null;
  ready: boolean;
  coin: number;
  isHost: boolean;
  connected: boolean;
};

/**
 * 抽奖奖品（与后端 protocol.Item 对齐）。
 *
 * - `id` 即 `prizeId`，与后端 `internal/game/lottery.go` 中的奖品池一一对应。
 * - `value` 为奖品官方估值（单位 W = 万），抽中后服务端会累加到房间 `guaranteedAmount`。
 */
export type LotteryItem = {
  id: string;
  name: string;
  rarity: string;
  value: number;
};

/**
 * 一次抽奖的权威记录。
 *
 * `guaranteeDelta` 等于 `item.value`；`guaranteeBefore` / `guaranteeAfter` 是
 * 服务端裁决该抽奖时记录的保底快照。前端的"抽奖记录"列表与"实时播报"应
 * 直接读取这些字段，不再做二次计算。
 */
export type LotteryRecord = {
  id: string;
  playerId: string;
  cost: number;
  item: LotteryItem;
  guaranteeDelta: number;
  guaranteeBefore: number;
  guaranteeAfter: number;
  at: number;
};

/** 本局结算结果枚举：撤离成功（扣减保底）/ 炸单（增加保底）。 */
export type SettlementOutcome = 'EVACUATED' | 'BOMBED';

/**
 * 单次本局结算中某位玩家的明细（含服务端最终计算的 coinDelta）。
 *
 * `role` 为结算时的角色；`rescue` 仅对护航有意义（true 表示救助板板 +1 币）。
 * 服务端按角色区分计币：板板看战绩（小红 / 6 格 / 9 格），护航看击杀与救助。
 */
export type SettlementPerPlayer = {
  playerId: string;
  role?: Role;
  kills: number;
  smallReds: number;
  sixGrids: number;
  nineGrids: number;
  rescue?: boolean;
  coinDelta: number;
};

/**
 * 一次完整的本局结算记录：含保底变化与每位玩家金币变化。
 * 来源于服务端广播 `ROUND_SETTLED.record` 与快照 `settlementRecords`。
 */
export type SettlementRecord = {
  id: string;
  at: number;
  outcome: SettlementOutcome;
  guaranteeDelta: number;
  guaranteeBefore: number;
  guaranteeAfter: number;
  perPlayer: SettlementPerPlayer[];
};

/**
 * 后端的 FeedEvent 使用通用的 { kind, payload } 结构，
 * 前端以判别联合（discriminated union）方式映射，便于在模板里使用。
 *
 * 已知 kind：PLAYER_JOINED / PLAYER_LEFT / GAME_STARTED / LOTTERY / PURCHASE / SETTLEMENT。
 */
export type FeedEvent = {
  id: string;
  at: number;
  kind: string;
  payload: Record<string, unknown>;
};

/**
 * 房间快照。
 *
 * `guaranteedAmount` 是房间级别的"保底金额"（单位 W = 万），由房主在创建房间时设定。
 * PLAYING 期间房主可通过 `SETTLE_ROUND` 调整该值，其余时机不可修改。该字段是独立
 * 的展示性数值，**不可与玩家金币（player.coin）合并展示或参与运算**。
 */
export type RoomSnapshot = {
  code: string;
  status: RoomStatus;
  hostId: string;
  players: Player[];
  guaranteedAmount: number;
  lotteryRecords: LotteryRecord[];
  settlementRecords: SettlementRecord[];
  feed: FeedEvent[];
};

/* ------------------------------ Envelope ------------------------------ */

export type Envelope<T extends string = string, P = unknown> = {
  type: T;
  payload: P;
  reqId?: string;
  ts: number;
};

/* ------------------------------ Client → Server (Intent) ------------------------------ */

/**
 * SETTLE_ROUND 中每位玩家的统计输入；服务端校验后再回填 coinDelta。
 *
 * 字段范围：每个统计字段 0 ~ 99 的整数。
 */
export type SettlePlayerInput = {
  playerId: string;
  kills: number;
  smallReds: number;
  sixGrids: number;
  nineGrids: number;
  /** 仅护航生效：是否救助板板（true 时 +1 币）。 */
  rescue?: boolean;
};

export type IntentMap = {
  JOIN: { nickname: string; avatar?: string };
  LEAVE: Record<string, never>;
  DISMISS_ROOM: Record<string, never>;
  CHANGE_ROLE: { role: Role };
  READY: { ready: boolean };
  START_GAME: Record<string, never>;
  BUY_EQUIPMENT: { equipmentId: string; pick?: string };
  /**
   * 老板单抽，消耗 `SINGLE_DRAW_COST` 币。
   * 当前仅支持 `times: 1`；若后续扩展多连抽，请同步收窄/放宽该字段
   * 并更新后端 `game.CostForTimes`。
   */
  DRAW_LOTTERY: { times: 1 };
  /**
   * 房主结算本局：
   *  - `outcome`        撤离成功 / 炸单。
   *  - `guaranteeDelta` 非负整数（单位 W），EVACUATED 表示扣减额，BOMBED 表示增加额。
   *  - `perPlayer`      必须涵盖房间内全部 3 位玩家（含老板），不重复不遗漏。
   */
  SETTLE_ROUND: {
    outcome: SettlementOutcome;
    guaranteeDelta: number;
    perPlayer: SettlePlayerInput[];
  };
  PING: Record<string, never>;
};

export type IntentType = keyof IntentMap;
export type IntentEnvelope<T extends IntentType = IntentType> = Envelope<T, IntentMap[T]>;

/* ------------------------------ Server → Client (Event) ------------------------------ */

export type JoinRejectReason = 'NOT_FOUND' | 'FULL' | 'PLAYING' | 'DUP' | 'OTHER';
export type KickReason = 'DISMISSED' | 'IDLE' | 'KICKED_BY_HOST' | 'REPLACED' | string;

export type EventMap = {
  JOIN_ACCEPTED: { snapshot: RoomSnapshot; selfPlayerId: string; sessionToken: string };
  JOIN_REJECTED: { reason: JoinRejectReason; message?: string };
  STATE_SNAPSHOT: { snapshot: RoomSnapshot };
  PLAYER_JOINED: { player: Player };
  PLAYER_LEFT: { playerId: string };
  ROLE_CHANGED: { playerId: string; role: Role | null };
  READY_CHANGED: { playerId: string; ready: boolean };
  HOST_CHANGED: { newHostId: string };
  GAME_STARTED: { snapshot: RoomSnapshot };
  COIN_CHANGED: { changes: Array<{ playerId: string; delta: number; balance: number }> };
  EQUIPMENT_PURCHASED: { playerId: string; equipmentId: string; pick?: string };
  LOTTERY_DRAWN: { records: LotteryRecord[] };
  /**
   * 保底金额变化广播。
   *
   * 由 DRAW_LOTTERY（按奖品 `item.value` 累加）或 SETTLE_ROUND（撤离扣减 / 炸单增加）触发；
   * 当 before == after（即 guaranteeDelta == 0）时不发送。
   */
  GUARANTEE_CHANGED: { before: number; after: number };
  /**
   * 本局结算结果广播；与配套的 GUARANTEE_CHANGED / COIN_CHANGED 同 tick 内连续广播。
   */
  ROUND_SETTLED: { record: SettlementRecord };
  ACTION_REJECTED: { reqId?: string; reason: string; message?: string };
  KICK: { reason: KickReason };
  PONG: Record<string, never>;
};

export type EventType = keyof EventMap;
export type EventEnvelope<T extends EventType = EventType> = Envelope<T, EventMap[T]>;

/* ------------------------------ Helpers ------------------------------ */

export function buildEnvelope<T extends string, P>(
  type: T,
  payload: P,
  reqId?: string,
): Envelope<T, P> {
  return { type, payload, reqId, ts: Date.now() };
}

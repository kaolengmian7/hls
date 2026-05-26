// Package protocol 定义服务端与客户端之间所有 WebSocket 消息的结构体。
//
// 所有消息统一打包到 Envelope 中：
//
//	{ "type": "JOIN", "payload": {...}, "reqId": "abc", "ts": 1700000000000 }
//
// type 枚举：见 IntentType（客户端 → 服务端）与 EventType（服务端 → 客户端）。
package protocol

import "encoding/json"

// Envelope 统一消息封套。
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	ReqID   string          `json:"reqId,omitempty"`
	TS      int64           `json:"ts"`
}

// Outgoing 是构造服务端 → 客户端消息的辅助类型。
// 与 Envelope 的区别仅在于 Payload 直接是 any，便于序列化时一次性 marshal。
type Outgoing struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
	ReqID   string `json:"reqId,omitempty"`
	TS      int64  `json:"ts"`
}

// RoomStatus 房间生命周期状态。
type RoomStatus string

const (
	RoomStatusWaiting RoomStatus = "WAITING"
	RoomStatusPlaying RoomStatus = "PLAYING"
	RoomStatusClosed  RoomStatus = "CLOSED"
)

// Role 角色枚举。
type Role string

const (
	RoleBoss   Role = "BOSS"
	RoleEscort Role = "ESCORT"
)

// Player 单玩家快照视图。
type Player struct {
	ID        string `json:"id"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar,omitempty"`
	Role      Role   `json:"role,omitempty"`
	Ready     bool   `json:"ready"`
	Coin      int    `json:"coin"`
	IsHost    bool   `json:"isHost"`
	Connected bool   `json:"connected"`
}

// Item 物品最小描述。
//
// Value 为奖品官方估值（单位 W = 万），正整数。老板每抽中一件奖品，
// 服务端会在裁决抽奖的同 tick 内把 Value 累加到房间 GuaranteedAmount。
// Value == 0 视为"无价值奖品"，不会推高保底；当前奖品池不使用该形态。
type Item struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Rarity string `json:"rarity"`
	Value  int    `json:"value"`
}

// LotteryRecord 抽奖结果记录。
//
// GuaranteeDelta == Item.Value，单位 W；
// GuaranteeBefore / GuaranteeAfter 由服务端在裁决抽奖时记录，是该条抽奖
// 对房间保底金额产生影响的权威快照。前端的"抽奖记录"列表与"实时播报"
// 都应直接读取这些字段，不做二次计算。
type LotteryRecord struct {
	ID              string `json:"id"`
	PlayerID        string `json:"playerId"`
	Cost            int    `json:"cost"`
	Item            Item   `json:"item"`
	GuaranteeDelta  int    `json:"guaranteeDelta"`
	GuaranteeBefore int    `json:"guaranteeBefore"`
	GuaranteeAfter  int    `json:"guaranteeAfter"`
	At              int64  `json:"at"`
}

// SettlementOutcome 本局结算结果枚举。
type SettlementOutcome string

const (
	// SettlementOutcomeEvacuated 撤离成功：扣减保底金额、每位玩家额外 +3 币。
	SettlementOutcomeEvacuated SettlementOutcome = "EVACUATED"
	// SettlementOutcomeBombed 炸单：增加保底金额、不发额外基础币。
	SettlementOutcomeBombed SettlementOutcome = "BOMBED"
)

// SettlementPerPlayer 单次本局结算中某位玩家的明细，含服务端最终计算的 CoinDelta。
type SettlementPerPlayer struct {
	PlayerID  string `json:"playerId"`
	Kills     int    `json:"kills"`
	SmallReds int    `json:"smallReds"`
	SixGrids  int    `json:"sixGrids"`
	NineGrids int    `json:"nineGrids"`
	CoinDelta int    `json:"coinDelta"`
}

// SettlementRecord 一次完整的本局结算记录（含保底变化与每人金币变化）。
//
// 通过 STATE_SNAPSHOT.SettlementRecords 与事件 ROUND_SETTLED 同步给客户端。
type SettlementRecord struct {
	ID              string                `json:"id"`
	At              int64                 `json:"at"`
	Outcome         SettlementOutcome     `json:"outcome"`
	GuaranteeDelta  int                   `json:"guaranteeDelta"`
	GuaranteeBefore int                   `json:"guaranteeBefore"`
	GuaranteeAfter  int                   `json:"guaranteeAfter"`
	PerPlayer       []SettlementPerPlayer `json:"perPlayer"`
}

// FeedEvent 是「实时播报」中通用事件结构。
//
// Kind 取值：PLAYER_JOINED / PLAYER_LEFT / GAME_STARTED / LOTTERY / PURCHASE / SETTLEMENT。
type FeedEvent struct {
	ID      string         `json:"id"`
	At      int64          `json:"at"`
	Kind    string         `json:"kind"`
	Payload map[string]any `json:"payload"`
}

// RoomSnapshot 给客户端的房间快照。
//
// GuaranteedAmount 是房间级别的"保底金额"，单位 W（万），由房主在创建房间时设定。
// PLAYING 期间房主可通过 SETTLE_ROUND 调整该值，其余时机不可修改。
// 该字段是独立的展示性数值，与玩家金币（Player.Coin）解耦，不参与任何金币结算 /
// 抽奖 / 装备扣费等金币运算。
type RoomSnapshot struct {
	Code              string             `json:"code"`
	Status            RoomStatus         `json:"status"`
	HostID            string             `json:"hostId"`
	Players           []Player           `json:"players"`
	GuaranteedAmount  int                `json:"guaranteedAmount"`
	LotteryRecords    []LotteryRecord    `json:"lotteryRecords"`
	SettlementRecords []SettlementRecord `json:"settlementRecords"`
	Feed              []FeedEvent        `json:"feed"`
}

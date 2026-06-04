package protocol

// Event 枚举：服务端 → 客户端。
const (
	EventJoinAccepted       = "JOIN_ACCEPTED"
	EventJoinRejected       = "JOIN_REJECTED"
	EventStateSnapshot      = "STATE_SNAPSHOT"
	EventPlayerJoined       = "PLAYER_JOINED"
	EventPlayerLeft         = "PLAYER_LEFT"
	EventRoleChanged        = "ROLE_CHANGED"
	EventReadyChanged       = "READY_CHANGED"
	EventHostChanged        = "HOST_CHANGED"
	EventGameStarted        = "GAME_STARTED"
	EventCoinChanged        = "COIN_CHANGED"
	EventEquipmentPurchased = "EQUIPMENT_PURCHASED"
	EventLotteryDrawn       = "LOTTERY_DRAWN"
	EventGuaranteeChanged   = "GUARANTEE_CHANGED"
	EventRoundSettled       = "ROUND_SETTLED"
	EventActionRejected     = "ACTION_REJECTED"
	EventKick               = "KICK"
	EventPong               = "PONG"
)

// JoinRejectReason / KickReason 枚举。
const (
	RejectNotFound = "NOT_FOUND"
	RejectFull     = "FULL"
	RejectPlaying  = "PLAYING"
	RejectDup      = "DUP"
	RejectOther    = "OTHER"

	KickDismissed = "DISMISSED"
	KickIdle      = "IDLE"
	KickByHost    = "KICKED_BY_HOST"
)

// JoinAcceptedPayload 服务端 → 客户端 准入成功。
type JoinAcceptedPayload struct {
	Snapshot     RoomSnapshot `json:"snapshot"`
	SelfPlayerID string       `json:"selfPlayerId"`
	SessionToken string       `json:"sessionToken"`
}

// JoinRejectedPayload 服务端拒绝。
type JoinRejectedPayload struct {
	Reason  string `json:"reason"`
	Message string `json:"message,omitempty"`
}

// StateSnapshotPayload 房间完整快照。
type StateSnapshotPayload struct {
	Snapshot RoomSnapshot `json:"snapshot"`
}

// PlayerJoinedPayload 增量：玩家加入。
type PlayerJoinedPayload struct {
	Player Player `json:"player"`
}

// PlayerLeftPayload 增量：玩家离开。
type PlayerLeftPayload struct {
	PlayerID string `json:"playerId"`
}

// RoleChangedPayload 增量：角色变化。
type RoleChangedPayload struct {
	PlayerID string `json:"playerId"`
	Role     Role   `json:"role,omitempty"`
}

// ReadyChangedPayload 增量：准备状态变化。
type ReadyChangedPayload struct {
	PlayerID string `json:"playerId"`
	Ready    bool   `json:"ready"`
}

// HostChangedPayload 房主迁移。
type HostChangedPayload struct {
	NewHostID string `json:"newHostId"`
}

// GameStartedPayload 游戏开始。
type GameStartedPayload struct {
	Snapshot RoomSnapshot `json:"snapshot"`
}

// CoinChange 单个金币变更条目。
type CoinChange struct {
	PlayerID string `json:"playerId"`
	Delta    int    `json:"delta"`
	Balance  int    `json:"balance"`
}

// CoinChangedPayload 金币变化广播。
type CoinChangedPayload struct {
	Changes []CoinChange `json:"changes"`
}

// EquipmentPurchasedPayload 装备购买广播。
type EquipmentPurchasedPayload struct {
	PlayerID    string `json:"playerId"`
	EquipmentID string `json:"equipmentId"`
	Pick        string `json:"pick,omitempty"`
}

// LotteryDrawnPayload 抽奖结果广播。
//
// 当前每次 DRAW_LOTTERY 仅产出 1 条记录（单抽）；若后续扩展多连抽，
// Records 可携带多条，并按抽奖顺序排列。
// 每条记录的 GuaranteeDelta / GuaranteeBefore / GuaranteeAfter 已经反映
// 服务端裁决该抽奖时对房间 GuaranteedAmount 的影响，与同 tick 内先行
// 下发的 GUARANTEE_CHANGED 一致。
type LotteryDrawnPayload struct {
	Records []LotteryRecord `json:"records"`
}

// GuaranteeChangedPayload 保底金额变化广播。
//
// 由 DRAW_LOTTERY（按奖品 Item.Value 累加）或 SETTLE_ROUND（撤离扣减 / 炸单增加）触发；
// 当 Before == After（GuaranteeDelta == 0）时不发送。
type GuaranteeChangedPayload struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

// RoundSettledPayload 本局结算结果广播。
//
// Record 内含本次保底变化、每位玩家最终的 CoinDelta，与配套的
// GUARANTEE_CHANGED / COIN_CHANGED 同 tick 内连续广播。
type RoundSettledPayload struct {
	Record SettlementRecord `json:"record"`
}

// ActionRejectedPayload 通用动作拒绝回执。
type ActionRejectedPayload struct {
	ReqID   string `json:"reqId,omitempty"`
	Reason  string `json:"reason"`
	Message string `json:"message,omitempty"`
}

// KickPayload 被强制下线。
type KickPayload struct {
	Reason string `json:"reason"`
}

package protocol

// Intent 枚举：客户端 → 服务端。
const (
	IntentJoin         = "JOIN"
	IntentLeave        = "LEAVE"
	IntentDismissRoom  = "DISMISS_ROOM"
	IntentChangeRole   = "CHANGE_ROLE"
	IntentReady        = "READY"
	IntentStartGame    = "START_GAME"
	IntentBuyEquipment = "BUY_EQUIPMENT"
	IntentDrawLottery  = "DRAW_LOTTERY"
	IntentSettleRound  = "SETTLE_ROUND"
	IntentCloseGame    = "CLOSE_GAME"
	IntentPing         = "PING"
)

// JoinPayload 客户端 JOIN 首帧。
type JoinPayload struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar,omitempty"`
}

// ChangeRolePayload 客户端切换角色。
type ChangeRolePayload struct {
	Role Role `json:"role"`
}

// ReadyPayload 客户端切换准备状态。
type ReadyPayload struct {
	Ready bool `json:"ready"`
}

// BuyEquipmentPayload 客户端购买装备。
type BuyEquipmentPayload struct {
	EquipmentID string `json:"equipmentId"`
	Pick        string `json:"pick,omitempty"`
}

// DrawLotteryPayload 客户端发起抽奖。
//
// Times 当前仅支持 1（单抽，消耗 SingleDrawCost 币）；其它值会被服务端拒绝。
type DrawLotteryPayload struct {
	Times int `json:"times"`
}

// SettlePlayerInput 房主结算时为每位玩家提交的统计输入。
//
// 字段含义：
//   - Kills      击杀人数 (0 ~ 99)
//   - SmallReds  摸小红数量 (0 ~ 99)
//   - SixGrids   摸 6 格数量 (0 ~ 99)
//   - NineGrids  摸 9 格及以上数量 (0 ~ 99)
//   - Rescue     是否救助板板（仅护航生效，true 时 +1 币）
//
// 服务端按角色区分计币规则：
//   - 板板（老板）：Kills*1 + SmallReds*1 + SixGrids*2 + NineGrids*3。
//   - 护航：Kills*1 + (Rescue ? 1 : 0)；不再通过小红 / 6 格 / 9 格获得币。
//
// EVACUATED 结果再额外为每位玩家追加 3 币基础奖励。
type SettlePlayerInput struct {
	PlayerID  string `json:"playerId"`
	Kills     int    `json:"kills"`
	SmallReds int    `json:"smallReds"`
	SixGrids  int    `json:"sixGrids"`
	NineGrids int    `json:"nineGrids"`
	Rescue    bool   `json:"rescue,omitempty"`
}

// SettleRoundPayload 客户端发起本局结算（仅房主在 PLAYING 状态有效）。
//
// GuaranteeDelta 单位 W，非负整数；EVACUATED 表示本次得吃增加额，BOMBED 表示保底增加额。
// BossCovered 仅 EVACUATED 生效：true 表示老板兜底，得吃增加但保底不扣减。
// PerPlayer 必须涵盖房间内全部 3 位玩家（含老板），不允许重复或遗漏。
type SettleRoundPayload struct {
	Outcome        SettlementOutcome   `json:"outcome"`
	GuaranteeDelta int                 `json:"guaranteeDelta"`
	BossCovered    bool                `json:"bossCovered,omitempty"`
	PerPlayer      []SettlePlayerInput `json:"perPlayer"`
}

// Package game 持有所有业务规则（角色名额、装备扣费、抽奖随机）。
//
// 该包是 *纯函数* 实现，不感知房间/连接等基础设施，便于单元测试与复用。
package game

import (
	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/errors"
)

const (
	// MaxPlayers 房间最大人数。
	MaxPlayers = 3
	// MaxEscorts 护航名额。
	MaxEscorts = 2
	// MaxBosses 老板名额。
	MaxBosses = 1
	// EscortInitialCoin 护航初始金币。
	EscortInitialCoin = 12
	// BossInitialCoin 老板初始金币。
	BossInitialCoin = 0

	// 保底金额（房间级展示性数值，单位 W = 万）。
	// 与玩家金币（Player.Coin）完全解耦，不参与任何金币结算 / 抽奖 / 装备扣费。
	//
	// MinGuaranteedAmount / MaxGuaranteedAmount 为合法范围（含端点）；
	// DefaultGuaranteedAmount 为客户端未指定时的服务端兜底默认值。
	MinGuaranteedAmount     = 1
	MaxGuaranteedAmount     = 999_999
	DefaultGuaranteedAmount = 3188
)

// NormalizeGuaranteedAmount 规范化客户端传入的保底金额：
//   - 0 视为未指定，回落到 DefaultGuaranteedAmount。
//   - 落在 [Min, Max] 内：原值返回。
//   - 越界：返回 NewExposable 错误，由调用方决定是回 400 还是 ACTION_REJECTED。
func NormalizeGuaranteedAmount(amount int) (int, error) {
	if amount == 0 {
		return DefaultGuaranteedAmount, nil
	}
	if amount < MinGuaranteedAmount || amount > MaxGuaranteedAmount {
		return 0, errors.NewExposable(errors.CodeInvalidParam,
			"INVALID_GUARANTEE: guaranteedAmount must be in [%d, %d]",
			MinGuaranteedAmount, MaxGuaranteedAmount)
	}
	return amount, nil
}

// CountByRole 统计 players 中每个角色已占用数量。
func CountByRole(players []protocol.Player) (bosses, escorts int) {
	for _, p := range players {
		switch p.Role {
		case protocol.RoleBoss:
			bosses++
		case protocol.RoleEscort:
			escorts++
		}
	}
	return
}

// CanAssignRole 判断 player 是否还能切换到 role。
// exceptPlayerID 表示自己原先占用的角色不计入名额校验。
func CanAssignRole(players []protocol.Player, role protocol.Role, exceptPlayerID string) bool {
	occupied := 0
	for _, p := range players {
		if p.ID == exceptPlayerID {
			continue
		}
		if p.Role == role {
			occupied++
		}
	}
	switch role {
	case protocol.RoleBoss:
		return occupied < MaxBosses
	case protocol.RoleEscort:
		return occupied < MaxEscorts
	}
	return false
}

// InitialCoin 返回 role 对应的初始金币。
func InitialCoin(role protocol.Role) int {
	switch role {
	case protocol.RoleEscort:
		return EscortInitialCoin
	case protocol.RoleBoss:
		return BossInitialCoin
	}
	return 0
}

// CanStartGame 校验是否可以从 WAITING 切到 PLAYING。
// 返回非空 *Error 表示不可开始；nil 表示满足条件。
func CanStartGame(players []protocol.Player) error {
	if len(players) != MaxPlayers {
		return errors.NewExposable(errors.CodeStartCondition, "NOT_FULL")
	}
	bosses, escorts := CountByRole(players)
	if bosses != MaxBosses || escorts != MaxEscorts {
		return errors.NewExposable(errors.CodeStartCondition, "ROLE_INCOMPLETE")
	}
	for _, p := range players {
		if !p.Ready {
			return errors.NewExposable(errors.CodeStartCondition, "NOT_ALL_READY")
		}
	}
	return nil
}

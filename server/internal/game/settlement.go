package game

import (
	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/errors"
)

// 本局结算金币计算常量。
//
// 金币计算按角色区分（每位玩家）：
//
//	板板（老板）：
//	  coinDelta = kills*SettlementCoinKill
//	            + smallReds*SettlementCoinSmallRed
//	            + sixGrids*SettlementCoinSixGrid
//	            + nineGrids*SettlementCoinNineGrid
//	            + (outcome == EVACUATED ? SettlementEvacuationBonus : 0)
//
//	护航：
//	  coinDelta = kills*SettlementCoinKill
//	            + (rescue ? SettlementCoinRescue : 0)
//	            + (outcome == EVACUATED ? SettlementEvacuationBonus : 0)
//	  （护航不再通过小红 / 6 格 / 9 格获得币）
const (
	// SettlementEvacuationBonus 撤离成功时每位玩家额外获得的基础币（适用于全部 3 位玩家）。
	SettlementEvacuationBonus = 3
	// SettlementCoinKill 每击杀 1 人获得的金币。
	SettlementCoinKill = 1
	// SettlementCoinSmallRed 每摸 1 个小红获得的金币（仅板板）。
	SettlementCoinSmallRed = 1
	// SettlementCoinSixGrid 每摸 1 个 6 格获得的金币（仅板板）。
	SettlementCoinSixGrid = 2
	// SettlementCoinNineGrid 每摸 1 个 9 格及以上获得的金币（仅板板）。
	SettlementCoinNineGrid = 3
	// SettlementCoinRescue 护航救助板板获得的金币。
	SettlementCoinRescue = 1

	// MinSettlementStat / MaxSettlementStat 每位玩家明细字段的合法范围（含端点）。
	MinSettlementStat = 0
	MaxSettlementStat = 99
)

// 暴露给上层的 reason 字符串（直接写入 ACTION_REJECTED.reason）。
const (
	settlementReasonInvalidParams    = "INVALID_PARAMS"
	settlementReasonInvalidGuarantee = "INVALID_GUARANTEE"
)

// SettlementResult 是 ComputeSettlement 的产出，由 Room Loop 据此写入权威状态。
//
// PerPlayer 顺序与入参 playerIDs 完全一致，便于客户端按 STATE_SNAPSHOT.players
// 顺序对齐展示。
type SettlementResult struct {
	Outcome         protocol.SettlementOutcome
	GuaranteeDelta  int
	GuaranteeBefore int
	GuaranteeAfter  int
	BossCovered     bool
	PerPlayer       []protocol.SettlementPerPlayer
}

// ComputeSettlement 校验一次本局结算输入并计算金币奖励。
//
// 校验规则：
//   - outcome 必须为 EVACUATED 或 BOMBED；
//   - guaranteeDelta 落在 [0, MaxGuaranteedAmount]；
//   - 计算后的新保底金额不得超过 MaxGuaranteedAmount（撤离扣减允许为负数），越界视为 INVALID_GUARANTEE；
//   - perPlayer 必须涵盖 playerIDs 中全部 ID 且不重复，每个统计字段落在 [0, 99]。
//
// roleByID 提供每位玩家结算时的角色，用于区分计币规则（板板看战绩、护航看救助）。
// 缺省（无对应条目）按板板规则计算。
//
// 调用者职责：在调用前完成「房主权限」与「房间状态为 PLAYING」的校验。
func ComputeSettlement(
	outcome protocol.SettlementOutcome,
	guaranteeDelta int,
	bossCovered bool,
	currentGuarantee int,
	perPlayer []protocol.SettlePlayerInput,
	playerIDs []string,
	roleByID map[string]protocol.Role,
) (SettlementResult, error) {
	if outcome != protocol.SettlementOutcomeEvacuated && outcome != protocol.SettlementOutcomeBombed {
		return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidParams)
	}
	if guaranteeDelta < 0 || guaranteeDelta > MaxGuaranteedAmount {
		return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidParams)
	}

	var newGuarantee int
	switch outcome {
	case protocol.SettlementOutcomeEvacuated:
		if bossCovered {
			newGuarantee = currentGuarantee
		} else {
			// 撤离扣减保底允许结果为负数，不做下限校验。
			newGuarantee = currentGuarantee - guaranteeDelta
		}
	case protocol.SettlementOutcomeBombed:
		newGuarantee = currentGuarantee + guaranteeDelta
	}
	if newGuarantee > MaxGuaranteedAmount {
		return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidGuarantee)
	}

	if len(perPlayer) != len(playerIDs) {
		return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidParams)
	}
	inputByID := make(map[string]protocol.SettlePlayerInput, len(perPlayer))
	for _, item := range perPlayer {
		if _, dup := inputByID[item.PlayerID]; dup {
			return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidParams)
		}
		if !validStat(item.Kills) || !validStat(item.SmallReds) ||
			!validStat(item.SixGrids) || !validStat(item.NineGrids) {
			return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidParams)
		}
		inputByID[item.PlayerID] = item
	}

	out := make([]protocol.SettlementPerPlayer, 0, len(playerIDs))
	for _, pid := range playerIDs {
		in, ok := inputByID[pid]
		if !ok {
			return SettlementResult{}, errors.NewExposable(errors.CodeInvalidParam, settlementReasonInvalidParams)
		}
		role := roleByID[pid]
		out = append(out, protocol.SettlementPerPlayer{
			PlayerID:  pid,
			Role:      role,
			Kills:     in.Kills,
			SmallReds: in.SmallReds,
			SixGrids:  in.SixGrids,
			NineGrids: in.NineGrids,
			Rescue:    in.Rescue,
			CoinDelta: computeCoinDelta(outcome, role, in),
		})
	}

	return SettlementResult{
		Outcome:         outcome,
		GuaranteeDelta:  guaranteeDelta,
		GuaranteeBefore: currentGuarantee,
		GuaranteeAfter:  newGuarantee,
		BossCovered:     outcome == protocol.SettlementOutcomeEvacuated && bossCovered,
		PerPlayer:       out,
	}, nil
}

func computeCoinDelta(outcome protocol.SettlementOutcome, role protocol.Role, in protocol.SettlePlayerInput) int {
	var delta int
	switch role {
	case protocol.RoleEscort:
		// 护航：仅击杀与「救助板板」计币，不再通过小红 / 6 格 / 9 格获得币。
		delta = in.Kills * SettlementCoinKill
		if in.Rescue {
			delta += SettlementCoinRescue
		}
	default:
		// 板板（老板）及兜底：保留完整战绩计币规则。
		delta = in.Kills*SettlementCoinKill +
			in.SmallReds*SettlementCoinSmallRed +
			in.SixGrids*SettlementCoinSixGrid +
			in.NineGrids*SettlementCoinNineGrid
	}
	if outcome == protocol.SettlementOutcomeEvacuated {
		delta += SettlementEvacuationBonus
	}
	return delta
}

func validStat(v int) bool {
	return v >= MinSettlementStat && v <= MaxSettlementStat
}

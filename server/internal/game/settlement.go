package game

import (
	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/errors"
)

// 本局结算金币计算常量。
//
// 金币计算公式（每位玩家）：
//
//	coinDelta = kills*SettlementCoinKill
//	          + smallReds*SettlementCoinSmallRed
//	          + sixGrids*SettlementCoinSixGrid
//	          + nineGrids*SettlementCoinNineGrid
//	          + (outcome == EVACUATED ? SettlementEvacuationBonus : 0)
const (
	// SettlementEvacuationBonus 撤离成功时每位玩家额外获得的基础币（适用于全部 3 位玩家）。
	SettlementEvacuationBonus = 3
	// SettlementCoinKill 每击杀 1 人获得的金币。
	SettlementCoinKill = 1
	// SettlementCoinSmallRed 每摸 1 个小红获得的金币。
	SettlementCoinSmallRed = 1
	// SettlementCoinSixGrid 每摸 1 个 6 格获得的金币。
	SettlementCoinSixGrid = 2
	// SettlementCoinNineGrid 每摸 1 个 9 格及以上获得的金币。
	SettlementCoinNineGrid = 3

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
	PerPlayer       []protocol.SettlementPerPlayer
}

// ComputeSettlement 校验一次本局结算输入并计算金币奖励。
//
// 校验规则：
//   - outcome 必须为 EVACUATED 或 BOMBED；
//   - guaranteeDelta 落在 [0, MaxGuaranteedAmount]；
//   - 计算后的新保底金额落在 [0, MaxGuaranteedAmount]，越界视为 INVALID_GUARANTEE；
//   - perPlayer 必须涵盖 playerIDs 中全部 ID 且不重复，每个统计字段落在 [0, 99]。
//
// 调用者职责：在调用前完成「房主权限」与「房间状态为 PLAYING」的校验。
func ComputeSettlement(
	outcome protocol.SettlementOutcome,
	guaranteeDelta int,
	currentGuarantee int,
	perPlayer []protocol.SettlePlayerInput,
	playerIDs []string,
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
		newGuarantee = currentGuarantee - guaranteeDelta
	case protocol.SettlementOutcomeBombed:
		newGuarantee = currentGuarantee + guaranteeDelta
	}
	if newGuarantee < 0 || newGuarantee > MaxGuaranteedAmount {
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
		out = append(out, protocol.SettlementPerPlayer{
			PlayerID:  pid,
			Kills:     in.Kills,
			SmallReds: in.SmallReds,
			SixGrids:  in.SixGrids,
			NineGrids: in.NineGrids,
			CoinDelta: computeCoinDelta(outcome, in),
		})
	}

	return SettlementResult{
		Outcome:         outcome,
		GuaranteeDelta:  guaranteeDelta,
		GuaranteeBefore: currentGuarantee,
		GuaranteeAfter:  newGuarantee,
		PerPlayer:       out,
	}, nil
}

func computeCoinDelta(outcome protocol.SettlementOutcome, in protocol.SettlePlayerInput) int {
	delta := in.Kills*SettlementCoinKill +
		in.SmallReds*SettlementCoinSmallRed +
		in.SixGrids*SettlementCoinSixGrid +
		in.NineGrids*SettlementCoinNineGrid
	if outcome == protocol.SettlementOutcomeEvacuated {
		delta += SettlementEvacuationBonus
	}
	return delta
}

func validStat(v int) bool {
	return v >= MinSettlementStat && v <= MaxSettlementStat
}

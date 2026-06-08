package game

import (
	"testing"

	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/errors"
)

func mkInput(pid string, k, sr, sg, ng int) protocol.SettlePlayerInput {
	return protocol.SettlePlayerInput{
		PlayerID:  pid,
		Kills:     k,
		SmallReds: sr,
		SixGrids:  sg,
		NineGrids: ng,
	}
}

// rolesAllBoss 为给定 id 列表构造「全部按板板规则计币」的角色表，
// 用于复用历史用例对完整战绩计币的期望。
func rolesAllBoss(ids ...string) map[string]protocol.Role {
	m := make(map[string]protocol.Role, len(ids))
	for _, id := range ids {
		m[id] = protocol.RoleBoss
	}
	return m
}

func TestComputeSettlementEvacuated(t *testing.T) {
	playerIDs := []string{"a", "b", "c"}
	result, err := ComputeSettlement(
		protocol.SettlementOutcomeEvacuated,
		200,
		2500,
		[]protocol.SettlePlayerInput{
			mkInput("a", 1, 2, 1, 1),
			mkInput("c", 0, 0, 0, 0),
			mkInput("b", 0, 2, 0, 0),
		},
		playerIDs,
		rolesAllBoss(playerIDs...),
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.Outcome != protocol.SettlementOutcomeEvacuated {
		t.Errorf("outcome=%v", result.Outcome)
	}
	if result.GuaranteeBefore != 2500 || result.GuaranteeAfter != 2300 || result.GuaranteeDelta != 200 {
		t.Errorf("guarantee got before=%d after=%d delta=%d",
			result.GuaranteeBefore, result.GuaranteeAfter, result.GuaranteeDelta)
	}
	if len(result.PerPlayer) != 3 {
		t.Fatalf("perPlayer len=%d", len(result.PerPlayer))
	}
	// 顺序必须按 playerIDs：a, b, c
	if result.PerPlayer[0].PlayerID != "a" ||
		result.PerPlayer[1].PlayerID != "b" ||
		result.PerPlayer[2].PlayerID != "c" {
		t.Fatalf("perPlayer order wrong: %#v", result.PerPlayer)
	}
	// 撤离成功每人 +3 基础 + 战绩
	// a: 1 + 2 + 1*2 + 1*3 + 3 = 11
	// b: 0 + 2 + 0   + 0   + 3 = 5
	// c: 0 + 0 + 0   + 0   + 3 = 3
	wantCoins := []int{11, 5, 3}
	for i, want := range wantCoins {
		if result.PerPlayer[i].CoinDelta != want {
			t.Errorf("perPlayer[%d].CoinDelta=%d want %d",
				i, result.PerPlayer[i].CoinDelta, want)
		}
	}
}

func TestComputeSettlementBombed(t *testing.T) {
	playerIDs := []string{"a", "b", "c"}
	result, err := ComputeSettlement(
		protocol.SettlementOutcomeBombed,
		60,
		2500,
		[]protocol.SettlePlayerInput{
			mkInput("a", 2, 0, 0, 0),
			mkInput("b", 0, 0, 1, 1),
			mkInput("c", 0, 0, 0, 0),
		},
		playerIDs,
		rolesAllBoss(playerIDs...),
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.GuaranteeAfter != 2560 {
		t.Errorf("guarantee after=%d want 2560", result.GuaranteeAfter)
	}
	// 炸单不发基础 +3
	// a: 2*1 = 2
	// b: 1*2 + 1*3 = 5
	// c: 0
	wantCoins := []int{2, 5, 0}
	for i, want := range wantCoins {
		if result.PerPlayer[i].CoinDelta != want {
			t.Errorf("perPlayer[%d].CoinDelta=%d want %d",
				i, result.PerPlayer[i].CoinDelta, want)
		}
	}
}

func TestComputeSettlementRoleDifferentiated(t *testing.T) {
	playerIDs := []string{"boss", "e1", "e2"}
	roleByID := map[string]protocol.Role{
		"boss": protocol.RoleBoss,
		"e1":   protocol.RoleEscort,
		"e2":   protocol.RoleEscort,
	}
	// e1 救助板板（+1），e2 未救助。护航的小红 / 6 格 / 9 格不计币。
	e1 := mkInput("e1", 1, 5, 5, 5)
	e1.Rescue = true
	e2 := mkInput("e2", 0, 9, 9, 9)

	result, err := ComputeSettlement(
		protocol.SettlementOutcomeEvacuated,
		0,
		2500,
		[]protocol.SettlePlayerInput{
			mkInput("boss", 1, 2, 1, 1),
			e1,
			e2,
		},
		playerIDs,
		roleByID,
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// 撤离成功每人 +3 基础。
	// boss : 1 + 2 + 1*2 + 1*3 + 3 = 11
	// e1   : 1(击杀) + 1(救助) + 3 = 5（小红/6格/9格不计）
	// e2   : 0 + 0 + 3 = 3
	wantCoins := []int{11, 5, 3}
	for i, want := range wantCoins {
		if result.PerPlayer[i].CoinDelta != want {
			t.Errorf("perPlayer[%d].CoinDelta=%d want %d",
				i, result.PerPlayer[i].CoinDelta, want)
		}
	}
	if result.PerPlayer[0].Role != protocol.RoleBoss {
		t.Errorf("boss role not recorded: %v", result.PerPlayer[0].Role)
	}
	if result.PerPlayer[1].Role != protocol.RoleEscort || !result.PerPlayer[1].Rescue {
		t.Errorf("escort rescue not recorded: %#v", result.PerPlayer[1])
	}
	if result.PerPlayer[2].Rescue {
		t.Errorf("e2 should not be marked rescued")
	}
}

func TestComputeSettlementEvacuatedNegativeGuarantee(t *testing.T) {
	playerIDs := []string{"a", "b", "c"}
	result, err := ComputeSettlement(
		protocol.SettlementOutcomeEvacuated,
		100,
		50,
		[]protocol.SettlePlayerInput{
			mkInput("a", 0, 0, 0, 0),
			mkInput("b", 0, 0, 0, 0),
			mkInput("c", 0, 0, 0, 0),
		},
		playerIDs,
		rolesAllBoss(playerIDs...),
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.GuaranteeAfter != -50 {
		t.Errorf("guarantee after=%d want -50", result.GuaranteeAfter)
	}
}

func TestComputeSettlementZeroDelta(t *testing.T) {
	playerIDs := []string{"a", "b", "c"}
	result, err := ComputeSettlement(
		protocol.SettlementOutcomeEvacuated,
		0,
		2500,
		[]protocol.SettlePlayerInput{
			mkInput("a", 0, 0, 0, 0),
			mkInput("b", 0, 0, 0, 0),
			mkInput("c", 0, 0, 0, 0),
		},
		playerIDs,
		rolesAllBoss(playerIDs...),
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.GuaranteeBefore != result.GuaranteeAfter {
		t.Errorf("guarantee should not change with delta=0")
	}
	for _, pp := range result.PerPlayer {
		if pp.CoinDelta != SettlementEvacuationBonus {
			t.Errorf("EVACUATED zero stats should give %d, got %d",
				SettlementEvacuationBonus, pp.CoinDelta)
		}
	}
}

func TestComputeSettlementInvalid(t *testing.T) {
	playerIDs := []string{"a", "b", "c"}
	good := []protocol.SettlePlayerInput{
		mkInput("a", 0, 0, 0, 0),
		mkInput("b", 0, 0, 0, 0),
		mkInput("c", 0, 0, 0, 0),
	}

	tests := []struct {
		name           string
		outcome        protocol.SettlementOutcome
		guaranteeDelta int
		current        int
		perPlayer      []protocol.SettlePlayerInput
		ids            []string
		wantReason     string
	}{
		{
			name:           "未知结果",
			outcome:        "WHATEVER",
			guaranteeDelta: 10,
			current:        2500,
			perPlayer:      good,
			ids:            playerIDs,
			wantReason:     "INVALID_PARAMS",
		},
		{
			name:           "delta 负数",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: -1,
			current:        2500,
			perPlayer:      good,
			ids:            playerIDs,
			wantReason:     "INVALID_PARAMS",
		},
		{
			name:           "delta 超过最大值",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: MaxGuaranteedAmount + 1,
			current:        2500,
			perPlayer:      good,
			ids:            playerIDs,
			wantReason:     "INVALID_PARAMS",
		},
		{
			name:           "炸单累加超过上限",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: 100,
			current:        MaxGuaranteedAmount,
			perPlayer:      good,
			ids:            playerIDs,
			wantReason:     "INVALID_GUARANTEE",
		},
		{
			name:           "perPlayer 数量不匹配",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: 60,
			current:        2500,
			perPlayer: []protocol.SettlePlayerInput{
				mkInput("a", 0, 0, 0, 0),
				mkInput("b", 0, 0, 0, 0),
			},
			ids:        playerIDs,
			wantReason: "INVALID_PARAMS",
		},
		{
			name:           "perPlayer 重复 playerId",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: 60,
			current:        2500,
			perPlayer: []protocol.SettlePlayerInput{
				mkInput("a", 0, 0, 0, 0),
				mkInput("a", 0, 0, 0, 0),
				mkInput("b", 0, 0, 0, 0),
			},
			ids:        playerIDs,
			wantReason: "INVALID_PARAMS",
		},
		{
			name:           "perPlayer 缺失玩家",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: 60,
			current:        2500,
			perPlayer: []protocol.SettlePlayerInput{
				mkInput("a", 0, 0, 0, 0),
				mkInput("b", 0, 0, 0, 0),
				mkInput("d", 0, 0, 0, 0),
			},
			ids:        playerIDs,
			wantReason: "INVALID_PARAMS",
		},
		{
			name:           "字段超出 0~99",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: 60,
			current:        2500,
			perPlayer: []protocol.SettlePlayerInput{
				mkInput("a", 100, 0, 0, 0),
				mkInput("b", 0, 0, 0, 0),
				mkInput("c", 0, 0, 0, 0),
			},
			ids:        playerIDs,
			wantReason: "INVALID_PARAMS",
		},
		{
			name:           "字段为负",
			outcome:        protocol.SettlementOutcomeBombed,
			guaranteeDelta: 60,
			current:        2500,
			perPlayer: []protocol.SettlePlayerInput{
				mkInput("a", -1, 0, 0, 0),
				mkInput("b", 0, 0, 0, 0),
				mkInput("c", 0, 0, 0, 0),
			},
			ids:        playerIDs,
			wantReason: "INVALID_PARAMS",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ComputeSettlement(tt.outcome, tt.guaranteeDelta, tt.current, tt.perPlayer, tt.ids, rolesAllBoss(tt.ids...))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.IsExposable(err) {
				t.Fatalf("error should be exposable: %v", err)
			}
			if errors.CodeOf(err) != errors.CodeInvalidParam {
				t.Fatalf("wrong code: %d", errors.CodeOf(err))
			}
			if err.Error() != tt.wantReason {
				t.Fatalf("reason=%q want %q", err.Error(), tt.wantReason)
			}
		})
	}
}

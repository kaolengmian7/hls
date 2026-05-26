package game

import (
	"testing"

	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/errors"
)

func mkPlayer(id string, role protocol.Role, ready bool) protocol.Player {
	return protocol.Player{ID: id, Role: role, Ready: ready}
}

func TestCanAssignRole(t *testing.T) {
	players := []protocol.Player{
		mkPlayer("a", protocol.RoleEscort, false),
		mkPlayer("b", protocol.RoleEscort, false),
		mkPlayer("c", protocol.RoleBoss, false),
	}
	tests := []struct {
		name   string
		role   protocol.Role
		except string
		want   bool
	}{
		{"老板已满", protocol.RoleBoss, "", false},
		{"护航已满", protocol.RoleEscort, "", false},
		{"老板可重选自己的老板位", protocol.RoleBoss, "c", true},
		{"护航想换老板但老板满员", protocol.RoleBoss, "a", false},
		{"自己已是护航 仍允许换护航", protocol.RoleEscort, "a", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanAssignRole(players, tt.role, tt.except); got != tt.want {
				t.Errorf("CanAssignRole=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanStartGame(t *testing.T) {
	tests := []struct {
		name    string
		players []protocol.Player
		wantOK  bool
	}{
		{
			name: "全部满足",
			players: []protocol.Player{
				mkPlayer("a", protocol.RoleEscort, true),
				mkPlayer("b", protocol.RoleEscort, true),
				mkPlayer("c", protocol.RoleBoss, true),
			},
			wantOK: true,
		},
		{
			name: "人数不足",
			players: []protocol.Player{
				mkPlayer("a", protocol.RoleEscort, true),
				mkPlayer("b", protocol.RoleBoss, true),
			},
			wantOK: false,
		},
		{
			name: "角色错误",
			players: []protocol.Player{
				mkPlayer("a", protocol.RoleEscort, true),
				mkPlayer("b", protocol.RoleEscort, true),
				mkPlayer("c", protocol.RoleEscort, true),
			},
			wantOK: false,
		},
		{
			name: "未全部准备",
			players: []protocol.Player{
				mkPlayer("a", protocol.RoleEscort, true),
				mkPlayer("b", protocol.RoleEscort, false),
				mkPlayer("c", protocol.RoleBoss, true),
			},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CanStartGame(tt.players)
			gotOK := err == nil
			if gotOK != tt.wantOK {
				t.Errorf("CanStartGame ok=%v, want %v (err=%v)", gotOK, tt.wantOK, err)
			}
			if err != nil && errors.CodeOf(err) != errors.CodeStartCondition {
				t.Errorf("wrong error code: %v", errors.CodeOf(err))
			}
		})
	}
}

func TestInitialCoin(t *testing.T) {
	if InitialCoin(protocol.RoleEscort) != EscortInitialCoin {
		t.Errorf("escort initial coin")
	}
	if InitialCoin(protocol.RoleBoss) != BossInitialCoin {
		t.Errorf("boss initial coin")
	}
	if InitialCoin("UNKNOWN") != 0 {
		t.Errorf("unknown role should be 0")
	}
}

func TestNormalizeGuaranteedAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{"零回落到默认值", 0, DefaultGuaranteedAmount, false},
		{"下限", MinGuaranteedAmount, MinGuaranteedAmount, false},
		{"上限", MaxGuaranteedAmount, MaxGuaranteedAmount, false},
		{"常规默认", 2500, 2500, false},
		{"常规自定义", 5000, 5000, false},
		{"负数", -1, 0, true},
		{"超过上限", MaxGuaranteedAmount + 1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeGuaranteedAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeGuaranteedAmount err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("NormalizeGuaranteedAmount got=%d want=%d", got, tt.want)
			}
			if tt.wantErr {
				if !errors.IsExposable(err) {
					t.Fatalf("error should be exposable: %v", err)
				}
				if errors.CodeOf(err) != errors.CodeInvalidParam {
					t.Fatalf("wrong code: %d", errors.CodeOf(err))
				}
			}
		})
	}
}

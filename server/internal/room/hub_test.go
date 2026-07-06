package room

import (
	"context"
	"testing"

	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/pkg/code"
)

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gen := code.NewGenerator()
	return NewHub(ctx, gen, nil, logx.WithContext(ctx), nil, 1.0) // nil shopRepo = no DB; no VIP users; default boost
}

// TestQueryRoomMembership 覆盖回归判定：携带 playerId 查询时，
// 房主（已占座）应被识别为 member，陌生 playerId 与空串均不应。
func TestQueryRoomMembership(t *testing.T) {
	hub := newTestHub(t)
	host := "1"
	r, err := hub.CreateRoom("房主", "", host, 0, 0)
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	c := r.Code()

	tests := []struct {
		name       string
		playerID   string
		wantMember bool
		wantExists bool
	}{
		{"房主本人", host, true, true},
		{"陌生玩家", "99", false, true},
		{"空 playerId", "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := hub.QueryRoom(c, tt.playerID)
			if summary.Exists != tt.wantExists {
				t.Errorf("Exists got %v, want %v", summary.Exists, tt.wantExists)
			}
			if summary.IsMember != tt.wantMember {
				t.Errorf("IsMember got %v, want %v", summary.IsMember, tt.wantMember)
			}
		})
	}
}

// TestQueryRoomNotExists 不存在的房间应返回 Exists=false 且 IsMember=false。
func TestQueryRoomNotExists(t *testing.T) {
	hub := newTestHub(t)
	summary := hub.QueryRoom("000000", "42")
	if summary.Exists {
		t.Errorf("Exists got true, want false")
	}
	if summary.IsMember {
		t.Errorf("IsMember got true, want false")
	}
}

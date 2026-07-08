package room

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/protocol"
)

// wsTestPair creates a matched server/client websocket pair for tests.
func wsTestPair(t *testing.T) (server *websocket.Conn, client *websocket.Conn) {
	t.Helper()
	ch := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("ws upgrade: %v", err)
			return
		}
		ch <- conn
	}))
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cconn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("wsTestPair dial: %v", err)
	}
	sconn := <-ch
	t.Cleanup(func() { cconn.Close(); sconn.Close() })
	return sconn, cconn
}

// newTestRoom returns a room with no DB and a minimal Hub, suitable for direct method calls.
func newTestRoom(t *testing.T) *Room {
	t.Helper()
	ctx := context.Background()
	hub := &Hub{vipUsers: map[int64]struct{}{}, vipHighBoost: 1.0}
	return newRoom(hub, "123456", "host", "", "h1", 0, 0, nil, logx.WithContext(ctx))
}

// TestHandleLeaveClosesClientNotSlot verifies LEAVE closes the WS connection
// but does NOT remove the player from r.players.
func TestHandleLeaveClosesClientNotSlot(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	rm.players["h1"].Connected = true

	sconn, _ := wsTestPair(t)
	c := NewClient(sconn, rm, "h1", 0, logx.WithContext(ctx))
	rm.clients["h1"] = c

	rm.handleLeave(ctx, "h1")

	c.mu.Lock()
	closed := c.isClosed
	c.mu.Unlock()
	if !closed {
		t.Error("client should be closed after LEAVE")
	}
	if _, ok := rm.players["h1"]; !ok {
		t.Error("player slot should be preserved after LEAVE")
	}
}

// TestHandleClientGoneMarksOffline verifies that when WS drops, the slot
// stays in r.players with Connected=false.
func TestHandleClientGoneMarksOffline(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	rm.players["h1"].Connected = true

	sconn, _ := wsTestPair(t)
	c := NewClient(sconn, rm, "h1", 0, logx.WithContext(ctx))
	rm.clients["h1"] = c

	rm.handleClientGone(ctx, c)

	p, ok := rm.players["h1"]
	if !ok {
		t.Fatal("player slot must be preserved after disconnect")
	}
	if p.Connected {
		t.Error("player should be marked offline")
	}
	if _, stillInClients := rm.clients["h1"]; stillInClients {
		t.Error("client entry should be removed from r.clients")
	}
}

// drainSend discards all pending messages on a client's send channel.
func drainSend(c *Client) {
	for {
		select {
		case <-c.send:
		default:
			return
		}
	}
}

// TestHandleJoinSlotTakeover verifies that a new player joining when there is
// an offline slot inherits that slot's role, coin, ready, and isHost.
func TestHandleJoinSlotTakeover(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)

	// Set up offline host slot with game state
	rm.players["h1"].Role = protocol.RoleBoss
	rm.players["h1"].Coin = 7
	rm.players["h1"].Ready = true
	rm.players["h1"].Connected = false
	rm.players["h1"].IsHost = true

	// New player connects
	sconn, _ := wsTestPair(t)
	newClient := NewClient(sconn, rm, "p2", 2, logx.WithContext(ctx))

	payload, _ := json.Marshal(protocol.JoinPayload{Nickname: "新玩家"})
	rm.handleJoin(ctx, Intent{
		Type:    protocol.IntentJoin,
		Payload: payload,
		From:    newClient,
	})
	drainSend(newClient)

	// Old slot must be gone
	if _, ok := rm.players["h1"]; ok {
		t.Error("old playerID should be removed after takeover")
	}

	// New slot must exist with inherited state
	p, ok := rm.players["p2"]
	if !ok {
		t.Fatal("new player should occupy the taken-over slot")
	}
	if p.Role != protocol.RoleBoss {
		t.Errorf("role: got %v, want BOSS", p.Role)
	}
	if p.Coin != 7 {
		t.Errorf("coin: got %d, want 7", p.Coin)
	}
	if !p.Ready {
		t.Error("ready should be inherited (true)")
	}
	if !p.IsHost {
		t.Error("isHost should be inherited (true)")
	}
	if rm.hostID != "p2" {
		t.Errorf("hostID: got %q, want %q", rm.hostID, "p2")
	}
	// playerOrder must be updated
	found := false
	for _, pid := range rm.playerOrder {
		if pid == "p2" {
			found = true
		}
		if pid == "h1" {
			t.Error("old playerID should not remain in playerOrder")
		}
	}
	if !found {
		t.Error("new playerID should appear in playerOrder")
	}
}

// TestHandleJoinNoTakeoverWhenAllConnected verifies FULL rejection when every
// slot is occupied by a connected player.
func TestHandleJoinNoTakeoverWhenAllConnected(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	rm.players["h1"].Connected = true

	// Fill remaining slots with connected players
	for _, pid := range []string{"p2", "p3"} {
		rm.players[pid] = &protocol.Player{ID: pid, Connected: true}
		rm.states[pid] = &playerState{}
		rm.playerOrder = append(rm.playerOrder, pid)
	}

	sconn, _ := wsTestPair(t)
	newClient := NewClient(sconn, rm, "p4", 4, logx.WithContext(ctx))
	payload, _ := json.Marshal(protocol.JoinPayload{Nickname: "多余玩家"})

	rm.handleJoin(ctx, Intent{
		Type:    protocol.IntentJoin,
		Payload: payload,
		From:    newClient,
	})
	drainSend(newClient)

	if _, ok := rm.players["p4"]; ok {
		t.Error("full room should reject new player")
	}
}

// TestHandleJoinAllowsPlayingRoom verifies that new players can join a
// PLAYING room that has an offline slot.
func TestHandleJoinAllowsPlayingRoom(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	rm.status = protocol.RoomStatusPlaying
	rm.players["h1"].Connected = false // offline slot available

	sconn, _ := wsTestPair(t)
	newClient := NewClient(sconn, rm, "p2", 2, logx.WithContext(ctx))
	payload, _ := json.Marshal(protocol.JoinPayload{Nickname: "后来者"})

	rm.handleJoin(ctx, Intent{
		Type:    protocol.IntentJoin,
		Payload: payload,
		From:    newClient,
	})
	drainSend(newClient)

	if _, ok := rm.players["p2"]; !ok {
		t.Error("player should be able to join a PLAYING room via slot takeover")
	}
}

// TestHandleTickNoGC verifies that long-disconnected players and idle rooms
// are NOT cleaned up by the tick after removing the GC logic.
func TestHandleTickNoGC(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	// Host has been "disconnected" for 24h
	rm.players["h1"].Connected = false
	rm.states["h1"].disconnectedAt = time.Now().Add(-24 * 365 * time.Hour)
	rm.lastActivity = time.Now().Add(-24 * 365 * time.Hour)

	rm.handleTick(ctx)

	if _, ok := rm.players["h1"]; !ok {
		t.Error("player should NOT be removed by tick")
	}
	select {
	case <-rm.doneCh:
		t.Error("room should NOT be stopped by tick")
	default:
	}
}

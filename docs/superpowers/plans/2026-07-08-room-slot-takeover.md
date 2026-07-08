# Room Slot Takeover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rooms persist forever (closed only by 结单), players keep their slot on disconnect, new players take over offline slots inheriting role/coin/ready/isHost.

**Architecture:** Backend room loop changes: LEAVE marks offline instead of freeing slot; JOIN scans for offline slot first (takeover) before falling back to empty-slot join; tick GC and host-migration machinery removed entirely. Frontend removes all pre-join guards and the DISMISS_ROOM intent.

**Tech Stack:** Go 1.21, gorilla/websocket, Pinia/Vue 3, TypeScript

---

## File Map

| File | Change |
|------|--------|
| `server/internal/room/room.go` | `handleLeave` → close WS only; `handleJoin` → slot-takeover path; remove `migrateHost`, `removePlayer`, `hasNoConnectedClients`, tick GC; remove unused constants |
| `server/internal/room/room_test.go` | New — unit tests for leave/takeover/tick behaviour |
| `server/internal/handler/http.go` | `CreateRoomHandler` remove 409; `QueryRoomHandler` remove `activeRoomCode` |
| `src/games/shop/stores/room.ts` | `leaveRoom()` delete `DISMISS_ROOM` branch |
| `src/games/shop/net/api.ts` | `RoomSummary` remove `activeRoomCode` field |
| `src/games/shop/pages/ShopHomePage.vue` | `handleJoin` remove three pre-join guards + add cross-room leave; `handleCreate` remove 409 block |
| `src/games/shop/pages/RoomPage.vue` | Leave-dialog text; CLOSED-reason dialog shows "已结单" |
| `src/games/shop/pages/PlayPage.vue` | CLOSED-reason dialog shows "已结单" |

---

## Task 1 — Backend: LEAVE preserves slot

**Files:**
- Modify: `server/internal/room/room.go` (`handleLeave`)
- Create: `server/internal/room/room_test.go`

- [ ] **Step 1: Create test file with WS-pair helper and first two failing tests**

```go
// server/internal/room/room_test.go
package room

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

// newTestRoom returns a room with no DB and no hub, suitable for direct method calls.
func newTestRoom(t *testing.T) *Room {
	t.Helper()
	ctx := context.Background()
	return newRoom(nil, "123456", "host", "", "h1", 0, 0, nil, logx.WithContext(ctx))
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
```

- [ ] **Step 2: Run tests — expect FAIL (handleLeave currently calls removePlayer)**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run "TestHandleLeave|TestHandleClientGone" -v
```

Expected: `TestHandleLeaveClosesClientNotSlot` FAIL — player is removed by current code.

- [ ] **Step 3: Change `handleLeave` in `room.go`**

Replace the entire `handleLeave` function:

```go
func (r *Room) handleLeave(ctx context.Context, playerID string) {
	if c, ok := r.clients[playerID]; ok && c != nil {
		c.Close() // readPump exit triggers handleClientGone, which marks offline
	}
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run "TestHandleLeave|TestHandleClientGone" -v
```

- [ ] **Step 5: Run full test suite — no regressions**

```bash
cd /Users/kratos/hls/server && go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add server/internal/room/room.go server/internal/room/room_test.go
git commit -m "feat(room): LEAVE marks slot offline instead of removing it"
```

---

## Task 2 — Backend: handleJoin slot-takeover path

**Files:**
- Modify: `server/internal/room/room.go` (`handleJoin`)
- Modify: `server/internal/room/room_test.go` (add tests)

- [ ] **Step 1: Add failing tests for slot takeover**

Append to `room_test.go`:

```go
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
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run "TestHandleJoin" -v
```

Expected: `TestHandleJoinSlotTakeover` and `TestHandleJoinAllowsPlayingRoom` FAIL.

- [ ] **Step 3: Rewrite the new-player section of `handleJoin` in `room.go`**

Find the block starting with `// 全新玩家：对局已开始则不允许中途加入。` and replace everything from that line to the end of the function (the regular new-player join logic):

```go
	// 全新玩家：优先顶替离线位置，其次占用空位。
	// （已移除 PLAYING 状态拦截：游戏中同样允许顶替离线位）

	// 扫描第一个离线位置。
	var takeoverPID string
	for _, pid := range r.playerOrder {
		if pl, ok := r.players[pid]; ok && !pl.Connected {
			takeoverPID = pid
			break
		}
	}

	if takeoverPID != "" {
		// 顶替路径：继承角色、鼠币、ready、isHost。
		old := r.players[takeoverPID]
		newPlayer := &protocol.Player{
			ID:        playerID,
			Nickname:  p.Nickname,
			Avatar:    p.Avatar,
			Role:      old.Role,
			Coin:      old.Coin,
			Ready:     old.Ready,
			IsHost:    old.IsHost,
			Connected: true,
		}
		delete(r.players, takeoverPID)
		r.players[playerID] = newPlayer
		for i, pid := range r.playerOrder {
			if pid == takeoverPID {
				r.playerOrder[i] = playerID
				break
			}
		}
		if r.hostID == takeoverPID {
			r.hostID = playerID
		}
		delete(r.states, takeoverPID)
		r.states[playerID] = &playerState{}
		r.clients[playerID] = c
		c.hasJoined = true
		c.playerID = playerID
		c.nickname = p.Nickname
		c.avatar = p.Avatar
		delete(r.userIDs, takeoverPID)
		r.userIDs[playerID] = c.userID
		oldPID, newUID := takeoverPID, c.userID
		isHost2 := newPlayer.IsHost
		role2, coin2, ready2 := string(newPlayer.Role), newPlayer.Coin, newPlayer.Ready
		r.dbWriteAsync(func() error {
			bctx := context.Background()
			if err2 := r.shopRepo.RemovePlayer(bctx, r.code, oldPID); err2 != nil {
				return err2
			}
			if err2 := r.shopRepo.AddPlayer(bctx, r.code, newUID, playerID, isHost2); err2 != nil {
				return err2
			}
			return r.shopRepo.UpdatePlayerState(bctx, r.code, playerID, role2, coin2, ready2)
		})
		snap := r.buildSnapshot()
		c.SendEvent(protocol.EventJoinAccepted, protocol.JoinAcceptedPayload{
			Snapshot:     snap,
			SelfPlayerID: playerID,
		}, it.ReqID)
		r.broadcastSnapshot()
		logger.Infow("player took over slot",
			logx.Field("code", r.code),
			logx.Field("newPlayerId", playerID),
			logx.Field("oldPlayerId", oldPID))
		return
	}

	// 空位路径：无离线位置，尝试占用新位置。
	if len(r.players) >= game.MaxPlayers {
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason: protocol.RejectFull,
		}, it.ReqID)
		c.Close()
		return
	}

	player := &protocol.Player{
		ID:        playerID,
		Nickname:  p.Nickname,
		Avatar:    p.Avatar,
		Ready:     false,
		Coin:      0,
		IsHost:    false,
		Connected: true,
	}
	r.players[playerID] = player
	r.states[playerID] = &playerState{}
	r.playerOrder = append(r.playerOrder, playerID)
	r.clients[playerID] = c
	c.hasJoined = true
	c.playerID = playerID
	c.nickname = p.Nickname
	c.avatar = p.Avatar
	r.userIDs[playerID] = c.userID
	uid := c.userID
	pid := playerID
	r.dbWriteAsync(func() error {
		return r.shopRepo.AddPlayer(context.Background(), r.code, uid, pid, false)
	})

	r.appendFeed("PLAYER_JOINED", map[string]any{
		"playerId": playerID,
		"nickname": p.Nickname,
	})
	snap := r.buildSnapshot()
	c.SendEvent(protocol.EventJoinAccepted, protocol.JoinAcceptedPayload{
		Snapshot:     snap,
		SelfPlayerID: playerID,
	}, it.ReqID)
	r.broadcastExcept(playerID, protocol.EventPlayerJoined, protocol.PlayerJoinedPayload{Player: *player})
	r.broadcastSnapshot()
	logger.Infow("player joined",
		logx.Field("code", r.code),
		logx.Field("playerId", playerID))
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run "TestHandleJoin" -v
```

- [ ] **Step 5: Full suite**

```bash
cd /Users/kratos/hls/server && go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add server/internal/room/room.go server/internal/room/room_test.go
git commit -m "feat(room): new player takes over offline slot on join"
```

---

## Task 3 — Backend: Remove tick GC and dead code

**Files:**
- Modify: `server/internal/room/room.go`
- Modify: `server/internal/room/room_test.go`

- [ ] **Step 1: Add failing test for tick not removing players**

Append to `room_test.go`:

```go
// TestHandleTickNoGC verifies that long-disconnected players and idle rooms
// are NOT cleaned up by the tick after removing the GC logic.
func TestHandleTickNoGC(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	// Host has been "disconnected" for 24h
	rm.players["h1"].Connected = false
	rm.states["h1"].disconnectedAt = rm.states["h1"].disconnectedAt.Add(-24 * 365 * time.Hour)
	rm.lastActivity = rm.lastActivity.Add(-24 * 365 * time.Hour)

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
```

- [ ] **Step 2: Run test — expect FAIL (current tick removes players and closes rooms)**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run "TestHandleTickNoGC" -v
```

- [ ] **Step 3: Remove GC machinery from `room.go`**

**3a. Remove constants** — delete `hostMigrationGrace`, `playerDropGrace`, `idleRoomTimeout`, `tickInterval` from the `const` block. Keep `feedCapacity`.

Before:
```go
const (
	hostMigrationGrace = 8 * time.Hour
	playerDropGrace    = 8 * time.Hour
	idleRoomTimeout    = 8 * time.Hour
	tickInterval       = 30 * time.Second
	feedCapacity       = 50
)
```

After:
```go
const (
	feedCapacity = 50
)
```

**3b. Remove the `time` import** if it is now unused. Check the file — `time` is still used in `playerState.disconnectedAt` field and `shortEventID`. Keep the import.

Actually `playerState.disconnectedAt` is `time.Time` and is still set in `handleClientGone`. Keep `"time"` in imports.

**3c. Gut `handleTick`** — replace with empty body:

```go
func (r *Room) handleTick(ctx context.Context) {
	// GC removed: rooms never auto-close, slots never auto-drop.
}
```

**3d. Remove `migrateHost`** — delete the entire function:

```go
func (r *Room) migrateHost(ctx context.Context) {
	// ...entire function...
}
```

**3e. Remove `removePlayer`** — delete the entire function (no longer called anywhere).

**3f. Remove `hasNoConnectedClients`** — delete the entire function.

**3g. Remove ticker from `Run`** — the ticker is no longer useful (handleTick is a no-op). Remove:

```go
ticker := time.NewTicker(tickInterval)
defer ticker.Stop()
```

and the select case:

```go
case <-ticker.C:
    r.handleTick(ctx)
```

The `Run` function's select becomes:

```go
for {
    select {
    case <-ctx.Done():
        return
    case <-r.doneCh:
        return
    case reg := <-r.regCh:
        r.handleRegistration(ctx, reg)
    case it := <-r.inbox:
        r.dispatch(ctx, it)
    }
}
```

- [ ] **Step 4: Compile check**

```bash
cd /Users/kratos/hls/server && go build ./...
```

Fix any "declared and not used" or "undefined" errors.

- [ ] **Step 5: Run all room tests**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -v
```

- [ ] **Step 6: Full suite**

```bash
cd /Users/kratos/hls/server && go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add server/internal/room/room.go server/internal/room/room_test.go
git commit -m "feat(room): remove tick GC, migrateHost, removePlayer — rooms persist until 结单"
```

---

## Task 4 — Backend: HTTP handler cleanup

**Files:**
- Modify: `server/internal/handler/http.go`

- [ ] **Step 1: Remove 409 check from `CreateRoomHandler`**

Find and delete this block in `CreateRoomHandler`:

```go
activeCode, err := ctx.ShopRepo.FindActiveRoomForUser(r.Context(), userID)
if err != nil {
    writeError(w, r, http.StatusInternalServerError, "db error", err)
    return
}
if activeCode != "" {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusConflict)
    _ = json.NewEncoder(w).Encode(map[string]any{
        "ok": false, "message": "already_in_game", "activeRoomCode": activeCode,
    })
    return
}
```

- [ ] **Step 2: Remove `activeRoomCode` from `QueryRoomHandler`**

In `QueryRoomHandler`, delete the `QueryRoomResponse` struct and the block that populates `activeRoomCode`:

Delete the struct:
```go
type QueryRoomResponse struct {
	room.RoomSummary
	ActiveRoomCode string `json:"activeRoomCode,omitempty"`
}
```

Replace the handler response logic. Find:
```go
resp := QueryRoomResponse{RoomSummary: summary}
if !summary.IsMember && errKey == "" && ctx.ShopRepo != nil {
    if ac, err := ctx.ShopRepo.FindActiveRoomForUser(r.Context(), uid); err == nil && ac != "" && ac != c {
        resp.ActiveRoomCode = ac
    }
}
httpx.OkJsonCtx(r.Context(), w, resp)
```

Replace with:
```go
httpx.OkJsonCtx(r.Context(), w, summary)
```

- [ ] **Step 3: Compile check**

```bash
cd /Users/kratos/hls/server && go build ./...
```

- [ ] **Step 4: Run handler tests**

```bash
cd /Users/kratos/hls/server && go test ./internal/handler/... -v
```

- [ ] **Step 5: Full suite**

```bash
cd /Users/kratos/hls/server && go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add server/internal/handler/http.go
git commit -m "feat(handler): remove 409 room conflict check and activeRoomCode response"
```

---

## Task 5 — Frontend: leaveRoom and api.ts cleanup

**Files:**
- Modify: `src/games/shop/stores/room.ts`
- Modify: `src/games/shop/net/api.ts`

- [ ] **Step 1: Remove `DISMISS_ROOM` branch from `leaveRoom()` in `room.ts`**

Find in `room.ts`:

```ts
function leaveRoom(): void {
    if (snapshot.value && me.value?.isHost) {
      // 房主：通知服务端解散
      client?.send('DISMISS_ROOM', {});
    } else {
      client?.send('LEAVE', {});
    }
    teardownClient();
```

Replace with:

```ts
function leaveRoom(): void {
    teardownClient(); // destroy() inside sends LEAVE automatically
```

(`teardownClient` calls `client.destroy('STORE_TEARDOWN')`, and `GameClient.destroy()` already sends `LEAVE` before closing the WS.)

- [ ] **Step 2: Remove `activeRoomCode` from `RoomSummary` in `api.ts`**

Find in `api.ts`:

```ts
export type RoomSummary = {
  exists: boolean;
  status?: RoomStatus;
  playerCount?: number;
  /** 携带 playerId 查询时回填：该玩家当前是否仍在房间内（可凭口令回归恢复身份/鼠币）。 */
  isMember?: boolean;
  activeRoomCode?: string;
};
```

Replace with:

```ts
export type RoomSummary = {
  exists: boolean;
  status?: RoomStatus;
  playerCount?: number;
  isMember?: boolean;
};
```

- [ ] **Step 3: TypeScript compile check**

```bash
cd /Users/kratos/hls && npx vue-tsc --noEmit
```

Fix any type errors (there may be references to `activeRoomCode` in ShopHomePage.vue — they will be removed in Task 6).

- [ ] **Step 4: Commit**

```bash
git add src/games/shop/stores/room.ts src/games/shop/net/api.ts
git commit -m "feat(frontend): leaveRoom sends LEAVE via destroy, remove activeRoomCode type"
```

---

## Task 6 — Frontend: ShopHomePage join flow

**Files:**
- Modify: `src/games/shop/pages/ShopHomePage.vue`

- [ ] **Step 1: Update `handleJoin` — remove three guards, add cross-room leave**

Find the `handleJoin` function. Replace the body from after the initial `joining.value = true` guard to the `roomStore.connect` call:

**Before:**
```ts
  try {
    const summary = await queryRoom(codeInput.value);
    if (!summary.exists) {
      roomStore.pushToast('error', '这家商店不存在，或已经关闭');
      return;
    }

    if (summary.activeRoomCode && summary.activeRoomCode !== codeInput.value) {
      roomStore.pushToast('warn', `您已在商店 ${summary.activeRoomCode} 中，请先离开当前商店`);
      return;
    }

    const isMember = summary.isMember === true;
    if (!isMember) {
      if (summary.status === 'PLAYING') {
        roomStore.pushToast('warn', '商店已经开张，暂时不能中途加入');
        return;
      }
      if ((summary.playerCount ?? 0) >= 3) {
        roomStore.pushToast('warn', '店内 3 个位置已满');
        return;
      }
    }

    await roomStore.connect({
```

**After:**
```ts
  try {
    const summary = await queryRoom(codeInput.value);
    if (!summary.exists) {
      roomStore.pushToast('error', '这家商店不存在，或已经关闭');
      return;
    }

    // Leave current room (if any) before joining new one.
    if (roomStore.status === 'connected') {
      roomStore.leaveRoom();
    }

    await roomStore.connect({
```

- [ ] **Step 2: Remove 409 block from `handleCreate`**

Find and delete this block inside `handleCreate`'s catch:

```ts
    if (err instanceof ApiError && err.status === 409) {
      const body = err.body as { activeRoomCode?: string } | null;
      const ac = body?.activeRoomCode;
      if (ac) {
        roomStore.pushToast('warn', `您已在商店 ${ac} 中，请先离开当前商店`);
        codeInput.value = ac;
      } else {
        roomStore.pushToast('error', '您已在另一个商店中');
      }
      return;
    }
```

- [ ] **Step 3: Remove unused `ApiError` import if no longer referenced**

Check the top of the file. If `ApiError` is now only referenced in the catch for `handleCreate` (which still uses it for the generic error message), keep it. If the only 409 block was removed, verify `ApiError` is still used by the generic catch:

```ts
    const message =
      err instanceof ApiError ? err.message : (err as Error).message;
```

`ApiError` is still used here — keep the import.

- [ ] **Step 4: TypeScript compile check**

```bash
cd /Users/kratos/hls && npx vue-tsc --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add src/games/shop/pages/ShopHomePage.vue
git commit -m "feat(shop): remove pre-join guards, allow cross-room join"
```

---

## Task 7 — Frontend: Dialog text updates

**Files:**
- Modify: `src/games/shop/pages/RoomPage.vue`
- Modify: `src/games/shop/pages/PlayPage.vue`

- [ ] **Step 1: Update `RoomPage.vue` — leave-dialog text**

Find the leave `ConfirmDialog` in the template:

```vue
<ConfirmDialog
    :open="showLeaveConfirm"
    :title="isHost ? '关闭商店？' : '离开商店？'"
    :message="
      isHost
        ? '您是房主，离开后这家神秘商店会立刻关闭。'
        : '离开后仍可凭商店口令再次进入。'
    "
    :confirm-text="isHost ? '关闭' : '离开'"
    danger
    @confirm="confirmLeave"
    @cancel="showLeaveConfirm = false"
/>
```

Replace with:

```vue
<ConfirmDialog
    :open="showLeaveConfirm"
    title="离开商店？"
    message="离开后您的位置将保留，可以凭口令重新回来。"
    confirm-text="离开"
    danger
    @confirm="confirmLeave"
    @cancel="showLeaveConfirm = false"
/>
```

- [ ] **Step 2: Update `RoomPage.vue` — room-closed dialog text for CLOSED reason**

Find the host-closed `ConfirmDialog` in the template:

```vue
<ConfirmDialog
    :open="showHostClosedDialog"
    title="商店已关闭"
    :message="`商店已关闭（${hostClosedReason || 'CLOSED'}），3 秒后返回首页。`"
    confirm-text="返回首页"
    cancel-text="稍候"
    @confirm="dismissHostClosed"
    @cancel="dismissHostClosed"
/>
```

Replace with:

```vue
<ConfirmDialog
    :open="showHostClosedDialog"
    :title="hostClosedReason === 'CLOSED' ? '神秘商店已结单' : '商店已关闭'"
    :message="hostClosedReason === 'CLOSED'
      ? '神秘商店已结单，3 秒后返回主页。'
      : `商店已关闭（${hostClosedReason || 'CLOSED'}），3 秒后返回首页。`"
    confirm-text="返回主页"
    cancel-text="稍候"
    @confirm="dismissHostClosed"
    @cancel="dismissHostClosed"
/>
```

- [ ] **Step 3: Update `PlayPage.vue` — closed dialog text for CLOSED reason**

Find the `ConfirmDialog` for closed state in `PlayPage.vue`:

```vue
<ConfirmDialog
    :open="showClosedDialog"
    title="连接已断开"
    :message="`商店已关闭（${closedReason || 'CLOSED'}），3 秒后返回首页。`"
    confirm-text="返回首页"
    cancel-text="稍候"
    @confirm="dismissClosed"
    @cancel="dismissClosed"
/>
```

Replace with:

```vue
<ConfirmDialog
    :open="showClosedDialog"
    :title="closedReason === 'CLOSED' ? '神秘商店已结单' : '连接已断开'"
    :message="closedReason === 'CLOSED'
      ? '神秘商店已结单，3 秒后返回主页。'
      : `商店已关闭（${closedReason || 'CLOSED'}），3 秒后返回首页。`"
    confirm-text="返回主页"
    cancel-text="稍候"
    @confirm="dismissClosed"
    @cancel="dismissClosed"
/>
```

- [ ] **Step 4: TypeScript compile check**

```bash
cd /Users/kratos/hls && npx vue-tsc --noEmit
```

- [ ] **Step 5: Commit**

```bash
git add src/games/shop/pages/RoomPage.vue src/games/shop/pages/PlayPage.vue
git commit -m "feat(ui): show '神秘商店已结单' dialog on CLOSE_GAME, update leave dialog text"
```

---

## Self-Review Notes

**Spec coverage check:**

| Spec requirement | Task |
|-----------------|------|
| LEAVE preserves slot | Task 1 |
| Slot takeover on join (inherit role/coin/ready/isHost) | Task 2 |
| PLAYING rooms joinable | Task 2 (removed PLAYING guard) |
| Remove tick GC / migrateHost | Task 3 |
| Remove 409 in CreateRoom | Task 4 |
| Remove activeRoomCode in QueryRoom | Task 4 |
| leaveRoom() no DISMISS_ROOM | Task 5 |
| Remove activeRoomCode from RoomSummary type | Task 5 |
| handleJoin pre-join guards removed | Task 6 |
| Cross-room switch: leave then join | Task 6 |
| handleCreate 409 block removed | Task 6 |
| Leave dialog text update | Task 7 |
| CLOSED dialog shows "已结单" | Task 7 |

**Type consistency:** `protocol.JoinPayload`, `protocol.RoleBoss`, `protocol.RoomStatusPlaying` all referenced in Task 2 tests — these are defined in `server/internal/protocol/`. `RoomSummary` field removed in Task 5 matches the reference removed in Task 6. ✓

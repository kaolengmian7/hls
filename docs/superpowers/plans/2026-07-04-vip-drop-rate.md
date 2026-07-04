# VIP Account Drop Rate Boost Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a VIP user (configurable list) plays as BOSS in the mystery shop, all `Rarity=HIGH` item weights are multiplied by a configurable factor (default 5), dramatically increasing their drop rate.

**Architecture:** Config layer feeds a VIP user set into `Hub` → `Room`. `game.DrawLottery` gains a `highBoost float64` parameter; `handleDrawLottery` looks up the drawing player's userID and passes the boost factor. A pure `applyHighBoost` function handles the weight multiplication without mutating the shared pool.

**Tech Stack:** Go, go-zero config (yaml tags), `crypto/rand`-based weighted selection.

---

## File Map

| File | Change |
|------|--------|
| `server/internal/config/config.go` | Add `VIPUsers []int64`, `VIPHighBoost float64` to `Config` |
| `server/etc/gameserver.yaml` | Add `VIPUsers` and `VIPHighBoost` entries |
| `server/internal/game/lottery.go` | Add `applyHighBoost`; add `highBoost` param to `DrawLottery` |
| `server/internal/game/lottery_test.go` | Add `TestApplyHighBoost`, `TestDrawLottery_VIPBoost`; fix existing callers |
| `server/internal/room/hub.go` | Add `vipUsers map[int64]struct{}`, `vipHighBoost float64` to `Hub`; update `NewHub` |
| `server/internal/room/room.go` | Add same fields to `Room`; copy from hub in `newRoom`; boost lookup in `handleDrawLottery` |
| `server/internal/svc/service_context.go` | Pass VIP config to `room.NewHub` |

---

## Task 1: Add VIP fields to Config

**Files:**
- Modify: `server/internal/config/config.go`
- Modify: `server/etc/gameserver.yaml`

- [ ] **Step 1: Add fields to Config struct**

In `server/internal/config/config.go`, add to the `Config` struct after `TencentSMS TencentSMSConfig`:

```go
Shop ShopConfig
```

And add the new type at the bottom of the file:

```go
// ShopConfig 神秘商店专项配置。
type ShopConfig struct {
	VIPUsers     []int64 `json:",optional"`
	VIPHighBoost float64 `json:",default=5"`
}
```

- [ ] **Step 2: Add entries to gameserver.yaml**

In `server/etc/gameserver.yaml`, append after the `TencentSMS` block:

```yaml
Shop:
  VIPUsers:
    - 900006000
  VIPHighBoost: 5
```

- [ ] **Step 3: Verify config compiles**

```bash
cd server && go build ./internal/config/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add server/internal/config/config.go server/etc/gameserver.yaml
git commit -m "feat(config): add Shop.VIPUsers and Shop.VIPHighBoost"
```

---

## Task 2: Add `applyHighBoost` and update `DrawLottery` signature

**Files:**
- Modify: `server/internal/game/lottery.go`
- Modify: `server/internal/game/lottery_test.go`

- [ ] **Step 1: Write failing tests**

Add the following two test functions to `server/internal/game/lottery_test.go` (append after the existing functions):

```go
func TestApplyHighBoost(t *testing.T) {
	pool := []LotteryItem{
		{ID: "h1", Name: "H1", Rarity: rarityHigh, Value: 100, Weight: 2.0},
		{ID: "m1", Name: "M1", Rarity: rarityMedium, Value: 50, Weight: 6.0},
		{ID: "l1", Name: "L1", Rarity: rarityLow, Value: 10, Weight: 6.0},
	}
	boosted := applyHighBoost(pool, 5.0)

	// HIGH weight multiplied
	if boosted[0].Weight != 10.0 {
		t.Errorf("HIGH weight got %f, want 10.0", boosted[0].Weight)
	}
	// MEDIUM unchanged
	if boosted[1].Weight != 6.0 {
		t.Errorf("MEDIUM weight got %f, want 6.0", boosted[1].Weight)
	}
	// LOW unchanged
	if boosted[2].Weight != 6.0 {
		t.Errorf("LOW weight got %f, want 6.0", boosted[2].Weight)
	}
	// original pool not mutated
	if pool[0].Weight != 2.0 {
		t.Errorf("original HIGH weight mutated, got %f", pool[0].Weight)
	}
}

func TestDrawLottery_VIPBoost(t *testing.T) {
	// With a huge boost, HIGH items must dominate.
	const runs = 1000
	highCount := 0
	for i := 0; i < runs; i++ {
		records, _, err := DrawLottery("p1", 1, 0, 1000.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if records[0].Item.Rarity == rarityHigh {
			highCount++
		}
	}
	ratio := float64(highCount) / runs
	if ratio < 0.95 {
		t.Errorf("HIGH ratio=%.2f with boost=1000, want >= 0.95", ratio)
	}
}
```

- [ ] **Step 2: Run tests — expect compile failure**

```bash
cd server && go test ./internal/game/... 2>&1 | head -20
```

Expected: compile error — `DrawLottery` called with wrong number of arguments and `applyHighBoost` undefined.

- [ ] **Step 3: Add `applyHighBoost` to lottery.go**

Add after the `rollOne` function in `server/internal/game/lottery.go`:

```go
// applyHighBoost returns a new pool slice with HIGH item weights multiplied by
// factor. MEDIUM and LOW items are unchanged. The original slice is not mutated.
func applyHighBoost(src []LotteryItem, factor float64) []LotteryItem {
	out := make([]LotteryItem, len(src))
	copy(out, src)
	for i := range out {
		if out[i].Rarity == rarityHigh {
			out[i].Weight *= factor
		}
	}
	return out
}
```

- [ ] **Step 4: Update `DrawLottery` signature and body**

Replace the `DrawLottery` function signature and the `rollOne` call inside it.

Old signature:
```go
func DrawLottery(playerID string, times int, currentGuarantee int) ([]protocol.LotteryRecord, int, error) {
```

New signature:
```go
// highBoost is the weight multiplier for HIGH-rarity items. Pass 1.0 for no
// boost. Values > 1 increase the probability of HIGH items proportionally.
func DrawLottery(playerID string, times int, currentGuarantee int, highBoost float64) ([]protocol.LotteryRecord, int, error) {
```

Inside `DrawLottery`, replace the line:
```go
	item, err := rollOne(lotteryPool)
```
with:
```go
	pool := lotteryPool
	if highBoost != 1.0 {
		pool = applyHighBoost(lotteryPool, highBoost)
	}
	item, err := rollOne(pool)
```

- [ ] **Step 5: Fix existing test callers**

In `server/internal/game/lottery_test.go`, update every existing call to `DrawLottery` to pass `1.0` as the last argument.

The lines to update (search for `DrawLottery`):

```go
// TestDrawLotterySingle — line ~37
records, newGuarantee, err := DrawLottery("p1", 1, current, 1.0)

// TestDrawLotteryInvalidTimes — two occurrences
records, guarantee, err := DrawLottery("p1", tt.times, 2500, 1.0)

// TestDrawLotteryRollbackOnOverflow — line ~99
records, guarantee, err := DrawLottery("p1", 1, MaxGuaranteedAmount, 1.0)

// TestDrawLotteryPropagatesImageURL — line ~174
records, _, err := DrawLottery("player-test", 1, 0, 1.0)
```

- [ ] **Step 6: Run all game tests — expect PASS**

```bash
cd server && go test ./internal/game/... -v -count=1
```

Expected: all tests PASS including the two new ones.

- [ ] **Step 7: Commit**

```bash
git add server/internal/game/lottery.go server/internal/game/lottery_test.go
git commit -m "feat(game): add applyHighBoost; DrawLottery accepts highBoost param"
```

---

## Task 3: Add VIP fields to Hub and wire config

**Files:**
- Modify: `server/internal/room/hub.go`
- Modify: `server/internal/svc/service_context.go`

- [ ] **Step 1: Add fields to Hub struct**

In `server/internal/room/hub.go`, add two fields to the `Hub` struct after `logger logx.Logger`:

```go
vipUsers     map[int64]struct{}
vipHighBoost float64
```

- [ ] **Step 2: Update NewHub to accept and store VIP config**

Replace the `NewHub` function:

```go
// NewHub 创建 Hub。vipUsers 是专属高爆率账号集合；vipHighBoost 是 HIGH 物品权重倍数。
func NewHub(ctx context.Context, gen *code.Generator, tokens TokenIssuer, shopRepo *repo.ShopRepo, logger logx.Logger, vipUserIDs []int64, vipHighBoost float64) *Hub {
	vipSet := make(map[int64]struct{}, len(vipUserIDs))
	for _, uid := range vipUserIDs {
		vipSet[uid] = struct{}{}
	}
	return &Hub{
		ctx:          ctx,
		generator:    gen,
		tokens:       tokens,
		shopRepo:     shopRepo,
		logger:       logger,
		vipUsers:     vipSet,
		vipHighBoost: vipHighBoost,
		rooms:        map[string]*Room{},
	}
}
```

- [ ] **Step 3: Update service_context.go to pass VIP config**

In `server/internal/svc/service_context.go`, find the `NewHub` call:

```go
hub := room.NewHub(rootCtx, gen, signer, shopRepo, logx.WithContext(rootCtx))
```

Replace with:

```go
hub := room.NewHub(rootCtx, gen, signer, shopRepo, logx.WithContext(rootCtx), cfg.Shop.VIPUsers, cfg.Shop.VIPHighBoost)
```

- [ ] **Step 4: Verify compilation**

```bash
cd server && go build ./...
```

Expected: no errors. (Hub callers outside svc — like tests — will fail if they call `NewHub` directly; check with the command and fix any remaining callers.)

- [ ] **Step 5: Fix any remaining NewHub callers in tests**

```bash
cd server && grep -rn "NewHub(" --include="*.go" .
```

If any test files call `NewHub`, update them to pass two extra arguments: `nil, 1.0` (no VIP users, no boost).

- [ ] **Step 6: Verify all tests still compile and pass**

```bash
cd server && go test ./... 2>&1 | tail -20
```

Expected: all tests PASS (or SKIP if DB-dependent).

- [ ] **Step 7: Commit**

```bash
git add server/internal/room/hub.go server/internal/svc/service_context.go
git commit -m "feat(hub): accept VIP user set and highBoost; wire from config"
```

---

## Task 4: Add VIP boost lookup to Room and handleDrawLottery

**Files:**
- Modify: `server/internal/room/room.go`

- [ ] **Step 1: Add fields to Room struct**

In `server/internal/room/room.go`, add two fields to the `Room` struct after `userIDs map[string]int64`:

```go
vipUsers     map[int64]struct{} // read-only after newRoom; shared reference from Hub
vipHighBoost float64
```

- [ ] **Step 2: Initialize fields in newRoom**

In `newRoom`, inside the `r := &Room{...}` literal, add after `userIDs: map[string]int64{hostPlayerID: hostUserID},`:

```go
vipUsers:          hub.vipUsers,
vipHighBoost:      hub.vipHighBoost,
```

- [ ] **Step 3: Update handleDrawLottery to compute boost**

In `handleDrawLottery`, find the existing call to `game.DrawLottery`:

```go
records, newGuarantee, err := game.DrawLottery(pid, p.Times, guaranteeBefore)
```

Replace with:

```go
boost := 1.0
if uid, ok := r.userIDs[pid]; ok {
    if _, isVIP := r.vipUsers[uid]; isVIP {
        boost = r.vipHighBoost
    }
}
records, newGuarantee, err := game.DrawLottery(pid, p.Times, guaranteeBefore, boost)
```

- [ ] **Step 4: Build the whole server**

```bash
cd server && go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run all tests**

```bash
cd server && go test ./... 2>&1 | tail -20
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/room/room.go
git commit -m "feat(room): apply VIP high-boost when BOSS draws lottery"
```

---

## Self-Review Checklist

- [x] **Spec §1 Config** → Task 1 covers `VIPUsers`, `VIPHighBoost` in config struct and yaml.
- [x] **Spec §2 DrawLottery signature** → Task 2 adds `highBoost float64` param and `applyHighBoost`.
- [x] **Spec §2 applyHighBoost** → Task 2 Step 3 implements it as a pure function; Task 2 Step 5 (existing tests) pass `1.0`.
- [x] **Spec §3 Room VIP lookup** → Task 4 adds fields and the boost lookup in `handleDrawLottery`.
- [x] **Spec §4 Hub transparent pass-through** → Task 3 wires config → Hub → Room.
- [x] **Spec Tests** → Task 2 adds `TestApplyHighBoost` and `TestDrawLottery_VIPBoost`.
- [x] **No placeholders** → every step includes exact code.
- [x] **Type consistency** → `applyHighBoost` defined in Task 2 Step 3, referenced in Task 2 Step 4 — names match. `vipUsers map[int64]struct{}` used consistently across Hub (Task 3) and Room (Task 4).

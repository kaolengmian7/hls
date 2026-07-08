# 神秘商店结单弹窗 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 房主点击"结单"后，服务端先广播 `GAME_CLOSED` 事件，所有玩家收到后立刻弹出"神秘商店已结单"对话框，点击"确认回到主页"才跳首页。

**Architecture:** 服务端新增 `GAME_CLOSED` 事件类型，在 `handleCloseGame` 踢人前广播；前端 `room.ts` 监听该事件并置 `gameClosed` 为 true；`PlayPage.vue` 监听 `gameClosed`，弹出仅有确认按钮且背板不可关闭的对话框。

**Tech Stack:** Go (gorilla/websocket), Vue 3, TypeScript, Pinia

---

### Task 1: 后端 — 新增 GAME_CLOSED 事件类型

**Files:**
- Modify: `server/internal/protocol/event.go`

- [ ] **Step 1: 在 event.go 末尾的常量块中添加 EventGameClosed**

打开 `server/internal/protocol/event.go`，在 `EventPong = "PONG"` 之后新增一行：

```go
const (
	EventJoinAccepted       = "JOIN_ACCEPTED"
	EventJoinRejected       = "JOIN_REJECTED"
	EventStateSnapshot      = "STATE_SNAPSHOT"
	EventPlayerJoined       = "PLAYER_JOINED"
	EventPlayerLeft         = "PLAYER_LEFT"
	EventRoleChanged        = "ROLE_CHANGED"
	EventReadyChanged       = "READY_CHANGED"
	EventHostChanged        = "HOST_CHANGED"
	EventGameStarted        = "GAME_STARTED"
	EventCoinChanged        = "COIN_CHANGED"
	EventEquipmentPurchased = "EQUIPMENT_PURCHASED"
	EventLotteryDrawn       = "LOTTERY_DRAWN"
	EventGuaranteeChanged   = "GUARANTEE_CHANGED"
	EventRoundSettled       = "ROUND_SETTLED"
	EventActionRejected     = "ACTION_REJECTED"
	EventKick               = "KICK"
	EventPong               = "PONG"
	EventGameClosed         = "GAME_CLOSED"
)
```

- [ ] **Step 2: 在 event.go 末尾添加 GameClosedPayload 结构体**

在 `KickPayload` 结构体之后追加：

```go
// GameClosedPayload 结单通知；在 KICK 之前广播，让所有客户端先弹弹窗。
type GameClosedPayload struct{}
```

- [ ] **Step 3: 编译验证**

```bash
cd /Users/kratos/hls/server && go build ./...
```

Expected: 无错误输出。

- [ ] **Step 4: Commit**

```bash
git add server/internal/protocol/event.go
git commit -m "feat(protocol): add GAME_CLOSED event type"
```

---

### Task 2: 后端 — handleCloseGame 先广播 GAME_CLOSED 再踢人

**Files:**
- Modify: `server/internal/room/room.go`
- Test: `server/internal/room/room_test.go`

- [ ] **Step 1: 先写失败测试**

在 `room_test.go` 末尾追加：

```go
// TestHandleCloseGameBroadcastsGameClosedBeforeKick verifies that GAME_CLOSED
// is sent to all clients before the KICK message when the host closes the game.
func TestHandleCloseGameBroadcastsGameClosedBeforeKick(t *testing.T) {
	ctx := context.Background()
	rm := newTestRoom(t)
	rm.status = protocol.RoomStatusPlaying
	rm.players["h1"].Connected = true

	sconn, _ := wsTestPair(t)
	c := NewClient(sconn, rm, "h1", 0, logx.WithContext(ctx))
	rm.clients["h1"] = c

	rm.handleCloseGame(ctx, Intent{Type: protocol.IntentCloseGame, From: c})

	// Read two messages directly from the buffered send channel.
	// writePump is never started in tests, so messages stay buffered.
	var m1, m2 map[string]any
	if err := json.Unmarshal(<-c.send, &m1); err != nil {
		t.Fatalf("parse msg1: %v", err)
	}
	if err := json.Unmarshal(<-c.send, &m2); err != nil {
		t.Fatalf("parse msg2: %v", err)
	}

	if got := m1["type"]; got != protocol.EventGameClosed {
		t.Errorf("first message: got %q, want %q", got, protocol.EventGameClosed)
	}
	if got := m2["type"]; got != protocol.EventKick {
		t.Errorf("second message: got %q, want %q", got, protocol.EventKick)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run TestHandleCloseGameBroadcastsGameClosedBeforeKick -v
```

Expected: FAIL，因为 GAME_CLOSED 尚未广播，第一条消息是 KICK。

- [ ] **Step 3: 在 handleCloseGame 中添加广播**

定位 `room.go` 的 `handleCloseGame`（约第 1035 行），在 `for _, c := range r.clients` 循环之前插入一行：

```go
func (r *Room) handleCloseGame(ctx context.Context, it Intent) {
	if it.From.playerID != r.hostID {
		r.rejectAction(it.From, it.ReqID, "FORBIDDEN", "仅房主可结单")
		return
	}
	if r.status != protocol.RoomStatusPlaying {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "结单仅在游戏进行中有效")
		return
	}
	logx.WithContext(ctx).Infow("close game", logx.Field("code", r.code), logx.Field("by", it.From.playerID))
	r.broadcast(protocol.EventGameClosed, protocol.GameClosedPayload{})
	for _, c := range r.clients {
		if c == nil {
			continue
		}
		c.SendEvent(protocol.EventKick, protocol.KickPayload{Reason: protocol.KickClosed}, "")
		c.Close()
	}
	r.stop()
}
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
cd /Users/kratos/hls/server && go test ./internal/room/... -run TestHandleCloseGameBroadcastsGameClosedBeforeKick -v
```

Expected: PASS。

- [ ] **Step 5: 跑全量测试，确认无回归**

```bash
cd /Users/kratos/hls/server && go test ./...
```

Expected: 全部 PASS。

- [ ] **Step 6: Commit**

```bash
git add server/internal/room/room.go server/internal/room/room_test.go
git commit -m "feat(room): broadcast GAME_CLOSED before KICK on handleCloseGame"
```

---

### Task 3: 前端 — protocol.ts 注册 GAME_CLOSED

**Files:**
- Modify: `src/games/shop/net/protocol.ts`

- [ ] **Step 1: 在 EventMap 中添加 GAME_CLOSED**

打开 `src/games/shop/net/protocol.ts`，在 `EventMap` 类型定义（约第 184 行）内的 `PONG: Record<string, never>` 之后添加：

```ts
export type EventMap = {
  JOIN_ACCEPTED: { snapshot: RoomSnapshot; selfPlayerId: string };
  JOIN_REJECTED: { reason: JoinRejectReason; message?: string };
  STATE_SNAPSHOT: { snapshot: RoomSnapshot };
  PLAYER_JOINED: { player: Player };
  PLAYER_LEFT: { playerId: string };
  ROLE_CHANGED: { playerId: string; role: Role | null };
  READY_CHANGED: { playerId: string; ready: boolean };
  HOST_CHANGED: { newHostId: string };
  GAME_STARTED: { snapshot: RoomSnapshot };
  COIN_CHANGED: { changes: Array<{ playerId: string; delta: number; balance: number }> };
  EQUIPMENT_PURCHASED: { playerId: string; equipmentId: string; pick?: string };
  LOTTERY_DRAWN: { records: LotteryRecord[] };
  GUARANTEE_CHANGED: { before: number; after: number };
  ROUND_SETTLED: { record: SettlementRecord };
  ACTION_REJECTED: { reqId?: string; reason: string; message?: string };
  KICK: { reason: KickReason };
  PONG: Record<string, never>;
  GAME_CLOSED: Record<string, never>;
};
```

- [ ] **Step 2: 类型检查**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: 无 TypeScript 错误。

- [ ] **Step 3: Commit**

```bash
git add src/games/shop/net/protocol.ts
git commit -m "feat(protocol): add GAME_CLOSED to EventMap"
```

---

### Task 4: 前端 — room store 处理 GAME_CLOSED 事件

**Files:**
- Modify: `src/games/shop/stores/room.ts`

- [ ] **Step 1: 在 store 状态区新增 gameClosed ref**

打开 `src/games/shop/stores/room.ts`，在 `const closedReason = ref<string>('');`（约第 54 行）之后添加：

```ts
const gameClosed = ref(false);
```

- [ ] **Step 2: 在 handleEvent 的 switch 中添加 GAME_CLOSED case**

在 `handleEvent` 函数（约第 202 行）的 `switch (type)` 里，`default: break;` 之前添加：

```ts
case 'GAME_CLOSED':
  gameClosed.value = true;
  break;
```

完整的 switch 结构如下（只展示变更部分，其他 case 不变）：

```ts
switch (type) {
  // ... 已有的 case ...
  case 'GAME_CLOSED':
    gameClosed.value = true;
    break;
  default:
    break;
}
```

- [ ] **Step 3: 在 leaveRoom 中重置 gameClosed**

在 `leaveRoom` 函数（约第 181 行）的 `closedReason.value = '';` 之后添加：

```ts
function leaveRoom(): void {
  teardownClient();
  snapshot.value = null;
  latestLotteryReveal.value = [];
  awaitingSelfLotteryReveal = false;
  heldLotteryFeedIds.value = new Set<string>();
  code.value = '';
  selfPlayerId.value = '';
  status.value = 'idle';
  closedReason.value = '';
  gameClosed.value = false;
}
```

- [ ] **Step 4: 在 return 中 expose gameClosed**

在 `return { ... }` 块（约第 306 行）中的 `closedReason,` 之后添加：

```ts
return {
  snapshot,
  selfPlayerId,
  code,
  status,
  closedReason,
  gameClosed,        // ← 新增
  toasts,
  // ... 其余不变 ...
};
```

- [ ] **Step 5: 类型检查**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: 无错误。

- [ ] **Step 6: Commit**

```bash
git add src/games/shop/stores/room.ts
git commit -m "feat(store): handle GAME_CLOSED event, expose gameClosed ref"
```

---

### Task 5: 前端 — ConfirmDialog 新增 persistent 和 noCancel props

**Files:**
- Modify: `src/shared/components/ConfirmDialog.vue`

- [ ] **Step 1: 替换 script setup 的 defineProps**

将 `src/shared/components/ConfirmDialog.vue` 中的 `defineProps` 替换为：

```ts
defineProps<{
  open: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  /** 为 true 时背板点击不触发 cancel */
  persistent?: boolean;
  /** 为 true 时隐藏取消按钮 */
  noCancel?: boolean;
}>();
```

- [ ] **Step 2: 修改背板的 @click.self**

将模板中 overlay div 的 `@click.self="emit('cancel')"` 改为：

```html
@click.self="!persistent && emit('cancel')"
```

- [ ] **Step 3: 为取消按钮添加 v-if**

将取消按钮改为：

```html
<button v-if="!noCancel" class="btn-ghost px-4 py-2" @click="emit('cancel')">
  {{ cancelText ?? '稍候' }}
</button>
```

完整更新后的模板如下供参考：

```vue
<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      @click.self="!persistent && emit('cancel')"
    >
      <div class="card rpg-corners w-[92%] max-w-sm p-7 animate-slide-up text-center">
        <div class="rune-mark text-2xl text-gold-300 animate-flicker">⚜</div>
        <h3 class="rpg-title text-xl mt-1 mb-2">{{ title }}</h3>
        <div class="mx-auto max-w-[60%] rpg-divider mb-3" />
        <p class="font-body text-sm text-parchment-200/90 leading-relaxed">
          {{ message }}
        </p>
        <div class="mt-6 flex gap-3 justify-center">
          <button v-if="!noCancel" class="btn-ghost px-4 py-2" @click="emit('cancel')">
            {{ cancelText ?? '稍候' }}
          </button>
          <button :class="danger ? 'btn-danger' : 'btn-primary'" @click="emit('confirm')">
            {{ confirmText ?? '确认' }}
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>
```

- [ ] **Step 4: 类型检查**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: 无错误。

- [ ] **Step 5: Commit**

```bash
git add src/shared/components/ConfirmDialog.vue
git commit -m "feat(ui): add persistent and noCancel props to ConfirmDialog"
```

---

### Task 6: 前端 — PlayPage 添加 GAME_CLOSED 弹窗

**Files:**
- Modify: `src/games/shop/pages/PlayPage.vue`

- [ ] **Step 1: 新增 showGameClosedDialog ref**

在 script setup 的 ref 声明区（约第 22 行），在 `const showClosedDialog = ref(false);` 之后添加：

```ts
const showGameClosedDialog = ref(false);
```

- [ ] **Step 2: 给现有的 status watch 加守卫，防止双弹窗**

GAME_CLOSED 之后服务端还会发 KICK，KICK 会让 `roomStore.status` 变为 `'closed'`，从而触发现有的 closed 弹窗（会和 GAME_CLOSED 弹窗同时出现）。在现有 watch 条件里加一个守卫：

将约第 72 行的：

```ts
watch(
  () => roomStore.status,
  (status) => {
    if (status === 'closed' && !restoring.value) {
```

改为：

```ts
watch(
  () => roomStore.status,
  (status) => {
    if (status === 'closed' && !restoring.value && !showGameClosedDialog.value) {
```

- [ ] **Step 3: 新增监听 roomStore.gameClosed 的 watch**

在修改后的 `watch(() => roomStore.status, ...)` 之后添加：

```ts
watch(
  () => roomStore.gameClosed,
  (closed) => {
    if (closed) showGameClosedDialog.value = true;
  },
);
```

- [ ] **Step 4: 新增 handleGameClosed 函数**

在 `handleCloseGame` 函数（约第 89 行）之后添加：

```ts
function handleGameClosed(): void {
  showGameClosedDialog.value = false;
  roomStore.leaveRoom();
  router.replace({ name: 'home' });
}
```

- [ ] **Step 5: 在模板中添加 GAME_CLOSED 弹窗**

在现有 `</ConfirmDialog>` 闭合标签（约第 298 行）之后、`</div>` 之前添加：

```html
<ConfirmDialog
  :open="showGameClosedDialog"
  title="神秘商店已结单"
  message="商店已结束营业，感谢参与！"
  confirm-text="确认回到主页"
  persistent
  no-cancel
  @confirm="handleGameClosed"
/>
```

- [ ] **Step 6: 类型检查**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: 无错误。

- [ ] **Step 7: Commit**

```bash
git add src/games/shop/pages/PlayPage.vue
git commit -m "feat(shop): show GAME_CLOSED dialog for all players on 结单"
```

---

### Task 7: 端到端验证

- [ ] **Step 1: 启动服务（后端 + 前端）**

```bash
# 后端
cd /Users/kratos/hls/server && go run .

# 前端（新终端）
cd /Users/kratos/hls && npm run dev
```

- [ ] **Step 2: 用两个浏览器标签页测试**

1. 标签页 A：以房主身份创建并进入游戏（PlayPage）
2. 标签页 B：以另一玩家身份加入同一房间并进入游戏
3. 在标签页 A 点击"结单"按钮
4. **预期结果：**
   - 标签页 A 和 B 几乎同时弹出"神秘商店已结单"对话框
   - 弹窗只有"确认回到主页"按钮，点击背板不关闭弹窗
   - 点击"确认回到主页"后跳转到首页

- [ ] **Step 3: 验证无回归**

检查离开按钮（"离开"）行为不受影响，点击后直接离开，不弹 GAME_CLOSED 对话框。

# 神秘商店结单弹窗 Design

## 问题

点击"结单"后，所有玩家应立刻看到"神秘商店已结单"弹窗，确认后跳转首页。现状存在两个 bug：

1. `client.ts` 用硬编码字符串 `'已被服务端断开'` 覆盖了服务端 KICK 的 reason，导致弹窗条件 `closedReason === 'CLOSED'` 永远不成立，标题显示"商店已关闭"而非"神秘商店已结单"。
2. `ConfirmDialog` 背板有 `@click.self="emit('cancel')"` + 3 秒自动跳转，房主点"结单"后鼠标弹起落在背板上即立刻跳走，非房主玩家也因 3 秒定时器看不到弹窗。

## 方案：新增 `GAME_CLOSED` 服务端事件

不依赖 KICK / WebSocket 断开来触发弹窗，改为在踢人之前广播专属事件，前端收到后立刻显示弹窗。

## 时序

```
房主点击"结单"
  → 客户端发送 CLOSE_GAME intent
  → 服务端验证（仅房主 + PLAYING 状态）
  → 服务端 broadcast GAME_CLOSED  ← 所有客户端收到，显示弹窗
  → 服务端 loop: SendEvent(KICK) + Close()  ← WebSocket 断开
  → 玩家点"确认回到主页" → leaveRoom() + router 跳首页
```

`GAME_CLOSED` 在 KICK 之前发出，保证所有客户端在连接断开前就已收到事件并渲染弹窗。

## 后端改动

### `server/internal/protocol/event.go`

新增常量和 payload：

```go
EventGameClosed = "GAME_CLOSED"

type GameClosedPayload struct{}
```

### `server/internal/room/room.go` — `handleCloseGame`

在踢人循环之前，先广播 `GAME_CLOSED`：

```go
func (r *Room) handleCloseGame(ctx context.Context, it Intent) {
    // ... 鉴权检查不变 ...
    r.broadcast(protocol.EventGameClosed, protocol.GameClosedPayload{})
    for _, c := range r.clients {
        if c == nil { continue }
        c.SendEvent(protocol.EventKick, protocol.KickPayload{Reason: protocol.KickClosed}, "")
        c.Close()
    }
    r.stop()
}
```

## 前端改动

### `src/games/shop/net/protocol.ts`

`EventMap` 新增：

```ts
GAME_CLOSED: Record<string, never>;
```

### `src/games/shop/stores/room.ts`

新增 ref 并在 `handleEvent` 里处理：

```ts
const gameClosed = ref(false);

// handleEvent switch:
case 'GAME_CLOSED':
  gameClosed.value = true;
  break;

// leaveRoom() 里重置：
gameClosed.value = false;

// expose:
gameClosed,
```

### `src/shared/components/ConfirmDialog.vue`

新增 `persistent` prop：当 `persistent=true` 时，背板点击不 emit cancel。

```ts
defineProps<{
  // ...existing...
  persistent?: boolean;
}>();
```

模板：`@click.self` 改为 `@click.self="!persistent && emit('cancel')"`

### `src/games/shop/pages/PlayPage.vue`

1. 新增 `showGameClosedDialog = ref(false)`
2. 新增 watch：

```ts
watch(
  () => roomStore.gameClosed,
  (closed) => {
    if (closed) showGameClosedDialog.value = true;
  },
);
```

3. 新增弹窗（`persistent` + `noCancel`，仅确认按钮）：

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

4. 新增处理函数：

```ts
function handleGameClosed(): void {
  showGameClosedDialog.value = false;
  roomStore.leaveRoom();  // leaveRoom 内部会重置 gameClosed
  router.replace({ name: 'home' });
}
```

5. 现有的 `status === 'closed'` watch 保留，用于处理网络异常等其他断连场景；`GAME_CLOSED` 弹窗独立，不干扰。

### ConfirmDialog 单按钮处理

新增 `noCancel` 布尔 prop，为 true 时隐藏取消按钮：

```ts
defineProps<{
  // ...existing...
  noCancel?: boolean;
}>();
```

```html
<button v-if="!noCancel" class="btn-ghost px-4 py-2" @click="emit('cancel')">
  {{ cancelText ?? '稍候' }}
</button>
```

## 范围

- 仅 `PlayPage.vue`：`GAME_CLOSED` 只在 PLAYING 状态下触发，玩家此时都在 PlayPage
- `RoomPage.vue` 不需要修改（结单前提是 PLAYING，不在 RoomPage）
- 现有的 KICK 路径不修改，保留作为断连兜底

## 不在范围内

- 弹窗中显示保底金额等结算信息
- RoomPage 的断连弹窗文案修复（另立 issue）

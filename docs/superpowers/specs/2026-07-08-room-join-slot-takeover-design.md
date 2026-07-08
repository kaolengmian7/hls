# 神秘商店：房间加入、位置继承与结单关闭 设计文档

**日期**：2026-07-08

## 1. 背景与目标

当前系统对房间加入存在多项限制：游戏中不可加入、已在其他房间中不可加入、满员不可加入。同时，玩家离开后位置立即释放，无法恢复身份。本次改动旨在：

- 允许用户随时通过房间码加入未满员的房间（无游戏状态限制、无跨房限制）
- 玩家离开或断线后，角色与鼠币数据保留，新玩家可顶替其位置继续游戏
- 房间只能被房主显式结单（CLOSE_GAME）关闭，不会自动超时关闭
- 结单后所有人弹出提示对话框，确认后跳转网站主页

## 2. 核心概念：位置继承（Slot Takeover）

房间内的「位置」（slot）独立于玩家身份存在，携带：

- 角色（`role`: `BOSS` / `ESCORT` / `null`）
- 鼠币（`coin`）
- 准备状态（`ready`）

玩家 ID 只是「当前占着该位置的人」。当位置持有者离线后，任意新玩家可顶替进入，继承角色、鼠币与准备状态（role、coin、ready、isHost 全部继承）。

## 3. 功能详细设计

### 3.1 离开 = 断开连接，位置不释放

**当前行为**：
- 非房主 `LEAVE` → `removePlayer` → 位置释放
- 房主 `LEAVE` → `DISMISS_ROOM` → 房间关闭

**新行为**：
- 任意玩家（含房主）发送 `LEAVE` → 仅关闭 WS 连接，`handleClientGone` 标记 `connected=false`
- 位置保留（角色、鼠币不变），快照中该玩家显示为 offline
- 不广播 `PLAYER_LEFT` feed 事件

**后端**：`handleLeave` 仅调 `c.Close()`；`handleClientGone` 负责标记 `connected=false` 并广播快照（已有逻辑，无需修改）。

**前端**：`leaveRoom()` 删除 `DISMISS_ROOM` 分支；`teardownClient() → destroy()` 自动发送 `LEAVE`，行为不变。「离开商店」确认弹窗去掉「商店会立刻关闭」措辞。

### 3.2 新玩家加入：优先顶替离线位置

`handleJoin` 对新 playerID 的处理顺序：

```
1. 若 playerID 已在房间内（同一人重连）→ 走现有 slot-reuse 路径（优先级最高，不变）

2. 若 playerID 为全新用户：
   a. 扫描 playerOrder，找第一个 connected=false 的离线位置
      → 找到离线位置 → 顶替：
          新玩家继承 role、coin、ready、isHost
          若 isHost=true → r.hostID 更新为新 playerID
          旧 playerID 从 players / states / clients / userIDs 中删除，写入新 playerID
          DB：RemovePlayer(旧) + AddPlayer(新) + UpdatePlayerState(role, coin, ready)
          发 JOIN_ACCEPTED（含当前快照），广播 STATE_SNAPSHOT
      → 未找到离线位置 → 检查空位：
          len(players) < MaxPlayers → 正常新玩家加入（现有逻辑，role=null, coin=0）
          len(players) >= MaxPlayers → REJECT(FULL)
```

移除对新玩家的 `r.status != WAITING` 拦截，PLAYING 状态同样允许顶替或空位加入。

### 3.3 房主机制

- **无自动房主迁移**：移除 `migrateHost` 函数及所有调用点
- **房主更替**：仅通过顶替自然发生——新玩家顶替房主的离线位置时，继承 `isHost=true`，`r.hostID` 同步更新为新 playerID
- **原房主重连**：其 playerID 已被替换时，走普通新玩家路径（顶替其他离线位或 FULL 拒绝）
- **无继任者时**：房间保持原 `hostID`，等待下一次顶替；不关闭房间

### 3.4 房间永不自动关闭

移除 `handleTick` 中的三项时间清理：

| 常量 | 原作用 | 新处理 |
|------|--------|--------|
| `playerDropGrace` | 断线玩家 8h 后清座 | 删除 |
| `hostMigrationGrace` | 房主断线 8h 后迁移 | 删除 |
| `idleRoomTimeout` | 空房 8h 后 GC | 删除 |

Tick 定时器保留（可用于未来扩展），清理逻辑全部删除。

**房间唯一关闭路径**：房主在 `PLAYING` 状态发送 `CLOSE_GAME`。

### 3.5 加入限制放开

**前端 `ShopHomePage.handleJoin` 删除以下前置拦截**：
- `summary.status === 'PLAYING'` 非成员拦截
- `summary.activeRoomCode` 跨房拦截
- `summary.playerCount >= 3` 满员前置检查

改为：直接尝试连接，由后端 `FULL` 拒绝兜底（现有错误处理保留）。

若当前已连接某房间（`roomStore.status === 'connected'`），先调 `roomStore.leaveRoom()` 静默离开，再连接新房间。

**前端 `ShopHomePage.handleCreate` 删除**：
- 409 冲突处理逻辑（后端不再返回 409）

**后端 `CreateRoomHandler` 删除**：
- `FindActiveRoomForUser` 的 409 检查，允许同一用户同时主持多个房间

**后端 `QueryRoomHandler`**：
- 移除响应中的 `activeRoomCode` 字段（前端不再使用）

### 3.6 结单后弹对话框，跳转主页

**新行为**：收到 `KICK(CLOSED)` → `roomStore.status = 'closed'`，`closedReason = 'CLOSED'`

- `RoomPage` / `PlayPage` 的 `watch(status)` 中，识别 `closedReason === 'CLOSED'` → 弹出「神秘商店已结单」对话框
- 点击确认按钮，或 3 秒后自动跳转 `router.replace({ name: 'home' })`
- `closedReason === 'DISMISSED'`：保留现有弹窗行为（前端理论上不再触发，后端保留兜底）

## 4. 变更文件清单

### 后端（Go）

| 文件 | 变更 |
|------|------|
| `server/internal/room/room.go` | `handleLeave` 改为仅 `c.Close()`；`handleJoin` 加顶替逻辑 + 移除 PLAYING 拦截；移除 `migrateHost` 及全部调用；`handleTick` 移除三项 GC 逻辑 |
| `server/internal/handler/http.go` | `CreateRoomHandler` 移除 409 检查；`QueryRoomHandler` 移除 `activeRoomCode` 字段 |

### 前端（Vue / TS）

| 文件 | 变更 |
|------|------|
| `src/games/shop/stores/room.ts` | `leaveRoom()` 删除 `DISMISS_ROOM` 分支 |
| `src/games/shop/net/api.ts` | `RoomSummary` 类型移除 `activeRoomCode` 字段 |
| `src/games/shop/pages/ShopHomePage.vue` | `handleJoin` 删除三项前置拦截，加已连接时先离开逻辑；`handleCreate` 删除 409 处理 |
| `src/games/shop/pages/RoomPage.vue` | 结单弹窗文案（`CLOSED` reason）；「离开商店」弹窗去掉「商店会立刻关闭」措辞 |
| `src/games/shop/pages/PlayPage.vue` | `closedReason === 'CLOSED'` 弹「神秘商店已结单」对话框后跳首页 |

## 5. 不变部分

- 满员（`FULL`）拒绝：所有位置均由在线玩家占满时拒绝新加入
- `CLOSE_GAME` 仅限 `PLAYING` 状态、仅限房主
- 同一玩家不能在同一房间开两个连接（`DUP` 拒绝）
- 同一用户可同时加入两个不同房间（两 Tab 独立 store，互不干扰）
- 断线重连（slot reuse by same playerID）优先于顶替，顺序不变

## 6. 边界情况

| 场景 | 处理 |
|------|------|
| 3 人全部离线（LEAVE 或关浏览器），新玩家加入 | 顶替 `playerOrder` 第一个离线位置（通常为房主位），新玩家成为房主 |
| 原房主重连，位置已被顶替 | 走普通新玩家路径，顶替其他离线位或 FULL 拒绝 |
| PLAYING 中新玩家顶替离线 BOSS 位 | 新玩家成为 BOSS，继承鼠币，可立即参与抽奖 |
| 房间只有 1 人（房主），房主离线，新玩家加入 | 顶替房主位，新玩家成为新房主；原房主重连后走顶替逻辑 |
| 房主刚创建房间尚未 WS 连接，第三方抢先加入 | 第三方顶替房主离线槽位，成为新房主；原创建者重连后视空位情况加入（实践中极小概率，因创建后立即发起 WS 连接） |
| 用户同时主持/加入两个房间 | 两个房间独立运行，不冲突 |
| 所有位置满员且全部在线 | REJECT(FULL)，前端展示「店内位置已满」 |

# 设计文档：用 userId 替换随机 playerId，并移除 session token

**日期：** 2026-07-06  
**状态：** 已批准  

---

## 背景

神秘商店游戏原来用随机生成的 `p_<nanoid>` 作为玩家身份标识，在没有用户系统时用于匿名区分玩家。同时用 HMAC-signed session token 来区分"断线重连"与"新加入"。现在系统已有用户账号体系（JWT 认证），所有玩家都需要登录：

- 随机 playerId 成了冗余的平行身份层
- session token 的唯一作用（区分重连 vs 新加入）可完全用 JWT 身份 + 房间内成员查找替代

---

## 目标

- 所有玩家身份（playerID）统一从 JWT 派生：`strconv.FormatInt(userID, 10)`
- 删除随机生成、存储、传递 playerId 的所有代码
- 各接口（REST、WebSocket）不再接受 `playerId` 参数
- **移除 session token 机制**：重连判断改为 JWT 身份 + 服务端房间成员查找

---

## 核心约定

```
playerID := strconv.FormatInt(userID, 10)
```

例：用户 ID 为 `42`，则 playerID 为字符串 `"42"`。

- `player_id` 列在 DB 中仍为 VARCHAR 类型，不需要 schema 变更
- 历史数据无需迁移（无需保留）

---

## 服务端变更

### `server/internal/handler/http.go`

**CreateRoomHandler**
- 删除 `hostPlayerID, err := newPlayerID()` 调用
- 改为：`hostPlayerID := strconv.FormatInt(userID, 10)`
- `CreateRoomResponse` struct：删除 `PlayerID string` 字段
- 删除 `newPlayerID()` helper 函数

**QueryRoomHandler**
- 删除从 query 参数读取 `playerId` 的逻辑
- 改为：从 JWT header 读取 userID（调用 `extractUserID`），派生 playerID
- 若未携带 JWT：`IsMember` 返回 `false`，正常响应其他字段
- 删除 `playerId` query param 的处理

### `server/internal/handler/ws.go`

**WebSocketHandler**
- JWT 变为必填：无 JWT 或 JWT 无效时返回 `401 Unauthorized`
- 删除从 query 读取 `playerId` 的代码
- 改为：`playerID := strconv.FormatInt(userID, 10)`
- 删除 session token 读取与验证逻辑（`query.Get("token")`、`ctx.Signer.Verify(...)`）
- 重连判断改为：`isReturningMember := rm.HasPlayer(playerID)`
- `rm.RegisterClient(client, playerID, isReturningMember)` 不变（`authenticated` 参数语义保留，值来源改变）

### `server/pkg/token/`

- 删除整个 `token` 包（`token.go`）

### `server/internal/svc/servicecontext.go`

- 删除 `Signer` 字段及其初始化

### `server/internal/room/room.go`

- `issueToken()` 方法删除
- `handleReconnect` 中 `issueToken` 调用删除（`JOIN_ACCEPTED` 不再携带 `sessionToken`）
- `TokenIssuer` interface 引用删除

### `server/internal/room/hub.go`

- `tokens TokenIssuer` 字段删除
- `NewHub` 参数中 `tokens` 删除

### `server/internal/room/client.go`

- `tokenVerified` 字段删除（如有）

---

## 前端变更

### `src/shared/stores/player.ts`

**`ensureIdentity()`**
- 删除生成随机 playerId 及写入/读取 localStorage 的逻辑
- `playerId` ref 改为从 `useAuthStore().user?.id` 派生：`playerId.value = String(authStore.user?.id ?? '')`
- 删除 `setPlayerId()` 方法（服务端不再返回 playerId）
- 删除 `STORAGE_KEY_ID` 常量

### `src/shared/utils/id.ts`

- 删除 `generatePlayerId()` 函数及相关 nanoid import（如无其他使用）

### `src/games/shop/net/api.ts`

**`CreateRoomResponse` type**
- 删除 `playerId: string` 字段
- 删除 `sessionToken: string` 字段（服务端不再签发）

**`queryRoom()`**
- 删除 `playerId?` 参数
- 请求改为携带 `Authorization: Bearer <token>` header（与 `createRoom` 一致）

**`request()` helper**
- 可选：统一加入 auth header 注入逻辑，避免每个接口单独处理

### `src/games/shop/net/client.ts`

**`ConnectInput` type**
- 删除 `playerId: string` 字段
- 删除 `sessionToken?: string` 字段

**`connect()`**
- WS URL 构建：删除 `playerId` 和 `token` query 参数，只保留 `jwt`
- 删除 `currentSessionToken` 字段及相关逻辑
- `handleOpen()`：删除 `if (this.currentSessionToken) { return; }` 判断——客户端始终发送 JOIN；服务端根据房间成员查找决定是走重连还是新加入路径

### `src/games/shop/stores/room.ts`

**`connect()` 方法**
- 参数 interface：删除 `playerId` 和 `sessionToken` 字段
- `selfPlayerId` 初始值从 `useAuthStore().user?.id.toString()` 取，JOIN_ACCEPTED 后由服务端 `selfPlayerId` 覆盖
- 删除 `playerStore.setSessionToken(...)` 调用

**`tryRestoreSession()`**
- 删除对 `playerStore.playerId`、`playerStore.getSessionToken` 的依赖
- 简化为：若 `authStore.user` 存在则尝试连接；服务端自动判断重连

### `src/shared/stores/player.ts`

- 删除 `session` ref、`SessionRecord` type
- 删除 `setSessionToken()`、`getSessionToken()`、`clearSession()` 方法
- 删除 `STORAGE_KEY_SESSION` 常量

---

## 测试影响

- `server/internal/room/hub_test.go`：硬编码的 `p_xxx` 格式 playerID 改为数字字符串（如 `"1"`、`"2"`）
- `server/internal/game/settlement_test.go`：同上

---

## 重连逻辑变化

| | 旧 | 新 |
|---|---|---|
| 判断依据 | session token 有效 | `rm.HasPlayer(playerID)` |
| 客户端行为 | 有 token → 不发 JOIN | 始终发 JOIN |
| 服务端行为 | `authenticated=true` → `handleReconnect` | `isReturningMember` → `handleReconnect`（先于 JOIN 执行） |
| 服务端收到 JOIN（重连玩家） | 不会发生（client 跳过） | JOIN 在 `handleReconnect` 发送 `JOIN_ACCEPTED` 后到达；`handleJoin` 检测已加入则忽略或幂等处理 |

---

## 错误处理

| 场景 | 行为 |
|------|------|
| WS 连接无 JWT | HTTP 401，前端跳登录页 |
| WS JWT 已过期/无效 | HTTP 401 |
| `createRoom` 无 JWT | 已有 401，不变 |
| `queryRoom` 无 JWT | `IsMember: false`，正常返回房间信息 |

---

## 不在范围内

- DB schema 不变（`player_id` 保持 VARCHAR）
- 用户昵称、头像等 player store 其余字段不变（昵称迁移见独立 spec）
- 房间号生成逻辑不变

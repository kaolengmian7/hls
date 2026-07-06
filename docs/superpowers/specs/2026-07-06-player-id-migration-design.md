# 设计文档：用 userId 替换随机 playerId

**日期：** 2026-07-06  
**状态：** 已批准  

---

## 背景

神秘商店游戏原来用随机生成的 `p_<nanoid>` 作为玩家身份标识，在没有用户系统时用于匿名区分玩家。现在系统已有用户账号体系（JWT 认证），所有玩家都需要登录，因此随机 playerId 成了冗余的平行身份层，需要统一清除。

---

## 目标

- 所有玩家身份（playerID）统一从 JWT 派生：`strconv.FormatInt(userID, 10)`
- 删除随机生成、存储、传递 playerId 的所有代码
- 各接口（REST、WebSocket）不再接受 `playerId` 参数
- 保持 session token 机制不变（仍用 `roomCode|playerID|exp` 签名）

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
- Session token 验证逻辑不变（`ctx.Signer.Verify(code, playerID, sessionToken)`）

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

**`queryRoom()`**
- 删除 `playerId?` 参数
- 请求改为携带 `Authorization: Bearer <token>` header（与 `createRoom` 一致）

**`request()` helper**
- 可选：统一加入 auth header 注入逻辑，避免每个接口单独处理

### `src/games/shop/net/client.ts`

**`ConnectInput` type**
- 删除 `playerId: string` 字段

**`connect()`**
- WS URL 构建：删除 `playerId` query 参数，仍保留 `jwt` 和 `token`

### `src/games/shop/stores/room.ts`

**`connect()` 方法**
- 参数 interface：删除 `playerId` 字段
- `selfPlayerId` 初始值从 `useAuthStore().user?.id.toString()` 取，JOIN_ACCEPTED 后由服务端 `selfPlayerId` 覆盖

**`tryRestoreSession()`**
- 删除对 `playerStore.playerId` 的依赖
- `playerId` 改为从 `useAuthStore()` 取

---

## 测试影响

- `server/internal/room/hub_test.go`：硬编码的 `p_xxx` 格式 playerID 改为数字字符串（如 `"1"`、`"2"`）
- `server/internal/game/settlement_test.go`：同上

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

- session token 签名算法不变
- DB schema 不变（`player_id` 保持 VARCHAR）
- 用户昵称、头像等 player store 其余字段不变
- 房间号生成逻辑不变

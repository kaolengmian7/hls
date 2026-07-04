# 神秘商店：认证门禁 + MySQL 持久化 + 测试账号 设计文档

**日期**：2026-07-04  
**状态**：待实现

---

## 1. 背景与目标

当前神秘商店存在三个缺口：

1. 无认证门禁：任何人无需登录即可进入准备页
2. 游戏状态全在内存：服务重启丢失所有对局
3. 无多开限制：同一用户可同时加入多个对局

本次设计涵盖：
- 预插入 300 个测试账号（手机号 900006000–900006299）
- 后端支持测试号登录（绕过短信，验证码 = 手机号本身）
- 前端登录门禁（访问 `/shop/**` 必须先登录）
- MySQL 持久化神秘商店对局状态（支持服务重启恢复）
- 单局约束（用户同时只能在一个对局中）
- 「结单」按钮（房主触发，结束整个房间）

---

## 2. 数据库 Schema

### 2.1 命名约定

所有神秘商店专属表以 `shop_` 开头。`users` / `auth_states` 为全局共用表，不加前缀。

### 2.2 新增表

```sql
-- 对局主表
CREATE TABLE IF NOT EXISTS shop_rooms (
  code               VARCHAR(6)   NOT NULL PRIMARY KEY,
  status             ENUM('waiting','playing','closed') NOT NULL DEFAULT 'waiting',
  host_player_id     VARCHAR(20)  NOT NULL,
  guaranteed_amount  INT          NOT NULL DEFAULT 0,
  created_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_status (status)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 玩家对局状态表
CREATE TABLE IF NOT EXISTS shop_room_players (
  id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  room_code   VARCHAR(6)   NOT NULL,
  user_id     BIGINT       NOT NULL,
  player_id   VARCHAR(20)  NOT NULL,   -- 协议层随机 ID (p_xxx)
  role        VARCHAR(20)  NOT NULL DEFAULT '',
  coin        INT          NOT NULL DEFAULT 0,
  is_host     TINYINT(1)   NOT NULL DEFAULT 0,
  ready       TINYINT(1)   NOT NULL DEFAULT 0,
  left_at     DATETIME     NULL,       -- NULL = 仍在局中
  joined_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_room_user (room_code, user_id),
  INDEX idx_user_active (user_id, left_at)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 抽奖流水
CREATE TABLE IF NOT EXISTS shop_lottery_records (
  id               BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  room_code        VARCHAR(6)   NOT NULL,
  player_id        VARCHAR(20)  NOT NULL,
  item_id          VARCHAR(64)  NOT NULL,
  item_name        VARCHAR(128) NOT NULL DEFAULT '',
  rarity           VARCHAR(32)  NOT NULL DEFAULT '',
  item_value       INT          NOT NULL DEFAULT 0,
  guarantee_before INT          NOT NULL DEFAULT 0,
  guarantee_after  INT          NOT NULL DEFAULT 0,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_room (room_code)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 结算流水
CREATE TABLE IF NOT EXISTS shop_settlement_records (
  id               VARCHAR(20)  NOT NULL PRIMARY KEY,   -- shortEventID
  room_code        VARCHAR(6)   NOT NULL,
  outcome          VARCHAR(20)  NOT NULL,
  guarantee_delta  INT          NOT NULL DEFAULT 0,
  guarantee_before INT          NOT NULL DEFAULT 0,
  guarantee_after  INT          NOT NULL DEFAULT 0,
  per_player       JSON         NOT NULL,               -- []PerPlayerSettlement
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_room (room_code)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2.3 单局约束查询

```sql
SELECT r.code
FROM shop_room_players p
JOIN shop_rooms r ON r.code = p.room_code
WHERE p.user_id = ?
  AND r.status IN ('waiting', 'playing')
  AND p.left_at IS NULL
LIMIT 1;
```

返回非空 → 用户已在对局，拒绝加入并返回 `room_code` 供前端提示重连。

---

## 3. 测试账号

### 3.1 预插入脚本

新增文件 `server/db/seed_test_users.sql`，插入 300 条记录：

| 手机号范围            | 昵称       |
|--------------------|-----------|
| 900006000–900006099 | 小小花栗鼠 |
| 900006100–900006199 | 魔王鼠鼠   |
| 900006200–900006299 | 修罗鼠鼠   |

脚本使用 `INSERT IGNORE INTO users (phone, nickname, avatar_url)` 保证幂等（重复执行不报错）。

### 3.2 服务端测试号绕过

**识别函数**（`server/internal/handler/sms.go` 或新增 `testphone.go`）：

```go
// isTestPhone 返回 true 表示该号码是测试账号（900006000–900006299）。
var testPhoneRe = regexp.MustCompile(`^9000060[0-2]\d{2}$`)

func isTestPhone(phone string) bool {
    return testPhoneRe.MatchString(phone)
}
```

**`SmsSendHandler` 改动**：在 `phoneRe` 校验前插入：

```go
if isTestPhone(req.Phone) {
    httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
    return
}
```

**`SmsVerifyHandler` 改动**：在 `phoneRe` 校验前插入：

```go
if isTestPhone(req.Phone) {
    if req.Code != req.Phone {
        writeSmsError(w, http.StatusBadRequest, "invalid_code", 0)
        return
    }
    user, err := ctx.UserRepo.UpsertByPhone(r.Context(), req.Phone)
    // ... 后续与正常流程相同（签发 JWT）
}
```

`UpsertByPhone` 会找到预插入的用户，昵称保持预设值不被覆盖（当前实现：phone 已存在时直接返回，不更新昵称）。

### 3.3 前端改动（LoginPage）

- 发送按钮启用条件：`phone.length === 9 || phone.length === 11`
- 验证码输入框 `maxlength`：动态跟随手机号长度（`phone.length`）
- 验证码提交按钮启用条件：`code.length === phone.length`（9位号接受9位码）

---

## 4. 登录门禁

### 4.1 前端路由守卫

`src/router/index.ts` 增加全局 `beforeEach`：

```ts
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.path.startsWith('/shop') && !auth.token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
})
```

### 4.2 登录后跳回

`usePhoneLogin.ts` 中 `verifyCode` 成功后：

```ts
const redirect = route.query.redirect as string | undefined
router.push(redirect || '/')
```

### 4.3 后端接口鉴权

创建房间（`POST /api/rooms`）和加入房间（WebSocket 握手 JOIN 阶段）均要求携带 `Authorization: Bearer <jwt>`。  
服务端从 JWT 解出 `user_id`，注入到 `room_players` 写入逻辑和单局约束检查。

---

## 5. MySQL 持久化

### 5.1 写入时机

| 事件                  | DB 操作                                                   |
|---------------------|----------------------------------------------------------|
| 创建房间               | INSERT shop_rooms + INSERT shop_room_players (host)      |
| 玩家加入               | INSERT shop_room_players                                 |
| 角色 / 准备状态变更     | UPDATE shop_room_players                                 |
| 游戏开始               | UPDATE shop_rooms.status='playing' + UPDATE coins        |
| 购买装备               | UPDATE shop_room_players.coin                            |
| 抽奖                  | UPDATE coin + UPDATE shop_rooms.guaranteed_amount + INSERT shop_lottery_records (批量) |
| 结算                  | UPDATE coin + UPDATE shop_rooms.guaranteed_amount + INSERT shop_settlement_records |
| 玩家离开 / 超时 GC     | UPDATE shop_room_players.left_at                         |
| 结单 / 解散            | UPDATE shop_rooms.status='closed' + UPDATE left_at 全员  |

所有 DB 写入均在 Room Loop 单 goroutine 内发起（异步写，不阻塞广播）；使用 `context.Background()` 而非请求 context，避免客户端断开导致写入中止。

### 5.2 重启恢复

服务启动时（`NewServiceContext`）：

1. 查询 `shop_rooms WHERE status IN ('waiting','playing')`
2. 对每个房间，查询 `shop_room_players WHERE left_at IS NULL`
3. 查询最新的 `shop_lottery_records` 和 `shop_settlement_records`（按 room_code）
4. 重建 `Room` 结构体，写入 `Hub.rooms` map，启动各 Room goroutine

恢复后的房间：玩家均处于 `Connected=false` 状态，等待客户端重连；房间状态（金币、角色、抽奖记录）完整还原。

### 5.3 新增 Repo 层

`server/internal/repo/shop.go` 提供：

```go
type ShopRepo struct { db sqlx.SqlConn }

func (r *ShopRepo) CreateRoom(...)
func (r *ShopRepo) UpdateRoomStatus(code, status string)
func (r *ShopRepo) UpsertPlayer(...)
func (r *ShopRepo) UpdatePlayerState(...)
func (r *ShopRepo) InsertLotteryRecords(records []...)
func (r *ShopRepo) InsertSettlement(...)
func (r *ShopRepo) CloseRoom(code string)
func (r *ShopRepo) FindActiveRoomForUser(userID int64) (string, error)
func (r *ShopRepo) LoadActiveRooms() ([]RoomSnapshot, error)
```

`ShopRepo` 注入到 `ServiceContext`，再注入到 `Hub` → `Room`。

---

## 6. 「结单」按钮

### 6.1 前端

`PlayPage.vue` 结算区域新增「结单」按钮，仅 `selfPlayer.isHost` 时渲染：

```html
<button v-if="selfIsHost" @click="closeGame">结单</button>
```

点击后发送 WebSocket Intent：`{ type: "CLOSE_GAME" }`

### 6.2 后端

`room.go` 新增 intent 处理：

```go
case protocol.IntentCloseGame:
    r.handleCloseGame(ctx, it)
```

`handleCloseGame` 行为：
1. 校验：仅房主、仅 PLAYING 状态（WAITING 状态房主用解散）
2. 广播 `EventKick`（reason: `CLOSED`）给所有在线客户端
3. 调用 `r.shopRepo.CloseRoom(r.code)`（status='closed', left_at=NOW() 全员）
4. `r.stop()`

### 6.3 结单与单局约束的联动

结单后 `shop_rooms.status='closed'`，单局约束查询不再匹配该房间，用户可以重新创建或加入新对局。

---

## 7. 影响范围

| 模块                              | 改动类型 |
|----------------------------------|--------|
| `server/db/schema.sql`           | 追加 4 张新表 |
| `server/db/seed_test_users.sql`  | 新增文件 |
| `server/internal/handler/sms.go` | 测试号绕过 |
| `server/internal/handler/http.go`| JWT 鉴权 + 单局检查 |
| `server/internal/handler/ws.go`  | JWT 鉴权注入 user_id |
| `server/internal/repo/shop.go`   | 新增文件 |
| `server/internal/room/room.go`   | 注入 ShopRepo + handleCloseGame + DB 写入 |
| `server/internal/room/hub.go`    | 启动恢复逻辑 |
| `server/internal/svc/service_context.go` | 注入 ShopRepo |
| `src/router/index.ts`            | 登录守卫 |
| `src/composables/usePhoneLogin.ts` | redirect 跳回 |
| `src/pages/LoginPage.vue`        | 9位号/码支持 |
| `src/games/shop/pages/PlayPage.vue` | 结单按钮 |
| `src/games/shop/net/api.ts`      | 创建房间携带 JWT |

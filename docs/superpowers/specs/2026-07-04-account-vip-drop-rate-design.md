# 账号专属爆率设计

**日期**：2026-07-04  
**状态**：待实现

## 目标

为可配置的 VIP 账号列表提供神秘商店抽奖时的专属高爆率：当 VIP 用户担任 BOSS（抽奖方）时，所有 `Rarity = HIGH` 物品的权重按配置倍数放大，其余逻辑不变。

## 背景

- 抽奖核心在 `server/internal/game/lottery.go`，`rollOne` 按权重随机选取物品。
- HIGH 物品权重极低（0.05–6.08），远低于 MEDIUM/LOW（均为 6.08）。
- `room.handleDrawLottery` 已持有 `r.userIDs`（playerID → auth userID），可直接判断身份。

## 方案：`highBoost` 参数注入

### §1 配置层

`server/internal/config/config.go` 新增字段：

```go
VIPUsers     []int64 `json:",optional"`    // 专属爆率账号 userID 列表
VIPHighBoost float64 `json:",default=5"`   // HIGH 物品权重倍数，默认 5
```

`etc/gameserver.yaml` 示例：

```yaml
VIPUsers:
  - 900006000
VIPHighBoost: 5
```

`ServiceContext` → `Hub` → `Room` 透传这两个字段，与现有 config 注入路径一致。

### §2 `game.DrawLottery` 签名变更

```go
// highBoost 为 HIGH 物品权重倍数。正常玩家传 1.0，VIP 传配置值。
func DrawLottery(playerID string, times int, currentGuarantee int, highBoost float64) ([]protocol.LotteryRecord, int, error)
```

内部新增纯函数：

```go
// applyHighBoost 返回新 slice，HIGH 物品权重乘以 factor，MEDIUM/LOW 不变。
func applyHighBoost(src []LotteryItem, factor float64) []LotteryItem
```

`DrawLottery` 在 roll 前：

```go
pool := lotteryPool
if highBoost != 1.0 {
    pool = applyHighBoost(lotteryPool, highBoost)
}
item, err := rollOne(pool)
```

`applyHighBoost` 不修改原池（copy-on-write），每次抽奖分配 51 元素 slice，开销忽略不计。

### §3 Room 层 VIP 判断

`Room` 结构体新增：

```go
vipUsers     map[int64]struct{}  // 只读，启动时从 config 构建
vipHighBoost float64
```

`handleDrawLottery` 调用 `DrawLottery` 前：

```go
boost := 1.0
if uid, ok := r.userIDs[pid]; ok {
    if _, isVIP := r.vipUsers[uid]; isVIP {
        boost = r.vipHighBoost
    }
}
records, newGuarantee, err := game.DrawLottery(pid, p.Times, guaranteeBefore, boost)
```

VIP 爆率仅影响 roll 结果，不影响保底逻辑、金币扣除、DB 写入。

### §4 Hub 透传

`Hub.CreateRoom` / `newRoom` 接收 config 中的 VIP 字段，构建 `vipUsers` set 后注入 `Room`。

## 测试

| 测试 | 断言 |
|------|------|
| `TestApplyHighBoost` | HIGH 权重 × factor，MEDIUM/LOW 权重不变 |
| `TestDrawLottery_VIPBoost` | `highBoost=1000.0` 下 1000 次抽奖，HIGH 占比 > 95% |
| 现有 `DrawLottery` 测试 | 传 `1.0`，行为与原来一致 |

## 变更范围

| 文件 | 变更 |
|------|------|
| `server/internal/config/config.go` | 新增 `VIPUsers`、`VIPHighBoost` |
| `etc/gameserver.yaml` | 配置 VIP 账号和倍数 |
| `server/internal/game/lottery.go` | `DrawLottery` 加参数；新增 `applyHighBoost` |
| `server/internal/game/lottery_test.go` | 新增两个测试；现有测试传 `1.0` |
| `server/internal/room/room.go` | `Room` 新增字段；`handleDrawLottery` 判断 VIP |
| `server/internal/room/hub.go` | 透传 VIP config 给 `newRoom` |

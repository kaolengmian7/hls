# 神秘商店奖品池可配置化设计

**日期：** 2026-07-14  
**范围：** `server/` 后端；前端不涉及

## 背景

神秘商店的抽奖奖品池（共 51 件"红"）当前硬编码在 `server/internal/game/lottery.go` 的 `var lotteryPool`。修改奖品需要重新编译，且无法在其他游戏中直接复用同一批奖品数据。目标是将奖品池迁移到 `server/etc/gameserver.yaml`，通过配置注入，不修改代码即可替换整个池子。

## 设计方案

采用**包级 `InitPool()` 初始化**（方案 A）：启动时由 `NewServiceContext` 从配置读取奖品池并注入 `game` 包，`GetPool()` / `DrawLottery()` 签名不变。放弃回退到硬编码默认池——若配置为空则启动失败，强制运维显式声明奖品池。

## 变更文件清单

| 文件 | 变更类型 | 说明 |
|---|---|---|
| `server/internal/config/config.go` | 新增结构体 + 字段 | `LotteryItemConfig`；`ShopConfig.LotteryPool` |
| `server/etc/gameserver.yaml` | 新增内容 | `Shop.LotteryPool` 下 51 条奖品 |
| `server/internal/game/lottery.go` | 替换 | 删除硬编码 `var lotteryPool = ...`；加 `InitPool()` |
| `server/internal/svc/service_context.go` | 小改 | 最先调 `game.InitPool`，失败即 `logx.Must` |
| `server/internal/game/lottery_test.go` | 小改 | `TestMain` 里调 `InitPool` 初始化测试数据 |

## 详细设计

### 1. config.go

新增 `LotteryItemConfig`，字段与 `game.LotteryItem` 一一对应：

```go
type LotteryItemConfig struct {
    ID       string  `json:",optional"`
    Name     string  `json:",optional"`
    Rarity   string  `json:",optional"` // HIGH | MEDIUM | LOW
    Value    int     `json:",optional"` // 万元估值，正整数
    Weight   float64 `json:",optional"` // 抽奖权重，仅服务端可见
    ImageURL string  `json:",optional"`
}
```

`ShopConfig` 追加：

```go
LotteryPool []LotteryItemConfig `json:",optional"`
```

字段标 `optional` 是 go-zero 解析约定；非空校验由 `InitPool` 在运行时完成，而非 schema 层面。

### 2. gameserver.yaml

在现有 `Shop:` 块下追加 `LotteryPool:` 列表，每项示例：

```yaml
Shop:
  VIPUsers: []
  VIPHighBoost: 5
  LotteryPool:
    - ID: TEAR_OF_THE_OCEAN
      Name: 海洋之泪
      Rarity: HIGH
      Value: 2800
      Weight: 0.05
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tear-of-the-ocean.webp
    # ...其余 50 件
```

完整 51 件数据直接从 `lottery.go` 的 `lotteryPool` 字面量迁移，字段一一对应，无数据丢失。

### 3. lottery.go

`InitPool` 接受已导出的 `[]LotteryItem`，不引入 `config` 包，保持 `game` 包自包含（方便复用）。

**删除**：

```go
var lotteryPool = []LotteryItem{ /* 51件硬编码 */ }
```

**替换为**：

```go
var lotteryPool []LotteryItem

// InitPool 由 service_context 在启动时唯一调用。
// 池为空时返回 error，调用方应 logx.Must 使进程立即退出。
func InitPool(items []LotteryItem) error {
    if len(items) == 0 {
        return errors.New(errors.CodeInternal,
            "lottery pool is empty: configure Shop.LotteryPool in gameserver.yaml")
    }
    lotteryPool = append([]LotteryItem(nil), items...)
    return nil
}
```

`GetPool()`、`DrawLottery()`、`rollOne()`、`applyHighBoost()` 签名与逻辑**完全不变**。

### 4. service_context.go

config → game 的类型转换在此处完成，`game` 包无需知道 `config` 包的存在：

```go
func NewServiceContext(cfg config.Config) *ServiceContext {
    logx.Must(game.InitPool(toLotteryItems(cfg.Shop.LotteryPool))) // 失败即退出

    rootCtx, cancel := context.WithCancel(context.Background())
    // ... 其余不变
}

func toLotteryItems(cfgItems []config.LotteryItemConfig) []game.LotteryItem {
    items := make([]game.LotteryItem, len(cfgItems))
    for i, it := range cfgItems {
        items[i] = game.LotteryItem{
            ID: it.ID, Name: it.Name, Rarity: it.Rarity,
            Value: it.Value, Weight: it.Weight, ImageURL: it.ImageURL,
        }
    }
    return items
}
```

`InitPool` 放在最前，确保后续 hub / room 恢复等逻辑能正常使用奖品池。

### 5. lottery_test.go

测试文件当前直接引用包级 `lotteryPool`（51 件）做断言。迁移后需在 `TestMain` 中初始化：

```go
func TestMain(m *testing.M) {
    // 用原有硬编码数据初始化，保持现有测试不变
    if err := InitPool(defaultTestPool()); err != nil {
        panic(err)
    }
    os.Exit(m.Run())
}
```

`defaultTestPool()` 返回与原 51 件一致的 `[]config.LotteryItemConfig`，放在测试文件内。现有各测试函数无需修改。

## 数据流

```
gameserver.yaml
    └─ Shop.LotteryPool ([]LotteryItemConfig)
           │
           ▼
NewServiceContext(cfg)
    └─ game.InitPool(cfg.Shop.LotteryPool)
           │ 写入
           ▼
      game.lotteryPool ([]LotteryItem)
           │
    ┌──────┴──────────────┐
    ▼                     ▼
GetPool()           DrawLottery()
(HTTP handler)      (room intent)
```

## 错误处理

- `LotteryPool` 为空 → `InitPool` 返回 error → `logx.Must` 打印原因后进程退出，exit code 非零，容器自动重启并告警
- 单件奖品字段缺失（如 Weight=0）→ 当前不做额外校验，权重为 0 的奖品不会被抽中（`rollOne` 按权重累加），行为可预期

## 复用说明

新游戏需要同一批"红"时，只需：
1. 将 `Shop.LotteryPool` 块从本项目 `gameserver.yaml` 复制到新项目的配置文件
2. 新项目实现同样的 `InitPool([]LotteryItemConfig)` 接口（或直接复用 `game` 包）

奖品数据与业务逻辑解耦，无需触碰 Go 源码。

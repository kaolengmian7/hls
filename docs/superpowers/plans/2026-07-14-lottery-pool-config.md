# 神秘商店奖品池可配置化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `lottery.go` 中硬编码的 51 件奖品池迁移到 `gameserver.yaml`，通过 `game.InitPool()` 在启动时注入，奖品数据与代码解耦。

**Architecture:** `config.LotteryItemConfig` 承载 YAML 字段；`NewServiceContext` 将其转换为 `[]game.LotteryItem` 后调 `game.InitPool`；`InitPool` 写入包级 `lotteryPool`，`GetPool` / `DrawLottery` 不改签名。`game` 包不引入 `config` 包，保持自包含。

**Tech Stack:** Go 1.21+、go-zero conf（YAML 解析）、标准库 `os`

---

## 文件清单

| 文件 | 操作 |
|---|---|
| `server/internal/config/config.go` | 新增 `LotteryItemConfig` 结构体；`ShopConfig` 加 `LotteryPool` 字段 |
| `server/internal/game/lottery.go` | 删除硬编码 `var lotteryPool = []LotteryItem{...}`；加 `InitPool([]LotteryItem) error` |
| `server/internal/game/lottery_test.go` | 加 `TestMain` + `defaultTestPool()` + `TestInitPool` |
| `server/internal/svc/service_context.go` | 加 `toLotteryItems`；`NewServiceContext` 首行调 `game.InitPool` |
| `server/etc/gameserver.yaml` | 追加 `Shop.LotteryPool`（51 条） |

---

### Task 1: config.go — 添加 LotteryItemConfig

**Files:**
- Modify: `server/internal/config/config.go`

- [ ] **Step 1: 添加 `LotteryItemConfig` 结构体和 `ShopConfig.LotteryPool` 字段**

在 `config.go` 的 `ShopConfig` 定义之前插入新类型，并在 `ShopConfig` 末尾加字段。最终结果：

```go
// LotteryItemConfig 是奖品池单件的 YAML 配置映射，与 game.LotteryItem 字段一一对应。
type LotteryItemConfig struct {
	ID       string  `json:",optional"`
	Name     string  `json:",optional"`
	Rarity   string  `json:",optional"` // HIGH | MEDIUM | LOW
	Value    int     `json:",optional"` // 万元估值，正整数
	Weight   float64 `json:",optional"` // 抽奖权重（客户端不可见）
	ImageURL string  `json:",optional"`
}

// ShopConfig 神秘商店专项配置。
type ShopConfig struct {
	VIPUsers     []int64             `json:",optional"`
	VIPHighBoost float64             `json:",default=5"`
	LotteryPool  []LotteryItemConfig `json:",optional"`
}
```

- [ ] **Step 2: 编译验证**

```bash
cd server && go build ./...
```

期望：无错误。

- [ ] **Step 3: Commit**

```bash
git add server/internal/config/config.go
git commit -m "feat(config): add LotteryItemConfig and ShopConfig.LotteryPool field"
```

---

### Task 2: lottery.go + lottery_test.go — TDD InitPool

**Files:**
- Modify: `server/internal/game/lottery.go`
- Modify: `server/internal/game/lottery_test.go`

- [ ] **Step 1: 在 lottery_test.go 写 `TestInitPool` + `TestMain` + `defaultTestPool()`（RED）**

将 `lottery_test.go` 顶部的 `import` 块更新为：

```go
import (
	"os"
	"testing"

	"hls.dev/server/pkg/errors"
)
```

然后在文件末尾追加以下内容：

```go
func TestMain(m *testing.M) {
	if err := InitPool(defaultTestPool()); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestInitPool(t *testing.T) {
	t.Run("nil pool returns error", func(t *testing.T) {
		if err := InitPool(nil); err == nil {
			t.Fatal("expected error for nil pool")
		}
	})
	t.Run("empty pool returns error", func(t *testing.T) {
		if err := InitPool([]LotteryItem{}); err == nil {
			t.Fatal("expected error for empty pool")
		}
	})
	t.Run("valid pool sets lotteryPool", func(t *testing.T) {
		orig := lotteryPool
		t.Cleanup(func() { lotteryPool = orig })
		items := []LotteryItem{
			{ID: "X", Name: "X Item", Rarity: rarityHigh, Value: 100, Weight: 1.0,
				ImageURL: "https://cdn.example.com/x.webp"},
		}
		if err := InitPool(items); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lotteryPool) != 1 {
			t.Fatalf("want 1 item, got %d", len(lotteryPool))
		}
		if lotteryPool[0].ID != "X" {
			t.Errorf("ID=%s, want X", lotteryPool[0].ID)
		}
		// 确保 InitPool 做了拷贝，外部修改不影响包级 pool
		items[0].ID = "mutated"
		if lotteryPool[0].ID != "X" {
			t.Error("InitPool must copy items, not reference input slice")
		}
	})
}

// defaultTestPool 返回与生产池一致的 51 件奖品，供 TestMain 初始化。
func defaultTestPool() []LotteryItem {
	return []LotteryItem{
		{ID: "TEAR_OF_THE_OCEAN", Name: "海洋之泪", Rarity: rarityHigh, Value: 2800, Weight: 0.05,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tear-of-the-ocean.webp"},
		{ID: "HEART_OF_AFRICA", Name: "非洲之心", Rarity: rarityHigh, Value: 1300, Weight: 0.1,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/heart-of-africa.webp"},
		{ID: "PORTABLE_VENTILATOR", Name: "复苏呼吸机", Rarity: rarityHigh, Value: 530, Weight: 0.3,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/portable-ventilator.webp"},
		{ID: "MICRO_REACTOR", Name: "微型反应炉", Rarity: rarityHigh, Value: 470, Weight: 0.4,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/micro-reactor.webp"},
		{ID: "ARMORED_VEHICLE_BATTERY", Name: "装甲车电池", Rarity: rarityHigh, Value: 380, Weight: 0.55,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/armored-vehicle-battery.webp"},
		{ID: "ECMO", Name: "ECMO", Rarity: rarityHigh, Value: 300, Weight: 0.7,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ecmo.webp"},
		{ID: "RADAR_PORTABLE", Name: "便携军用雷达", Rarity: rarityHigh, Value: 215, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/radar-portable.webp"},
		{ID: "MANDEL_SUPERCOMPUTER", Name: "曼德尔超算单元", Rarity: rarityHigh, Value: 200, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mandel-supercomputer.webp"},
		{ID: "DISK_ARRAY_HS", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 170, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/disk-array-hs.webp"},
		{ID: "GOLDEN_CROWN_OF_TEARS", Name: "万金泪冠", Rarity: rarityHigh, Value: 230, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-crown-of-tears.webp"},
		{ID: "VERTICAL_HORIZONTAL", Name: "纵横", Rarity: rarityHigh, Value: 240, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/vertical-horizontal.webp"},
		{ID: "IMPRESSIONIST_PAINTING", Name: "印象派名画", Rarity: rarityHigh, Value: 210, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/impressionist-painting.webp"},
		{ID: "ROBOT_VAC_MOP", Name: "扫拖一体机器人", Rarity: rarityHigh, Value: 200, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/robot-vac-mop.webp"},
		{ID: "REINFORCED_CARBON_FIBER_PLATE", Name: "强化碳纤维板", Rarity: rarityHigh, Value: 200, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/reinforced-carbon-fiber-plate.webp"},
		{ID: "FLIGHT_RECORDER", Name: "飞行记录仪", Rarity: rarityHigh, Value: 170, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/flight-recorder.webp"},
		{ID: "FAMOUS_KILN_PORCELAIN", Name: "名窑瓷器", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/famous-kiln-porcelain.webp"},
		{ID: "BATTERY_PACK", Name: "动力电池组", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/battery-pack.webp"},
		{ID: "MIL_SHELL", Name: "军用炮弹", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-shell.webp"},
		{ID: "DISK_ARRAY_HS_COMPACT", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/disk-array-hs-compact.webp"},
		{ID: "AED", Name: "自动体外除颤仪", Rarity: rarityHigh, Value: 140, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/aed.webp"},
		{ID: "POWERFUL_VACUUM", Name: "强力吸尘器", Rarity: rarityHigh, Value: 140, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/powerful-vacuum.webp"},
		{ID: "BLADE_SERVER", Name: "刀片服务器", Rarity: rarityHigh, Value: 140, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/blade-server.webp"},
		{ID: "IFV_MODEL_DELUXE", Name: "步战车模型", Rarity: rarityHigh, Value: 220, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ifv-model-deluxe.webp"},
		{ID: "IFV_MODEL", Name: "步战车模型", Rarity: rarityMedium, Value: 130, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ifv-model.webp"},
		{ID: "MIL_DRONE", Name: "军用无人机", Rarity: rarityMedium, Value: 50, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-drone.webp"},
		{ID: "MIL_INFO_TERMINAL", Name: "军用信息终端", Rarity: rarityMedium, Value: 38, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-info-terminal.webp"},
		{ID: "CHAMPAGNE", Name: "香槟", Rarity: rarityMedium, Value: 34, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/champagne.webp"},
		{ID: "GPU", Name: "显卡", Rarity: rarityMedium, Value: 33, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gpu.webp"},
		{ID: "QUANTUM_STORAGE", Name: "量子存储", Rarity: rarityMedium, Value: 30, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/quantum-storage.webp"},
		{ID: "GOLD_BAR", Name: "万足金条", Rarity: rarityMedium, Value: 30, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gold-bar.webp"},
		{ID: "FOSSIL", Name: "化石", Rarity: rarityMedium, Value: 35, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/fossil.webp"},
		{ID: "GOLDEN_GAZELLE", Name: "黄金瞪羚", Rarity: rarityMedium, Value: 40, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-gazelle.webp"},
		{ID: "ARCHIVED_AUDIO_SOURCE", Name: "已封存音源", Rarity: rarityMedium, Value: 50, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/archived-audio-source.webp"},
		{ID: "GOLDEN_CROCODILE_BUST", Name: "黄金鳄鱼头雕像", Rarity: rarityMedium, Value: 50, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-crocodile-bust.webp"},
		{ID: "TIANYUAN_DIFANG", Name: "天圆地方", Rarity: rarityMedium, Value: 60, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tianyuan-difang.webp"},
		{ID: "CAMERA", Name: "摄影机", Rarity: rarityMedium, Value: 65, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/camera.webp"},
		{ID: "MUSKET", Name: "滑膛枪", Rarity: rarityMedium, Value: 70, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/musket.webp"},
		{ID: "MIL_RADIO", Name: "军用电台", Rarity: rarityMedium, Value: 100, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-radio.webp"},
		{ID: "ELECTRONIC_ANKLET", Name: "电子脚镣", Rarity: rarityMedium, Value: 100, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/electronic-anklet.webp"},
		{ID: "MEDICAL_ROBOT", Name: "医疗机器人", Rarity: rarityMedium, Value: 120, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/medical-robot.webp"},
		{ID: "LAPTOP", Name: "笔记本电脑", Rarity: rarityMedium, Value: 120, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/laptop.webp"},
		{ID: "CLOUD_STORAGE_ARRAY", Name: "云储存阵列", Rarity: rarityMedium, Value: 130, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/cloud-storage-array.webp"},
		{ID: "VENTILATOR", Name: "呼吸机", Rarity: rarityMedium, Value: 45, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ventilator.webp"},
		{ID: "GPS_RECEIVER", Name: "定位接收器", Rarity: rarityLow, Value: 18, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gps-receiver.webp"},
		{ID: "STAR_SENSOR", Name: "恒星敏感器", Rarity: rarityLow, Value: 18, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/star-sensor.webp"},
		{ID: "CAVIAR", Name: "鱼子酱", Rarity: rarityLow, Value: 17, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/caviar.webp"},
		{ID: "GILT_CARD", Name: "鎏金卡牌", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gilt-card.webp"},
		{ID: "ROLEX", Name: "劳力士", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/rolex.webp"},
		{ID: "SAYYID_POCKET_WATCH", Name: "赛义德的怀表", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/sayyid-pocket-watch.webp"},
		{ID: "ARCHIVED_EXPERIMENT_DATA", Name: "实验数据", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/archived-experiment-data.webp"},
		{ID: "ULTRASONIC_CUTTER", Name: "超声波切割刀", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ultrasonic-cutter.webp"},
	}
}
```

- [ ] **Step 2: 运行测试确认 RED（编译失败）**

```bash
cd server && go test ./internal/game/...
```

期望输出包含：`undefined: InitPool`

- [ ] **Step 3: 在 lottery.go 删除硬编码池，加 `InitPool`**

将 `lottery.go` 中从 `// lotteryPool 老板抽奖奖品池。` 到 `}` 的整个 `var lotteryPool = []LotteryItem{...}` 块（约第 49–157 行）替换为：

```go
// lotteryPool 由 InitPool 在服务启动时从配置注入；零值表示未初始化。
var lotteryPool []LotteryItem

// InitPool 在服务启动时由 svc.NewServiceContext 调用，写入奖品池。
// 池为空时返回错误；调用方应 logx.Must，使进程立即退出。
// InitPool 对 items 做防御性拷贝，外部修改不影响已注入的池。
func InitPool(items []LotteryItem) error {
	if len(items) == 0 {
		return errors.New(errors.CodeInternal,
			"lottery pool is empty: configure Shop.LotteryPool in gameserver.yaml")
	}
	lotteryPool = append([]LotteryItem(nil), items...)
	return nil
}
```

- [ ] **Step 4: 运行测试确认 GREEN**

```bash
cd server && go test ./internal/game/... -v -count=1
```

期望：所有测试 PASS，包括 `TestInitPool`、`TestLotteryPoolDefinition`（51件）、`TestDrawLotterySingle`、`TestGetPool` 等。

- [ ] **Step 5: Commit**

```bash
git add server/internal/game/lottery.go server/internal/game/lottery_test.go
git commit -m "feat(game): replace hardcoded lotteryPool with InitPool injection"
```

---

### Task 3: service_context.go — 接入 InitPool

**Files:**
- Modify: `server/internal/svc/service_context.go`

- [ ] **Step 1: 在 `service_context.go` 添加 `game` import，追加 `toLotteryItems`，并在 `NewServiceContext` 首行调用 `InitPool`**

`service_context.go` 当前 import 块里已有 `config` 但没有 `game`，需加一行。将 import 块更新为：

```go
import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/auth"
	"hls.dev/server/internal/config"
	"hls.dev/server/internal/game"
	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/room"
	"hls.dev/server/pkg/code"
)
```

将 `NewServiceContext` 函数的开头从：

```go
func NewServiceContext(cfg config.Config) *ServiceContext {
	rootCtx, cancel := context.WithCancel(context.Background())
```

改为：

```go
func NewServiceContext(cfg config.Config) *ServiceContext {
	logx.Must(game.InitPool(toLotteryItems(cfg.Shop.LotteryPool)))

	rootCtx, cancel := context.WithCancel(context.Background())
```

然后在文件末尾（`Stop` 方法之后）追加：

```go
// toLotteryItems 将 config 层的奖品配置转换为 game 层类型，隔离两个包的依赖。
func toLotteryItems(cfgItems []config.LotteryItemConfig) []game.LotteryItem {
	items := make([]game.LotteryItem, len(cfgItems))
	for i, it := range cfgItems {
		items[i] = game.LotteryItem{
			ID:       it.ID,
			Name:     it.Name,
			Rarity:   it.Rarity,
			Value:    it.Value,
			Weight:   it.Weight,
			ImageURL: it.ImageURL,
		}
	}
	return items
}
```

- [ ] **Step 2: 编译验证**

```bash
cd server && go build ./...
```

期望：无错误。

- [ ] **Step 3: Commit**

```bash
git add server/internal/svc/service_context.go
git commit -m "feat(svc): wire game.InitPool from config at startup"
```

---

### Task 4: gameserver.yaml — 追加 51 件奖品

**Files:**
- Modify: `server/etc/gameserver.yaml`

- [ ] **Step 1: 在 gameserver.yaml 末尾追加 Shop 配置块**

当前文件内容：

```yaml
Name: gameserver
Host: 0.0.0.0
Port: 8088
Mode: dev
Timeout: 30000

Log:
  Mode: console
  Encoding: plain
  Level: info
```

在文件末尾追加以下内容（完整 51 件）：

```yaml

Shop:
  LotteryPool:
    - ID: TEAR_OF_THE_OCEAN
      Name: 海洋之泪
      Rarity: HIGH
      Value: 2800
      Weight: 0.05
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tear-of-the-ocean.webp
    - ID: HEART_OF_AFRICA
      Name: 非洲之心
      Rarity: HIGH
      Value: 1300
      Weight: 0.1
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/heart-of-africa.webp
    - ID: PORTABLE_VENTILATOR
      Name: 复苏呼吸机
      Rarity: HIGH
      Value: 530
      Weight: 0.3
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/portable-ventilator.webp
    - ID: MICRO_REACTOR
      Name: 微型反应炉
      Rarity: HIGH
      Value: 470
      Weight: 0.4
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/micro-reactor.webp
    - ID: ARMORED_VEHICLE_BATTERY
      Name: 装甲车电池
      Rarity: HIGH
      Value: 380
      Weight: 0.55
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/armored-vehicle-battery.webp
    - ID: ECMO
      Name: ECMO
      Rarity: HIGH
      Value: 300
      Weight: 0.7
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ecmo.webp
    - ID: RADAR_PORTABLE
      Name: 便携军用雷达
      Rarity: HIGH
      Value: 215
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/radar-portable.webp
    - ID: MANDEL_SUPERCOMPUTER
      Name: 曼德尔超算单元
      Rarity: HIGH
      Value: 200
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mandel-supercomputer.webp
    - ID: DISK_ARRAY_HS
      Name: 高速磁盘阵列
      Rarity: HIGH
      Value: 170
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/disk-array-hs.webp
    - ID: GOLDEN_CROWN_OF_TEARS
      Name: 万金泪冠
      Rarity: HIGH
      Value: 230
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-crown-of-tears.webp
    - ID: VERTICAL_HORIZONTAL
      Name: 纵横
      Rarity: HIGH
      Value: 240
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/vertical-horizontal.webp
    - ID: IMPRESSIONIST_PAINTING
      Name: 印象派名画
      Rarity: HIGH
      Value: 210
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/impressionist-painting.webp
    - ID: ROBOT_VAC_MOP
      Name: 扫拖一体机器人
      Rarity: HIGH
      Value: 200
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/robot-vac-mop.webp
    - ID: REINFORCED_CARBON_FIBER_PLATE
      Name: 强化碳纤维板
      Rarity: HIGH
      Value: 200
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/reinforced-carbon-fiber-plate.webp
    - ID: FLIGHT_RECORDER
      Name: 飞行记录仪
      Rarity: HIGH
      Value: 170
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/flight-recorder.webp
    - ID: FAMOUS_KILN_PORCELAIN
      Name: 名窑瓷器
      Rarity: HIGH
      Value: 150
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/famous-kiln-porcelain.webp
    - ID: BATTERY_PACK
      Name: 动力电池组
      Rarity: HIGH
      Value: 150
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/battery-pack.webp
    - ID: MIL_SHELL
      Name: 军用炮弹
      Rarity: HIGH
      Value: 150
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-shell.webp
    - ID: DISK_ARRAY_HS_COMPACT
      Name: 高速磁盘阵列
      Rarity: HIGH
      Value: 150
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/disk-array-hs-compact.webp
    - ID: AED
      Name: 自动体外除颤仪
      Rarity: HIGH
      Value: 140
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/aed.webp
    - ID: POWERFUL_VACUUM
      Name: 强力吸尘器
      Rarity: HIGH
      Value: 140
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/powerful-vacuum.webp
    - ID: BLADE_SERVER
      Name: 刀片服务器
      Rarity: HIGH
      Value: 140
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/blade-server.webp
    - ID: IFV_MODEL_DELUXE
      Name: 步战车模型
      Rarity: HIGH
      Value: 220
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ifv-model-deluxe.webp
    - ID: IFV_MODEL
      Name: 步战车模型
      Rarity: MEDIUM
      Value: 130
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ifv-model.webp
    - ID: MIL_DRONE
      Name: 军用无人机
      Rarity: MEDIUM
      Value: 50
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-drone.webp
    - ID: MIL_INFO_TERMINAL
      Name: 军用信息终端
      Rarity: MEDIUM
      Value: 38
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-info-terminal.webp
    - ID: CHAMPAGNE
      Name: 香槟
      Rarity: MEDIUM
      Value: 34
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/champagne.webp
    - ID: GPU
      Name: 显卡
      Rarity: MEDIUM
      Value: 33
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gpu.webp
    - ID: QUANTUM_STORAGE
      Name: 量子存储
      Rarity: MEDIUM
      Value: 30
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/quantum-storage.webp
    - ID: GOLD_BAR
      Name: 万足金条
      Rarity: MEDIUM
      Value: 30
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gold-bar.webp
    - ID: FOSSIL
      Name: 化石
      Rarity: MEDIUM
      Value: 35
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/fossil.webp
    - ID: GOLDEN_GAZELLE
      Name: 黄金瞪羚
      Rarity: MEDIUM
      Value: 40
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-gazelle.webp
    - ID: ARCHIVED_AUDIO_SOURCE
      Name: 已封存音源
      Rarity: MEDIUM
      Value: 50
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/archived-audio-source.webp
    - ID: GOLDEN_CROCODILE_BUST
      Name: 黄金鳄鱼头雕像
      Rarity: MEDIUM
      Value: 50
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-crocodile-bust.webp
    - ID: TIANYUAN_DIFANG
      Name: 天圆地方
      Rarity: MEDIUM
      Value: 60
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tianyuan-difang.webp
    - ID: CAMERA
      Name: 摄影机
      Rarity: MEDIUM
      Value: 65
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/camera.webp
    - ID: MUSKET
      Name: 滑膛枪
      Rarity: MEDIUM
      Value: 70
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/musket.webp
    - ID: MIL_RADIO
      Name: 军用电台
      Rarity: MEDIUM
      Value: 100
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-radio.webp
    - ID: ELECTRONIC_ANKLET
      Name: 电子脚镣
      Rarity: MEDIUM
      Value: 100
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/electronic-anklet.webp
    - ID: MEDICAL_ROBOT
      Name: 医疗机器人
      Rarity: MEDIUM
      Value: 120
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/medical-robot.webp
    - ID: LAPTOP
      Name: 笔记本电脑
      Rarity: MEDIUM
      Value: 120
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/laptop.webp
    - ID: CLOUD_STORAGE_ARRAY
      Name: 云储存阵列
      Rarity: MEDIUM
      Value: 130
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/cloud-storage-array.webp
    - ID: VENTILATOR
      Name: 呼吸机
      Rarity: MEDIUM
      Value: 45
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ventilator.webp
    - ID: GPS_RECEIVER
      Name: 定位接收器
      Rarity: LOW
      Value: 18
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gps-receiver.webp
    - ID: STAR_SENSOR
      Name: 恒星敏感器
      Rarity: LOW
      Value: 18
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/star-sensor.webp
    - ID: CAVIAR
      Name: 鱼子酱
      Rarity: LOW
      Value: 17
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/caviar.webp
    - ID: GILT_CARD
      Name: 鎏金卡牌
      Rarity: LOW
      Value: 20
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gilt-card.webp
    - ID: ROLEX
      Name: 劳力士
      Rarity: LOW
      Value: 20
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/rolex.webp
    - ID: SAYYID_POCKET_WATCH
      Name: 赛义德的怀表
      Rarity: LOW
      Value: 20
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/sayyid-pocket-watch.webp
    - ID: ARCHIVED_EXPERIMENT_DATA
      Name: 实验数据
      Rarity: LOW
      Value: 20
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/archived-experiment-data.webp
    - ID: ULTRASONIC_CUTTER
      Name: 超声波切割刀
      Rarity: LOW
      Value: 20
      Weight: 6.08
      ImageURL: https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ultrasonic-cutter.webp
```

- [ ] **Step 2: 验证配置可被解析（smoke test）**

```bash
cd server && go run cmd/gameserver/main.go -f etc/gameserver.yaml 2>&1 | head -5
```

期望：服务尝试启动（可能因 DB 连接失败而退出，但不应出现 `lottery pool is empty` 错误）。若无 DB 环境，只要不出现 pool 相关 panic 即可。

- [ ] **Step 3: 运行全部 server 测试确认无回归**

```bash
cd server && go test ./... -count=1
```

期望：所有测试 PASS。

- [ ] **Step 4: Commit**

```bash
git add server/etc/gameserver.yaml
git commit -m "feat(config): add Shop.LotteryPool with 51 items to gameserver.yaml"
```

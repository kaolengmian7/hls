# 老虎机奖品图片 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将神秘商店老虎机卷轴的文字标签替换为腾讯云 CDN 托管的奖品图片，无图时显示 🎁 fallback。

**Architecture:** 在 Go 的 `protocol.Item` 和 `game.LotteryItem` 结构体加 `ImageURL` 字段，`DrawLottery` 负责传递；前端 `LotteryItem` 类型同步加 `imageUrl?`；`LotteryPanel.vue` 卷轴行改为 `<img>` 渲染，用 `reactive Set` 追踪加载失败的 URL 实现 fallback。

**Tech Stack:** Go 1.25（后端结构体 + 数据）、TypeScript / Vue 3（前端组件）

---

## 前置条件（Task 3 前完成）

将 51 件奖品图片上传至腾讯云 COS，获取 CDN 加速域名后，图片 URL 格式示例：

```
https://your-bucket.file.myqcloud.com/prizes/tear-of-the-ocean.webp
```

建议文件名：奖品 ID 小写加连字符，如 `TEAR_OF_THE_OCEAN` → `tear-of-the-ocean.webp`。图片规格：256×256px WebP，质量 85。

---

## 文件地图

| 文件 | 变更 |
|---|---|
| `server/internal/protocol/envelope.go` | `Item` 结构体加 `ImageURL` 字段 |
| `server/internal/game/lottery.go` | `LotteryItem` 加 `ImageURL`，填充 `lotteryPool`，`DrawLottery` 传递 `ImageURL` |
| `server/internal/game/lottery_test.go` | 修正 pool 数量断言（11→51），加 `ImageURL` 传递测试 |
| `src/games/shop/net/protocol.ts` | `LotteryItem` 加 `imageUrl?` |
| `src/games/shop/components/LotteryPanel.vue` | `ITEM_H` 改 96，`DECOR_FILLERS` 加 CDN URL，卷轴行改图片渲染 |

---

### Task 1：Go 协议层 — `Item` 加 `ImageURL`

**Files:**
- Modify: `server/internal/protocol/envelope.go:63-68`

- [ ] **Step 1：更新 `Item` 结构体**

将 `server/internal/protocol/envelope.go` 第 63–68 行替换为：

```go
type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Rarity   string `json:"rarity"`
	Value    int    `json:"value"`
	ImageURL string `json:"imageUrl,omitempty"`
}
```

- [ ] **Step 2：验证编译通过**

```bash
cd server && go build ./...
```

期望：无报错。

- [ ] **Step 3：提交**

```bash
git add server/internal/protocol/envelope.go
git commit -m "feat: add ImageURL to protocol Item struct"
```

---

### Task 2：Go 游戏层 — `LotteryItem` 加字段并传递

**Files:**
- Modify: `server/internal/game/lottery.go:31-37`（结构体），`149-158`（DrawLottery 构造）
- Modify: `server/internal/game/lottery_test.go`

- [ ] **Step 1：写失败测试**

在 `server/internal/game/lottery_test.go` 末尾追加：

```go
func TestDrawLotteryPropagatesImageURL(t *testing.T) {
	// 给每件奖品设可预测的 URL，保证无论 roll 到哪件都能断言。
	for i := range lotteryPool {
		lotteryPool[i].ImageURL = "https://cdn.example.com/prizes/" + lotteryPool[i].ID + ".webp"
	}
	t.Cleanup(func() {
		for i := range lotteryPool {
			lotteryPool[i].ImageURL = ""
		}
	})

	records, _, err := DrawLottery("player-test", 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantURL := "https://cdn.example.com/prizes/" + records[0].Item.ID + ".webp"
	if records[0].Item.ImageURL != wantURL {
		t.Errorf("ImageURL mismatch: got %q, want %q", records[0].Item.ImageURL, wantURL)
	}
}
```

同时修正 `TestLotteryPoolDefinition`（第 119 行），pool 实际有 51 件，将断言改为：

```go
if len(lotteryPool) != 51 {
    t.Fatalf("expect 51 prizes in pool, got %d", len(lotteryPool))
}
```

- [ ] **Step 2：运行确认测试失败**

```bash
cd server && go test ./internal/game/... -run TestDrawLotteryPropagatesImageURL -v
```

期望：`FAIL`，报错 `lotteryPool[i].ImageURL undefined`（字段尚不存在）。

- [ ] **Step 3：给 `LotteryItem` 加 `ImageURL` 字段**

将 `server/internal/game/lottery.go` 第 31–37 行替换为：

```go
type LotteryItem struct {
	ID       string
	Name     string
	Rarity   string
	Value    int
	Weight   float64
	ImageURL string
}
```

- [ ] **Step 4：`DrawLottery` 中传递 `ImageURL`**

将 `server/internal/game/lottery.go` 第 149–158 行替换为：

```go
out = append(out, protocol.LotteryRecord{
	ID:       id,
	PlayerID: playerID,
	Cost:     cost,
	Item: protocol.Item{
		ID:       item.ID,
		Name:     item.Name,
		Rarity:   item.Rarity,
		Value:    item.Value,
		ImageURL: item.ImageURL,
	},
	GuaranteeDelta:  item.Value,
	GuaranteeBefore: before,
	GuaranteeAfter:  after,
	At:              now,
})
```

- [ ] **Step 5：运行确认目标测试通过**

```bash
cd server && go test ./internal/game/... -run TestDrawLotteryPropagatesImageURL -v
```

期望：`PASS`。

- [ ] **Step 6：运行全部 game 测试**

```bash
cd server && go test ./internal/game/... -v
```

期望：所有测试 `PASS`（包括修正后的 `TestLotteryPoolDefinition`）。

- [ ] **Step 7：提交**

```bash
git add server/internal/game/lottery.go server/internal/game/lottery_test.go
git commit -m "feat: propagate ImageURL through LotteryItem and DrawLottery"
```

---

### Task 3：填充 CDN URL 到 `lotteryPool`

**前置：已将全部图片上传至 COS，获取到 CDN 域名。**

**Files:**
- Modify: `server/internal/game/lottery.go:53-105`

- [ ] **Step 1：为每件奖品加 `ImageURL`**

将 `server/internal/game/lottery.go` 第 53–105 行的 `lotteryPool` 替换，每条加 `ImageURL`。以实际 CDN 域名替换 `your-bucket.file.myqcloud.com`：

```go
var lotteryPool = []LotteryItem{
	{ID: "TEAR_OF_THE_OCEAN", Name: "海洋之泪", Rarity: rarityHigh, Value: 2800, Weight: 0.05,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/tear-of-the-ocean.webp"},
	{ID: "HEART_OF_AFRICA", Name: "非洲之心", Rarity: rarityHigh, Value: 1300, Weight: 0.1,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/heart-of-africa.webp"},
	{ID: "PORTABLE_VENTILATOR", Name: "复苏呼吸机", Rarity: rarityHigh, Value: 530, Weight: 0.3,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/portable-ventilator.webp"},
	{ID: "MICRO_REACTOR", Name: "微型反应炉", Rarity: rarityHigh, Value: 470, Weight: 0.4,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/micro-reactor.webp"},
	{ID: "ARMORED_VEHICLE_BATTERY", Name: "装甲车电池", Rarity: rarityHigh, Value: 380, Weight: 0.55,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/armored-vehicle-battery.webp"},
	{ID: "ECMO", Name: "ECMO", Rarity: rarityHigh, Value: 300, Weight: 0.7,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/ecmo.webp"},
	{ID: "RADAR_PORTABLE", Name: "便携军用雷达", Rarity: rarityHigh, Value: 215, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/radar-portable.webp"},
	{ID: "MANDEL_SUPERCOMPUTER", Name: "曼德尔超算单元", Rarity: rarityHigh, Value: 200, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/mandel-supercomputer.webp"},
	{ID: "DISK_ARRAY_HS", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 170, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/disk-array-hs.webp"},
	{ID: "GOLDEN_CROWN_OF_TEARS", Name: "万金泪冠", Rarity: rarityHigh, Value: 230, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/golden-crown-of-tears.webp"},
	{ID: "VERTICAL_HORIZONTAL", Name: "纵横", Rarity: rarityHigh, Value: 240, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/vertical-horizontal.webp"},
	{ID: "IMPRESSIONIST_PAINTING", Name: "印象派名画", Rarity: rarityHigh, Value: 210, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/impressionist-painting.webp"},
	{ID: "ROBOT_VAC_MOP", Name: "扫拖一体机器人", Rarity: rarityHigh, Value: 200, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/robot-vac-mop.webp"},
	{ID: "REINFORCED_CARBON_FIBER_PLATE", Name: "强化碳纤维板", Rarity: rarityHigh, Value: 200, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/reinforced-carbon-fiber-plate.webp"},
	{ID: "FLIGHT_RECORDER", Name: "飞行记录仪", Rarity: rarityHigh, Value: 170, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/flight-recorder.webp"},
	{ID: "FAMOUS_KILN_PORCELAIN", Name: "名窑瓷器", Rarity: rarityHigh, Value: 150, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/famous-kiln-porcelain.webp"},
	{ID: "BATTERY_PACK", Name: "动力电池组", Rarity: rarityHigh, Value: 150, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/battery-pack.webp"},
	{ID: "MIL_SHELL", Name: "军用炮弹", Rarity: rarityHigh, Value: 150, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/mil-shell.webp"},
	{ID: "DISK_ARRAY_HS_COMPACT", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 150, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/disk-array-hs-compact.webp"},
	{ID: "AED", Name: "自动体外除颤仪", Rarity: rarityHigh, Value: 140, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/aed.webp"},
	{ID: "POWERFUL_VACUUM", Name: "强力吸尘器", Rarity: rarityHigh, Value: 140, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/powerful-vacuum.webp"},
	{ID: "BLADE_SERVER", Name: "刀片服务器", Rarity: rarityHigh, Value: 140, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/blade-server.webp"},
	{ID: "IFV_MODEL_DELUXE", Name: "步战车模型", Rarity: rarityHigh, Value: 220, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/ifv-model-deluxe.webp"},
	{ID: "MIL_DRONE", Name: "军用无人机", Rarity: rarityMedium, Value: 50, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/mil-drone.webp"},
	{ID: "MIL_INFO_TERMINAL", Name: "军用信息终端", Rarity: rarityMedium, Value: 38, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/mil-info-terminal.webp"},
	{ID: "CHAMPAGNE", Name: "香槟", Rarity: rarityMedium, Value: 34, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/champagne.webp"},
	{ID: "GPU", Name: "显卡", Rarity: rarityMedium, Value: 33, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/gpu.webp"},
	{ID: "QUANTUM_STORAGE", Name: "量子存储", Rarity: rarityMedium, Value: 30, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/quantum-storage.webp"},
	{ID: "GOLD_BAR", Name: "万足金条", Rarity: rarityMedium, Value: 30, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/gold-bar.webp"},
	{ID: "FOSSIL", Name: "化石", Rarity: rarityMedium, Value: 35, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/fossil.webp"},
	{ID: "GOLDEN_GAZELLE", Name: "黄金瞪羚", Rarity: rarityMedium, Value: 40, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/golden-gazelle.webp"},
	{ID: "ARCHIVED_AUDIO_SOURCE", Name: "已封存音源", Rarity: rarityMedium, Value: 50, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/archived-audio-source.webp"},
	{ID: "GOLDEN_CROCODILE_BUST", Name: "黄金鳄鱼头雕像", Rarity: rarityMedium, Value: 50, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/golden-crocodile-bust.webp"},
	{ID: "TIANYUAN_DIFANG", Name: "天圆地方", Rarity: rarityMedium, Value: 60, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/tianyuan-difang.webp"},
	{ID: "CAMERA", Name: "摄影机", Rarity: rarityMedium, Value: 65, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/camera.webp"},
	{ID: "MUSKET", Name: "滑膛枪", Rarity: rarityMedium, Value: 70, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/musket.webp"},
	{ID: "MIL_RADIO", Name: "军用电台", Rarity: rarityMedium, Value: 100, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/mil-radio.webp"},
	{ID: "ELECTRONIC_ANKLET", Name: "电子脚镣", Rarity: rarityMedium, Value: 100, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/electronic-anklet.webp"},
	{ID: "MEDICAL_ROBOT", Name: "医疗机器人", Rarity: rarityMedium, Value: 120, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/medical-robot.webp"},
	{ID: "LAPTOP", Name: "笔记本电脑", Rarity: rarityMedium, Value: 120, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/laptop.webp"},
	{ID: "IFV_MODEL", Name: "步战车模型", Rarity: rarityMedium, Value: 130, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/ifv-model.webp"},
	{ID: "CLOUD_STORAGE_ARRAY", Name: "云储存阵列", Rarity: rarityMedium, Value: 130, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/cloud-storage-array.webp"},
	{ID: "VENTILATOR", Name: "呼吸机", Rarity: rarityMedium, Value: 45, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/ventilator.webp"},
	{ID: "GPS_RECEIVER", Name: "定位接收器", Rarity: rarityLow, Value: 18, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/gps-receiver.webp"},
	{ID: "STAR_SENSOR", Name: "恒星敏感器", Rarity: rarityLow, Value: 18, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/star-sensor.webp"},
	{ID: "CAVIAR", Name: "鱼子酱", Rarity: rarityLow, Value: 17, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/caviar.webp"},
	{ID: "GILT_CARD", Name: "鎏金卡牌", Rarity: rarityLow, Value: 20, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/gilt-card.webp"},
	{ID: "ROLEX", Name: "劳力士", Rarity: rarityLow, Value: 20, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/rolex.webp"},
	{ID: "SAYYID_POCKET_WATCH", Name: "赛义德的怀表", Rarity: rarityLow, Value: 20, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/sayyid-pocket-watch.webp"},
	{ID: "ARCHIVED_EXPERIMENT_DATA", Name: "实验数据", Rarity: rarityLow, Value: 20, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/archived-experiment-data.webp"},
	{ID: "ULTRASONIC_CUTTER", Name: "超声波切割刀", Rarity: rarityLow, Value: 20, Weight: 6.08,
		ImageURL: "https://your-bucket.file.myqcloud.com/prizes/ultrasonic-cutter.webp"},
}
```

- [ ] **Step 2：验证编译通过**

```bash
cd server && go build ./...
```

期望：无报错。

- [ ] **Step 3：运行全部测试**

```bash
cd server && go test ./internal/game/... -v
```

期望：全部 `PASS`。

- [ ] **Step 4：提交**

```bash
git add server/internal/game/lottery.go
git commit -m "feat: fill CDN image URLs in lotteryPool"
```

---

### Task 4：前端协议类型加 `imageUrl`

**Files:**
- Modify: `src/games/shop/net/protocol.ts:30-35`

- [ ] **Step 1：更新 `LotteryItem` 类型**

将 `src/games/shop/net/protocol.ts` 第 30–35 行替换为：

```ts
export type LotteryItem = {
  id: string;
  name: string;
  rarity: string;
  value: number;
  imageUrl?: string;
};
```

- [ ] **Step 2：类型检查**

```bash
npm run typecheck
```

期望：无报错。

- [ ] **Step 3：提交**

```bash
git add src/games/shop/net/protocol.ts
git commit -m "feat: add imageUrl to frontend LotteryItem type"
```

---

### Task 5：`LotteryPanel.vue` — 常量与 `DECOR_FILLERS`

**Files:**
- Modify: `src/games/shop/components/LotteryPanel.vue:17`（`ITEM_H`），`33-46`（`DECOR_FILLERS`）

- [ ] **Step 1：更新 `ITEM_H`**

将第 17 行改为：

```ts
const ITEM_H = 96; // 每格高度（px）
```

`VIEWPORT_H`、渐隐遮罩高度、中间指示框偏移均通过 `ITEM_H` 计算，无需额外修改。

- [ ] **Step 2：给 `DECOR_FILLERS` 加 CDN URL**

`DECOR_FILLERS` 的 12 个装饰条目复用对应真实奖品的图片。将第 33–46 行替换，使用与 Task 3 相同的 CDN 域名：

```ts
const DECOR_FILLERS: LotteryItem[] = [
  { id: 'decor-1', name: '海洋之泪', rarity: 'HIGH', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/tear-of-the-ocean.webp' },
  { id: 'decor-2', name: '非洲之心', rarity: 'HIGH', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/heart-of-africa.webp' },
  { id: 'decor-3', name: '便携军用雷达', rarity: 'HIGH', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/radar-portable.webp' },
  { id: 'decor-4', name: '万金泪冠', rarity: 'HIGH', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/golden-crown-of-tears.webp' },
  { id: 'decor-5', name: '军用无人机', rarity: 'MEDIUM', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/mil-drone.webp' },
  { id: 'decor-6', name: '万足金条', rarity: 'MEDIUM', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/gold-bar.webp' },
  { id: 'decor-7', name: '医疗机器人', rarity: 'MEDIUM', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/medical-robot.webp' },
  { id: 'decor-8', name: '笔记本电脑', rarity: 'MEDIUM', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/laptop.webp' },
  { id: 'decor-9', name: '劳力士', rarity: 'LOW', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/rolex.webp' },
  { id: 'decor-10', name: '鱼子酱', rarity: 'LOW', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/caviar.webp' },
  { id: 'decor-11', name: '鎏金卡牌', rarity: 'LOW', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/gilt-card.webp' },
  { id: 'decor-12', name: '恒星敏感器', rarity: 'LOW', value: 0,
    imageUrl: 'https://your-bucket.file.myqcloud.com/prizes/star-sensor.webp' },
];
```

- [ ] **Step 3：类型检查**

```bash
npm run typecheck
```

期望：无报错。

- [ ] **Step 4：提交**

```bash
git add src/games/shop/components/LotteryPanel.vue
git commit -m "feat: update ITEM_H to 96px and add imageUrl to DECOR_FILLERS"
```

---

### Task 6：`LotteryPanel.vue` — 卷轴行改为图片渲染

**Files:**
- Modify: `src/games/shop/components/LotteryPanel.vue:2`（import），`~55` 之后（reactive 状态），`271-281`（模板卷轴行）

- [ ] **Step 1：在 Vue import 中加 `reactive`**

将第 2 行改为：

```ts
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
```

- [ ] **Step 2：在响应式状态声明区加 `failedImages`**

在第 55 行（`const result = ref<LotteryRecord | null>(null);`）之后插入：

```ts
const failedImages = reactive(new Set<string>());
```

- [ ] **Step 3：移除 `RARITY_COLOR` 和 `Rarity` 的 import（不再使用）**

将第 4–10 行的 import 改为：

```ts
import {
  SINGLE_DRAW_COST,
  formatGuaranteeShift,
  formatValueW,
} from '../game/lottery';
```

- [ ] **Step 4：替换模板中的卷轴行**

将模板中的卷轴行（第 271–281 行，`v-for="(row, idx) in rows"` 的 `<div>`）替换为：

```html
<div
  v-for="(row, idx) in rows"
  :key="idx"
  class="flex items-center justify-center"
  :style="{ height: ITEM_H + 'px' }"
>
  <img
    v-if="row.imageUrl && !failedImages.has(row.imageUrl)"
    :src="row.imageUrl"
    :alt="row.name"
    style="width:64px;height:64px;border-radius:10px;object-fit:contain;"
    @error="row.imageUrl && failedImages.add(row.imageUrl)"
  />
  <div
    v-else
    style="width:64px;height:64px;border-radius:10px;background:rgba(255,255,255,0.06);display:flex;align-items:center;justify-content:center;font-size:30px;"
  >
    🎁
  </div>
</div>
```

- [ ] **Step 5：类型检查**

```bash
npm run typecheck
```

期望：无报错。

- [ ] **Step 6：启动开发服务器，视觉验证**

```bash
npm run dev
```

打开神秘商店，点击「开启神秘商店」，验证：
- 卷轴滚动时显示图片（CDN 配置前显示 🎁）
- 卷轴停稳后中间行显示对应奖品图片
- 底部仍显示奖品名字、价值、保底变化
- 将某个 `imageUrl` 改为无效 URL，确认 fallback 显示 🎁

- [ ] **Step 7：提交**

```bash
git add src/games/shop/components/LotteryPanel.vue
git commit -m "feat: replace text labels with prize images in slot machine reel"
```

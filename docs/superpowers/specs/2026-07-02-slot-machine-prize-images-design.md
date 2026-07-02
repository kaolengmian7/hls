# 神秘商店老虎机：文字换图片

**日期**：2026-07-02  
**状态**：待实现

## 背景

神秘商店抽奖的老虎机卷轴目前每格显示奖品名字（文字），用稀有度颜色区分。目标是换成奖品实物图片，图片托管在腾讯云 CDN。

## 目标

老虎机卷轴滚动及停稳时，每格显示奖品图片而非文字。无图片时显示 🎁 作为 fallback。

## 方案

**前后端同步改**：协议、前端、Go 后端一并修改，上线即完整。

## 图片托管

- 存储：腾讯云 COS Bucket（选上海/广州地域）
- 加速：在 Bucket 上开启默认 CDN 加速域名，无需单独购买 CDN 产品
- 格式：WebP，尺寸建议 128×128px（2× 分辨率），文件名与奖品 ID 对应
- CDN URL 示例：`https://xxx.file.myqcloud.com/prizes/ocean-tear.webp`

## 协议变更

**前端**：`src/games/shop/net/protocol.ts`，第 30-35 行 `LotteryItem` 类型加可选字段：

```ts
export type LotteryItem = {
  id: string;
  name: string;
  rarity: string;
  value: number;
  imageUrl?: string; // 腾讯云 CDN 图片地址，后端下发，无则前端用 🎁 fallback
};
```

**后端**：`server/internal/protocol/envelope.go`，第 63-68 行 `Item` 结构体加字段：

```go
type Item struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Rarity   string `json:"rarity"`
    Value    int    `json:"value"`
    ImageURL string `json:"imageUrl,omitempty"` // 新增
}
```

字段均为可选，旧数据（无 imageUrl）自动走 fallback，前后端兼容。

## 前端变更

### 格子高度

`ITEM_H`：84px → 96px，给图片更多呼吸空间。其余动画参数（速度、着陆帧数等）不变。

### 卷轴行渲染

从文字 `<div>` 改为图片 `<img>`：

- 有 `imageUrl`：渲染 `<img>` 元素，72×72px（中间结果行）/ 64×64px（上下行）
- 无 `imageUrl`：渲染半透明背景 + 🎁 emoji，尺寸与图片行相同
- 非焦点行（上下各一行）opacity 0.5，强化"窗口"聚焦感
- 图片使用 `object-fit: contain`，圆角 10-12px，与现有卡片风格一致

### DECOR_FILLERS

12 个装饰性填充条目各自补充对应 CDN URL，让滚动过程中图片真实飞过。CDN URL 待图片上传后填入。

### 停稳后展示区

保持不变：仍显示奖品名字、价值（`formatValueW`）、保底变化。

## Go 后端变更

需改动三处：

**1. `server/internal/game/lottery.go`，第 31-37 行 — `LotteryItem` 结构体加字段**

```go
type LotteryItem struct {
    ID       string
    Name     string
    Rarity   string
    Value    int
    Weight   float64
    ImageURL string // 新增，腾讯云 CDN 地址，空串时前端走 🎁 fallback
}
```

**2. `server/internal/game/lottery.go`，第 53-105 行 — `lotteryPool` 数组填入 CDN URL**

共 51 件奖品，每条加 `ImageURL` 字段，例：

```go
{ID: "TEAR_OF_THE_OCEAN", Name: "海洋之泪", Rarity: rarityHigh, Value: 2800, Weight: 0.05,
 ImageURL: "https://xxx.file.myqcloud.com/prizes/tear-of-ocean.webp"},
```

**3. `server/internal/game/lottery.go`，第 149-158 行 — 构造 `protocol.Item` 时传入 `ImageURL`**

```go
Item: protocol.Item{
    ID:       item.ID,
    Name:     item.Name,
    Rarity:   item.Rarity,
    Value:    item.Value,
    ImageURL: item.ImageURL, // 新增
},
```

## Fallback 策略

| imageUrl 状态 | 显示内容 |
|---|---|
| 有效 URL，加载成功 | 奖品图片 |
| 有效 URL，加载失败（onerror）| 🎁 emoji |
| 空串 / undefined | 🎁 emoji |

## 发布顺序

1. 上传奖品图片到腾讯云 COS，获取 CDN URL
2. 更新 Go 后端奖品配置，加入 CDN URL，部署后端
3. 更新前端协议类型 + `LotteryPanel.vue` + `DECOR_FILLERS`，部署前端

前后端可分批发布：后端未部署前，前端因 imageUrl 全为 undefined 而全显示 🎁；后端部署后图片自动生效。

## 不在范围内

- 图片上传工具 / 管理后台
- 图片懒加载或预加载优化
- 多图片列（目前是单列卷轴）

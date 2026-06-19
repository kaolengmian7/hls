# 花栗鼠俱乐部官网改版设计方案

**日期：** 2026-06-19  
**状态：** 已确认，待实现

---

## 背景

hls.com 当前 `/` 直接是"花栗鼠神秘商店"的游戏入口。本次改版目标：

- 将 `/` 改为花栗鼠俱乐部的官方主页，展示俱乐部品牌与所有玩法入口
- 神秘商店整体迁移至 `/shop`（含房间等待、游戏页）
- 建立可扩展的目录结构，方便后续添加 1V1 教学、试音、摸红扑克牌等更多玩法

---

## 路由结构

```
/                          → LandingPage（官方首页，新建）
/shop                      → ShopHomePage（原 / 的内容）
/shop/room/:code           → 神秘商店等待室（原 /room/:code）
/shop/room/:code/play      → 神秘商店游戏页（原 /room/:code/play）
/teach                     → TeachPage（1V1 定制教学，占位页）
/audition                  → AuditionPage（试音，占位页）
/poker                     → PokerPage（摸红扑克牌，占位页）
```

后端 Go 服务端的所有路径（`/api/rooms`、`/ws/rooms/:code`）**不变**。`vercel.json` 的 SPA rewrite 规则不变。

---

## 目录结构

```
src/
├── App.vue
├── main.ts
├── env.d.ts
│
├── router/
│   └── index.ts                        ← 聚合所有玩法 router.ts
│
├── pages/
│   └── LandingPage.vue                 ← 官方首页 /
│
├── games/                              ← 每个玩法完全自包含
│   ├── shop/
│   │   ├── pages/
│   │   │   ├── ShopHomePage.vue        ← /shop
│   │   │   ├── RoomPage.vue            ← /shop/room/:code
│   │   │   └── PlayPage.vue            ← /shop/room/:code/play
│   │   ├── components/
│   │   │   ├── EquipmentShop.vue
│   │   │   ├── FeedStream.vue
│   │   │   ├── LotteryPanel.vue
│   │   │   ├── LotteryRecordModal.vue
│   │   │   ├── PlayerSlot.vue
│   │   │   ├── SettlementModal.vue
│   │   │   └── SettlementRecordModal.vue
│   │   ├── net/
│   │   │   ├── api.ts
│   │   │   ├── client.ts
│   │   │   ├── config.ts
│   │   │   ├── protocol.ts
│   │   │   └── ws.ts
│   │   ├── game/
│   │   │   ├── equipment.ts
│   │   │   ├── lottery.ts
│   │   │   └── rules.ts
│   │   ├── stores/
│   │   │   └── room.ts
│   │   └── router.ts
│   │
│   ├── teach/
│   │   ├── pages/
│   │   │   └── TeachPage.vue           ← /teach（占位）
│   │   └── router.ts
│   │
│   ├── audition/
│   │   ├── pages/
│   │   │   └── AuditionPage.vue        ← /audition（占位）
│   │   └── router.ts
│   │
│   └── poker/
│       ├── pages/
│       │   └── PokerPage.vue           ← /poker（占位）
│       └── router.ts
│
└── shared/                             ← 跨玩法公共代码
    ├── components/
    │   ├── Avatar.vue
    │   ├── ConfirmDialog.vue
    │   └── ToastContainer.vue
    ├── stores/
    │   └── player.ts                   ← 玩家身份（playerId/nickname，跨玩法复用）
    ├── utils/
    │   ├── avatar.ts
    │   ├── id.ts
    │   └── storage.ts
    └── styles/
        └── main.css
```

### 路径别名（vite.config.ts）

```ts
resolve: {
  alias: {
    '@': '/src',               // 保持现有
    '@shop': '/src/games/shop',
    '@shared': '/src/shared',
  }
}
```

---

## 首页设计（LandingPage.vue）

风格延续现有深色 RPG 主题（parchment / gold 色系，`rpg-title` / `rpg-subtitle` 等现有样式类）。

### 区块结构

**1. 顶部导航栏**
- 左：`✦ 花栗鼠俱乐部`（Logo 文字）
- 右：`玩法` | `微信服务号`（文字导航，点击平滑滚动到对应区块）

**2. Hero 区**
- 副标题：`✦ 花栗鼠俱乐部 · 三角洲行动 ✦`
- 主标题：`花栗鼠俱乐部`（大字）
- 分隔线
- 口号：`开一家最有温度的三角洲陪玩店`
- 副标题：`原创 1V1 定制教学单 · 神秘商店趣味单`

**3. 玩法卡片区**（锚点 `#games`，对应导航"玩法"）

2×2 网格，4 张卡片：

| 卡片 | 路由 | 状态 | 样式 |
|---|---|---|---|
| 🎲 神秘商店 | `/shop` | 上线 | 金色边框，"立即开始 →" |
| 🎓 1V1 定制教学 | `/teach` | 占位 | 暗色，"即将推出" badge |
| 🎵 试音 | `/audition` | 占位 | 暗色，"即将推出" badge |
| 🃏 摸红扑克牌 | `/poker` | 占位 | 暗色，"即将推出" badge |

点击任意卡片均跳转到对应路由（占位页有内容后再填充）。

**4. Footer**
- `✥ 微信服务号 · 花栗鼠俱乐部 ✥`

---

## 文件迁移映射

所有文件**仅移动位置，内容不变**，只需更新 import 路径。

| 原路径 | 新路径 |
|---|---|
| `src/pages/HomePage.vue` | `src/games/shop/pages/ShopHomePage.vue` |
| `src/pages/RoomPage.vue` | `src/games/shop/pages/RoomPage.vue` |
| `src/pages/PlayPage.vue` | `src/games/shop/pages/PlayPage.vue` |
| `src/components/EquipmentShop.vue` | `src/games/shop/components/EquipmentShop.vue` |
| `src/components/FeedStream.vue` | `src/games/shop/components/FeedStream.vue` |
| `src/components/LotteryPanel.vue` | `src/games/shop/components/LotteryPanel.vue` |
| `src/components/LotteryRecordModal.vue` | `src/games/shop/components/LotteryRecordModal.vue` |
| `src/components/PlayerSlot.vue` | `src/games/shop/components/PlayerSlot.vue` |
| `src/components/SettlementModal.vue` | `src/games/shop/components/SettlementModal.vue` |
| `src/components/SettlementRecordModal.vue` | `src/games/shop/components/SettlementRecordModal.vue` |
| `src/net/api.ts` | `src/games/shop/net/api.ts` |
| `src/net/client.ts` | `src/games/shop/net/client.ts` |
| `src/net/config.ts` | `src/games/shop/net/config.ts` |
| `src/net/protocol.ts` | `src/games/shop/net/protocol.ts` |
| `src/net/ws.ts` | `src/games/shop/net/ws.ts` |
| `src/game/equipment.ts` | `src/games/shop/game/equipment.ts` |
| `src/game/lottery.ts` | `src/games/shop/game/lottery.ts` |
| `src/game/rules.ts` | `src/games/shop/game/rules.ts` |
| `src/stores/room.ts` | `src/games/shop/stores/room.ts` |
| `src/stores/player.ts` | `src/shared/stores/player.ts` |
| `src/components/Avatar.vue` | `src/shared/components/Avatar.vue` |
| `src/components/ConfirmDialog.vue` | `src/shared/components/ConfirmDialog.vue` |
| `src/components/ToastContainer.vue` | `src/shared/components/ToastContainer.vue` |
| `src/utils/avatar.ts` | `src/shared/utils/avatar.ts` |
| `src/utils/id.ts` | `src/shared/utils/id.ts` |
| `src/utils/storage.ts` | `src/shared/utils/storage.ts` |
| `src/styles/main.css` | `src/shared/styles/main.css` |

### 新增文件

- `src/pages/LandingPage.vue`
- `src/games/shop/router.ts`
- `src/games/teach/pages/TeachPage.vue`
- `src/games/teach/router.ts`
- `src/games/audition/pages/AuditionPage.vue`
- `src/games/audition/router.ts`
- `src/games/poker/pages/PokerPage.vue`
- `src/games/poker/router.ts`

---

## 占位页规范

三个未上线玩法的占位页采用统一模板：
- 沿用深色主题
- 显示玩法名称 + 图标
- 文案：「即将推出，敬请期待」
- 提供返回首页链接

---

## 后续扩展约定

新增一个玩法的步骤：
1. 在 `src/games/<name>/` 下新建目录
2. 创建 `pages/`（至少一个页面）和 `router.ts`
3. 在 `src/router/index.ts` 中 import 并注册
4. 在 `LandingPage.vue` 的玩法卡片区添加入口卡片

不需要修改任何其他玩法的代码。

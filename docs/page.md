# 三人对局 Web 应用 · 页面设计方案

> 一个 3 人房间的实时对局 Web 应用，**前后端分离** 架构。角色分为 **护航 ×2** 与 **老板 ×1**。
> 技术形态：**前端** Vue 3 + Vite + TypeScript；**后端** Go + Gorilla WebSocket（搭配 go-zero 的 `logx`/`conf` 与项目内 `pkg/errors`）。
> 通信：基于 **WebSocket** 的全双工长连接，**服务端权威（server-authoritative）**，所有业务规则、随机源与状态裁决均由 Go 服务端持有。
> 下文按"页面 → 模块 → 字段/交互/校验"的顺序描述，并在第 4–6 节给出完整的技术架构、消息协议与脚手架。

---

## 0. 全局约定

### 0.1 房间生命周期

| 状态 | 触发条件 | 说明 |
| --- | --- | --- |
| `WAITING` | 服务端创建房间成功 | 选角准备阶段，可加入 / 离开 / 切换角色 |
| `PLAYING` | 房主点击「开始游戏」 → 服务端裁决通过 | 锁定房间，新玩家不可加入 |
| `CLOSED` | 房主主动解散 / 全员离开 / 长时间无活动超时 | 房间销毁，Code 释放 |

> 房间的"权威状态"完全持有在 **Go 服务端内存中**（每个房间对应一个独立 goroutine 的 Room Loop）。三名玩家仅持有服务端推送的最近一次快照 + 本地乐观 UI。

### 0.2 房间 Code

- 6 位数字 Code 由 **Go 服务端**生成，全局唯一（碰撞则重试）。
- 创建流程：
  1. 客户端调用 `POST /api/rooms` 申请创建。
  2. 服务端生成 Code、创建 Room 上下文（goroutine + channel）、签发 `hostToken` 与 `sessionToken`，返回 `{ code, playerId, sessionToken }`。
  3. 客户端建立 WebSocket：`wss://<host>/ws/rooms/:code?playerId=...&token=...`。
  4. 路由跳转 `/room/:code`，当前玩家身份为 **房主（Host）**。
- 加入流程：
  1. 客户端可选预检：`GET /api/rooms/:code` 返回房间是否存在 / 状态 / 当前人数。
  2. 建立 WebSocket：`wss://<host>/ws/rooms/:code?playerId=...`，首帧发送 `JOIN { nickname, avatar? }`。
  3. 服务端校验通过 → 回 `JOIN_ACCEPTED { snapshot, selfPlayerId, sessionToken }`；否则 `JOIN_REJECTED { reason }` 并主动关闭连接。

### 0.3 实时同步（WebSocket）

- 所有同步走每位玩家与 Go 服务端之间的 **单条 WebSocket 长连接**。
- **拓扑**：星型，服务端是唯一中枢；客户端**不**互联。
- **消息方向**：
  - 客户端 → 服务端：**Intent / Action**（`JOIN`、`LEAVE`、`CHANGE_ROLE`、`READY`、`START_GAME`、`DISMISS_ROOM`、`BUY_EQUIPMENT`、`DRAW_LOTTERY`、`PING`）。
  - 服务端 → 客户端：**State / Event / Ack**（`JOIN_ACCEPTED`、`JOIN_REJECTED`、`STATE_SNAPSHOT`、`PLAYER_JOINED`、`PLAYER_LEFT`、`ROLE_CHANGED`、`READY_CHANGED`、`HOST_CHANGED`、`GAME_STARTED`、`COIN_CHANGED`、`EQUIPMENT_PURCHASED`、`LOTTERY_DRAWN`、`ACTION_REJECTED`、`KICK`、`PONG`）。
- **一致性**：所有业务规则由 Go 服务端裁决；客户端仅做乐观 UI，最终以服务端下发的 `STATE_SNAPSHOT` 为准。
- **消息编码**：JSON。后续若有性能需求可平滑切换到 MessagePack / Protobuf。
- **背压**：每个连接独立的写 channel（带缓冲），若写阻塞超阈值 → 强制断开该连接，由其重连补全。

### 0.4 保底金额（Guaranteed Amount）

- **定义**：房间级别的"保底金额"是一个**独立的展示性数值**，由房主在创建房间时设定，全程对房间内 3 位玩家同步展示。
- **单位**：`W`（万），整数。例如 `2500` 表示 `2500W`，UI 上展示为 `2500W`。
- **默认值**：`2500`（即 `2500W`）。
- **取值约束**：正整数，建议范围 `1 ~ 999999`；服务端做硬上下限校验，超出范围回 `ACTION_REJECTED { reason: 'INVALID_GUARANTEE' }` 或在创建接口直接返回 400。
- **可变性**：创建房间时设置初始值；进入对局（`PLAYING`）后保底金额只能通过以下两类**服务端权威动作**变化：
  1. **老板抽奖**（`DRAW_LOTTERY`）：每次抽中奖品后，服务端把奖品 `value`（单位 `W`）**累加**到当前 `guaranteedAmount`，详见 §3.2.2。
  2. **本局结算**（`SETTLE_ROUND`）：房主在每轮收束时手动调整（撤离成功扣减 / 炸单增加），详见 §3.4。

  除以上两类动作外的任何时机一律不可修改。
- **与金币的关系**：保底金额**不参与**任何装备扣费 / 抽奖消耗等金币结算运算，与金币系统独立存储与渲染。仅在 `DRAW_LOTTERY` / `SETTLE_ROUND` 时由服务端原子地改动其展示数值。前端在所有出现"金币"的地方都不可与保底金额混合展示或运算，二者必须分开渲染、分开存储。
- **同步范围**：房间内 3 名玩家（含房主）通过 `STATE_SNAPSHOT.guaranteedAmount` 拿到同一份数据，新加入或断线重连的玩家在 `JOIN_ACCEPTED` 携带的 `snapshot` 中即可获取，无需额外请求；任何变更通过 `GUARANTEE_CHANGED` 事件实时广播。

### 0.5 异常与边界（server-authoritative 下的约定）

- **客户端断线重连**：客户端凭 localStorage 中的 `playerId` + `roomCode` + 服务端在 `JOIN_ACCEPTED` 时下发的 `sessionToken`，在 30 秒内重连同一房间；服务端校验后恢复其座位/角色/资产，并对其单播 `STATE_SNAPSHOT`；超时则视为离开并广播 `PLAYER_LEFT`。
- **房主离线**：服务端继续运行；若房主断线超 30s 且房间内仍有其他玩家，服务端**自动将房主权限迁移**给当前最早加入且在线的玩家，并广播 `HOST_CHANGED { newHostId }`，前端无需关心 host migration 细节。
- **服务端重启 / 节点宕机**：本期房间状态仅存内存，重启即丢失。客户端会收到 WebSocket `close`，弹窗"服务异常，请稍后再试"，3 秒后回首页。后续可扩展 Redis 持久化（见 §7）。
- **网络抖动**：客户端封装自动重连（指数退避，最多 5 次），重连成功后由服务端下发完整 `STATE_SNAPSHOT` 对账，本地乐观状态以快照覆盖。
- **心跳**：客户端每 20s 发 `PING`，服务端 60s 内无任何消息视为掉线并断开。

---

## 1. 首页 / 大厅页（Home Page）

进入应用的第一个页面，仅负责"创建"与"加入"。

### 1.1 视觉

- 居中两个主操作按钮，作为视觉中心。
- 顶部 Logo / 应用名，底部可放规则简介或版本号。

### 1.2 模块

#### 「创建房间」模块

- **昵称输入框**：必填，长度限制 1~16 字。
- **保底金额输入框**：必填，纯数字，单位 `W`（万），**默认值 `2500`**（即 `2500W`）。
  - 输入框右侧固定显示后缀 `W` 提示单位；下方副本提示"保底金额为对局展示值，与金币系统独立；房主在每局结算时可单独调整"，避免与金币混淆。
  - 本地校验：必须为正整数，范围 `1 ~ 999999`；非法时按钮置灰并行内提示"请输入 1~999999 之间的整数"。
- **创建按钮**：
  1. 调用 `POST /api/rooms { nickname, avatar?, guaranteedAmount }` → 服务端返回 `{ code, playerId, sessionToken }`。
  2. 建立 WebSocket → 收到 `JOIN_ACCEPTED` 后跳转 `/room/:code`，当前玩家身份为 **房主**。
  3. 创建失败（极少见：Code 池耗尽 / 服务端过载 / 保底金额非法）→ 提示"创建失败请重试"或服务端返回的具体原因。

#### 「加入房间」模块

- 输入框：6 位 Code，限制只输入数字。
- 「加入」按钮：
  1. 可选前置：`GET /api/rooms/:code` 返回 404 → 直接提示"房间不存在或已关闭"，不再建立连接。
  2. 建立 WebSocket → 首帧发送 `JOIN { nickname, avatar? }`。
  3. 收到 `JOIN_ACCEPTED` → 跳转 `/room/:code`；收到 `JOIN_REJECTED` → 显示对应提示并关闭连接。

### 1.3 校验提示（统一 Toast / 行内提示）

| 场景 | 触发判定 | 提示文案 |
| --- | --- | --- |
| 保底金额非法 | 本地正则校验 / 服务端 400 | "请输入 1~999999 之间的整数" |
| Code 格式错误（非 6 位数字） | 本地正则校验 | "请输入 6 位房间号" |
| Code 不存在 / 房间已关闭 | `GET /api/rooms/:code` 返回 404 或 `JOIN_REJECTED { reason: 'NOT_FOUND' }` | "房间不存在或已关闭" |
| 房间已满（3 / 3） | `JOIN_REJECTED { reason: 'FULL' }` | "房间已满" |
| 房间已开始游戏 | `JOIN_REJECTED { reason: 'PLAYING' }` | "游戏已开始，无法加入" |
| 重复进入（playerId 已在房间内且连接活跃） | `JOIN_REJECTED { reason: 'DUP' }` | "该账号已在房间内" |
| WebSocket 握手失败 / 鉴权失败 | HTTP 401/403 或异常关闭码 | "网络异常，请稍后重试" |

---

## 2. 选角与准备页（Room / Role Selection Page）

路由：`/room/:code`。玩家进入房间后分配角色并准备。

### 2.1 顶部信息栏

- **房间号**：`房间 Code: XXXXXX`，附「一键复制」按钮（复制纯 6 位数字）。
- **房间链接**：可选，复制后是直达加入链接（`/?code=XXXXXX`）。
- **当前人数**：`2 / 3`，来源于服务端下发的 `STATE_SNAPSHOT.players.length`。
- **保底金额**：显著位置展示 `保底：2500W`（数值取自 `STATE_SNAPSHOT.guaranteedAmount`，单位固定 `W`），3 位玩家看到的数据完全一致；该字段不可在此页修改，仅展示。
- **退出房间**：右上角次要按钮。
  - 房主点击：二次确认 → 发送 `DISMISS_ROOM`，服务端广播 `KICK { reason: 'DISMISSED' }` 后销毁房间。
  - 客户端点击：二次确认 → 发送 `LEAVE`，服务端广播 `PLAYER_LEFT`。

### 2.2 玩家列表区（中央）

- 3 个固定 Slot 卡片，水平或网格排列。
- 已加入的 Slot 显示：头像 / 昵称 / 当前角色徽标 / 是否准备 / 是否房主（皇冠图标）。
- 空 Slot 显示："等待加入..."。

### 2.3 角色选择区（底部）

| 按钮 | 名额 | 客户端动作 | 服务端裁决 |
| --- | --- | --- | --- |
| 护航 | 2 | 发送 `CHANGE_ROLE { role: 'ESCORT' }` | 名额未满 → 广播 `ROLE_CHANGED`；已满 → 回 `ACTION_REJECTED { reason: 'ROLE_FULL' }` |
| 老板 | 1 | 发送 `CHANGE_ROLE { role: 'BOSS' }` | 未被占用 → 广播 `ROLE_CHANGED`；已占用 → 回 `ACTION_REJECTED { reason: 'ROLE_TAKEN' }` |

- 玩家可以**反复切换**角色（在未点击准备前）。
- 自己当前角色按钮高亮显示。
- 客户端可做乐观 UI 反馈，但最终以服务端下发的 `STATE_SNAPSHOT` 为准。

### 2.4 操作区

- **「准备 / 取消准备」**：所有玩家可见，向服务端发送 `READY { ready: boolean }`；服务端广播 `READY_CHANGED`。切换角色会被服务端自动取消准备并广播。
- **「开始游戏」**：仅房主可见且**逻辑在服务端裁决**，需同时满足以下条件才亮起：
  1. 房间人数 = 3
  2. 角色配置为 1 老板 + 2 护航
  3. 所有玩家均已准备
- 不满足时按钮置灰，hover 时提示缺失的条件（"等待玩家准备" / "角色未分配完整"）。
- 房主点击 → 客户端发送 `START_GAME`，服务端二次校验通过后：房间状态切到 `PLAYING`，初始化资产（护航 10 币 / 老板 0 币），广播 `GAME_STARTED { snapshot }`，全员路由跳转 `/room/:code/play`。

---

## 3. 游戏主控制台（Main Game Dashboard）

路由：`/room/:code/play`。游戏开始后所有玩家进入。**根据角色动态渲染核心操作区**，公共信息区共享。

> 本页面所有"金币变化 / 抽奖结果 / 装备购买"等动作**都必须先发请求给服务端**，由服务端裁决并广播；任何客户端本地都不允许直接修改自己的金币/物品状态（防作弊）。

### 3.1 全局导航栏（顶部，所有人可见）

- **我的角色**：`我的角色：老板` / `我的角色：护航`，源自服务端下发的快照。
- **房间保底金额**：常驻展示 `保底：2500W`（取自 `STATE_SNAPSHOT.guaranteedAmount`），3 位玩家可见且一致；UI 与金币模块分开渲染，**不可与金币数值合并展示或参与运算**。
- **我的资产**：实时金币数量，源自快照。
  - 护航初始 **10 币**
  - 老板初始 **0 币**
- **队友资产**：另外 2 名玩家昵称 + 角色 + 当前金币数（小卡片或图标）。
- **本局结算（仅房主可见）**：主操作按钮，醒目色彩。仅当当前玩家 `isHost === true` 且 `status === 'PLAYING'` 时渲染；点击后打开「本局结算」Modal（详见 §3.4）。非房主完全不渲染，避免误触。
- **退出房间**：二次确认后返回首页。
  - 客户端：发 `LEAVE` 后关闭 WebSocket。
  - 房主：发 `DISMISS_ROOM`，全员被踢回首页。

### 3.2 核心操作区（左 / 中，按角色渲染）

#### 3.2.1 护航视角（俗称「打手」）

- **装备商店面板**：固定展示以下装备清单。
- 每件装备显示：名称 / 图标 / 价格 / 简介。
- 操作：「购买」→ 发送 `BUY_EQUIPMENT { equipmentId, pick? }`，服务端校验金币是否足够后扣款并广播 `COIN_CHANGED` + `EQUIPMENT_PURCHASED`。

##### 装备清单（护航可购买）

| 类别 | 装备 | `equipmentId` | 价格（币） | 说明 |
| --- | --- | --- | --- | --- |
| 护甲 | 4 级护甲 | `ARMOR_L4` | 1 | 基础护甲 |
| 护甲 | 5 级护甲 | `ARMOR_L5` | 2 | 进阶护甲 |
| 护甲 | 6 级护甲 | `ARMOR_L6` | 3 | 顶级护甲 |
| 武器组合 | AW + 任意枪 | `BUNDLE_AW` | 4 | AW（高级狙击）+ 任意枪，整组直接购买，无需二级选枪 |
| 武器组合 | M7 + 任意狙击枪 | `BUNDLE_M7` | 3 | M7（突击步枪）+ 任意狙击枪，整组直接购买，无需二级选枪 |
| 单件武器 | 基础枪械套装 | `WEAPON_SINGLE` | 2 | 除 T0 枪械以外的任意单枪 + 任意一把狙击枪（AW 除外），直接购买，无需二级选枪 |
| 单件武器 | 单枪匹马 | `WEAPON_LONE` | 1 | 除 T0 枪械以外的任意单枪，直接购买，无需二级选枪 |

> 备注：
> - 护航初始金币为 12，可任意组合上述装备，**不限购买次数**（受余额约束）。
> - 上述全部条目均为**直接购买**：客户端只发 `BUY_EQUIPMENT { equipmentId }`，扣费即可，不再弹出二级选枪面板、不再携带 `pick` 字段。服务端若收到多余的 `pick`，回 `ACTION_REJECTED { reason: 'INVALID_PARAM' }`（`PICK_NOT_REQUIRED`）。
> - 服务端在 `internal/game/equipment.go` 维护这份清单（价格、`equipmentId`），是装备校验的唯一权威来源。

#### 3.2.2 老板视角

- **盲盒 / 抽奖机面板**：突出视觉效果（开箱动画 / 转盘 / 卡牌翻转皆可）。
- 操作：
  - 「单抽（消耗 2 币）」 → 发送 `DRAW_LOTTERY { times: 1 }`
- 抽奖**结果由服务端**完成随机（`crypto/rand`，保证唯一可信随机源），通过 `LOTTERY_DRAWN { records }` 广播给全员。
- 每件抽中奖品都带有**官方估值 `value`（单位 `W`）**，服务端在裁决抽奖的同 tick **原子地**把 `value` 累加到房间 `guaranteedAmount` 上，并随后广播 `GUARANTEE_CHANGED { before, after }`（详见 §3.2.2 服务端裁决与 §4.4）。
- 客户端收到广播后播放开箱动画 + 物品详情（含价值标签），并将事件追加到「实时播报」（包含本次保底增量）。

##### 抽奖奖品池（老板专属）

> 该奖品池是**服务端权威配置**，前端不应硬编码；服务端在 `internal/game/lottery.go` 维护下表（`prizeId`、名称、`value`、稀有度、抽中权重），是抽奖随机与价值累计的唯一来源。每一次抽中的奖品，其 `value` 会原子地累加到当前房间的 `guaranteedAmount`，从而推高保底。

| 奖品名称 | `prizeId` | 价值（W） | 备注 |
| --- | --- | --- | --- |
| 便携军用雷达 | `RADAR_PORTABLE` | 215 | 高价值奖品 |
| 曼德尔超算单元 | `MANDEL_SUPERCOMPUTER` | 200 | 高价值奖品 |
| 高速磁盘阵列 | `DISK_ARRAY_HS` | 170 | 高价值奖品 |
| 军用无人机 | `MIL_DRONE` | 50 | 中价值奖品 |
| 军用信息终端 | `MIL_INFO_TERMINAL` | 38 | 中价值奖品 |
| 香槟 | `CHAMPAGNE` | 34 | 中价值奖品 |
| 显卡 | `GPU` | 33 | 中价值奖品 |
| 量子存储 | `QUANTUM_STORAGE` | 30 | 中价值奖品 |
| 定位接收器 | `GPS_RECEIVER` | 18 | 低价值奖品 |
| 恒星敏感器 | `STAR_SENSOR` | 18 | 低价值奖品 |
| 鱼子酱 | `CAVIAR` | 17 | 低价值奖品 |

> 备注：
> - 该 11 条目即为当前奖品池全集；新增 / 调整时仅修改 `internal/game/lottery.go`，前端通过 `LOTTERY_DRAWN` 与 `STATE_SNAPSHOT.lotteryRecords` 自然同步，无需重复维护静态文案。
> - 抽中权重 / 稀有度由服务端单独配置（建议高价值奖品权重更低），与 `value` 字段解耦；前端不感知权重。
> - `value` 必须为正整数；累加后 `newGuarantee` 仍受 §0.4 的 `1 ~ 999999` 范围约束，越界则服务端回 `ACTION_REJECTED { reason: 'INVALID_GUARANTEE' }`，本次抽奖整体回滚（金币不扣、保底不变、不下发 `LOTTERY_DRAWN`）。

##### 服务端裁决（抽奖 → 保底联动）

1. 校验老板身份、`status === 'PLAYING'`、`coin >= 2`，否则 `ACTION_REJECTED`。
2. 扣 2 币（`coin -= 2`），从奖品池按权重随机一件 `prize`。
3. 计算 `newGuarantee = currentGuarantee + prize.value`，越界回滚（见上）。
4. 原子提交：扣币 + 增保底 + 追加一条 `LotteryRecord`（含 `guaranteeDelta = prize.value`、`guaranteeBefore` / `guaranteeAfter`）+ 追加 `kind = 'LOTTERY'` 的 `FeedEvent`。
5. 广播顺序（同 tick 内连续）：
   1. `COIN_CHANGED { changes: [{ playerId: bossId, delta: -2, balance }] }`
   2. `GUARANTEE_CHANGED { before, after }`
   3. `LOTTERY_DRAWN { records: [LotteryRecord] }`

> **经济系统说明**：老板初始 0 币，必须依赖以下机制之一才能启动抽奖（在后续机制设计中明确）：
> - **时间/事件奖励**：老板每隔 N 秒或触发某事件后获得固定币。
> - **任务结算**：完成阶段任务后老板获得收益。

### 3.3 公共信息与记录区（右侧 / 抽屉，所有人可见）

#### 实时播报（Feed 流）

- 公屏滚动，最近 N 条事件，例如：
  - `[老板] 抽中了 [便携军用雷达]（价值 215W）· 保底 +215W（2500 → 2715）`
  - `[护航 B] 购买了 [装备 A]。`
  - `[本局结算] 撤离成功 · 保底 -200W · A 获得 8 币 / B 获得 5 币 / C 获得 3 币`
- 新事件高亮一次后归入普通样式。

#### 抽奖记录入口

- 「查看所有抽奖记录」按钮 → 弹出 Modal。
- Modal 内列表，按**时间倒序**排列：

| 时间 | 抽奖人 | 消耗金币 | 抽中物品 | 价值（W） | 保底变化 |
| --- | --- | --- | --- | --- | --- |
| 12:01:33 | 老板 | 2 | 鱼子酱 | 17 | +17W（2715 → 2732） |
| 12:01:10 | 老板 | 2 | 便携军用雷达 | 215 | +215W（2500 → 2715） |

- 该列表在 3 个玩家页面上**保持实时一致**（来源为服务端广播的 `LOTTERY_DRAWN` 流；服务端端持有完整记录数组，新加入/重连玩家通过 `STATE_SNAPSHOT.lotteryRecords` 一次性补齐）。
- "价值" 与 "保底变化" 两列都来自 `LotteryRecord` 自身字段（`item.value` / `guaranteeDelta` / `guaranteeBefore` / `guaranteeAfter`），前端不做二次计算，保证与服务端权威值一致。

#### 结算记录入口

- 「查看本局结算记录」按钮（所有玩家可见）→ 弹出 `SettlementRecordModal`，按**时间倒序**展示历史结算明细：

| 时间 | 结果 | 保底变化 | 玩家 A | 玩家 B | 玩家 C |
| --- | --- | --- | --- | --- | --- |
| 12:30:10 | 撤离成功 | -200W（2500 → 2300） | 8 币（击杀1/小红2/6格1/9格1） | 5 币（小红2） | 3 币（基础） |
| 12:42:01 | 炸单 | +60W（2300 → 2360） | 0 币 | 0 币 | 0 币 |

- 数据来源：服务端广播的 `ROUND_SETTLED` 流；服务端持有完整 `settlementRecords`，新加入 / 重连的玩家通过 `STATE_SNAPSHOT.settlementRecords` 一次性补齐。

### 3.4 本局结算（房主专属）

> 房主在 `PLAYING` 状态可手动触发本局结算：调整保底金额并依据本局战绩为 3 位玩家发放金币。结算只是房间生命周期内的一次"对局收束"事件，不会自动改变房间状态（仍为 `PLAYING`），房主可继续游戏直到 `DISMISS_ROOM`。

#### 3.4.1 入口

- 入口按钮位于全局导航栏（详见 §3.1），仅 `isHost && status === 'PLAYING'` 时渲染。
- 点击后打开 `SettlementModal`，背景遮罩；Modal 不阻塞服务端事件接收（Feed / 资产仍会实时更新）。

#### 3.4.2 Modal 结构

##### A. 本局结果（必选，单选）

| 选项 | `outcome` | 保底金额变化 | 每人额外金币 |
| --- | --- | --- | --- |
| 撤离成功 | `EVACUATED` | **扣减**输入金额（单位 `W`） | **+3 币 / 人**（含老板） |
| 炸单 | `BOMBED` | **增加**输入金额（单位 `W`，**默认 60**） | 0 |

- 选中后下方动态显示对应输入框：
  - `EVACUATED` → 「扣减保底（W）」，无默认值，placeholder 为示例（如 `200`）。
  - `BOMBED` → 「增加保底（W）」，**默认填入 `60`**，房主可改写。
- 输入约束（前端先校验，服务端再次裁决）：
  - 必须为非负整数；范围 `0 ~ 999999`。
  - `EVACUATED`：要求 `currentGuarantee - delta >= 0`，否则按钮置灰并行内提示"保底金额不足"。
  - `BOMBED`：要求 `currentGuarantee + delta <= 999999`，否则置灰并提示"超出保底上限"。

##### B. 三位玩家结算明细（必填表格，按角色区分录入）

**板板（老板）行**：录入完整战绩。

| 玩家（昵称 + 角色徽标） | 击杀（人） | 小红（个） | 6 格（个） | 9 格及以上（个） |
| --- | --- | --- | --- | --- |
| 板板 | 输入 | 输入 | 输入 | 输入 |

**护航行**：仅录入击杀，并以「救助板板：是 / 否」开关替代小红 / 6 格 / 9 格三列。

| 玩家（昵称 + 角色徽标） | 击杀（人） | 救助板板 |
| --- | --- | --- |
| 护航 | 输入 | 是 / 否 |

- 默认击杀为 `0`，救助默认为「否」。
- 校验：数值格必须为非负整数，范围 `0 ~ 99`；非法时按钮置灰并行内高亮该单元格。
- 行序按 `STATE_SNAPSHOT.players` 顺序渲染，便于房主对照实际玩家。
- 表格需覆盖房间内**全部 3 位玩家**，不能遗漏（漏填即视为 0）。

##### C. 奖励规则提示卡片（只读）

- **板板**：击杀 1 人 +1 币，摸 1 个小红 +1 币，摸 1 个 6 格 +2 币，摸 1 个 9 格及以上 +3 币。
- **护航**：击杀 1 人 +1 币，救助板板 +1 币（**不再通过小红 / 6 格 / 9 格获得币**）。
- 撤离成功额外 +3 币 / 人（板板与护航通用）。

##### D. 底部预览栏（实时计算，纯前端展示）

- 实时显示三位玩家本次结算预计获得的金币（**仅前端预览，最终以服务端 `ROUND_SETTLED` 为准**）。
  - 板板：`coinDelta = kills*1 + smallReds*1 + sixGrids*2 + nineGrids*3 + (outcome === 'EVACUATED' ? 3 : 0)`
  - 护航：`coinDelta = kills*1 + (rescue ? 1 : 0) + (outcome === 'EVACUATED' ? 3 : 0)`
- 实时显示新的保底金额（`currentGuarantee ± delta`）与变化量。

##### E. 操作按钮

- **「取消」**：关闭 Modal，不发送任何 Intent。
- **「确认结算」**：
  1. 前端做一次最终校验，未通过则置灰并提示。
  2. 通过则发送 `SETTLE_ROUND` Intent（payload 见 §4.4），按钮在等待服务端响应期间置灰，文案变为"结算中…"。
  3. 收到 `ROUND_SETTLED` 广播 → 关闭 Modal 并 Toast 提示"结算成功"；收到 `ACTION_REJECTED` → 保留 Modal、显示错误原因。

#### 3.4.3 服务端裁决

- **权限**：仅房主可发起；非房主或 `status != PLAYING` → `ACTION_REJECTED { reason: 'FORBIDDEN' }`。
- **保底金额校验**：
  - `EVACUATED`：`newGuarantee = current - delta`，要求 `newGuarantee >= 0`，否则 `ACTION_REJECTED { reason: 'INVALID_GUARANTEE' }`。
  - `BOMBED`：`newGuarantee = current + delta`，要求 `newGuarantee <= 999999`，否则同样 `ACTION_REJECTED { reason: 'INVALID_GUARANTEE' }`。
  - `delta` 必须为 `0 ~ 999999` 的整数，否则 `ACTION_REJECTED { reason: 'INVALID_PARAMS' }`。
- **玩家明细校验**：必须涵盖房间内全部 3 位玩家（按 `playerId` 比对），每个字段为 `0 ~ 99` 的整数；不合规 → `ACTION_REJECTED { reason: 'INVALID_PARAMS' }`。
- **金币计算**：`coinDelta = kills + smallReds + sixGrids*2 + nineGrids*3 + (outcome === EVACUATED ? 3 : 0)`，整数算术，不允许负值。
- **原子更新（在 Room Loop 单 goroutine 内一次性完成）**：
  1. 更新 `guaranteedAmount` 为 `newGuarantee`。
  2. 更新各玩家 `coin += coinDelta`。
  3. 追加一条 `SettlementRecord` 到 `settlementRecords`。
  4. 追加一条 `kind = 'SETTLEMENT'` 的 `FeedEvent`。
- **广播顺序**（同一 tick 内连续发出，客户端按顺序应用）：
  1. `GUARANTEE_CHANGED { before, after }`（仅当 `delta != 0` 时发送）。
  2. `COIN_CHANGED { changes: [{ playerId, delta, balance }] }`（3 条变更，零变化也发送以保证客户端一致性）。
  3. `ROUND_SETTLED { record }`：完整结算记录广播，作为本次结算的"权威收束"。

> 房间状态保持 `PLAYING`，不会因结算自动结束；房主可在多次结算之间继续游戏 / 继续抽奖 / 继续购买装备，直到主动 `DISMISS_ROOM`。

---

## 4. 技术架构（前端 Vue 3 + 后端 Go · server-authoritative）

### 4.1 拓扑

```
                    ┌──────────────────────────────────┐
                    │       Go WebSocket Server         │
                    │  (Hub + per-Room Goroutine Loop)  │
                    │                                   │
                    │   ┌─────┐  ┌─────┐  ┌─────┐       │
                    │   │ R1  │  │ R2  │  │ R3  │  ...  │
                    │   └─────┘  └─────┘  └─────┘       │
                    └────────────────┬──────────────────┘
                                     │ WebSocket (wss)
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
        ┌─────▼─────┐           ┌────▼─────┐          ┌─────▼─────┐
        │ Escort A  │           │   Host    │         │ Escort B  │
        │ (Vue 3)   │           │  (Vue 3)  │         │ (Vue 3)   │
        └───────────┘           └───────────┘         └───────────┘
```

- 三名玩家**各自**与 Go 服务端建立独立 WebSocket，不存在客户端之间的直连。
- 服务端为每个房间维护一个 goroutine（Room Loop），所有 Intent 串行排队处理，无需在房间内部加锁。
- 业务规则、随机源、状态裁决全部由服务端持有；房主（Host）仅是房间内享有 `START_GAME` / `DISMISS_ROOM` 权限的特殊玩家。

### 4.2 服务端职责（Authoritative Server）

| 模块 | 职责 |
| --- | --- |
| `Hub` | 全局房间表（`map[code]*Room`），创建/销毁房间、分配 Code、按 Code 路由入站连接 |
| `Room` | 房间核心。一个 goroutine + 一个 inbox channel，串行处理所有 Intent；维护 `status` / `players` / `lotteryRecords` / `feed`；负责广播 |
| `Client` | 单条 WebSocket 连接对应的读 goroutine + 写 goroutine + 写 channel；只做"反序列化 → 投递到 Room.inbox"和"消费写队列 → 序列化下发" |
| `protocol` | 统一定义 Envelope + 所有 Intent / Event 的 Go 结构体（与前端 TS 类型一一对应） |
| `game` | 纯函数业务规则：角色名额校验、抽奖随机、装备扣费 |
| 准入控制 | `JOIN` 时校验房间状态 / 人数 / playerId 重复 / sessionToken |
| 心跳与超时 | 读超时 60s；30s 无新消息触发 `PING`；超时无响应断连，30s 不重连视为 `PLAYER_LEFT` |
| 房主迁移 | 房主断线超时 → 选最早加入的在线玩家继承 `hostId`，广播 `HOST_CHANGED` |
| 日志 | 全链路 `logx.WithContext(ctx)`，每个 Room 携带 `code` field |
| 错误 | 统一项目 `pkg/errors`，对前端暴露的错误经 `ACTION_REJECTED` 的 `reason` 枚举传递 |

### 4.3 客户端职责（Thin Client）

- 仅维护服务端下发的最新快照（Pinia store）。
- 把所有用户操作转换成 Intent 通过 WebSocket 发出，并维护 `reqId → resolver` 的临时回执表用于错误回滚。
- 渲染层可做 100~300ms 的乐观 UI（如点击「准备」立即灰按钮），收到 `ACTION_REJECTED` 则回滚到快照值。
- 监听 WebSocket `close` → 自动重连（指数退避，最多 5 次）；最终失败 → 弹窗 + 回首页。

### 4.4 消息协议（WebSocket 上的 JSON）

所有消息统一采用 envelope：

```ts
type Envelope<T extends string = string, P = unknown> = {
  type: T;          // 消息类型，全大写 SNAKE_CASE
  payload: P;       // 业务负载
  reqId?: string;   // 客户端发起 Intent 时携带，便于服务端回执对账
  ts: number;       // 发送时间戳（毫秒）
};
```

#### Client → Server（Intent）

| type | payload | 说明 |
| --- | --- | --- |
| `JOIN` | `{ nickname, avatar? }` | 首帧。WS 连接的 query 已带 `playerId` / `sessionToken` |
| `LEAVE` | `{}` | 主动离开（非房主） |
| `DISMISS_ROOM` | `{}` | 房主解散房间 |
| `CHANGE_ROLE` | `{ role: 'BOSS' \| 'ESCORT' }` | 选/换角色 |
| `READY` | `{ ready: boolean }` | 准备/取消 |
| `START_GAME` | `{}` | 仅房主有效 |
| `BUY_EQUIPMENT` | `{ equipmentId: string }` | 护航购买装备；所有条目均直接扣费购买，无需 `pick` |
| `DRAW_LOTTERY` | `{ times: 1 }` | 老板单次抽奖（消耗 2 币） |
| `SETTLE_ROUND` | `{ outcome: 'EVACUATED' \| 'BOMBED', guaranteeDelta: number, perPlayer: { playerId: string, kills: number, smallReds: number, sixGrids: number, nineGrids: number, rescue?: boolean }[] }` | 房主结算本局。`guaranteeDelta` 为非负整数（单位 W）；`outcome === 'EVACUATED'` 时表示扣减、`'BOMBED'` 时表示增加。`perPlayer` 必须涵盖房间内全部 3 位玩家。服务端按角色区分计币：板板看战绩，护航看击杀与 `rescue`（救助板板 +1 币，护航不再通过小红 / 6 格 / 9 格获得币）。仅 `status === 'PLAYING'` 且发起者为房主时有效。 |
| `PING` | `{}` | 心跳 |

#### Server → Client（Event / Ack）

| type | payload | 说明 |
| --- | --- | --- |
| `JOIN_ACCEPTED` | `{ snapshot: RoomSnapshot, selfPlayerId, sessionToken }` | 准入成功 + 初始快照 |
| `JOIN_REJECTED` | `{ reason: 'NOT_FOUND' \| 'FULL' \| 'PLAYING' \| 'DUP' \| 'OTHER', message? }` | 拒绝加入并主动关闭连接 |
| `STATE_SNAPSHOT` | `{ snapshot: RoomSnapshot }` | 完整快照（重连/关键状态变更后下发） |
| `PLAYER_JOINED` | `{ player: Player }` | 增量：新成员 |
| `PLAYER_LEFT` | `{ playerId: string }` | 增量：离开 |
| `ROLE_CHANGED` | `{ playerId, role }` | 增量：角色变更 |
| `READY_CHANGED` | `{ playerId, ready }` | 增量：准备状态 |
| `HOST_CHANGED` | `{ newHostId }` | 房主迁移 |
| `GAME_STARTED` | `{ snapshot: RoomSnapshot }` | 进入对局 |
| `COIN_CHANGED` | `{ changes: { playerId, delta, balance }[] }` | 资产变化（扣费、奖励都走这里） |
| `EQUIPMENT_PURCHASED` | `{ playerId, equipmentId }` | 装备购买广播 |
| `LOTTERY_DRAWN` | `{ records: LotteryRecord[] }` | 抽奖结果广播（单抽 1 条）。每条 `LotteryRecord` 自带 `item.value` 与 `guaranteeDelta`，与同 tick 内先行下发的 `GUARANTEE_CHANGED` 一致 |
| `GUARANTEE_CHANGED` | `{ before: number, after: number }` | 保底金额变化广播；由 `DRAW_LOTTERY`（按奖品 `value` 累加）或 `SETTLE_ROUND`（撤离扣减 / 炸单增加）触发；`before == after` 时不发送 |
| `ROUND_SETTLED` | `{ record: SettlementRecord }` | 本局结算结果广播（含本次保底变化与每位玩家金币变化），与配套的 `GUARANTEE_CHANGED` / `COIN_CHANGED` 同 tick 内连续发出 |
| `ACTION_REJECTED` | `{ reqId, reason, message? }` | 通用动作拒绝回执，`reason` 为枚举（含 `FORBIDDEN` / `INVALID_GUARANTEE` / `INVALID_PARAMS` 等） |
| `KICK` | `{ reason: 'DISMISSED' \| 'IDLE' \| 'KICKED_BY_HOST' }` | 被服务端强制下线 |
| `PONG` | `{}` | 心跳回应 |

### 4.5 REST 接口（仅初始化与运维用）

| Method | Path | 说明 |
| --- | --- | --- |
| `POST` | `/api/rooms` | 创建房间。Body: `{ nickname, avatar?, guaranteedAmount }`（`guaranteedAmount` 单位 `W`，整数，默认 `2500`，范围 `1~999999`），返回 `{ code, playerId, sessionToken, guaranteedAmount }` |
| `GET`  | `/api/rooms/:code` | 预查房间状态，返回 `{ exists, status, playerCount }`；不存在返回 404 |
| `GET`  | `/api/healthz` | 健康检查（K8s 探针） |
| `GET`  | `/ws/rooms/:code` | WebSocket 升级入口，query: `playerId` / `sessionToken`（重连必带） |

### 4.6 异常兜底

| 场景 | 处理 |
| --- | --- |
| WebSocket 单次断连 | 客户端指数退避自动重连（最多 5 次），重连成功后服务端单播 `STATE_SNAPSHOT` |
| 房主离线超 30s | 服务端把房主权限迁移给最早加入的在线玩家，广播 `HOST_CHANGED` |
| 所有玩家离线 | 房间空闲 60s 后服务端清理，释放 Code |
| 服务端重启 | 房间全部销毁，客户端弹窗"服务异常，请稍后再试"后回首页 |
| 写阻塞 / 慢客户端 | 写 channel 缓冲满 → 强制关闭该连接，由其重连补全 |

---

## 5. 数据模型

### 5.1 前端类型（TypeScript，运行时快照）

```ts
type Room = {
  code: string;                              // 6 位数字
  status: 'WAITING' | 'PLAYING' | 'CLOSED';
  hostId: string;                            // 房主 playerId（服务端裁决，可能因迁移而变化）
  players: Player[];                         // 长度 0~3
  guaranteedAmount: number;                  // 保底金额（单位 W，整数；创建时设定初值，PLAYING 期间仅可经 SETTLE_ROUND 调整；与金币系统隔离）
  lotteryRecords: LotteryRecord[];           // 全量抽奖记录
  settlementRecords: SettlementRecord[];     // 全量本局结算记录
  feed: FeedEvent[];                         // 实时播报队列（保留最近 N 条）
  createdAt: number;
};

type Player = {
  id: string;                                // playerId（本地 nanoid，首次访问生成并持久化到 localStorage）
  nickname: string;
  avatar?: string;
  role: 'BOSS' | 'ESCORT' | null;
  ready: boolean;
  coin: number;
  isHost: boolean;
  connected: boolean;                        // 服务端视角下的连接状态
};

type LotteryItem = {
  id: string;          // prizeId，例如 'RADAR_PORTABLE'
  name: string;        // 中文奖品名，例如 '便携军用雷达'
  rarity: string;      // 稀有度（服务端配置）
  value: number;       // 奖品价值（单位 W，正整数），抽中即累加到 guaranteedAmount
};

type LotteryRecord = {
  id: string;
  playerId: string;
  cost: number;                 // 本次抽奖消耗的金币（当前固定 2）
  item: LotteryItem;
  guaranteeDelta: number;       // = item.value，单位 W；服务端权威值
  guaranteeBefore: number;      // 抽奖前保底金额
  guaranteeAfter: number;       // 抽奖后保底金额（= before + guaranteeDelta）
  at: number;
};

type SettlementOutcome = 'EVACUATED' | 'BOMBED';

type SettlementPerPlayer = {
  playerId: string;
  kills: number;        // 0~99
  smallReds: number;    // 0~99
  sixGrids: number;     // 0~99
  nineGrids: number;    // 0~99
  coinDelta: number;    // 服务端最终计算的本次结算金币变化
};

type SettlementRecord = {
  id: string;
  at: number;
  outcome: SettlementOutcome;
  guaranteeDelta: number;       // 非负整数（W）；EVACUATED 表示扣减额，BOMBED 表示增加额
  guaranteeBefore: number;      // 结算前保底金额
  guaranteeAfter: number;       // 结算后保底金额
  perPlayer: SettlementPerPlayer[]; // 必为 3 项，按 players 顺序
};

type FeedEvent =
  | { id: string; at: number; kind: 'LOTTERY'; playerId: string; itemName: string; rarity: string; itemValue: number; guaranteeDelta: number; guaranteeAfter: number }
  | { id: string; at: number; kind: 'PURCHASE'; playerId: string; equipmentId: string }
  | { id: string; at: number; kind: 'SETTLEMENT'; outcome: SettlementOutcome; guaranteeDelta: number };

type RoomSnapshot = Pick<Room, 'code' | 'status' | 'hostId' | 'players' | 'guaranteedAmount' | 'lotteryRecords' | 'settlementRecords' | 'feed'>;
```

### 5.2 后端类型（Go，权威状态）

```go
package protocol

type RoomStatus string

const (
    RoomStatusWaiting RoomStatus = "WAITING"
    RoomStatusPlaying RoomStatus = "PLAYING"
    RoomStatusClosed  RoomStatus = "CLOSED"
)

type Role string

const (
    RoleBoss   Role = "BOSS"
    RoleEscort Role = "ESCORT"
)

type Player struct {
    ID        string `json:"id"`
    Nickname  string `json:"nickname"`
    Avatar    string `json:"avatar,omitempty"`
    Role      Role   `json:"role,omitempty"`
    Ready     bool   `json:"ready"`
    Coin      int    `json:"coin"`
    IsHost    bool   `json:"isHost"`
    Connected bool   `json:"connected"`
}

type Item struct {
    ID     string `json:"id"`     // prizeId
    Name   string `json:"name"`
    Rarity string `json:"rarity"`
    Value  int    `json:"value"`  // 奖品价值，单位 W（万），正整数，抽中即累加到房间 GuaranteedAmount
}

type LotteryRecord struct {
    ID              string `json:"id"`
    PlayerID        string `json:"playerId"`
    Cost            int    `json:"cost"`            // 本次消耗金币（当前固定为 2）
    Item            Item   `json:"item"`
    GuaranteeDelta  int    `json:"guaranteeDelta"`  // = Item.Value，单位 W
    GuaranteeBefore int    `json:"guaranteeBefore"` // 抽奖前保底金额
    GuaranteeAfter  int    `json:"guaranteeAfter"`  // 抽奖后保底金额
    At              int64  `json:"at"`
}

type SettlementOutcome string

const (
    SettlementOutcomeEvacuated SettlementOutcome = "EVACUATED"
    SettlementOutcomeBombed    SettlementOutcome = "BOMBED"
)

type SettlementPerPlayer struct {
    PlayerID  string `json:"playerId"`
    Kills     int    `json:"kills"`     // 0~99
    SmallReds int    `json:"smallReds"` // 0~99
    SixGrids  int    `json:"sixGrids"`  // 0~99
    NineGrids int    `json:"nineGrids"` // 0~99
    CoinDelta int    `json:"coinDelta"` // 服务端裁决得到的本次结算金币变化
}

type SettlementRecord struct {
    ID              string                `json:"id"`
    At              int64                 `json:"at"`
    Outcome         SettlementOutcome     `json:"outcome"`
    GuaranteeDelta  int                   `json:"guaranteeDelta"`  // 非负整数；EVACUATED 表示扣减额，BOMBED 表示增加额
    GuaranteeBefore int                   `json:"guaranteeBefore"` // 结算前保底金额
    GuaranteeAfter  int                   `json:"guaranteeAfter"`  // 结算后保底金额
    PerPlayer       []SettlementPerPlayer `json:"perPlayer"`       // 必含房间内全部 3 位玩家
}

type FeedEvent struct {
    ID      string         `json:"id"`
    At      int64          `json:"at"`
    Kind    string         `json:"kind"`   // LOTTERY | PURCHASE | SETTLEMENT
    Payload map[string]any `json:"payload"`
}

type RoomSnapshot struct {
    Code              string             `json:"code"`
    Status            RoomStatus         `json:"status"`
    HostID            string             `json:"hostId"`
    Players           []Player           `json:"players"`
    GuaranteedAmount  int                `json:"guaranteedAmount"`  // 单位 W（万），整数；创建时设定初值，PLAYING 期间仅可经 SETTLE_ROUND 调整
    LotteryRecords    []LotteryRecord    `json:"lotteryRecords"`
    SettlementRecords []SettlementRecord `json:"settlementRecords"` // 本局结算流水
    Feed              []FeedEvent        `json:"feed"`
}

type Envelope struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
    ReqID   string          `json:"reqId,omitempty"`
    TS      int64           `json:"ts"`
}
```

---

## 6. 项目脚手架建议

### 6.1 仓库结构（Monorepo）

```
hls/
├── web/                              # 前端
│   ├── src/
│   │   ├── main.ts
│   │   ├── App.vue
│   │   ├── router/
│   │   │   └── index.ts
│   │   ├── pages/
│   │   │   ├── HomePage.vue
│   │   │   ├── RoomPage.vue          # 选角与准备
│   │   │   └── PlayPage.vue          # 游戏主控制台
│   │   ├── components/
│   │   │   ├── PlayerSlot.vue
│   │   │   ├── EquipmentShop.vue
│   │   │   ├── LotteryPanel.vue
│   │   │   ├── FeedStream.vue
│   │   │   ├── LotteryRecordModal.vue
│   │   │   ├── SettlementModal.vue       # 房主专属：本局结算（§3.4）
│   │   │   └── SettlementRecordModal.vue # 所有人可见：结算历史（§3.3）
│   │   ├── stores/
│   │   │   ├── player.ts             # 本地身份（playerId / nickname / sessionToken）
│   │   │   └── room.ts               # 房间快照
│   │   ├── net/
│   │   │   ├── ws.ts                 # WebSocket 封装（自动重连 + 心跳 + reqId 回执）
│   │   │   ├── client.ts             # 发 Intent、接收 Event、回滚乐观 UI
│   │   │   ├── api.ts                # REST 调用：createRoom / queryRoom
│   │   │   └── protocol.ts           # Envelope / Intent / Event 类型（与后端 protocol 包对齐）
│   │   └── utils/
│   │       ├── id.ts                 # nanoid
│   │       └── storage.ts            # localStorage 封装
│   ├── package.json
│   └── vite.config.ts
└── server/                           # 后端 (Go)
    ├── cmd/
    │   └── gameserver/
    │       └── main.go               # 入口：加载配置 → 注册路由 → 启动 Hub
    ├── internal/
    │   ├── config/
    │   │   └── config.go             # go-zero conf 结构
    │   ├── handler/
    │   │   ├── http.go               # POST /api/rooms 等 REST
    │   │   └── ws.go                 # GET /ws/rooms/:code 升级 + 鉴权
    │   ├── room/
    │   │   ├── hub.go                # 全局 Hub：创建/销毁房间、Code 分配
    │   │   ├── room.go               # Room Loop：Intent 分发、广播、超时
    │   │   ├── client.go             # 单连接读/写 goroutine
    │   │   ├── snapshot.go           # 快照构造
    │   │   └── dispatch.go           # Intent → handler 函数表
    │   ├── game/
    │   │   ├── rules.go              # 角色/装备校验（纯函数）
    │   │   ├── lottery.go            # 抽奖奖品池（11 件，含 prizeId / name / value / rarity / weight）+ 抽奖随机（crypto/rand）+ value→保底累加
    │   │   ├── equipment.go          # 装备数据
    │   │   └── settlement.go         # 本局结算：保底变化校验 / 每人金币计算（纯函数）
    │   ├── protocol/
    │   │   ├── envelope.go
    │   │   ├── intent.go             # Client → Server payload 类型
    │   │   └── event.go              # Server → Client payload 类型
    │   └── svc/
    │       └── service_context.go    # 服务上下文（Hub、配置等）
    ├── pkg/
    │   ├── errors/                   # 项目错误包（NewExposable / WithMessage / WithStack）
    │   ├── code/                     # 6 位数字 Code 生成 + 唯一性管理
    │   └── token/                    # sessionToken 签发与校验
    ├── etc/
    │   └── gameserver.yaml
    ├── go.mod
    └── go.sum
```

### 6.2 前端依赖（建议占位）

```json
{
  "dependencies": {
    "vue": "^3",
    "vue-router": "^4",
    "pinia": "^2"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "latest",
    "typescript": "^5",
    "vite": "^5"
  }
}
```

> 不再依赖 `peerjs`。WebSocket 直接使用浏览器原生 `WebSocket`，封装在 `src/net/ws.ts`。

### 6.3 后端依赖（建议占位）

```go
require (
    github.com/zeromicro/go-zero v1.6.x   // rest / logx / conf
    github.com/gorilla/websocket v1.5.x   // WebSocket 升级与读写
    github.com/google/uuid v1.6.x         // playerId / sessionToken 备选
)
```

- **路由层**：可选 go-zero `rest.Server`，也可用更轻量的 `chi` + `net/http`；本方案推荐前者，便于与项目其它服务对齐。
- **WebSocket**：`gorilla/websocket`。
- **日志**：统一 `logx.WithContext(ctx)`，所有 Room 的 ctx 携带 `code` field。
- **错误**：统一 `pkg/errors`，禁止 `fmt.Errorf`；前端可见错误用 `errors.NewExposable` 包装。

### 6.4 Room Loop 设计要点（推荐写法）

> 一个房间内的所有状态变更只由 Room Loop 的同一个 goroutine 触达，避免任何业务侧锁。Client 侧只负责"反序列化 + 投递"和"消费写队列 + 序列化"，与业务完全解耦。

```go
package room

type Intent struct {
    Type    string
    Payload json.RawMessage
    ReqID   string
    From    *Client
}

type Room struct {
    code     string
    status   protocol.RoomStatus
    players  map[string]*protocol.Player   // playerId -> player
    hostID   string
    clients  map[string]*Client            // playerId -> client（活跃连接）

    inbox    chan Intent                   // 入站 Intent（带缓冲）
    register chan *Client                  // 新连接（已通过 JOIN 校验）
    leave    chan *Client                  // 主动/被动离开
    done     chan struct{}
}

func (r *Room) Run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second) // 巡检：心跳超时、空房 GC、房主迁移
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            r.cleanup(ctx)
            return
        case c := <-r.register:
            r.handleRegister(ctx, c)
        case c := <-r.leave:
            r.handleLeave(ctx, c)
        case it := <-r.inbox:
            r.dispatch(ctx, it)
        case <-ticker.C:
            r.handleTick(ctx)
        }
    }
}

func (r *Room) dispatch(ctx context.Context, it Intent) {
    switch it.Type {
    case "CHANGE_ROLE":
        r.onChangeRole(ctx, it)
    case "READY":
        r.onReady(ctx, it)
    case "START_GAME":
        r.onStartGame(ctx, it)
    case "BUY_EQUIPMENT":
        r.onBuyEquipment(ctx, it)
    case "DRAW_LOTTERY":
        r.onDrawLottery(ctx, it)
    case "SETTLE_ROUND":
        r.onSettleRound(ctx, it)
    case "LEAVE":
        r.leave <- it.From
    case "DISMISS_ROOM":
        r.onDismiss(ctx, it)
    case "PING":
        r.reply(ctx, it.From, "PONG", nil, it.ReqID)
    default:
        logx.WithContext(ctx).Errorw("unknown intent",
            logx.Field("type", it.Type),
            logx.Field("code", r.code))
    }
}
```

广播由 Room 向每个 Client 的 `send` channel 写消息，Client 自己的 write loop 完成序列化与发送，达到"业务 goroutine 永不阻塞在 IO"的目标。

### 6.5 鉴权与会话

- `playerId` 由前端在首次访问时生成（nanoid）并持久化到 `localStorage`，作为玩家身份的稳定 ID。
- `sessionToken` 由服务端在 `JOIN_ACCEPTED` 时签发（HMAC，TTL 与房间生命周期一致），用于断线重连时证明身份。
- 重连 query 必带 `playerId` + `sessionToken`；服务端校验失败 → 关闭连接并要求重新走 `JOIN` 流程。

---

## 7. 后续可扩展点

- **装备系统**：装备属性、品质、对老板抽奖的影响。
- **经济结算**：阶段性结算、胜负判定。
- **战斗系统**：护航 vs 老板 的对抗回合或事件。
- **匹配模式**：除房间码以外的快速匹配（lobby 服务）。
- **战绩系统**：玩家历史对局、胜率、稀有物品收藏（需引入持久化 DB）。
- **Redis 持久化**：Room 状态 + 抽奖记录写入 Redis，支持服务端滚动重启、跨进程恢复。
- **水平扩展**：多实例部署时按 Code Hash 一致性路由，结合 Redis Pub/Sub 跨实例广播；或将房间整体迁移到单一实例（粘性路由）。
- **观战模式**：第 4+ 个 WebSocket 连接以只读方式订阅 `STATE_SNAPSHOT`，不参与裁决。
- **回放系统**：服务端记录 Intent 流到对象存储，前端按时间轴重放。
- **可观测性**：接入 Prometheus（房间数 / 在线连接数 / Intent QPS / 写阻塞次数）+ OpenTelemetry trace。

# 订单系统设计

**日期：** 2026-08-04
**范围：** 订单表 · 后台订单管理 · 扣单分享页 · 打手「我的订单」· 收益计算

---

## 1. 背景与目标

系统已有用户角色（owner/service/enforcer/customer）、账密登录与 `/admin` 后台用户管理页（当前仅 owner 可访问，为单页用户列表）。本期新增订单系统，与用户管理同属后台，改造为「左侧 tab + 右侧内容」的常规后台形态。

目标：
- 店长/客服可建单、分享订单链接到微信群、改派打手、改订单状态、处理客诉。
- 打手通过订单链接扣单（需先登录），扣单后其 id+昵称写入订单；打手可在「我的订单」查看自己的订单与收益。
- 收益按约定公式由后端计算。

沿用现有技术栈：后端 go-zero + MySQL（`sqlx`，路由注册在 `server/cmd/gameserver/main.go`）；前端 Vue 3 + TS + Tailwind（`hls-` 主题类）+ Pinia + axios。

---

## 2. 权限模型（复用现有 4 角色）

| 角色 | 可见 Tab | 能力 |
|---|---|---|
| `owner` 店长 | 订单管理、用户管理 | 建单、改派、改状态、处理客诉、管理用户 |
| `service` 客服 | 订单管理 | 建单、改派、改状态、处理客诉 |
| `enforcer` 打手 | 我的订单 | 查看自己的订单与收益、通过链接扣单 |
| `customer` 客户 | 无 | 不能进后台 |

- `/admin` 路由守卫从「仅 owner」放宽为「owner / service / enforcer」；customer 与未登录跳走（未登录带 `redirect` 回跳）。
- **用户管理 tab 保持 owner-only**（建号涉及敏感权限），客服不可见。
- 后端新增 `requireAnyRole(ctx, roles, next)` 中间件，推广现有 `requireOwner`；`requireOwner` 可基于它重写或保留。

---

## 3. 数据库变更

新增迁移 `server/migrations/002_create_orders.sql`：

```sql
CREATE TABLE orders (
  id               BIGINT       AUTO_INCREMENT PRIMARY KEY,
  code             VARCHAR(20)  NOT NULL UNIQUE,           -- 唯一编号，time.Now().UnixMilli()
  title            VARCHAR(100) NOT NULL,                  -- 单子名称
  order_type       ENUM('guarantee','fun','companion','teaching') NOT NULL, -- 保底/趣味/陪玩/教学
  price            DECIMAL(10,2) NOT NULL,                 -- 价格
  commission_rate  DECIMAL(5,2)  NOT NULL DEFAULT 20.00,   -- 抽成 %
  bonus_rate       DECIMAL(5,2)  NOT NULL DEFAULT 3.00,    -- 鸡腿 %（奖金比例）
  player_count     TINYINT       NOT NULL,                 -- 需要打手数 1 或 2
  dispatcher_id    BIGINT        NOT NULL,                 -- 派单人 user id
  dispatcher_name  VARCHAR(50)   NOT NULL,                 -- 派单人昵称快照
  enforcer_a_id    BIGINT        NULL,
  enforcer_a_name  VARCHAR(50)   NULL,
  enforcer_b_id    BIGINT        NULL,
  enforcer_b_name  VARCHAR(50)   NULL,
  remark           VARCHAR(500)  NULL,                     -- 备注
  status           ENUM('dispatching','ongoing','settled','refunded') NOT NULL DEFAULT 'dispatching',
  complaint_status ENUM('none','complained','resolved')   NOT NULL DEFAULT 'none',
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_enforcer_a (enforcer_a_id),
  INDEX idx_enforcer_b (enforcer_b_id),
  INDEX idx_status (status)
);
```

设计要点：
- **金额/比例存原始值**（price、commission_rate、bonus_rate、player_count），收益**不落库、按需计算**，避免数据不一致。
- **昵称存快照**（dispatcher_name / enforcer_*_name），打手或派单人后续改名不影响历史订单展示。
- `code` 由后端生成 `time.Now().UnixMilli()` 的字符串；唯一约束 + 冲突时重试（极小概率）。
- 客诉初始态 `none`（无客诉）为原需求两态（被投诉、已处理）之外必须补的初始态。

---

## 4. 收益公式（后端计算，唯一真源）

```
每位打手收益 = 价格 × (1 − 抽成% + 鸡腿%) ÷ 打手人数N
```

- `N` = 建单时指定的 `player_count`（1 或 2），即「按打手人数均分」；`N=1` 时该打手获得全部收益。
- 抽成默认 20%，鸡腿默认 3%。
- 展开：`price - price*commission_rate/100` 为扣抽成后净额，按 N 均分；再加 `price*bonus_rate/100` 的奖金按 N 均分。
- 例：价格 800、抽成 20%、鸡腿 3%、2 人单 → 每人 `(800×0.83)/2 = 332`；1 人单 → `664`。

收益仅用于展示（「我的订单」列表、扣单页预估收益），由后端在响应中计算返回；前端不重复计算业务逻辑。

---

## 5. 订单状态机

```
派单中(dispatching) ──扣满所需打手──▶ 进行中(ongoing) ──店长/客服──▶ 结单(settled)
       ▲                                                         └────────▶ 退单(refunded)
       └── 移除打手 / 更换打手（清空对应槽，未招满则回派单中）
```

- **2 人单：招满 A+B 才**变「进行中」；1 人单：扣单即「进行中」。
- 扣单写入第一个空槽：A 优先，再 B；**同一打手不能占两个槽**；订单非「派单中」或已招满时拒绝扣单。
- 「更换打手」= **可单独移除** A 或 B 某个槽 → 该槽重新开放扣单；若因此未招满，状态回落「派单中」。
- 结单 / 退单由店长/客服手动设置。
- 客诉状态 `none → complained → resolved`，由店长/客服设置。

---

## 6. 后端 API（go-zero，注册在 `main.go`）

| 方法 路径 | 权限 | 作用 |
|---|---|---|
| POST `/api/orders` | owner/service | 建单（派单人自动取当前登录者 id+昵称） |
| GET `/api/orders` | owner/service | 全部订单列表 |
| GET `/api/orders/mine` | enforcer | 我的订单（含计算收益字段） |
| GET `/api/orders/:code` | 登录任意角色 | 订单详情（扣单页用，返回完整信息 + 预估收益） |
| POST `/api/orders/:code/claim` | enforcer | 扣单 |
| DELETE `/api/orders/:code/enforcer/:slot` | owner/service | 移除/更换打手（`slot` = `a`/`b`） |
| PUT `/api/orders/:code/status` | owner/service | 改订单状态（settled/refunded/ongoing…） |
| PUT `/api/orders/:code/complaint` | owner/service | 改客诉状态 |

新增文件：
- `server/internal/repo/order.go` — `Order` 结构 + `Create` / `GetByCode` / `ListAll` / `ListByEnforcer` / `AssignEnforcer(code, slot, uid, name)` / `RemoveEnforcer(code, slot)` / `UpdateStatus` / `UpdateComplaint`。
- `server/internal/handler/order.go` — 上述各 handler，复用现有 `extractUserID` / `writeError` / `httpx.OkJsonCtx` 模式。
- `server/internal/svc/service_context.go` — 注入 `OrderRepo`。

`order_type` / `status` / `complaint_status` 在后端用常量集合校验，非法值返回 400。

---

## 7. 前端

### 7.1 后台外壳改造

把现有单页 `src/pages/AdminPage.vue` 改造成**左侧 tab 导航 + 右侧内容区**的外壳：
- Tab 根据登录角色动态渲染（见权限表）。
- 内容区按选中 tab 渲染对应子面板组件。
- 复用现有 `hls-` 主题类（`hls-panel` / `hls-btn` / `hls-badge` / `hls-input` 等）与 `ConfirmDialog`。

子面板（`src/pages/admin/` 或就近目录）：
- `OrdersPanel.vue` — 订单列表表格 + 「创建订单」弹窗表单 + 每行操作（复制分享链接、移除/更换 A/B 打手、改状态、处理客诉）。
- `UsersPanel.vue` — 从现有 `AdminPage.vue` 抽取的用户管理逻辑（列表 + 建号 + 重置密码/昵称）。
- `MyOrdersPanel.vue` — 打手自己的订单列表，列：订单编号、订单类型、单子名称、收益、抽成、派单人、备注、鸡腿。

### 7.2 建单表单字段

单子名称、订单类型（4 选 1）、价格、抽成%（默认 20）、鸡腿%（默认 3）、打手人数（1/2）、备注。派单人不在表单中，后端自动填当前登录者。

### 7.3 扣单/分享页（新路由 `/order/:code`）

- 新增 `src/pages/OrderClaimPage.vue`，路由 `/order/:code`。
- 展示**完整订单信息**（单子名称、类型、价格、抽成、鸡腿、备注、状态、当前打手 A/B）+ 该打手扣单后的**预估收益**。
- 「扣单」按钮：未登录 → 跳登录页带 `redirect` 回本页；已登录 → 调 `POST /api/orders/:code/claim`。
- 订单已招满 / 非派单中 / 自己已在单中时，按钮禁用并提示原因。
- 页面提供「复制分享链接」入口供店长/客服使用（`${origin}/order/${code}`）。

### 7.4 路由守卫

`src/router/index.ts` 中 `/admin` 守卫改为：未登录跳登录；已登录但角色为 `customer` 跳首页；`owner/service/enforcer` 放行。`/order/:code` 为公开路由（扣单动作内部再校验登录）。

---

## 8. 展示映射（枚举 → 中文）

- 订单类型：`guarantee`=保底单、`fun`=趣味单、`companion`=陪玩单、`teaching`=教学单。
- 订单状态：`dispatching`=派单中、`ongoing`=进行中、`settled`=结单、`refunded`=退单。
- 客诉状态：`none`=无客诉、`complained`=被投诉、`resolved`=已处理。

前端集中维护 label 映射（参考现有 `AdminPage.vue` 的 `roleLabels`）。

---

## 9. 不做（YAGNI）

- 不做订单编辑（建错则退单重建）。
- 不做分页/复杂筛选（初版全量列表 + 前端简单按状态过滤即可）。
- 收益不落库、不做结算流水表。
- 不改动微信/短信登录与游戏房间相关代码。

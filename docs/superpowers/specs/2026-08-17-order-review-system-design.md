# 订单评价系统设计

- **日期**: 2026-08-17
- **状态**: 已确认，待实现
- **关联**: 建立在 `2026-08-04-order-system-design.md`（订单系统）之上

## 背景与目标

现有陪玩/打手接单系统里，订单有派单人、最多两个打手（enforcerA/enforcerB）、`status`（派单中/进行中/已结算/已退款）和 `complaint_status`（none/complained/resolved）两套状态，但**没有任何评价功能**。

本设计新增一个评价系统：由后台管理员（owner/service）在订单管理页给每个订单代录一份评价（态度分、技术分、若干扣分项标记、自由评语），并据此联动**鸡腿奖励**（影响收益）与**客诉状态**。

## 需求确认（已与需求方逐条确认）

1. **评分粒度**：按订单，一个订单一条评价（非按打手）。
2. **鸡腿奖励**：总分 ≥ 8 才发鸡腿，且**真实影响收益**——鸡腿部分从"无条件计入"改为"仅达标时计入"。
3. **客诉联动**：保存评价时，总分 < 6 自动把订单 `complaint_status` 置为 `complained`。
4. **录入方**：owner/service 代录，每单一条，可覆盖修改（upsert）。
5. **9 个 bool 字段不自动扣分**：仅作定性记录，总分只由 态度+技术 决定。
6. **客诉自动回退规则**：总分 ≥ 6 时，仅当当前是 `complained` 才自动改回 `none`，不覆盖人工设置的 `resolved`。
7. **评价详情界面**：用弹窗（modal）实现，不新增独立路由。

## 数据模型

### 新增 `reviews` 表

迁移文件 `server/migrations/003_create_reviews.sql`，沿用 `002_create_orders.sql` 风格（`utf8mb4`、`BIGINT AUTO_INCREMENT` 主键、`TIMESTAMP DEFAULT CURRENT_TIMESTAMP`）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT AUTO_INCREMENT PRIMARY KEY | 自增主键 |
| `order_code` | VARCHAR(20) NOT NULL **UNIQUE** | 关联 `orders.code`；UNIQUE 保证每单一条 |
| `attitude_score` | TINYINT NOT NULL | 态度评分 1–5 |
| `technical_score` | TINYINT NOT NULL | 技术评分 1–5 |
| `blame_boss` | TINYINT(1) NOT NULL DEFAULT 0 | 压力指责老板 |
| `late` | TINYINT(1) NOT NULL DEFAULT 0 | 迟到 |
| `temp_unavailable` | TINYINT(1) NOT NULL DEFAULT 0 | 约好时间临时有事 |
| `slow_reply` | TINYINT(1) NOT NULL DEFAULT 0 | 群里长时间不回复老板消息 |
| `no_backpack` | TINYINT(1) NOT NULL DEFAULT 0 | 未携带背包 |
| `no_channel_card` | TINYINT(1) NOT NULL DEFAULT 0 | 未携带通道卡 |
| `no_supply_card` | TINYINT(1) NOT NULL DEFAULT 0 | 未携带物资卡 |
| `no_emotional_value` | TINYINT(1) NOT NULL DEFAULT 0 | 未提供情绪价值 |
| `grab_container` | TINYINT(1) NOT NULL DEFAULT 0 | 抢老板的容器 |
| `comment` | TEXT | 自由评语（utf8mb4，支持汉字），可空 |
| `reviewer_id` | BIGINT NOT NULL | 代录人 userID |
| `reviewer_name` | VARCHAR(64) NOT NULL | 代录人昵称 |
| `created_at` | TIMESTAMP DEFAULT CURRENT_TIMESTAMP | |
| `updated_at` | TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP | |

索引：`order_code` UNIQUE 索引即可满足按单查询与幂等 upsert。

### 修改 `orders` 表

在同一迁移文件 `003` 中 `ALTER TABLE orders`：

```sql
ALTER TABLE orders ADD COLUMN bonus_qualified TINYINT(1) NOT NULL DEFAULT 0 AFTER complaint_status;
```

`bonus_qualified` 是从 review 冗余下来的达标标记，供收益计算读取，避免每次算收益都 JOIN reviews。

**说明：`total_score` 不落库**，由 `attitude + technical` 在代码里即时计算，避免冗余不一致。

## 计分与联动规则（后端集中实现，单一事实来源）

设 `total = attitude_score + technical_score`（范围 2–10）。保存评价（upsert）时，在**同一个数据库事务**内：

1. 写入/更新 `reviews` 行。
2. `bonus_qualified = (total >= 8) ? 1 : 0`，更新 `orders.bonus_qualified`。
3. 客诉联动：
   - `total < 6` → `orders.complaint_status = 'complained'`
   - `total >= 6` → **仅当** `orders.complaint_status` 当前为 `'complained'` 时改回 `'none'`；为 `'resolved'` 或 `'none'` 时不动。

### 收益公式变更

现状 `Order.Earnings()`（`server/internal/repo/order.go`）：

```
earnings = (price*(1 - commission_rate/100) + price*bonus_rate/100) / player_count
```

改为鸡腿部分有条件计入：

```
base  = price * (1 - commission_rate/100)
bonus = bonus_qualified ? (price * bonus_rate/100) : 0
earnings = (base + bonus) / player_count      // 保留四舍五入两位
```

实现上给 `Order` struct 增加 `BonusQualified` 字段（`db:"bonus_qualified"`），`Earnings()` 读取该字段。

⚠️ **行为变更 / 迁移影响**：现有历史订单迁移后 `bonus_qualified` 默认为 0，其展示收益会**减少鸡腿部分**，直到被评价且总分 ≥ 8。这是新规则的预期结果，但会改变已有订单的收益展示，实现和上线时需知悉（如需对历史订单做数据回填，另行处理，不在本设计范围）。

## 后端实现（go-zero + 手写 SQL，仿现有 order 模块）

### `server/internal/repo/review.go`（新建）

- `Review` struct，带 `db:"..."` tag，字段对应上表。
- `reviewCols` 常量（列名列表）。
- `NewReviewRepo(conn sqlx.SqlConn) *ReviewRepo`，复用同一 `conn`。
- 方法：
  - `Upsert(ctx, r *Review) error`：`INSERT ... ON DUPLICATE KEY UPDATE ...`（以 `order_code` UNIQUE 触发覆盖）。
  - `GetByOrderCode(ctx, code string) (*Review, error)`：单条完整查询，`errors.Is(err, sqlx.ErrNotFound)` 表示无评价。
  - `ListByOrderCodes(ctx, codes []string) (map[string]*ReviewSummary, error)`：供列表批量取评分摘要（或在订单列表 SQL 里 LEFT JOIN，见下）。

### `server/internal/repo/order.go`（修改）

- `Order` struct 增加 `BonusQualified bool \`db:"bonus_qualified"\``。
- `orderCols` 常量加入 `bonus_qualified`。
- `Earnings()` 按上文改为条件计入鸡腿。
- 新增 `UpdateReviewLinkedFields(ctx, code string, bonusQualified bool, complaintStatus string) error`（或在评价 handler 的事务里直接执行 SQL），用于同事务更新 `orders.bonus_qualified` 与 `orders.complaint_status`。
- `ListOrders` 的查询改为 `LEFT JOIN reviews r ON r.order_code = o.code`，在订单视图里带出评分摘要（态度/技术/总分/是否达标/是否已评价）。

### `server/internal/handler/review.go`（新建）

沿用 order.go 约定：私有请求 DTO（小驼峰 json tag）、`requireAnyRole` 包权限、`writeError` + `httpx.OkJsonCtx`、`toReviewView` 手工视图。

- **`PUT /api/orders/:code/review`** — upsert 评价。
  - 权限：`requireAnyRole(ctx, []string{"owner","service"}, ...)`。
  - 解析路径 code（复用 `parseOrderPath` 或类似）。
  - 校验：`attitude`、`technical` 均 ∈ [1,5]；9 个 bool 直接接收；`comment` 去空白、长度上限（如 500 字）。非法 → `writeError(... "invalid_review")`。
  - 校验订单存在（`OrderRepo.GetByCode`，不存在 → 404 `order_not_found`）。
  - 取当前 user 作 `reviewer_id/name`。
  - 计算 `total`、`bonusQualified`、目标 `complaint_status`（按联动规则，读当前 complaint_status 决定是否回退）。
  - **事务**内：`ReviewRepo.Upsert` + 更新 orders 两字段。
  - 返回 `toReviewView(review)`（可附带 `total`、`bonusQualified`）。
- **`GET /api/orders/:code/review`** — 取单条完整评价。
  - 权限：`requireAnyRole(ctx, []string{"owner","service"}, ...)`。
  - 无评价 → 404 `review_not_found`（前端据此显示"暂无评价"）。
  - 返回 `toReviewView(review)`（含全部 9 个 bool、评语、代录人、时间、total）。

### `server/internal/svc/service_context.go`（修改）

`ServiceContext` 增加 `ReviewRepo *repo.ReviewRepo`，`NewServiceContext` 里 `repo.NewReviewRepo(conn)`（复用现有 `conn`）。

### `server/cmd/gameserver/main.go`（修改）

`AddRoutes` 列表追加：

```go
{Method: http.MethodPut, Path: "/api/orders/:code/review", Handler: handler.UpsertReviewHandler(ctx)},
{Method: http.MethodGet, Path: "/api/orders/:code/review", Handler: handler.OrderReviewHandler(ctx)},
```

### 错误 message key（沿用短 key 风格）

`invalid_review`、`order_not_found`、`review_not_found`、`forbidden`、`db_error`、`unauthorized`、`user_not_found`。

## 前端实现（Vue3 `<script setup lang="ts">` + Tailwind 自研组件）

### `src/shared/orderLabels.ts`（修改）

- 新增 `OrderReview` 类型（态度、技术、9 个 bool、comment、reviewerName、createdAt、total）。
- 新增 `ReviewSummary` 类型（attitude、technical、total、qualified、hasReview）。
- 新增 `reviewFlagLabels: Record<string,string>` —— 9 个 bool key → 中文标签（压力指责老板 / 迟到 / 约好时间临时有事 / 群里长时间不回复老板消息 / 未携带背包 / 未携带通道卡 / 未携带物资卡 / 未提供情绪价值 / 抢老板的容器）。
- `OrderView` 增加可选 `review?: ReviewSummary`（列表摘要）。

### `src/api/orders.ts`（修改）

```ts
export async function getOrderReview(code: string): Promise<OrderReview> {
  const res = await http.get<OrderReview>(`/orders/${code}/review`)
  return res.data
}
export async function upsertOrderReview(code: string, payload: ReviewPayload): Promise<OrderReview> {
  const res = await http.put<OrderReview>(`/orders/${code}/review`, payload)
  return res.data
}
```

（复用现有 `http` 实例，无需改 axios 配置。）

### `src/pages/admin/OrdersPanel.vue`（修改）

订单管理页是卡片列表（`v-for` 渲染 `<article class="hls-panel">`），非 table。三处改动：

1. **卡片展示评分**：在指标网格里加评分区——展示 `态度 ★n / 技术 ★n / 总分 n`（用 pill），达标时显示 `🍗 鸡腿达标` 徽章，`total<6` 时显示 `已客诉` 提示。颜色 helper 仿现有 `getStatusClass` 就地写一个 `getRatingClass`（返回 Tailwind class 串）。未评价的订单显示"未评价"占位。
2. **录入/编辑评价**：卡片操作区加"评价"按钮 → 打开录入 modal（复用创建订单弹窗那套 `ref(false)` + `<transition name="fade">` + 固定遮罩范式）。表单：态度下拉 1–5、技术下拉 1–5、9 个复选框、评语 `<textarea>`。编辑时用已有评价预填。提交 → `upsertOrderReview(code, payload)` → `await load()` 刷新整表。
3. **评价详情**：卡片操作区加"评价详情"按钮 → 打开只读 modal，`openReview(code)` 里 `await getOrderReview(code)`，展示全部内容：两项评分、总分、勾选为真的扣分项（用 `reviewFlagLabels` 渲染成标签列表）、评语、代录人、时间。无评价时提示"暂无评价"。

### `src/pages/admin/MyOrdersPanel.vue`（打手视角）

不在本次改动范围新增评价展示；但因收益公式变更，打手看到的"我的收益"会自然反映鸡腿达标与否（收益由后端计算下发）。**无需前端改动**。

## 组件边界与测试

- **计分/联动规则**：全部集中在后端评价 handler + `Order.Earnings()`，是单一事实来源，前端不重复计算。
- **收益变更测试**：`Earnings()` 增加单测——`bonus_qualified=false` 不含鸡腿、`=true` 含鸡腿，覆盖 1/2 人分摊。
- **评价 upsert 测试**：总分 <6 / =6 / ≥8 三档，验证 `bonus_qualified` 与 `complaint_status` 联动正确，且 `resolved` 不被覆盖、第二次 upsert 覆盖首次。
- **前端**：录入 modal 提交后刷新、详情 modal 正确渲染勾选项与评语、未评价占位。

## 不做（YAGNI / 超出范围）

- 不做按打手（A/B）分别评价。
- 9 个 bool 不自动扣分。
- 不新增评价详情独立路由（用 modal）。
- 不做评价历史多条留存（每单一条覆盖）。
- 不做客户（customer 角色）自助评价入口。
- 不对历史订单做鸡腿收益数据回填（如需另行处理）。
- MyOrdersPanel 不新增评价展示。

# 打手管理 & 工资管理 设计文档

- 日期：2026-08-18
- 状态：已批准设计，待写实现计划
- 相关既有系统：订单系统（`2026-08-04-order-system-design.md`）、评价系统（`2026-08-17-order-review-system-design.md`）

## 1. 背景与目标

在既有「订单 + 评价」体系之上，新增两个后台子系统，并给打手（`role=enforcer`）补一个结单入口：

1. **打手管理**：以打手为单位，汇总押金（应缴/已缴/待缴）、接单数、好评数、差评数、好评率、待罚款金额，并支持管理员维护押金标准与罚款金额。
2. **工资管理**：订单进入「已结单」满 3 天后，按打手逐单生成工资条；发工资时自动抵缴打手的待缴押金、扣减待罚款，冻结并展示详细扣减逻辑；点「发工资」后状态转为「已发放」。
3. **打手结单**：打手打完单后，可在「扣单页（订单详情页）」和后台「我的订单」两处点「结单」按钮，把自己参与的 `ongoing` 订单改为 `settled`，并触发生成工资条。

### 既有领域约定（复用，不改）

- 订单状态：`dispatching → ongoing → settled → refunded`。招满打手时自动 `ongoing`。
- 每单最多 2 个打手槽：`enforcer_a`、`enforcer_b`。
- 评价：一单一条 `reviews`，`总分 = attitude_score + technical_score`（各 1–5，总分 2–10）。**好评 = 总分 ≥ 8，差评 = 总分 < 6**（与本需求定义一致）。
- 个人单笔收益：`Order.Earnings()`（已按 `player_count` 均分，含鸡腿加成，受 `bonus_qualified` 影响）。
- 角色：`owner / service / enforcer / customer`。

### 已确认的关键决策

- **罚款规则**：管理员手动录入（**不**由差评自动生成）。
- **押金模型**：全局默认 300、可配置；每个打手可覆盖；已缴纳押金由工资发放时自动抵缴累加。
- **好评率分母**：`好评数 / 已评价订单数`（该打手有评价的订单数）。
- **结单入口**：订单详情页与后台「我的订单」两处都放。
- **工资建模**：方案 A —— 结单时持久化工资条，发放时冻结扣减快照。
- **扣减顺序**：先补足待缴押金，再扣待罚款，实发下限为 0；不足抵扣的余额留到该打手下一单继续抵。
- **3 天窗口锚点**：用 `payroll_items.created_at`（=结单时刻），**不**给 orders 表加 `settled_at`。

## 2. 数据模型

新增迁移 `server/migrations/004_create_payroll.sql`，含两张表。不改动 orders / reviews / users 结构。

### 2.1 `enforcer_accounts`（打手账户，一人一行，稀疏表）

```sql
CREATE TABLE enforcer_accounts (
  user_id          BIGINT        PRIMARY KEY,      -- 对应 users.id (role=enforcer)
  deposit_required DECIMAL(10,2) NULL,             -- NULL = 用全局默认(300)
  deposit_paid     DECIMAL(10,2) NOT NULL DEFAULT 0,
  pending_fine     DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

- **稀疏**：打手默认无行。管理员设置押金覆盖 / 录入罚款、或工资抵缴时才 upsert 出行。列表中缺行的打手按默认值展示（押金应缴 = 全局默认 300、已缴 = 0、罚款 = 0）。
- 全局默认押金放 `config.go`：`PayrollConfig{ DefaultDeposit float64 // 默认 300 }`。

### 2.2 `payroll_items`（工资条，一打手一单一条）

```sql
CREATE TABLE payroll_items (
  id               BIGINT        AUTO_INCREMENT PRIMARY KEY,
  order_code       VARCHAR(20)   NOT NULL,
  enforcer_id      BIGINT        NOT NULL,
  enforcer_name    VARCHAR(50)   NOT NULL,
  status           ENUM('pending','paid') NOT NULL DEFAULT 'pending',
  gross            DECIMAL(10,2) NOT NULL DEFAULT 0,   -- 发放时冻结的个人收益
  deposit_deducted DECIMAL(10,2) NOT NULL DEFAULT 0,   -- 本单抵缴押金
  fine_deducted    DECIMAL(10,2) NOT NULL DEFAULT 0,   -- 本单扣罚款
  net_paid         DECIMAL(10,2) NOT NULL DEFAULT 0,   -- 实发
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,  -- =结单时刻，3天窗口锚点
  paid_at          TIMESTAMP     NULL,
  UNIQUE KEY uq_order_enforcer (order_code, enforcer_id),
  INDEX idx_status (status)
) DEFAULT CHARSET=utf8mb4;
```

- 结单时为每个**已填充**的打手槽（A/B）插一条 `pending`（`gross`/扣减先为 0）。
- `UNIQUE(order_code, enforcer_id)` + `INSERT IGNORE` 保证幂等：重复结单不会重复建条。
- 待发放列表展示的收益**实时算**（`Order.Earnings()`，反映评价晚到的鸡腿加成）；**发放时**才把当刻收益与扣减冻结进 `gross / deposit_deducted / fine_deducted / net_paid`。

## 3. 打手管理

### 3.1 后台入口

- `AdminPage.vue` tab 列表新增 `{ key: 'enforcers', label: '打手管理', roles: ['owner'] }`（仅 owner）。
- 新增面板 `src/pages/admin/EnforcersPanel.vue`。

### 3.2 列表字段（一行 = 一个 `role=enforcer` 用户）

| 字段 | 来源 / 算法 |
|---|---|
| 打手 | users.nickname / username |
| 押金应缴 | `enforcer_accounts.deposit_required ?? 全局默认 300` |
| 已缴纳押金 | `enforcer_accounts.deposit_paid`（默认 0） |
| 待缴押金 | `max(0, 应缴 − 已缴)`（前端计算） |
| 接单数 | 该打手占 A 或 B 槽的订单总数（**含所有状态**） |
| 好评数 | 上述订单中「有评价且总分 ≥ 8」的数量 |
| 差评数 | 上述订单中「有评价且总分 < 6」的数量 |
| 好评率 | `好评数 / 已评价订单数`（分母 = 该打手有评价的订单数）；分母为 0 时显示 `—` |
| 待罚款金额 | `enforcer_accounts.pending_fine`（默认 0） |

### 3.3 管理员操作（每行）

- **编辑押金标准**：设置或清除 `deposit_required` 覆盖值（清除 = 回到全局 300）。
- **录入罚款**：设置 `pending_fine` 为指定金额（**设置语义**，非增量）。

### 3.4 后端

- 新增 `repo/enforcer.go` → `EnforcerAccountRepo`：`GetOrDefault`、`ListAccounts`、`UpsertDepositRequired`、`SetPendingFine`（后两者按需 upsert 稀疏行）。
- 统计聚合（接单数/好评/差评/已评价数）：复用现有 `OrderRepo.ListAll` + `ReviewRepo.ListSummaries`，在 handler 层用**纯函数**按 enforcer 分组聚合（不写复杂 SQL，便于单测）。
- 新增 `handler/enforcer.go`：
  - `GET  /api/enforcers`（owner）→ 打手列表（统计 + 账户字段）。
  - `PUT  /api/enforcers/:id/deposit`（owner）→ 设置/清除押金标准。
  - `PUT  /api/enforcers/:id/fine`（owner）→ 设置待罚款金额。
- 前端 `src/api/enforcers.ts` 对应三个函数。

## 4. 工资管理

### 4.1 后台入口

- `AdminPage.vue` tab 新增 `{ key: 'payroll', label: '工资管理', roles: ['owner'] }`（owner）。
- 新增面板 `src/pages/admin/PayrollPanel.vue`，含「待发放 / 已发放」两个分区（或状态筛选）。

### 4.2 待发放筛选

```
status = 'pending' AND created_at <= NOW() - INTERVAL <SettleDelayDays> DAY
```

- 天数放 `PayrollConfig{ SettleDelayDays int // 默认 3 }`，可配置。

### 4.3 列表列（待发放 = 实时预测；已发放 = 冻结快照）

| 列 | 待发放（实时） | 已发放（快照） |
|---|---|---|
| 订单 / 打手 | order_code + 标题 + 打手名 | 同 |
| 应发工资 gross | `Order.Earnings()` 实时算 | `payroll_items.gross` |
| 抵缴押金 | `min(gross, 待缴押金)` | `deposit_deducted` |
| 扣罚款 | `min(gross − 抵押金, 待罚款)` | `fine_deducted` |
| 实发 net | `gross − 抵押金 − 扣罚款` | `net_paid` |
| 扣减说明 | 一句话文案（例：应发 86，抵缴押金 86，剩余待缴押金 214） | 用快照值同款文案 |
| 操作 / 时间 | **[发工资]** 按钮 | 已发放时间 `paid_at` |

- 待发放区顶部提示小字：「预测按当前欠缴额计算，实际以逐单发放为准」（同一打手多条待发放时，各条预测会重叠，实际发放按逐条顺序结算）。

### 4.4 扣减算法（点「发工资」，单条，事务内）

```
acc      = 账户.GetOrDefault(enforcerId)   // 缺行按默认
required = acc.deposit_required ?? DefaultDeposit(300)
owed     = max(0, required − acc.deposit_paid)      // 待缴押金
gross    = Order.Earnings()                          // 当刻冻结

depositDeduct = min(gross, owed)
remaining     = gross − depositDeduct
fineDeduct    = min(remaining, acc.pending_fine)
net           = remaining − fineDeduct               // ≥ 0

// 持久化（同一事务）
acc.deposit_paid += depositDeduct
acc.pending_fine −= fineDeduct
item ← { gross, depositDeduct, fineDeduct, net, status='paid', paid_at=NOW() }
```

- **顺序**：先补押金、再扣罚款，实发下限 0。不足抵扣的余额留到该打手下一单继续抵。
- **并发/幂等**：`UPDATE payroll_items ... WHERE id=? AND status='pending'`，影响行数 = 0 则报「已发放」，防重复点。
- 纯扣减逻辑抽为纯函数 `computeDeduction(gross, owed, fine) -> (depositDeduct, fineDeduct, net)`，便于单测。

### 4.5 后端

- 新增 `repo/payroll.go` → `PayrollRepo`：
  - `CreatePendingForOrder`（结单时调，见 §5）。
  - `ListPendingDue`（含 3 天过滤，联表 orders 取实时 gross）。
  - `ListPaid`。
  - `Pay(itemId)`（§4.4 事务算法，联动 `enforcer_accounts`）。
- 新增 `handler/payroll.go`：
  - `GET  /api/payroll?status=pending|paid`（owner）。
  - `POST /api/payroll/:id/pay`（owner）。
- 前端 `src/api/payroll.ts` 对应函数。

## 5. 结单流程

### 5.1 规则

- 打手（enforcer）对**自己参与**、状态 `ongoing` 的订单点「结单」→ 订单变 `settled`。
- 校验：调用者必须是该单 enforcer A 或 B，且当前 `status='ongoing'`（`dispatching` 未招满不能结、`settled/refunded` 不能重复结），否则报错。

### 5.2 副作用（同一事务）

```
UPDATE orders SET status='settled' WHERE code=? AND status='ongoing'   // 影响行=0 → 冲突
// 对每个已填充打手槽(A/B)：
INSERT IGNORE INTO payroll_items (order_code, enforcer_id, enforcer_name, status='pending')
```

- 新增 `OrderRepo.SettleWithPayroll(ctx, code, expectStatus)`：事务内改状态 + 建工资条。
- **既有** owner/service 的 `UpdateOrderStatusHandler` 把订单改为 `settled` 时，**复用**此方法（管理员手动结单同样生成工资条），单一入口不遗漏。

### 5.3 结单入口（两处）

1. **订单详情页 `OrderClaimPage.vue`（`/order/:code`，扣单页）**：当当前用户是 enforcer、是本单打手、且 `status==='ongoing'` 时，显示「✅ 结单」按钮 → 确认弹窗 → 调接口 → 刷新。
2. **后台「我的订单」`MyOrdersPanel.vue`**：每张属于自己且 `ongoing` 的订单卡片加「结单」按钮，逻辑同上。

### 5.4 后端接口

- `POST /api/orders/:code/settle`（enforcer）：校验本人是该单打手 → `SettleWithPayroll(code, 'ongoing')` → 返回订单视图。
- `src/api/orders.ts` 加 `settleOrder(code)`。

### 5.5 边界处理

- **已结单后又退款（settled → refunded）**：删除该单**尚未发放（pending）**的工资条；已发放的保留（钱已发，不回滚）。在 `UpdateOrderStatusHandler` 改为 refunded 时顺带处理。已发放的人工冲销不在本期范围（YAGNI）。

## 6. 接口 / 权限 汇总

| 方法 & 路径 | 角色 | 说明 |
|---|---|---|
| `GET /api/enforcers` | owner | 打手列表 + 统计 + 账户 |
| `PUT /api/enforcers/:id/deposit` | owner | 设置/清除押金标准 |
| `PUT /api/enforcers/:id/fine` | owner | 设置待罚款金额 |
| `GET /api/payroll?status=pending\|paid` | owner | 工资列表 |
| `POST /api/payroll/:id/pay` | owner | 发工资（事务：冻结扣减 + 联动账户） |
| `POST /api/orders/:code/settle` | enforcer | 打手结单，触发建工资条 |
| `PUT /api/orders/:code/status`（改） | owner/service | 改 settled 复用 `SettleWithPayroll`；改 refunded 清 pending 工资条 |

- 路由在 `server/internal/handler/http.go` 注册；权限沿用现有 `requireAnyRole` 中间件。

## 7. 新增 / 改动文件

**后端**
- 新增：`migrations/004_create_payroll.sql`、`repo/enforcer.go`、`repo/payroll.go`、`handler/enforcer.go`、`handler/payroll.go`。
- 改动：`config/config.go`（加 `PayrollConfig{ DefaultDeposit=300, SettleDelayDays=3 }`）、`repo/order.go`（加 `SettleWithPayroll`）、`handler/order.go`（settle handler + refunded 清条）、`handler/http.go`（注册路由）、`svc/service_context.go`（挂载新 repo）。

**前端**
- 新增：`api/enforcers.ts`、`api/payroll.ts`、`pages/admin/EnforcersPanel.vue`、`pages/admin/PayrollPanel.vue`。
- 改动：`pages/AdminPage.vue`（两个 tab）、`pages/admin/MyOrdersPanel.vue` 与 `pages/OrderClaimPage.vue`（结单按钮）、`api/orders.ts`（`settleOrder`）、`shared/orderLabels.ts`（如需新类型）。

## 8. 测试计划（沿用现有 `_test.go` 风格 + TDD）

- **纯函数 `computeDeduction`**：工资够抵全部押金+罚款、只够抵部分押金、押金已缴满只扣罚款、无欠缴无罚款全额发、实发触底 0。
- **统计聚合纯函数**：无评价（好评率 `—`）、混合好/中/差评的好评数/差评数/好评率。
- **仓储 / 事务**：`Pay` 幂等（重复 pay 第二次影响行 = 0）；`SettleWithPayroll` 幂等建条；settle 状态守卫（非 `ongoing` 拒绝）。
- **权限**：非 owner 访问 `/api/enforcers`、`/api/payroll` → 403；非本单打手 settle → 403 / 冲突。
- **3 天窗口**：`ListPendingDue` 用 `NOW() - INTERVAL ? DAY`，插入不同 `created_at` 验证过滤。

## 9. 非目标（本期不做）

- 差评自动生成罚款（罚款仅手动录入）。
- 已发放工资的人工冲销 / 回滚。
- 定时任务（cron）生成工资条（改用结单时即时生成）。
- 押金的独立「缴纳」入口（押金只经工资抵缴累加）。

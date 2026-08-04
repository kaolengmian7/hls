# 订单系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有后台中新增订单系统：店长/客服建单与分享、打手扫码扣单、打手「我的订单」查询与收益计算。

**Architecture:** 后端 go-zero + MySQL 新增 `orders` 表、`OrderRepo`、一组 REST handler，路由注册进 `main.go`；收益是纯函数（`Order.Earnings()`）不落库。前端把单页 `AdminPage.vue` 改造成「左侧 tab + 右侧内容」外壳，新增订单管理/我的订单面板与公开的 `/order/:code` 扣单页。

**Tech Stack:** Go (go-zero, `sqlx`), MySQL；Vue 3 + TypeScript + Tailwind（`hls-` 主题类）+ Pinia + axios + vue-router。

## Global Constraints

- 角色枚举固定为 `owner` / `service` / `enforcer` / `customer`（DB `users.role` ENUM，勿新增）。
- 金额/比例存原始值，**收益不落库，按需由 `Order.Earnings()` 计算**。
- 昵称在订单里存快照（`dispatcher_name` / `enforcer_*_name`），不随用户改名变化。
- 后端 handler 沿用现有模式：`extractUserID` 取 JWT、`writeError` 输出错误、`httpx.OkJsonCtx` 输出成功；错误响应体形如 `{"ok":false,"message":...}`。
- 后端测试：`cd server && go test ./...`；集成测试在 `TEST_DB_DSN` 未设置时 `t.Skip`。
- 前端无单测框架，验证用 `npm run typecheck` 与 `npm run build`；复用 `hls-` 主题类与 `ConfirmDialog`。
- 提交信息结尾附：`Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`。
- 枚举与中文映射：订单类型 `guarantee`保底/`fun`趣味/`companion`陪玩/`teaching`教学；订单状态 `dispatching`派单中/`ongoing`进行中/`settled`结单/`refunded`退单；客诉 `none`无客诉/`complained`被投诉/`resolved`已处理。
- **收益公式（唯一真源）**：`每位打手收益 = (价格×(1 − 抽成%/100) + 价格×鸡腿%/100) ÷ 打手人数N`，结果四舍五入到 2 位小数；`N` = `player_count`（1 或 2）；`N<=0` 返回 0。

---

## 文件结构

**新增（后端）**
- `server/migrations/002_create_orders.sql` — orders 表
- `server/internal/repo/order.go` — `Order` 结构、`Earnings()`、`OrderRepo` 及方法、错误哨兵
- `server/internal/repo/order_earnings_test.go` — `Earnings()` 纯函数单测（无 DB）
- `server/internal/repo/order_test.go` — `OrderRepo` 集成测试（无 DSN 跳过）
- `server/internal/handler/order.go` — 订单 handler + 视图 DTO + 路径解析/角色中间件
- `server/internal/handler/order_test.go` — handler 鉴权/解析单测

**修改（后端）**
- `server/internal/svc/service_context.go` — 注入 `OrderRepo`
- `server/cmd/gameserver/main.go` — 注册订单路由

**新增（前端）**
- `src/shared/orderLabels.ts` — 枚举→中文映射与类型
- `src/api/orders.ts` — 订单 HTTP 调用封装
- `src/pages/admin/OrdersPanel.vue` — 订单管理面板
- `src/pages/admin/UsersPanel.vue` — 用户管理面板（从 AdminPage 抽取）
- `src/pages/admin/MyOrdersPanel.vue` — 我的订单面板
- `src/pages/OrderClaimPage.vue` — 扣单/分享页

**修改（前端）**
- `src/pages/AdminPage.vue` — 改造成 tab 外壳
- `src/router/index.ts` — `/admin` 守卫放宽 + 新增 `/order/:code`

---

## Task 1: 建 orders 表迁移

**Files:**
- Create: `server/migrations/002_create_orders.sql`

**Interfaces:**
- Produces: 表 `orders`，列见下；供 Task 3 的 SQL 使用。

- [ ] **Step 1: 写迁移 SQL**

`server/migrations/002_create_orders.sql`:

```sql
-- server/migrations/002_create_orders.sql
CREATE TABLE orders (
  id               BIGINT       AUTO_INCREMENT PRIMARY KEY,
  code             VARCHAR(20)  NOT NULL UNIQUE,
  title            VARCHAR(100) NOT NULL,
  order_type       ENUM('guarantee','fun','companion','teaching') NOT NULL,
  price            DECIMAL(10,2) NOT NULL,
  commission_rate  DECIMAL(5,2)  NOT NULL DEFAULT 20.00,
  bonus_rate       DECIMAL(5,2)  NOT NULL DEFAULT 3.00,
  player_count     TINYINT       NOT NULL,
  dispatcher_id    BIGINT        NOT NULL,
  dispatcher_name  VARCHAR(50)   NOT NULL,
  enforcer_a_id    BIGINT        NULL,
  enforcer_a_name  VARCHAR(50)   NULL,
  enforcer_b_id    BIGINT        NULL,
  enforcer_b_name  VARCHAR(50)   NULL,
  remark           VARCHAR(500)  NULL,
  status           ENUM('dispatching','ongoing','settled','refunded') NOT NULL DEFAULT 'dispatching',
  complaint_status ENUM('none','complained','resolved') NOT NULL DEFAULT 'none',
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_enforcer_a (enforcer_a_id),
  INDEX idx_enforcer_b (enforcer_b_id),
  INDEX idx_status (status)
);
```

- [ ] **Step 2: 应用到本地/测试库并核验**

Run（把 `$DSN` 换成本地 MySQL 连接，如测试库）：
```bash
mysql --defaults-extra-file=<(printf '[client]\nuser=root\npassword=...\n') hls < server/migrations/002_create_orders.sql
mysql ... -e "DESCRIBE hls.orders;"
```
Expected: 建表成功，`DESCRIBE` 列出上述所有列。（若无法连库，至少用 `mysql --help` 校验 SQL 语法无误，实际建表在部署时执行。）

- [ ] **Step 3: Commit**

```bash
git add server/migrations/002_create_orders.sql
git commit -m "feat(db): add orders table migration

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Order 结构与 Earnings() 纯函数

**Files:**
- Create: `server/internal/repo/order.go`
- Test: `server/internal/repo/order_earnings_test.go`

**Interfaces:**
- Produces:
  - `type Order struct { ... }`（字段见下，`db`/`json` tag 齐全）
  - `func (o *Order) Earnings() float64` — 每位打手收益，四舍五入 2 位小数

- [ ] **Step 1: 写失败测试**

`server/internal/repo/order_earnings_test.go`:

```go
package repo_test

import (
	"math"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestOrderEarnings(t *testing.T) {
	cases := []struct {
		name       string
		price      float64
		commission float64
		bonus      float64
		players    int
		want       float64
	}{
		{"two players default rates", 800, 20, 3, 2, 332},
		{"one player default rates", 800, 20, 3, 1, 664},
		{"two players no fees", 100, 0, 0, 2, 50},
		{"one player no fees", 100, 0, 0, 1, 100},
		{"rounds to two decimals", 333, 20, 3, 2, 138.2},
		{"zero players guarded", 800, 20, 3, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := &repo.Order{
				Price:          c.price,
				CommissionRate: c.commission,
				BonusRate:      c.bonus,
				PlayerCount:    c.players,
			}
			got := o.Earnings()
			if math.Abs(got-c.want) > 0.001 {
				t.Fatalf("Earnings() = %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestOrderEarnings -v`
Expected: 编译失败（`repo.Order` 未定义 / `Earnings` 未定义）。

- [ ] **Step 3: 实现 Order 结构与 Earnings()**

`server/internal/repo/order.go`（先只放结构与纯函数，仓储方法在 Task 3 追加到同文件）:

```go
package repo

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Order mirrors a row in the orders table.
type Order struct {
	ID              int64          `db:"id"               json:"id"`
	Code            string         `db:"code"             json:"code"`
	Title           string         `db:"title"            json:"title"`
	OrderType       string         `db:"order_type"       json:"orderType"`
	Price           float64        `db:"price"            json:"price"`
	CommissionRate  float64        `db:"commission_rate"  json:"commissionRate"`
	BonusRate       float64        `db:"bonus_rate"       json:"bonusRate"`
	PlayerCount     int            `db:"player_count"     json:"playerCount"`
	DispatcherID    int64          `db:"dispatcher_id"    json:"dispatcherId"`
	DispatcherName  string         `db:"dispatcher_name"  json:"dispatcherName"`
	EnforcerAID     sql.NullInt64  `db:"enforcer_a_id"    json:"-"`
	EnforcerAName   sql.NullString `db:"enforcer_a_name"  json:"-"`
	EnforcerBID     sql.NullInt64  `db:"enforcer_b_id"    json:"-"`
	EnforcerBName   sql.NullString `db:"enforcer_b_name"  json:"-"`
	Remark          sql.NullString `db:"remark"           json:"-"`
	Status          string         `db:"status"           json:"status"`
	ComplaintStatus string         `db:"complaint_status" json:"complaintStatus"`
	CreatedAt       time.Time      `db:"created_at"       json:"createdAt"`
	UpdatedAt       time.Time      `db:"updated_at"       json:"updatedAt"`
}

// Earnings returns per-player earnings, rounded to two decimals.
// Formula: (price*(1-commission%/100) + price*bonus%/100) / playerCount.
func (o *Order) Earnings() float64 {
	if o.PlayerCount <= 0 {
		return 0
	}
	net := o.Price * (1 - o.CommissionRate/100)
	bonus := o.Price * o.BonusRate / 100
	per := (net + bonus) / float64(o.PlayerCount)
	return math.Round(per*100) / 100
}
```

> 注：`context` / `errors` / `sqlx` import 供 Task 3 的仓储方法使用；若本任务先编译，Task 3 未加方法前这些 import 会「未使用」。为避免中间态编译失败，本任务 import 只保留 `math`、`database/sql`、`time`；Task 3 再补 `context`/`errors`/`sqlx`。

按此注释，本任务的 import 块用：
```go
import (
	"database/sql"
	"math"
	"time"
)
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/repo/ -run TestOrderEarnings -v`
Expected: PASS（6 个子用例全绿）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/order.go server/internal/repo/order_earnings_test.go
git commit -m "feat(repo): add Order struct and Earnings() pure function

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: OrderRepo 仓储方法

**Files:**
- Modify: `server/internal/repo/order.go`（追加错误哨兵与 `OrderRepo`）
- Test: `server/internal/repo/order_test.go`

**Interfaces:**
- Consumes: `Order`（Task 2）、`orders` 表（Task 1）。
- Produces:
  - `var ErrSlotTaken = errors.New("slot taken")`
  - `func NewOrderRepo(db sqlx.SqlConn) *OrderRepo`
  - `func (r *OrderRepo) Create(ctx context.Context, o *Order) (*Order, error)`（按 `o.Code` 插入后回读整行返回）
  - `func (r *OrderRepo) GetByCode(ctx context.Context, code string) (*Order, error)`（无则 `sqlx.ErrNotFound`）
  - `func (r *OrderRepo) ListAll(ctx context.Context) ([]*Order, error)`（按 id 倒序）
  - `func (r *OrderRepo) ListByEnforcer(ctx context.Context, userID int64) ([]*Order, error)`
  - `func (r *OrderRepo) AssignEnforcer(ctx context.Context, code, slot string, uid int64, name string) error`（`slot` ∈ {"a","b"}，目标槽非空则 `ErrSlotTaken`）
  - `func (r *OrderRepo) RemoveEnforcer(ctx context.Context, code, slot string) error`
  - `func (r *OrderRepo) UpdateStatus(ctx context.Context, code, status string) error`
  - `func (r *OrderRepo) UpdateComplaint(ctx context.Context, code, complaint string) error`

- [ ] **Step 1: 写失败的集成测试**

`server/internal/repo/order_test.go`:

```go
package repo_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/repo"
)

func orderTestConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func newTestOrder(code string) *repo.Order {
	return &repo.Order{
		Code:           code,
		Title:          "上分单",
		OrderType:      "companion",
		Price:          800,
		CommissionRate: 20,
		BonusRate:      3,
		PlayerCount:    2,
		DispatcherID:   1,
		DispatcherName: "客服A",
		Status:         "dispatching",
	}
}

func TestOrderRepo_CreateAndGet(t *testing.T) {
	conn := orderTestConn(t)
	r := repo.NewOrderRepo(conn)
	ctx := context.Background()
	code := "test" + strconv.FormatInt(time.Now().UnixNano(), 10)

	created, err := r.Create(ctx, newTestOrder(code))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created.ID should be non-zero")
	}
	got, err := r.GetByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if got.Title != "上分单" || got.PlayerCount != 2 {
		t.Fatalf("unexpected row: %+v", got)
	}

	cleanupOrder(t, conn, code)
}

func TestOrderRepo_AssignAndRemoveEnforcer(t *testing.T) {
	conn := orderTestConn(t)
	r := repo.NewOrderRepo(conn)
	ctx := context.Background()
	code := "test" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := r.Create(ctx, newTestOrder(code)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer cleanupOrder(t, conn, code)

	if err := r.AssignEnforcer(ctx, code, "a", 10, "打手甲"); err != nil {
		t.Fatalf("AssignEnforcer(a) error = %v", err)
	}
	// second assign to same slot must fail
	if err := r.AssignEnforcer(ctx, code, "a", 11, "打手乙"); err != repo.ErrSlotTaken {
		t.Fatalf("AssignEnforcer(a) again = %v, want ErrSlotTaken", err)
	}
	got, _ := r.GetByCode(ctx, code)
	if !got.EnforcerAID.Valid || got.EnforcerAID.Int64 != 10 {
		t.Fatalf("enforcer_a_id = %+v, want 10", got.EnforcerAID)
	}
	if err := r.RemoveEnforcer(ctx, code, "a"); err != nil {
		t.Fatalf("RemoveEnforcer(a) error = %v", err)
	}
	got, _ = r.GetByCode(ctx, code)
	if got.EnforcerAID.Valid {
		t.Fatal("enforcer_a_id should be NULL after remove")
	}
}

func TestOrderRepo_UpdateStatusAndComplaint(t *testing.T) {
	conn := orderTestConn(t)
	r := repo.NewOrderRepo(conn)
	ctx := context.Background()
	code := "test" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := r.Create(ctx, newTestOrder(code)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer cleanupOrder(t, conn, code)

	if err := r.UpdateStatus(ctx, code, "settled"); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if err := r.UpdateComplaint(ctx, code, "complained"); err != nil {
		t.Fatalf("UpdateComplaint() error = %v", err)
	}
	got, _ := r.GetByCode(ctx, code)
	if got.Status != "settled" || got.ComplaintStatus != "complained" {
		t.Fatalf("status=%q complaint=%q", got.Status, got.ComplaintStatus)
	}
}

func TestOrderRepo_ListByEnforcer(t *testing.T) {
	conn := orderTestConn(t)
	r := repo.NewOrderRepo(conn)
	ctx := context.Background()
	code := "test" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := r.Create(ctx, newTestOrder(code)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer cleanupOrder(t, conn, code)
	_ = r.AssignEnforcer(ctx, code, "b", 77, "打手丙")

	list, err := r.ListByEnforcer(ctx, 77)
	if err != nil {
		t.Fatalf("ListByEnforcer() error = %v", err)
	}
	found := false
	for _, o := range list {
		if o.Code == code {
			found = true
		}
	}
	if !found {
		t.Fatal("expected order in enforcer 77's list")
	}
}

func cleanupOrder(t *testing.T, conn sqlx.SqlConn, code string) {
	t.Helper()
	db, _ := conn.RawDB()
	db.Exec("DELETE FROM orders WHERE code = ?", code)
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestOrderRepo -v`
Expected: 编译失败（`NewOrderRepo` / `ErrSlotTaken` 未定义）。设了 DSN 时后续会真正跑；未设则本步骤只需编译失败即证明测试就位。

- [ ] **Step 3: 实现 OrderRepo（追加到 `order.go`）**

先把 Task 2 里 `order.go` 的 import 补齐为：
```go
import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)
```

在 `order.go` 追加：

```go
// ErrSlotTaken is returned when assigning to an already-filled enforcer slot.
var ErrSlotTaken = errors.New("slot taken")

const orderCols = "id, code, title, order_type, price, commission_rate, bonus_rate, " +
	"player_count, dispatcher_id, dispatcher_name, enforcer_a_id, enforcer_a_name, " +
	"enforcer_b_id, enforcer_b_name, remark, status, complaint_status, created_at, updated_at"

// OrderRepo provides database access for orders.
type OrderRepo struct {
	db sqlx.SqlConn
}

// NewOrderRepo creates an OrderRepo.
func NewOrderRepo(db sqlx.SqlConn) *OrderRepo {
	return &OrderRepo{db: db}
}

// Create inserts a new order (by o.Code) and returns the stored row.
func (r *OrderRepo) Create(ctx context.Context, o *Order) (*Order, error) {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO orders (code, title, order_type, price, commission_rate, bonus_rate, "+
			"player_count, dispatcher_id, dispatcher_name, remark, status) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'dispatching')",
		o.Code, o.Title, o.OrderType, o.Price, o.CommissionRate, o.BonusRate,
		o.PlayerCount, o.DispatcherID, o.DispatcherName, o.Remark)
	if err != nil {
		return nil, err
	}
	return r.GetByCode(ctx, o.Code)
}

// GetByCode returns the order with the given code, or sqlx.ErrNotFound.
func (r *OrderRepo) GetByCode(ctx context.Context, code string) (*Order, error) {
	var o Order
	err := r.db.QueryRowCtx(ctx, &o, "SELECT "+orderCols+" FROM orders WHERE code = ?", code)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListAll returns all orders, newest first.
func (r *OrderRepo) ListAll(ctx context.Context) ([]*Order, error) {
	var list []*Order
	err := r.db.QueryRowsCtx(ctx, &list, "SELECT "+orderCols+" FROM orders ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListByEnforcer returns orders where the user occupies slot A or B, newest first.
func (r *OrderRepo) ListByEnforcer(ctx context.Context, userID int64) ([]*Order, error) {
	var list []*Order
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+orderCols+" FROM orders WHERE enforcer_a_id = ? OR enforcer_b_id = ? ORDER BY id DESC",
		userID, userID)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// AssignEnforcer fills slot "a" or "b" only if currently NULL; else ErrSlotTaken.
func (r *OrderRepo) AssignEnforcer(ctx context.Context, code, slot string, uid int64, name string) error {
	idCol, nameCol := "enforcer_a_id", "enforcer_a_name"
	if slot == "b" {
		idCol, nameCol = "enforcer_b_id", "enforcer_b_name"
	}
	res, err := r.db.ExecCtx(ctx,
		"UPDATE orders SET "+idCol+" = ?, "+nameCol+" = ? WHERE code = ? AND "+idCol+" IS NULL",
		uid, name, code)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSlotTaken
	}
	return nil
}

// RemoveEnforcer clears slot "a" or "b".
func (r *OrderRepo) RemoveEnforcer(ctx context.Context, code, slot string) error {
	idCol, nameCol := "enforcer_a_id", "enforcer_a_name"
	if slot == "b" {
		idCol, nameCol = "enforcer_b_id", "enforcer_b_name"
	}
	_, err := r.db.ExecCtx(ctx,
		"UPDATE orders SET "+idCol+" = NULL, "+nameCol+" = NULL WHERE code = ?", code)
	return err
}

// UpdateStatus sets the order status.
func (r *OrderRepo) UpdateStatus(ctx context.Context, code, status string) error {
	_, err := r.db.ExecCtx(ctx, "UPDATE orders SET status = ? WHERE code = ?", status, code)
	return err
}

// UpdateComplaint sets the complaint status.
func (r *OrderRepo) UpdateComplaint(ctx context.Context, code, complaint string) error {
	_, err := r.db.ExecCtx(ctx, "UPDATE orders SET complaint_status = ? WHERE code = ?", complaint, code)
	return err
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/repo/ -run 'TestOrderRepo|TestOrderEarnings' -v`
Expected: 未设 `TEST_DB_DSN` 时 `TestOrderRepo_*` 显示 SKIP、`TestOrderEarnings` PASS；设了 DSN 且已建表时全 PASS。

- [ ] **Step 5: 全量编译**

Run: `cd server && go build ./...`
Expected: 无错误。

- [ ] **Step 6: Commit**

```bash
git add server/internal/repo/order.go server/internal/repo/order_test.go
git commit -m "feat(repo): add OrderRepo CRUD and enforcer slot methods

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: 注入 OrderRepo 到 ServiceContext

**Files:**
- Modify: `server/internal/svc/service_context.go`

**Interfaces:**
- Consumes: `repo.NewOrderRepo`（Task 3）、已有 `conn`。
- Produces: `ServiceContext.OrderRepo *repo.OrderRepo`（供 Task 6/7 handler 使用）。

- [ ] **Step 1: 加字段**

在 `ServiceContext` struct 内 `ShopRepo` 一行下方加：
```go
	OrderRepo     *repo.OrderRepo
```

- [ ] **Step 2: 构造并赋值**

在 `NewServiceContext` 的 `return &ServiceContext{...}` 中，`ShopRepo: shopRepo,` 一行下方加：
```go
		OrderRepo:     repo.NewOrderRepo(conn),
```

- [ ] **Step 3: 编译**

Run: `cd server && go build ./...`
Expected: 无错误。

- [ ] **Step 4: Commit**

```bash
git add server/internal/svc/service_context.go
git commit -m "feat(svc): inject OrderRepo into ServiceContext

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: 角色中间件、订单编号与路径解析辅助

**Files:**
- Create: `server/internal/handler/order.go`（先只放辅助函数与常量）
- Test: `server/internal/handler/order_test.go`（先测辅助函数）

**Interfaces:**
- Consumes: 已有 `extractUserID`、`writeError`、`svc.ServiceContext`、`ctx.UserRepo.FindByID`。
- Produces:
  - `func requireAnyRole(ctx *svc.ServiceContext, roles []string, next http.HandlerFunc) http.HandlerFunc`
  - `func genOrderCode() string`（返回 `time.Now().UnixMilli()` 的十进制字符串）
  - `func parseOrderPath(path string) (code string, rest []string)`（从 `/api/orders/{code}/...` 解析）
  - 常量集合 `validOrderTypes` / `validOrderStatuses` / `validComplaintStatuses`（`map[string]bool`）

- [ ] **Step 1: 写失败测试**

`server/internal/handler/order_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestParseOrderPath(t *testing.T) {
	code, rest := handler.ParseOrderPathForTest("/api/orders/1699999999/enforcer/a")
	if code != "1699999999" {
		t.Fatalf("code = %q, want 1699999999", code)
	}
	if len(rest) != 2 || rest[0] != "enforcer" || rest[1] != "a" {
		t.Fatalf("rest = %v, want [enforcer a]", rest)
	}
}

func TestClaimHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/orders/123/claim", nil)
	w := httptest.NewRecorder()
	handler.ClaimOrderHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
```

> `ParseOrderPathForTest` 是导出的测试壳，见下方实现（Go 同包外测试需要导出符号；用一个薄导出包装避免暴露内部命名）。

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/handler/ -run 'TestParseOrderPath|TestClaimHandler_NoAuth' -v`
Expected: 编译失败（符号未定义）。

- [ ] **Step 3: 实现辅助与中间件**

`server/internal/handler/order.go`（本任务只放以下内容，handler 主体在 Task 6/7 追加）:

```go
package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"hls.dev/server/internal/svc"
)

var validOrderTypes = map[string]bool{
	"guarantee": true, "fun": true, "companion": true, "teaching": true,
}
var validOrderStatuses = map[string]bool{
	"dispatching": true, "ongoing": true, "settled": true, "refunded": true,
}
var validComplaintStatuses = map[string]bool{
	"none": true, "complained": true, "resolved": true,
}

// genOrderCode returns a timestamp-based unique order code.
func genOrderCode() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// parseOrderPath splits "/api/orders/{code}/a/b" into ("{code}", ["a","b"]).
func parseOrderPath(path string) (string, []string) {
	rest := strings.TrimPrefix(path, "/api/orders/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil
	}
	return parts[0], parts[1:]
}

// ParseOrderPathForTest exposes parseOrderPath to the external test package.
func ParseOrderPathForTest(path string) (string, []string) { return parseOrderPath(path) }

// requireAnyRole rejects users whose role is not in roles.
func requireAnyRole(ctx *svc.ServiceContext, roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		for _, role := range roles {
			if user.Role == role {
				next(w, r)
				return
			}
		}
		writeError(w, r, http.StatusForbidden, "forbidden", nil)
	}
}
```

> 注：`TestClaimHandler_NoAuth` 依赖 `ClaimOrderHandler` 存在。为让本任务可独立编译通过，在 `order.go` 末尾加一个**占位**导出（Task 7 会用真正实现替换其函数体）：
```go
// ClaimOrderHandler is implemented in Task 7; placeholder keeps the package compiling.
func ClaimOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotImplemented, "not_implemented", nil)
	})
}
```
> `requireAnyRole` 在无 auth 头时先返回 401（未触达 ctx），故 `handler.ClaimOrderHandler(nil)` 的 no-auth 测试可通过。

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/handler/ -run 'TestParseOrderPath|TestClaimHandler_NoAuth' -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/order.go server/internal/handler/order_test.go
git commit -m "feat(handler): add role middleware, order-code and path helpers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: 订单读取/创建 handler

**Files:**
- Modify: `server/internal/handler/order.go`（追加视图 DTO 与 4 个 handler）
- Test: `server/internal/handler/order_test.go`（追加 no-auth 测试）

**Interfaces:**
- Consumes: `ctx.OrderRepo`（Task 4）、`ctx.UserRepo`、`requireAnyRole`/`parseOrderPath`/`genOrderCode`/`validOrderTypes`（Task 5）、`Order.Earnings()`（Task 2）。
- Produces:
  - `func toOrderView(o *repo.Order) map[string]any`（统一 JSON 视图，含 `enforcerA/B`（`{id,name}`或 null）、`remark`、`earnings`）
  - `func CreateOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc`（owner/service）
  - `func ListOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc`（owner/service）
  - `func MyOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc`（enforcer）
  - `func OrderDetailHandler(ctx *svc.ServiceContext) http.HandlerFunc`（任意登录角色）

- [ ] **Step 1: 追加 no-auth 失败测试**

在 `server/internal/handler/order_test.go` 追加：

```go
func TestCreateOrderHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/orders", nil)
	w := httptest.NewRecorder()
	handler.CreateOrderHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestListOrdersHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	w := httptest.NewRecorder()
	handler.ListOrdersHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMyOrdersHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/my-orders", nil)
	w := httptest.NewRecorder()
	handler.MyOrdersHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/handler/ -run 'TestCreateOrderHandler_NoAuth|TestListOrdersHandler_NoAuth|TestMyOrdersHandler_NoAuth' -v`
Expected: 编译失败（handler 未定义）。

- [ ] **Step 3: 实现视图 DTO 与 handler**

在 `order.go` 顶部 import 补充（合并进已有 import 块）：
```go
	"database/sql"
	"encoding/json"

	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
```

追加：

```go
func nullEnforcer(id sql.NullInt64, name sql.NullString) any {
	if !id.Valid {
		return nil
	}
	return map[string]any{"id": id.Int64, "name": name.String}
}

// toOrderView builds the canonical JSON view of an order.
func toOrderView(o *repo.Order) map[string]any {
	return map[string]any{
		"code":            o.Code,
		"title":           o.Title,
		"orderType":       o.OrderType,
		"price":           o.Price,
		"commissionRate":  o.CommissionRate,
		"bonusRate":       o.BonusRate,
		"playerCount":     o.PlayerCount,
		"dispatcherName":  o.DispatcherName,
		"enforcerA":       nullEnforcer(o.EnforcerAID, o.EnforcerAName),
		"enforcerB":       nullEnforcer(o.EnforcerBID, o.EnforcerBName),
		"remark":          o.Remark.String,
		"status":          o.Status,
		"complaintStatus": o.ComplaintStatus,
		"earnings":        o.Earnings(),
		"createdAt":       o.CreatedAt,
	}
}

type createOrderRequest struct {
	Title          string  `json:"title"`
	OrderType      string  `json:"orderType"`
	Price          float64 `json:"price"`
	CommissionRate float64 `json:"commissionRate"`
	BonusRate      float64 `json:"bonusRate"`
	PlayerCount    int     `json:"playerCount"`
	Remark         string  `json:"remark"`
}

// CreateOrderHandler handles POST /api/orders.
func CreateOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" || !validOrderTypes[req.OrderType] || req.Price <= 0 ||
			(req.PlayerCount != 1 && req.PlayerCount != 2) {
			writeError(w, r, http.StatusBadRequest, "invalid_order", nil)
			return
		}
		o := &repo.Order{
			Title:          req.Title,
			OrderType:      req.OrderType,
			Price:          req.Price,
			CommissionRate: req.CommissionRate,
			BonusRate:      req.BonusRate,
			PlayerCount:    req.PlayerCount,
			DispatcherID:   userID,
			DispatcherName: user.Nickname,
			Remark:         sql.NullString{String: strings.TrimSpace(req.Remark), Valid: req.Remark != ""},
		}
		// retry a few times on code collision
		var created *repo.Order
		for i := 0; i < 5; i++ {
			o.Code = genOrderCode()
			created, err = ctx.OrderRepo.Create(r.Context(), o)
			if err == nil {
				break
			}
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(created))
	})
}

// ListOrdersHandler handles GET /api/orders.
func ListOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		orders, err := ctx.OrderRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(orders))
		for _, o := range orders {
			views = append(views, toOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// MyOrdersHandler handles GET /api/my-orders.
func MyOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		orders, err := ctx.OrderRepo.ListByEnforcer(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(orders))
		for _, o := range orders {
			views = append(views, toOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// OrderDetailHandler handles GET /api/orders/:code (any logged-in role).
func OrderDetailHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service", "enforcer", "customer"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusNotFound, "order_not_found", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/handler/ -run 'Order' -v && cd server && go build ./...`
Expected: no-auth 测试 PASS，编译通过。

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/order.go server/internal/handler/order_test.go
git commit -m "feat(handler): add order create/list/mine/detail handlers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: 订单生命周期 handler（扣单/换人/状态/客诉）

**Files:**
- Modify: `server/internal/handler/order.go`（用真实现替换 Task 5 的 `ClaimOrderHandler` 占位，追加另 3 个 handler）
- Test: `server/internal/handler/order_test.go`（追加 no-auth 测试）

**Interfaces:**
- Consumes: `ctx.OrderRepo`（含 `AssignEnforcer`/`RemoveEnforcer`/`UpdateStatus`/`UpdateComplaint`/`GetByCode`、`repo.ErrSlotTaken`）、`parseOrderPath`、`validOrderStatuses`/`validComplaintStatuses`。
- Produces:
  - `func ClaimOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc`（enforcer；替换占位）
  - `func RemoveEnforcerHandler(ctx *svc.ServiceContext) http.HandlerFunc`（owner/service）
  - `func UpdateOrderStatusHandler(ctx *svc.ServiceContext) http.HandlerFunc`（owner/service）
  - `func UpdateOrderComplaintHandler(ctx *svc.ServiceContext) http.HandlerFunc`（owner/service）

- [ ] **Step 1: 追加 no-auth 失败测试**

在 `order_test.go` 追加：

```go
func TestRemoveEnforcerHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/orders/123/enforcer/a", nil)
	w := httptest.NewRecorder()
	handler.RemoveEnforcerHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUpdateOrderStatusHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/orders/123/status", nil)
	w := httptest.NewRecorder()
	handler.UpdateOrderStatusHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUpdateOrderComplaintHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/orders/123/complaint", nil)
	w := httptest.NewRecorder()
	handler.UpdateOrderComplaintHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/handler/ -run 'RemoveEnforcer|UpdateOrderStatus|UpdateOrderComplaint' -v`
Expected: 编译失败（handler 未定义）。

- [ ] **Step 3: 实现 4 个 handler**

删除 Task 5 里 `ClaimOrderHandler` 的占位实现，替换为下面的真实现，并追加另 3 个。import 需补 `"hls.dev/server/internal/repo"`（Task 6 已补）。

```go
// ClaimOrderHandler handles POST /api/orders/:code/claim (enforcer).
// Fills the first open slot (A then B); rejects if the order is not dispatching,
// is already full, or the user already occupies a slot. When all required slots
// are filled, transitions the order to "ongoing".
func ClaimOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusNotFound, "order_not_found", nil)
			return
		}
		if o.Status != "dispatching" {
			writeError(w, r, http.StatusConflict, "order_not_claimable", nil)
			return
		}
		if (o.EnforcerAID.Valid && o.EnforcerAID.Int64 == userID) ||
			(o.EnforcerBID.Valid && o.EnforcerBID.Int64 == userID) {
			writeError(w, r, http.StatusConflict, "already_claimed", nil)
			return
		}
		// choose first open slot within required player_count
		slot := ""
		if !o.EnforcerAID.Valid {
			slot = "a"
		} else if o.PlayerCount == 2 && !o.EnforcerBID.Valid {
			slot = "b"
		}
		if slot == "" {
			writeError(w, r, http.StatusConflict, "order_full", nil)
			return
		}
		if err := ctx.OrderRepo.AssignEnforcer(r.Context(), code, slot, userID, user.Nickname); err != nil {
			if err == repo.ErrSlotTaken {
				writeError(w, r, http.StatusConflict, "slot_taken", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		// re-read to decide status transition
		o, err = ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if isFullyStaffed(o) {
			if err := ctx.OrderRepo.UpdateStatus(r.Context(), code, "ongoing"); err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			o.Status = "ongoing"
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

func isFullyStaffed(o *repo.Order) bool {
	if o.PlayerCount == 1 {
		return o.EnforcerAID.Valid
	}
	return o.EnforcerAID.Valid && o.EnforcerBID.Valid
}

// RemoveEnforcerHandler handles DELETE /api/orders/:code/enforcer/:slot (owner/service).
// Clears the slot; if the order is no longer fully staffed, reverts to "dispatching".
func RemoveEnforcerHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, rest := parseOrderPath(r.URL.Path)
		if len(rest) != 2 || rest[0] != "enforcer" || (rest[1] != "a" && rest[1] != "b") {
			writeError(w, r, http.StatusBadRequest, "invalid_slot", nil)
			return
		}
		if err := ctx.OrderRepo.RemoveEnforcer(r.Context(), code, rest[1]); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusNotFound, "order_not_found", nil)
			return
		}
		if !isFullyStaffed(o) && (o.Status == "ongoing") {
			_ = ctx.OrderRepo.UpdateStatus(r.Context(), code, "dispatching")
			o.Status = "dispatching"
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

type statusRequest struct {
	Status string `json:"status"`
}

// UpdateOrderStatusHandler handles PUT /api/orders/:code/status (owner/service).
func UpdateOrderStatusHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		var req statusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validOrderStatuses[req.Status] {
			writeError(w, r, http.StatusBadRequest, "invalid_status", nil)
			return
		}
		if err := ctx.OrderRepo.UpdateStatus(r.Context(), code, req.Status); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

type complaintRequest struct {
	ComplaintStatus string `json:"complaintStatus"`
}

// UpdateOrderComplaintHandler handles PUT /api/orders/:code/complaint (owner/service).
func UpdateOrderComplaintHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		var req complaintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validComplaintStatuses[req.ComplaintStatus] {
			writeError(w, r, http.StatusBadRequest, "invalid_complaint", nil)
			return
		}
		if err := ctx.OrderRepo.UpdateComplaint(r.Context(), code, req.ComplaintStatus); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}
```

- [ ] **Step 4: 运行确认通过 + 全量测试**

Run: `cd server && go test ./... `
Expected: 全 PASS（集成测试无 DSN 时 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/order.go server/internal/handler/order_test.go
git commit -m "feat(handler): add claim/remove-enforcer/status/complaint handlers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: 注册订单路由

**Files:**
- Modify: `server/cmd/gameserver/main.go`

**Interfaces:**
- Consumes: Task 6/7 的 handler。
- Produces: 8 条订单路由。

- [ ] **Step 1: 追加路由**

在 `server.AddRoutes([]rest.Route{ ... })` 内，用户路由那几行之后追加：

```go
		{Method: http.MethodPost, Path: "/api/orders", Handler: handler.CreateOrderHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/orders", Handler: handler.ListOrdersHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/my-orders", Handler: handler.MyOrdersHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/orders/:code", Handler: handler.OrderDetailHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/orders/:code/claim", Handler: handler.ClaimOrderHandler(ctx)},
		{Method: http.MethodDelete, Path: "/api/orders/:code/enforcer/:slot", Handler: handler.RemoveEnforcerHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/orders/:code/status", Handler: handler.UpdateOrderStatusHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/orders/:code/complaint", Handler: handler.UpdateOrderComplaintHandler(ctx)},
```

- [ ] **Step 2: 编译并启动冒烟**

Run: `cd server && go build ./... && go vet ./...`
Expected: 无错误。（`GET /api/orders`(list) 与 `GET /api/orders/:code`(detail) 属不同段深度，go-zero 路由树可区分；`my-orders` 独立路径避免与 `:code` 冲突。）

- [ ] **Step 3: Commit**

```bash
git add server/cmd/gameserver/main.go
git commit -m "feat(server): register order routes

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: 前端订单类型、标签映射与 API 封装

**Files:**
- Create: `src/shared/orderLabels.ts`
- Create: `src/api/orders.ts`

**Interfaces:**
- Consumes: 现有 `src/http.ts` 默认导出 `http`。
- Produces:
  - 类型 `OrderType` / `OrderStatus` / `ComplaintStatus` / `OrderView` / `EnforcerRef`
  - 映射 `orderTypeLabels` / `orderStatusLabels` / `complaintLabels`
  - API：`listOrders` / `myOrders` / `getOrder` / `createOrder` / `claimOrder` / `removeEnforcer` / `updateOrderStatus` / `updateComplaint`

- [ ] **Step 1: 写标签与类型**

`src/shared/orderLabels.ts`:

```ts
export type OrderType = 'guarantee' | 'fun' | 'companion' | 'teaching'
export type OrderStatus = 'dispatching' | 'ongoing' | 'settled' | 'refunded'
export type ComplaintStatus = 'none' | 'complained' | 'resolved'

export interface EnforcerRef { id: number; name: string }

export interface OrderView {
  code: string
  title: string
  orderType: OrderType
  price: number
  commissionRate: number
  bonusRate: number
  playerCount: number
  dispatcherName: string
  enforcerA: EnforcerRef | null
  enforcerB: EnforcerRef | null
  remark: string
  status: OrderStatus
  complaintStatus: ComplaintStatus
  earnings: number
  createdAt: string
}

export const orderTypeLabels: Record<OrderType, string> = {
  guarantee: '保底单',
  fun: '趣味单',
  companion: '陪玩单',
  teaching: '教学单',
}

export const orderStatusLabels: Record<OrderStatus, string> = {
  dispatching: '派单中',
  ongoing: '进行中',
  settled: '结单',
  refunded: '退单',
}

export const complaintLabels: Record<ComplaintStatus, string> = {
  none: '无客诉',
  complained: '被投诉',
  resolved: '已处理',
}
```

- [ ] **Step 2: 写 API 封装**

`src/api/orders.ts`:

```ts
import http from '../http'
import type { OrderView, OrderType, OrderStatus, ComplaintStatus } from '../shared/orderLabels'

export interface CreateOrderPayload {
  title: string
  orderType: OrderType
  price: number
  commissionRate: number
  bonusRate: number
  playerCount: 1 | 2
  remark?: string
}

export async function listOrders(): Promise<OrderView[]> {
  const res = await http.get<OrderView[]>('/orders')
  return res.data
}

export async function myOrders(): Promise<OrderView[]> {
  const res = await http.get<OrderView[]>('/my-orders')
  return res.data
}

export async function getOrder(code: string): Promise<OrderView> {
  const res = await http.get<OrderView>(`/orders/${code}`)
  return res.data
}

export async function createOrder(payload: CreateOrderPayload): Promise<OrderView> {
  const res = await http.post<OrderView>('/orders', payload)
  return res.data
}

export async function claimOrder(code: string): Promise<OrderView> {
  const res = await http.post<OrderView>(`/orders/${code}/claim`)
  return res.data
}

export async function removeEnforcer(code: string, slot: 'a' | 'b'): Promise<OrderView> {
  const res = await http.delete<OrderView>(`/orders/${code}/enforcer/${slot}`)
  return res.data
}

export async function updateOrderStatus(code: string, status: OrderStatus): Promise<void> {
  await http.put(`/orders/${code}/status`, { status })
}

export async function updateComplaint(code: string, complaintStatus: ComplaintStatus): Promise<void> {
  await http.put(`/orders/${code}/complaint`, { complaintStatus })
}
```

- [ ] **Step 3: 类型检查**

Run: `npm run typecheck`
Expected: 无错误。

- [ ] **Step 4: Commit**

```bash
git add src/shared/orderLabels.ts src/api/orders.ts
git commit -m "feat(fe): add order labels, types and API client

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: 路由守卫放宽 + 扣单页路由

**Files:**
- Modify: `src/router/index.ts`

**Interfaces:**
- Consumes: `useAuthStore`（已存在，`auth.user.role`）。
- Produces: `/order/:code` 命名路由 `order-claim`；`/admin` 守卫允许 owner/service/enforcer。

- [ ] **Step 1: 加路由**

在 `routes` 数组里 `/admin` 之后加：
```ts
    {
      path: '/order/:code',
      name: 'order-claim',
      component: () => import('@/pages/OrderClaimPage.vue'),
    },
```

- [ ] **Step 2: 改 /admin 守卫**

把 `router.beforeEach` 里 `/admin` 分支替换为：
```ts
  if (to.path === '/admin') {
    if (!auth.token) return { name: 'login', query: { redirect: to.fullPath } }
    const allowed = ['owner', 'service', 'enforcer']
    if (auth.user && !allowed.includes(auth.user.role)) return { name: 'home' }
  }
```

- [ ] **Step 3: 类型检查**

Run: `npm run typecheck`
Expected: 无错误（`OrderClaimPage.vue` 在 Task 14 创建；本步骤仅动态 import 字符串，不会因文件缺失导致 typecheck 失败，但为保险可先建空壳——见下）。

若 typecheck 因动态 import 报错，先建占位 `src/pages/OrderClaimPage.vue`：
```vue
<template><div /></template>
```
Task 14 再替换为完整实现。

- [ ] **Step 4: Commit**

```bash
git add src/router/index.ts src/pages/OrderClaimPage.vue
git commit -m "feat(fe): widen /admin guard and add order-claim route

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: AdminPage 改造成 tab 外壳 + 抽取 UsersPanel

**Files:**
- Create: `src/pages/admin/UsersPanel.vue`
- Modify: `src/pages/AdminPage.vue`

**Interfaces:**
- Consumes: `useAuthStore`（`auth.user.role`）。
- Produces: 外壳按角色渲染 tab；`UsersPanel.vue` 承载原用户管理逻辑。

- [ ] **Step 1: 抽取 UsersPanel**

新建 `src/pages/admin/UsersPanel.vue`，把**现有 `src/pages/AdminPage.vue` 的全部 `<script setup>` 与 `<template>`（用户列表 + 创建用户弹窗 + ConfirmDialog）原样搬入**。仅调整：`import` 路径由 `../http` → `../../http`、`../shared/...` → `../../shared/...`；根 `<div>` 去掉 `min-h-screen ... max-w-3xl mx-auto`（外层由外壳负责），改为 `class="space-y-6"`。

> 该组件内容与原 `AdminPage.vue` 一致（列表、`openCreate`/`submitCreate`/`askResetPassword`/`askResetNickname`/`handleConfirm`、`ConfirmDialog`），此处不重复粘贴——直接搬运原文件正文即可。

- [ ] **Step 2: 重写 AdminPage 为外壳**

`src/pages/AdminPage.vue`:

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import UsersPanel from './admin/UsersPanel.vue'
import OrdersPanel from './admin/OrdersPanel.vue'
import MyOrdersPanel from './admin/MyOrdersPanel.vue'

type TabKey = 'orders' | 'users' | 'my-orders'

const auth = useAuthStore()

interface TabDef { key: TabKey; label: string; roles: string[] }
const allTabs: TabDef[] = [
  { key: 'orders', label: '订单管理', roles: ['owner', 'service'] },
  { key: 'users', label: '用户管理', roles: ['owner'] },
  { key: 'my-orders', label: '我的订单', roles: ['enforcer'] },
]

const tabs = computed(() =>
  allTabs.filter((t) => auth.user && t.roles.includes(auth.user.role)),
)
const active = ref<TabKey>('orders')

onMounted(() => {
  if (tabs.value.length && !tabs.value.find((t) => t.key === active.value)) {
    active.value = tabs.value[0].key
  }
})
</script>

<template>
  <div class="min-h-screen flex">
    <aside class="w-48 shrink-0 border-r-2 border-hls-border/50 p-4 space-y-2">
      <h1 class="font-body font-bold text-lg text-hls-gold mb-4">后台管理</h1>
      <button
        v-for="t in tabs"
        :key="t.key"
        class="block w-full text-left px-3 py-2 rounded font-body text-sm"
        :class="active === t.key ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="active = t.key"
      >
        {{ t.label }}
      </button>
    </aside>

    <main class="flex-1 px-6 py-8 max-w-5xl">
      <OrdersPanel v-if="active === 'orders'" />
      <UsersPanel v-else-if="active === 'users'" />
      <MyOrdersPanel v-else-if="active === 'my-orders'" />
    </main>
  </div>
</template>
```

> 注：`OrdersPanel` / `MyOrdersPanel` 在 Task 12/13 创建。为让本任务 typecheck 通过，先建两文件的占位：`<template><div /></template>`，Task 12/13 再替换。

- [ ] **Step 3: 建占位并类型检查**

先建 `src/pages/admin/OrdersPanel.vue` 与 `src/pages/admin/MyOrdersPanel.vue` 占位（`<template><div /></template>`）。
Run: `npm run typecheck`
Expected: 无错误。

- [ ] **Step 4: Commit**

```bash
git add src/pages/AdminPage.vue src/pages/admin/
git commit -m "feat(fe): convert AdminPage to tabbed shell, extract UsersPanel

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: OrdersPanel（订单列表 + 建单 + 操作）

**Files:**
- Modify: `src/pages/admin/OrdersPanel.vue`（替换占位）

**Interfaces:**
- Consumes: `src/api/orders.ts`、`src/shared/orderLabels.ts`、`ConfirmDialog`。
- Produces: 订单管理完整界面。

- [ ] **Step 1: 实现组件**

`src/pages/admin/OrdersPanel.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listOrders, createOrder, removeEnforcer, updateOrderStatus, updateComplaint,
  type CreateOrderPayload,
} from '../../api/orders'
import {
  orderTypeLabels, orderStatusLabels, complaintLabels,
  type OrderView, type OrderType, type OrderStatus, type ComplaintStatus,
} from '../../shared/orderLabels'

const orders = ref<OrderView[]>([])
const loading = ref(false)
const listError = ref('')

async function load() {
  loading.value = true
  listError.value = ''
  try {
    orders.value = await listOrders()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ---- create ----
const showCreate = ref(false)
const form = ref<CreateOrderPayload>({
  title: '', orderType: 'companion', price: 0,
  commissionRate: 20, bonusRate: 3, playerCount: 1, remark: '',
})
const createError = ref('')
const creating = ref(false)

function openCreate() {
  form.value = { title: '', orderType: 'companion', price: 0, commissionRate: 20, bonusRate: 3, playerCount: 1, remark: '' }
  createError.value = ''
  showCreate.value = true
}

async function submitCreate() {
  createError.value = ''
  if (!form.value.title.trim() || form.value.price <= 0) {
    createError.value = '请填写单子名称与价格'
    return
  }
  creating.value = true
  try {
    await createOrder({ ...form.value, title: form.value.title.trim() })
    showCreate.value = false
    await load()
  } catch {
    createError.value = '创建失败，请重试'
  } finally {
    creating.value = false
  }
}

function shareLink(code: string) {
  const url = `${window.location.origin}/order/${code}`
  navigator.clipboard?.writeText(url)
  copiedCode.value = code
  setTimeout(() => { if (copiedCode.value === code) copiedCode.value = '' }, 1500)
}
const copiedCode = ref('')

async function onRemoveEnforcer(code: string, slot: 'a' | 'b') {
  try {
    await removeEnforcer(code, slot)
    await load()
  } catch { listError.value = '操作失败，请重试' }
}

async function onStatus(code: string, status: OrderStatus) {
  try { await updateOrderStatus(code, status); await load() }
  catch { listError.value = '操作失败，请重试' }
}

async function onComplaint(code: string, complaintStatus: ComplaintStatus) {
  try { await updateComplaint(code, complaintStatus); await load() }
  catch { listError.value = '操作失败，请重试' }
}

const orderTypeOptions = Object.entries(orderTypeLabels) as [OrderType, string][]
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">订单管理</h2>
      <button class="hls-btn py-2 px-5" @click="openCreate">▶ 创建订单</button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60">加载中…</p>

    <div v-if="!loading" class="hls-panel overflow-x-auto">
      <div class="hls-panel-title">订单列表</div>
      <table class="w-full text-sm whitespace-nowrap">
        <thead>
          <tr class="border-b-2 border-hls-border/50 text-hls-text-dark">
            <th class="text-left px-3 py-2 font-medium">编号</th>
            <th class="text-left px-3 py-2 font-medium">名称</th>
            <th class="text-left px-3 py-2 font-medium">类型</th>
            <th class="text-left px-3 py-2 font-medium">价格</th>
            <th class="text-left px-3 py-2 font-medium">人数</th>
            <th class="text-left px-3 py-2 font-medium">打手A/B</th>
            <th class="text-left px-3 py-2 font-medium">状态</th>
            <th class="text-left px-3 py-2 font-medium">客诉</th>
            <th class="text-left px-3 py-2 font-medium">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.code" class="border-b border-hls-border/25 last:border-0">
            <td class="px-3 py-2 text-hls-text-muted">{{ o.code }}</td>
            <td class="px-3 py-2 text-hls-text-dark">{{ o.title }}</td>
            <td class="px-3 py-2">{{ orderTypeLabels[o.orderType] }}</td>
            <td class="px-3 py-2">{{ o.price }}</td>
            <td class="px-3 py-2">{{ o.playerCount }}</td>
            <td class="px-3 py-2">
              <div>A：{{ o.enforcerA?.name || '—' }}
                <button v-if="o.enforcerA" class="text-xs text-red-400 ml-1" @click="onRemoveEnforcer(o.code, 'a')">移除</button>
              </div>
              <div v-if="o.playerCount === 2">B：{{ o.enforcerB?.name || '—' }}
                <button v-if="o.enforcerB" class="text-xs text-red-400 ml-1" @click="onRemoveEnforcer(o.code, 'b')">移除</button>
              </div>
            </td>
            <td class="px-3 py-2">
              <select :value="o.status" class="hls-input text-xs py-1"
                @change="onStatus(o.code, ($event.target as HTMLSelectElement).value as OrderStatus)">
                <option v-for="(label, key) in orderStatusLabels" :key="key" :value="key">{{ label }}</option>
              </select>
            </td>
            <td class="px-3 py-2">
              <select :value="o.complaintStatus" class="hls-input text-xs py-1"
                @change="onComplaint(o.code, ($event.target as HTMLSelectElement).value as ComplaintStatus)">
                <option v-for="(label, key) in complaintLabels" :key="key" :value="key">{{ label }}</option>
              </select>
            </td>
            <td class="px-3 py-2">
              <button class="hls-btn-secondary text-xs py-1 px-3" @click="shareLink(o.code)">
                {{ copiedCode === o.code ? '已复制' : '复制链接' }}
              </button>
            </td>
          </tr>
          <tr v-if="!orders.length"><td colspan="9" class="px-3 py-6 text-center text-hls-text-nav/60">暂无订单</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <transition name="fade">
      <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4" @click.self="showCreate = false">
        <div class="hls-panel w-[92%] max-w-md">
          <div class="hls-panel-title text-center">创建订单</div>
          <div class="p-6 space-y-3">
            <p v-if="createError" class="font-body text-xs text-red-400 text-center">{{ createError }}</p>
            <input v-model="form.title" placeholder="单子名称" class="hls-input w-full" />
            <select v-model="form.orderType" class="hls-input w-full">
              <option v-for="[key, label] in orderTypeOptions" :key="key" :value="key">{{ label }}</option>
            </select>
            <div class="flex gap-3">
              <input v-model.number="form.price" type="number" placeholder="价格" class="hls-input w-full" />
              <select v-model.number="form.playerCount" class="hls-input w-full">
                <option :value="1">1 人</option>
                <option :value="2">2 人</option>
              </select>
            </div>
            <div class="flex gap-3">
              <input v-model.number="form.commissionRate" type="number" placeholder="抽成%" class="hls-input w-full" />
              <input v-model.number="form.bonusRate" type="number" placeholder="鸡腿%" class="hls-input w-full" />
            </div>
            <textarea v-model="form.remark" placeholder="备注（可选）" class="hls-input w-full" rows="2" />
            <div class="flex gap-3 pt-1">
              <button class="flex-1 hls-btn-secondary py-2" @click="showCreate = false">取消</button>
              <button class="flex-1 hls-btn py-2 disabled:opacity-40" :disabled="creating" @click="submitCreate">
                {{ creating ? '创建中…' : '▶ 创建' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.18s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
```

- [ ] **Step 2: 类型检查 + 构建**

Run: `npm run typecheck && npm run build`
Expected: 均无错误。

- [ ] **Step 3: Commit**

```bash
git add src/pages/admin/OrdersPanel.vue
git commit -m "feat(fe): implement OrdersPanel with create and lifecycle actions

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: MyOrdersPanel（打手我的订单）

**Files:**
- Modify: `src/pages/admin/MyOrdersPanel.vue`（替换占位）

**Interfaces:**
- Consumes: `myOrders`（`src/api/orders.ts`）、标签映射。
- Produces: 打手订单列表，列：订单编号、订单类型、单子名称、收益、抽成、派单人、备注、鸡腿。

- [ ] **Step 1: 实现组件**

`src/pages/admin/MyOrdersPanel.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { myOrders } from '../../api/orders'
import { orderTypeLabels, type OrderView } from '../../shared/orderLabels'

const orders = ref<OrderView[]>([])
const loading = ref(false)
const listError = ref('')

async function load() {
  loading.value = true
  listError.value = ''
  try {
    orders.value = await myOrders()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <h2 class="font-body font-bold text-xl text-hls-gold">我的订单</h2>
    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60">加载中…</p>

    <div v-if="!loading" class="hls-panel overflow-x-auto">
      <div class="hls-panel-title">订单列表</div>
      <table class="w-full text-sm whitespace-nowrap">
        <thead>
          <tr class="border-b-2 border-hls-border/50 text-hls-text-dark">
            <th class="text-left px-3 py-2 font-medium">订单编号</th>
            <th class="text-left px-3 py-2 font-medium">类型</th>
            <th class="text-left px-3 py-2 font-medium">单子名称</th>
            <th class="text-left px-3 py-2 font-medium">收益</th>
            <th class="text-left px-3 py-2 font-medium">抽成</th>
            <th class="text-left px-3 py-2 font-medium">派单人</th>
            <th class="text-left px-3 py-2 font-medium">备注</th>
            <th class="text-left px-3 py-2 font-medium">鸡腿</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.code" class="border-b border-hls-border/25 last:border-0">
            <td class="px-3 py-2 text-hls-text-muted">{{ o.code }}</td>
            <td class="px-3 py-2">{{ orderTypeLabels[o.orderType] }}</td>
            <td class="px-3 py-2 text-hls-text-dark">{{ o.title }}</td>
            <td class="px-3 py-2 text-hls-gold-dark font-medium">{{ o.earnings }}</td>
            <td class="px-3 py-2">{{ o.commissionRate }}%</td>
            <td class="px-3 py-2">{{ o.dispatcherName }}</td>
            <td class="px-3 py-2 text-hls-text-muted">{{ o.remark || '—' }}</td>
            <td class="px-3 py-2">{{ o.bonusRate }}%</td>
          </tr>
          <tr v-if="!orders.length"><td colspan="8" class="px-3 py-6 text-center text-hls-text-nav/60">暂无订单</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 类型检查 + 构建**

Run: `npm run typecheck && npm run build`
Expected: 均无错误。

- [ ] **Step 3: Commit**

```bash
git add src/pages/admin/MyOrdersPanel.vue
git commit -m "feat(fe): implement MyOrdersPanel for enforcers

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 14: 扣单/分享页 OrderClaimPage

**Files:**
- Modify: `src/pages/OrderClaimPage.vue`（替换 Task 10 的占位）

**Interfaces:**
- Consumes: `getOrder` / `claimOrder`（`src/api/orders.ts`）、`useAuthStore`、`useRoute`/`useRouter`、标签映射。
- Produces: 完整订单信息展示 + 预估收益 + 扣单按钮（未登录跳登录带 redirect）。

- [ ] **Step 1: 实现页面**

`src/pages/OrderClaimPage.vue`:

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getOrder, claimOrder } from '../api/orders'
import { useAuthStore } from '../stores/auth'
import {
  orderTypeLabels, orderStatusLabels, type OrderView,
} from '../shared/orderLabels'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const code = route.params.code as string
const order = ref<OrderView | null>(null)
const loading = ref(true)
const loadError = ref('')
const actionError = ref('')
const claiming = ref(false)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    order.value = await getOrder(code)
  } catch {
    loadError.value = '订单不存在或已失效'
  } finally {
    loading.value = false
  }
}
onMounted(load)

const meId = computed(() => auth.user?.id ?? null)
const alreadyClaimed = computed(() =>
  !!order.value &&
  ((order.value.enforcerA?.id === meId.value) || (order.value.enforcerB?.id === meId.value)),
)
const isFull = computed(() => {
  const o = order.value
  if (!o) return false
  return o.playerCount === 1 ? !!o.enforcerA : !!o.enforcerA && !!o.enforcerB
})
const claimable = computed(() =>
  !!order.value && order.value.status === 'dispatching' && !isFull.value && !alreadyClaimed.value,
)

async function onClaim() {
  if (!auth.token) {
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  actionError.value = ''
  claiming.value = true
  try {
    order.value = await claimOrder(code)
  } catch {
    actionError.value = '扣单失败，可能已被他人接单'
    await load()
  } finally {
    claiming.value = false
  }
}
</script>

<template>
  <div class="min-h-screen px-4 py-8 max-w-md mx-auto">
    <p v-if="loading" class="font-body text-sm text-hls-text-nav/60">加载中…</p>
    <p v-else-if="loadError" class="font-body text-sm text-red-400">{{ loadError }}</p>

    <div v-else-if="order" class="hls-panel">
      <div class="hls-panel-title text-center">{{ order.title }}</div>
      <div class="p-6 space-y-3 font-body text-sm">
        <div class="flex justify-between"><span class="text-hls-text-muted">订单编号</span><span>{{ order.code }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">类型</span><span>{{ orderTypeLabels[order.orderType] }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">价格</span><span>{{ order.price }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">抽成</span><span>{{ order.commissionRate }}%</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">鸡腿</span><span>{{ order.bonusRate }}%</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">打手人数</span><span>{{ order.playerCount }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">派单人</span><span>{{ order.dispatcherName }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">备注</span><span>{{ order.remark || '—' }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">状态</span><span>{{ orderStatusLabels[order.status] }}</span></div>
        <div class="flex justify-between"><span class="text-hls-text-muted">打手A</span><span>{{ order.enforcerA?.name || '空缺' }}</span></div>
        <div v-if="order.playerCount === 2" class="flex justify-between"><span class="text-hls-text-muted">打手B</span><span>{{ order.enforcerB?.name || '空缺' }}</span></div>

        <div class="hls-stat text-center py-3">
          <p class="text-hls-text-muted text-xs">预估收益（每人）</p>
          <p class="text-hls-gold-dark font-bold text-lg">{{ order.earnings }}</p>
        </div>

        <p v-if="actionError" class="text-xs text-red-400 text-center">{{ actionError }}</p>
        <p v-if="alreadyClaimed" class="text-xs text-hls-gold-dark text-center">你已接此单</p>

        <button
          class="hls-btn w-full py-2 disabled:opacity-40"
          :disabled="claiming || !claimable"
          @click="onClaim"
        >
          <template v-if="!auth.token">登录并扣单</template>
          <template v-else-if="alreadyClaimed">已接单</template>
          <template v-else-if="isFull">已招满</template>
          <template v-else-if="order.status !== 'dispatching'">不可接单</template>
          <template v-else>{{ claiming ? '扣单中…' : '▶ 扣单' }}</template>
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: 类型检查 + 构建**

Run: `npm run typecheck && npm run build`
Expected: 均无错误。

- [ ] **Step 3: Commit**

```bash
git add src/pages/OrderClaimPage.vue
git commit -m "feat(fe): implement OrderClaimPage with claim flow

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 15: 端到端手动验收

**Files:** 无（验收）

- [ ] **Step 1: 起服务**

Run（两个终端）：
```bash
cd server && go run ./cmd/gameserver -f etc/gameserver.yaml
npm run dev
```
前置：`orders` 表已建（Task 1 的 SQL 应用到运行库）；至少各有一个 owner/service/enforcer 账号。

- [ ] **Step 2: 走查清单**

- [ ] 店长/客服登录 → `/admin` 见「订单管理」tab；店长另见「用户管理」。
- [ ] 建单（2 人单）→ 列表出现，状态「派单中」。
- [ ] 「复制链接」→ 粘贴得 `/order/<code>`。
- [ ] 打手打开该链接 → 见完整信息 + 预估收益；未登录点按钮跳登录并可回跳。
- [ ] 打手扣单 → 填入 A；再换一个打手扣单 → 填入 B，状态转「进行中」。
- [ ] 打手 `/admin` →「我的订单」→ 见该单，收益 = 价格×(1−抽成%+鸡腿%)/人数。
- [ ] 店长在订单管理移除 B → 状态回「派单中」；改状态为「结单」；客诉设「被投诉」→「已处理」。
- [ ] 客户账号访问 `/admin` → 被跳走。

- [ ] **Step 3: 若发现问题**

回到对应 Task 修复并重跑该 Task 的验证步骤。全部通过后本计划完成。

---

## Self-Review 记录

- **Spec 覆盖**：权限模型(Task 5/10/11)、页面结构(11/12/13)、扣单页(10/14)、状态机(3/7)、收益公式(2)、orders 表(1)、8 条 API(6/7/8)、枚举映射(9)、YAGNI(全程不做编辑/分页) — 均有对应任务。
- **占位符**：无 TBD/TODO；所有步骤含实际代码或确切命令。
- **类型一致**：`Order.Earnings()`、`OrderRepo.*` 签名、`OrderView`/`toOrderView` 字段（`enforcerA/B`、`earnings`、`complaintStatus`…）在前后端一致；路由 `/api/my-orders` 与 `:code` 不冲突。
- **偏离 spec 的小调整**：打手列表接口由 spec 的 `/api/orders/mine` 改为 `/api/my-orders`，避免与 `/api/orders/:code` 路由歧义（已在 Task 8 说明）。

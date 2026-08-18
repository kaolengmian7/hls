# 打手管理 & 工资管理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 打手管理 (per-enforcer deposit / order-count / review-rate / fine dashboard), 工资管理 (settled-order payroll that appears 3 days after settlement, auto-deducts deposit then fine, and marks paid), and an enforcer-facing 结单 button that transitions an order `ongoing → settled` and lazily creates payroll items.

**Architecture:** Go (go-zero) backend + Vue 3 / TS frontend. Two new tables (`enforcer_accounts`, `payroll_items`). Settling an order creates one `pending` payroll item per filled enforcer slot inside the same transaction; `payroll_items.created_at` is the 3-day window anchor (no `settled_at` column on orders). Deduction is a pure function frozen into the payroll row at pay time; the pending list shows a live projection.

**Tech Stack:** Go 1.x, go-zero (`rest`, `sqlx`), MySQL; Vue 3 `<script setup>`, Pinia, axios, Tailwind. Standard-library `testing` (table-driven; no testify). Integration repo tests read `TEST_DB_DSN` and skip when unset.

**Spec:** `docs/superpowers/specs/2026-08-18-enforcer-payroll-management-design.md`

## Global Constraints

- Money is `DECIMAL(10,2)` in SQL and `float64` in Go; round monetary results to 2 decimals (`math.Round(x*100)/100`).
- Global default deposit = **300**, settle-delay = **3 days**, both from `config.PayrollConfig` (go-zero `default` tags; no YAML edit required).
- 好评 = review total (`attitude_score + technical_score`) **≥ 8**; 差评 = total **< 6**. 好评率 = 好评数 / 已评价订单数 (`nil`/`—` when denominator is 0).
- Deduction order: deposit shortfall first, then fine; net is floored at 0; leftover owed/fine carries to the enforcer's next order.
- Role gates reuse existing middleware: `requireOwner` (owner-only) and `requireAnyRole(ctx, []string{"enforcer"}, ...)`. Path params are parsed from `r.URL.Path` manually (the codebase does not use go-zero's pathvar), matching the existing `parseOrderPath` / `extractCode` helpers.
- Frontend has no unit-test harness; frontend tasks verify with `npm run typecheck` (which is `vue-tsc --noEmit`).
- Commit after every task with a `feat:`/`test:` message.

---

## File Structure

**Backend (new)**
- `server/migrations/004_create_payroll.sql` — both tables.
- `server/internal/repo/enforcer.go` — `EnforcerAccount` + `EnforcerAccountRepo`.
- `server/internal/repo/payroll.go` — `PayrollItem`, `PayrollRepo`, `ComputeDeduction` pure fn, `round2`.
- `server/internal/handler/enforcer.go` — `computeEnforcerStats` pure fn + owner handlers.
- `server/internal/handler/payroll.go` — owner payroll handlers.

**Backend (modified)**
- `server/internal/config/config.go` — add `PayrollConfig`.
- `server/internal/repo/order.go` — add `SettleWithPayroll`, `ErrNotSettleable`.
- `server/internal/handler/order.go` — add `SettleOrderHandler`; branch `UpdateOrderStatusHandler` on `settled`/`refunded`.
- `server/internal/svc/service_context.go` — construct + hold the two new repos.
- `server/cmd/gameserver/main.go` — register 6 routes.

**Frontend (new)**
- `src/api/enforcers.ts`, `src/api/payroll.ts`
- `src/pages/admin/EnforcersPanel.vue`, `src/pages/admin/PayrollPanel.vue`

**Frontend (modified)**
- `src/api/orders.ts` — add `settleOrder`.
- `src/pages/AdminPage.vue` — add `enforcers` + `payroll` tabs.
- `src/pages/admin/MyOrdersPanel.vue`, `src/pages/OrderClaimPage.vue` — 结单 button.

---

## Task 1: Payroll config + pure deduction function

**Files:**
- Modify: `server/internal/config/config.go`
- Create: `server/internal/repo/payroll.go`
- Test: `server/internal/repo/payroll_deduction_test.go`

**Interfaces:**
- Produces: `config.PayrollConfig{ DefaultDeposit float64; SettleDelayDays int }`; `repo.ComputeDeduction(gross, owed, fine float64) (depositDeduct, fineDeduct, net float64)`; `repo.round2(float64) float64`.

- [ ] **Step 1: Write the failing test**

Create `server/internal/repo/payroll_deduction_test.go`:

```go
package repo_test

import (
	"math"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestComputeDeduction(t *testing.T) {
	cases := []struct {
		name                         string
		gross, owed, fine            float64
		wantDeposit, wantFine, wantNet float64
	}{
		{"covers deposit and fine fully", 300, 100, 50, 100, 50, 150},
		{"only partial deposit", 80, 300, 40, 80, 0, 0},
		{"deposit filled, only fine", 200, 0, 50, 0, 50, 150},
		{"no owed no fine full pay", 120, 0, 0, 0, 0, 120},
		{"net floors at zero exact", 90, 60, 30, 60, 30, 0},
		{"fine larger than remaining", 100, 40, 999, 40, 60, 0},
		{"negative gross guarded", -5, 100, 100, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, f, n := repo.ComputeDeduction(c.gross, c.owed, c.fine)
			if math.Abs(d-c.wantDeposit) > 0.001 || math.Abs(f-c.wantFine) > 0.001 || math.Abs(n-c.wantNet) > 0.001 {
				t.Fatalf("ComputeDeduction(%v,%v,%v) = (%v,%v,%v), want (%v,%v,%v)",
					c.gross, c.owed, c.fine, d, f, n, c.wantDeposit, c.wantFine, c.wantNet)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/repo/ -run TestComputeDeduction`
Expected: FAIL — `undefined: repo.ComputeDeduction`.

- [ ] **Step 3: Add `PayrollConfig` to config**

In `server/internal/config/config.go`, add the field to the `Config` struct (alongside `Shop ShopConfig`):

```go
	Payroll PayrollConfig
```

and add the type near `ShopConfig`:

```go
// PayrollConfig 打手工资/押金配置。
type PayrollConfig struct {
	DefaultDeposit  float64 `json:",default=300"`
	SettleDelayDays int     `json:",default=3"`
}
```

- [ ] **Step 4: Create `payroll.go` with the pure function**

Create `server/internal/repo/payroll.go`:

```go
package repo

import "math"

// round2 rounds a monetary value to two decimals.
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ComputeDeduction splits a gross salary into deposit repayment then fine
// repayment, flooring the net at zero. owed = outstanding deposit,
// fine = pending fine. Deposit is repaid before fine.
func ComputeDeduction(gross, owed, fine float64) (depositDeduct, fineDeduct, net float64) {
	if gross < 0 {
		gross = 0
	}
	if owed < 0 {
		owed = 0
	}
	if fine < 0 {
		fine = 0
	}
	depositDeduct = math.Min(gross, owed)
	remaining := gross - depositDeduct
	fineDeduct = math.Min(remaining, fine)
	net = remaining - fineDeduct
	return round2(depositDeduct), round2(fineDeduct), round2(net)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd server && go test ./internal/repo/ -run TestComputeDeduction`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/config/config.go server/internal/repo/payroll.go server/internal/repo/payroll_deduction_test.go
git commit -m "feat(payroll): add PayrollConfig and ComputeDeduction pure fn"
```

---

## Task 2: Migration + `EnforcerAccountRepo`

**Files:**
- Create: `server/migrations/004_create_payroll.sql`
- Create: `server/internal/repo/enforcer.go`
- Test: `server/internal/repo/enforcer_test.go`

**Interfaces:**
- Produces:
  - `repo.EnforcerAccount{ UserID int64; DepositRequired sql.NullFloat64; DepositPaid float64; PendingFine float64; CreatedAt, UpdatedAt time.Time }`
  - `repo.NewEnforcerAccountRepo(db sqlx.SqlConn) *EnforcerAccountRepo`
  - `(*EnforcerAccountRepo).GetOrDefault(ctx, userID int64) (*EnforcerAccount, error)` — never returns `ErrNotFound`; a missing row yields a zero-value account (`DepositRequired` invalid, paid/fine 0).
  - `(*EnforcerAccountRepo).ListAccounts(ctx) (map[int64]*EnforcerAccount, error)`
  - `(*EnforcerAccountRepo).UpsertDepositRequired(ctx, userID int64, required *float64) error` — `nil` sets SQL NULL.
  - `(*EnforcerAccountRepo).SetPendingFine(ctx, userID int64, fine float64) error`

- [ ] **Step 1: Write the migration**

Create `server/migrations/004_create_payroll.sql`:

```sql
-- server/migrations/004_create_payroll.sql
CREATE TABLE enforcer_accounts (
  user_id          BIGINT        PRIMARY KEY,
  deposit_required DECIMAL(10,2) NULL,
  deposit_paid     DECIMAL(10,2) NOT NULL DEFAULT 0,
  pending_fine     DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE payroll_items (
  id               BIGINT        AUTO_INCREMENT PRIMARY KEY,
  order_code       VARCHAR(20)   NOT NULL,
  enforcer_id      BIGINT        NOT NULL,
  enforcer_name    VARCHAR(50)   NOT NULL,
  status           ENUM('pending','paid') NOT NULL DEFAULT 'pending',
  gross            DECIMAL(10,2) NOT NULL DEFAULT 0,
  deposit_deducted DECIMAL(10,2) NOT NULL DEFAULT 0,
  fine_deducted    DECIMAL(10,2) NOT NULL DEFAULT 0,
  net_paid         DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paid_at          TIMESTAMP     NULL,
  UNIQUE KEY uq_order_enforcer (order_code, enforcer_id),
  INDEX idx_status (status)
) DEFAULT CHARSET=utf8mb4;
```

Apply it to your test DB (the same way `002`/`003` were applied — e.g. `mysql <db> < server/migrations/004_create_payroll.sql`) before running Step 3.

- [ ] **Step 2: Write the failing test**

Create `server/internal/repo/enforcer_test.go`:

```go
package repo_test

import (
	"context"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/repo"
)

func enforcerTestConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func TestEnforcerAccount_DefaultsAndUpsert(t *testing.T) {
	conn := enforcerTestConn(t)
	r := repo.NewEnforcerAccountRepo(conn)
	ctx := context.Background()
	const uid = 990001
	_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", uid)

	// missing row -> zero-value account, no error
	acc, err := r.GetOrDefault(ctx, uid)
	if err != nil {
		t.Fatalf("GetOrDefault error = %v", err)
	}
	if acc.DepositRequired.Valid || acc.DepositPaid != 0 || acc.PendingFine != 0 {
		t.Fatalf("expected empty defaults, got %+v", acc)
	}

	// set an override + fine
	req := 500.0
	if err := r.UpsertDepositRequired(ctx, uid, &req); err != nil {
		t.Fatalf("UpsertDepositRequired error = %v", err)
	}
	if err := r.SetPendingFine(ctx, uid, 88); err != nil {
		t.Fatalf("SetPendingFine error = %v", err)
	}
	acc, _ = r.GetOrDefault(ctx, uid)
	if !acc.DepositRequired.Valid || acc.DepositRequired.Float64 != 500 || acc.PendingFine != 88 {
		t.Fatalf("unexpected account: %+v", acc)
	}

	// clear override -> NULL
	if err := r.UpsertDepositRequired(ctx, uid, nil); err != nil {
		t.Fatalf("clear override error = %v", err)
	}
	acc, _ = r.GetOrDefault(ctx, uid)
	if acc.DepositRequired.Valid {
		t.Fatalf("expected NULL deposit_required, got %v", acc.DepositRequired.Float64)
	}

	m, err := r.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("ListAccounts error = %v", err)
	}
	if _, ok := m[uid]; !ok {
		t.Fatalf("ListAccounts missing uid %d", uid)
	}
	_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", uid)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd server && go test ./internal/repo/ -run TestEnforcerAccount`
Expected: FAIL — `undefined: repo.NewEnforcerAccountRepo` (or, without `TEST_DB_DSN`, it compiles once Step 4 exists and then SKIPs — set `TEST_DB_DSN` to get a real red/green).

- [ ] **Step 4: Implement `enforcer.go`**

Create `server/internal/repo/enforcer.go`:

```go
package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// EnforcerAccount mirrors a row in enforcer_accounts (sparse: one per enforcer
// that has an override, a fine, or accrued deposit). Missing rows are treated
// as zero-value accounts by GetOrDefault.
type EnforcerAccount struct {
	UserID          int64           `db:"user_id"`
	DepositRequired sql.NullFloat64 `db:"deposit_required"`
	DepositPaid     float64         `db:"deposit_paid"`
	PendingFine     float64         `db:"pending_fine"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

const enforcerAccountCols = "user_id, deposit_required, deposit_paid, pending_fine, created_at, updated_at"

// EnforcerAccountRepo provides database access for enforcer accounts.
type EnforcerAccountRepo struct {
	db sqlx.SqlConn
}

// NewEnforcerAccountRepo creates an EnforcerAccountRepo.
func NewEnforcerAccountRepo(db sqlx.SqlConn) *EnforcerAccountRepo {
	return &EnforcerAccountRepo{db: db}
}

// GetOrDefault returns the account for userID, or a zero-value account (no row)
// without error.
func (r *EnforcerAccountRepo) GetOrDefault(ctx context.Context, userID int64) (*EnforcerAccount, error) {
	var a EnforcerAccount
	err := r.db.QueryRowCtx(ctx, &a,
		"SELECT "+enforcerAccountCols+" FROM enforcer_accounts WHERE user_id = ?", userID)
	if err == sqlx.ErrNotFound {
		return &EnforcerAccount{UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAccounts returns all accounts keyed by user_id.
func (r *EnforcerAccountRepo) ListAccounts(ctx context.Context) (map[int64]*EnforcerAccount, error) {
	var list []*EnforcerAccount
	err := r.db.QueryRowsCtx(ctx, &list, "SELECT "+enforcerAccountCols+" FROM enforcer_accounts")
	if err != nil {
		return nil, err
	}
	m := make(map[int64]*EnforcerAccount, len(list))
	for _, a := range list {
		m[a.UserID] = a
	}
	return m, nil
}

// UpsertDepositRequired sets (or clears, when required is nil) the per-enforcer
// deposit override, creating the row if needed.
func (r *EnforcerAccountRepo) UpsertDepositRequired(ctx context.Context, userID int64, required *float64) error {
	var val sql.NullFloat64
	if required != nil {
		val = sql.NullFloat64{Float64: *required, Valid: true}
	}
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO enforcer_accounts (user_id, deposit_required) VALUES (?, ?) "+
			"ON DUPLICATE KEY UPDATE deposit_required = VALUES(deposit_required)",
		userID, val)
	return err
}

// SetPendingFine sets the pending fine (absolute value), creating the row if needed.
func (r *EnforcerAccountRepo) SetPendingFine(ctx context.Context, userID int64, fine float64) error {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO enforcer_accounts (user_id, pending_fine) VALUES (?, ?) "+
			"ON DUPLICATE KEY UPDATE pending_fine = VALUES(pending_fine)",
		userID, fine)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd server && TEST_DB_DSN=<your dsn> go test ./internal/repo/ -run TestEnforcerAccount`
Expected: PASS (or SKIP if you intentionally leave `TEST_DB_DSN` unset).

- [ ] **Step 6: Commit**

```bash
git add server/migrations/004_create_payroll.sql server/internal/repo/enforcer.go server/internal/repo/enforcer_test.go
git commit -m "feat(payroll): add payroll tables migration and EnforcerAccountRepo"
```

---

## Task 3: `PayrollRepo` + `OrderRepo.SettleWithPayroll`

**Files:**
- Modify: `server/internal/repo/payroll.go`
- Modify: `server/internal/repo/order.go`
- Test: `server/internal/repo/payroll_repo_test.go`

**Interfaces:**
- Produces:
  - `repo.PayrollItem{ ID int64; OrderCode string; EnforcerID int64; EnforcerName string; Status string; Gross, DepositDeducted, FineDeducted, NetPaid float64; CreatedAt time.Time; PaidAt sql.NullTime }`
  - `repo.NewPayrollRepo(db sqlx.SqlConn) *PayrollRepo`
  - `(*PayrollRepo).ListPendingDue(ctx, delayDays int) ([]*PayrollItem, error)`
  - `(*PayrollRepo).ListPaid(ctx) ([]*PayrollItem, error)`
  - `(*PayrollRepo).Pay(ctx, id int64, defaultDeposit float64) (*PayrollItem, error)` — errors `ErrPayrollNotFound`, `ErrAlreadyPaid`.
  - `(*PayrollRepo).DeletePendingByOrder(ctx, code string) error`
  - `(*OrderRepo).SettleWithPayroll(ctx, code, expectStatus string) error` — when `expectStatus != ""`, the status update is guarded and returns `ErrNotSettleable` on 0 rows; inserts one `pending` payroll item per filled enforcer slot (`INSERT IGNORE`).
  - `repo.ErrNotSettleable`, `repo.ErrPayrollNotFound`, `repo.ErrAlreadyPaid`.
- Consumes: `Order.Earnings()`, `orderCols`, `EnforcerAccount` semantics from Tasks 1–2.

- [ ] **Step 1: Write the failing test**

Create `server/internal/repo/payroll_repo_test.go`:

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

func payrollTestConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func TestSettleWithPayroll_And_Pay(t *testing.T) {
	conn := payrollTestConn(t)
	orderRepo := repo.NewOrderRepo(conn)
	payRepo := repo.NewPayrollRepo(conn)
	accRepo := repo.NewEnforcerAccountRepo(conn)
	ctx := context.Background()

	code := "pay" + strconv.FormatInt(time.Now().UnixNano(), 10)
	const uid = 990101
	o := &repo.Order{
		Code: code, Title: "结算单", OrderType: "companion", Price: 800,
		CommissionRate: 20, BonusRate: 3, PlayerCount: 1, DispatcherID: 1, DispatcherName: "客服A",
	}
	if _, err := orderRepo.Create(ctx, o); err != nil {
		t.Fatalf("Create order: %v", err)
	}
	if err := orderRepo.AssignEnforcer(ctx, code, "a", uid, "打手甲"); err != nil {
		t.Fatalf("AssignEnforcer: %v", err)
	}
	_ = orderRepo.UpdateStatus(ctx, code, "ongoing")
	_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", uid)

	// settle -> creates one pending item; idempotent on repeat
	if err := orderRepo.SettleWithPayroll(ctx, code, "ongoing"); err != nil {
		t.Fatalf("SettleWithPayroll: %v", err)
	}
	if err := orderRepo.SettleWithPayroll(ctx, code, ""); err != nil {
		t.Fatalf("SettleWithPayroll idempotent: %v", err)
	}
	var cnt int
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM payroll_items WHERE order_code = ?", code)
	if cnt != 1 {
		t.Fatalf("expected 1 payroll item, got %d", cnt)
	}

	// find the item id
	var id int64
	_ = conn.QueryRowCtx(ctx, &id, "SELECT id FROM payroll_items WHERE order_code = ? AND enforcer_id = ?", code, uid)

	// give the enforcer an owed deposit (300 default, 0 paid) and a fine of 50
	_ = accRepo.SetPendingFine(ctx, uid, 50)

	// gross for 1 player @800, 20% commission, unqualified bonus = 640
	item, err := payRepo.Pay(ctx, id, 300)
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if item.Status != "paid" || item.DepositDeducted != 300 || item.FineDeducted != 50 || item.NetPaid != 290 {
		t.Fatalf("unexpected paid item: %+v", item)
	}
	acc, _ := accRepo.GetOrDefault(ctx, uid)
	if acc.DepositPaid != 300 || acc.PendingFine != 0 {
		t.Fatalf("account not updated: %+v", acc)
	}

	// second pay is rejected
	if _, err := payRepo.Pay(ctx, id, 300); err != repo.ErrAlreadyPaid {
		t.Fatalf("expected ErrAlreadyPaid, got %v", err)
	}

	_, _ = conn.ExecCtx(ctx, "DELETE FROM payroll_items WHERE order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM orders WHERE code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", uid)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/repo/ -run TestSettleWithPayroll`
Expected: FAIL — `undefined: repo.NewPayrollRepo` / `SettleWithPayroll`.

- [ ] **Step 3: Add `SettleWithPayroll` + `ErrNotSettleable` to `order.go`**

In `server/internal/repo/order.go`, add the error var near `ErrSlotTaken`:

```go
// ErrNotSettleable is returned when a guarded settle finds no matching order/status.
var ErrNotSettleable = errors.New("not settleable")
```

and add the method (uses the existing `sqlx` import and `orderCols`):

```go
// SettleWithPayroll sets the order to "settled" and inserts one pending
// payroll item per filled enforcer slot, in a single transaction. When
// expectStatus is non-empty the status update is guarded on it and returns
// ErrNotSettleable if no row matches. INSERT IGNORE keeps item creation
// idempotent across repeated settles.
func (r *OrderRepo) SettleWithPayroll(ctx context.Context, code, expectStatus string) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		q := "UPDATE orders SET status = 'settled' WHERE code = ?"
		args := []any{code}
		if expectStatus != "" {
			q += " AND status = ?"
			args = append(args, expectStatus)
		}
		res, err := session.ExecCtx(ctx, q, args...)
		if err != nil {
			return err
		}
		if expectStatus != "" {
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotSettleable
			}
		}
		var o Order
		if err := session.QueryRowCtx(ctx, &o, "SELECT "+orderCols+" FROM orders WHERE code = ?", code); err != nil {
			return err
		}
		insert := func(id int64, name string) error {
			_, err := session.ExecCtx(ctx,
				"INSERT IGNORE INTO payroll_items (order_code, enforcer_id, enforcer_name, status) "+
					"VALUES (?, ?, ?, 'pending')", code, id, name)
			return err
		}
		if o.EnforcerAID.Valid {
			if err := insert(o.EnforcerAID.Int64, o.EnforcerAName.String); err != nil {
				return err
			}
		}
		if o.EnforcerBID.Valid {
			if err := insert(o.EnforcerBID.Int64, o.EnforcerBName.String); err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **Step 4: Add `PayrollItem` + `PayrollRepo` to `payroll.go`**

Append to `server/internal/repo/payroll.go` (add imports `context`, `database/sql`, `errors`, `time`, and `github.com/zeromicro/go-zero/core/stores/sqlx`):

```go
// ErrPayrollNotFound is returned when a payroll item id does not exist.
var ErrPayrollNotFound = errors.New("payroll item not found")

// ErrAlreadyPaid is returned when paying an item that is not pending.
var ErrAlreadyPaid = errors.New("payroll item already paid")

// PayrollItem mirrors a row in payroll_items (one per enforcer per order).
type PayrollItem struct {
	ID              int64        `db:"id"`
	OrderCode       string       `db:"order_code"`
	EnforcerID      int64        `db:"enforcer_id"`
	EnforcerName    string       `db:"enforcer_name"`
	Status          string       `db:"status"`
	Gross           float64      `db:"gross"`
	DepositDeducted float64      `db:"deposit_deducted"`
	FineDeducted    float64      `db:"fine_deducted"`
	NetPaid         float64      `db:"net_paid"`
	CreatedAt       time.Time    `db:"created_at"`
	PaidAt          sql.NullTime `db:"paid_at"`
}

const payrollCols = "id, order_code, enforcer_id, enforcer_name, status, gross, " +
	"deposit_deducted, fine_deducted, net_paid, created_at, paid_at"

// PayrollRepo provides database access for payroll items.
type PayrollRepo struct {
	db sqlx.SqlConn
}

// NewPayrollRepo creates a PayrollRepo.
func NewPayrollRepo(db sqlx.SqlConn) *PayrollRepo {
	return &PayrollRepo{db: db}
}

// ListPendingDue returns pending items whose created_at is at least delayDays old.
func (r *PayrollRepo) ListPendingDue(ctx context.Context, delayDays int) ([]*PayrollItem, error) {
	var list []*PayrollItem
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+payrollCols+" FROM payroll_items WHERE status = 'pending' "+
			"AND created_at <= NOW() - INTERVAL ? DAY ORDER BY created_at", delayDays)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListPaid returns paid items, newest first.
func (r *PayrollRepo) ListPaid(ctx context.Context) ([]*PayrollItem, error) {
	var list []*PayrollItem
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+payrollCols+" FROM payroll_items WHERE status = 'paid' ORDER BY paid_at DESC")
	if err != nil {
		return nil, err
	}
	return list, nil
}

// DeletePendingByOrder removes unpaid payroll items for an order (used on refund).
func (r *PayrollRepo) DeletePendingByOrder(ctx context.Context, code string) error {
	_, err := r.db.ExecCtx(ctx,
		"DELETE FROM payroll_items WHERE order_code = ? AND status = 'pending'", code)
	return err
}

// Pay freezes the deduction for a pending item and updates the enforcer account,
// all in one transaction. gross is recomputed live from the order (so late
// reviews affecting the bonus are reflected). defaultDeposit is the global
// fallback when the enforcer has no deposit override.
func (r *PayrollRepo) Pay(ctx context.Context, id int64, defaultDeposit float64) (*PayrollItem, error) {
	var out *PayrollItem
	err := r.db.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var item PayrollItem
		if err := session.QueryRowCtx(ctx, &item, "SELECT "+payrollCols+" FROM payroll_items WHERE id = ?", id); err != nil {
			if err == sqlx.ErrNotFound {
				return ErrPayrollNotFound
			}
			return err
		}
		if item.Status != "pending" {
			return ErrAlreadyPaid
		}
		var o Order
		if err := session.QueryRowCtx(ctx, &o, "SELECT "+orderCols+" FROM orders WHERE code = ?", item.OrderCode); err != nil {
			return err
		}
		gross := o.Earnings()

		var acc EnforcerAccount
		required := defaultDeposit
		err := session.QueryRowCtx(ctx, &acc,
			"SELECT "+enforcerAccountCols+" FROM enforcer_accounts WHERE user_id = ?", item.EnforcerID)
		if err != nil && err != sqlx.ErrNotFound {
			return err
		}
		if err == nil && acc.DepositRequired.Valid {
			required = acc.DepositRequired.Float64
		}
		owed := required - acc.DepositPaid
		if owed < 0 {
			owed = 0
		}
		depositDeduct, fineDeduct, net := ComputeDeduction(gross, owed, acc.PendingFine)

		res, err := session.ExecCtx(ctx,
			"UPDATE payroll_items SET status='paid', gross=?, deposit_deducted=?, fine_deducted=?, "+
				"net_paid=?, paid_at=NOW() WHERE id = ? AND status = 'pending'",
			gross, depositDeduct, fineDeduct, net, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrAlreadyPaid
		}
		if _, err := session.ExecCtx(ctx,
			"INSERT INTO enforcer_accounts (user_id, deposit_paid, pending_fine) VALUES (?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE deposit_paid = deposit_paid + ?, pending_fine = pending_fine - ?",
			item.EnforcerID, depositDeduct, round2(acc.PendingFine-fineDeduct), depositDeduct, fineDeduct); err != nil {
			return err
		}
		item.Status = "paid"
		item.Gross = gross
		item.DepositDeducted = depositDeduct
		item.FineDeducted = fineDeduct
		item.NetPaid = net
		out = &item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
```

Note: the `INSERT ... VALUES (?, deposit_paid, pending_fine)` initial branch fires only when the account row is absent — in that case `acc` is zero, so the inserted `pending_fine` is `acc.PendingFine - fineDeduct` which is `0` (a missing account has no fine, so `fineDeduct` is `0`). The `ON DUPLICATE` branch handles the normal existing-row case.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd server && TEST_DB_DSN=<your dsn> go test ./internal/repo/ -run TestSettleWithPayroll`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/repo/payroll.go server/internal/repo/order.go server/internal/repo/payroll_repo_test.go
git commit -m "feat(payroll): add PayrollRepo and OrderRepo.SettleWithPayroll"
```

---

## Task 4: Enforcer stats aggregation + owner handlers + wiring

**Files:**
- Create: `server/internal/handler/enforcer.go`
- Test: `server/internal/handler/enforcer_stats_test.go`
- Modify: `server/internal/svc/service_context.go`
- Modify: `server/cmd/gameserver/main.go`

**Interfaces:**
- Produces:
  - pure fn `computeEnforcerStats(users []*repo.UserSummary, orders []*repo.Order, reviews map[string]*repo.ReviewSummary, accounts map[int64]*repo.EnforcerAccount, defaultDeposit float64) []enforcerStatView`
  - `enforcerStatView` with JSON: `userId, nickname, username, depositRequired, depositPaid, depositOwed, orderCount, goodCount, badCount, reviewedCount, goodRate (*float64), pendingFine`
  - handlers `ListEnforcersHandler`, `UpdateEnforcerDepositHandler`, `UpdateEnforcerFineHandler`
  - `ServiceContext.EnforcerAccountRepo`, `ServiceContext.PayrollRepo`
- Consumes: `repo.UserSummary` (has `Role`), `repo.Order`, `repo.ReviewSummary`, `repo.EnforcerAccount`, `ctx.Config.Payroll.DefaultDeposit`, `requireOwner`.

- [ ] **Step 1: Write the failing test**

Create `server/internal/handler/enforcer_stats_test.go`:

```go
package handler

import (
	"database/sql"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestComputeEnforcerStats(t *testing.T) {
	users := []*repo.UserSummary{
		{ID: 1, Username: "hlsaaa", Nickname: "打手甲", Role: "enforcer"},
		{ID: 2, Username: "svc", Nickname: "客服", Role: "service"},
	}
	orders := []*repo.Order{
		{Code: "o1", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}},
		{Code: "o2", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}},
		{Code: "o3", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}},
	}
	reviews := map[string]*repo.ReviewSummary{
		"o1": {OrderCode: "o1", Attitude: 5, Technical: 5}, // 10 -> good
		"o2": {OrderCode: "o2", Attitude: 2, Technical: 3}, // 5  -> bad
		// o3 unreviewed
	}
	accounts := map[int64]*repo.EnforcerAccount{
		1: {UserID: 1, DepositPaid: 120, PendingFine: 40},
	}
	got := computeEnforcerStats(users, orders, reviews, accounts, 300)
	if len(got) != 1 {
		t.Fatalf("expected 1 enforcer, got %d", len(got))
	}
	s := got[0]
	if s.OrderCount != 3 || s.GoodCount != 1 || s.BadCount != 1 || s.ReviewedCount != 2 {
		t.Fatalf("counts wrong: %+v", s)
	}
	if s.GoodRate == nil || *s.GoodRate != 0.5 {
		t.Fatalf("goodRate wrong: %+v", s.GoodRate)
	}
	if s.DepositRequired != 300 || s.DepositPaid != 120 || s.DepositOwed != 180 || s.PendingFine != 40 {
		t.Fatalf("account fields wrong: %+v", s)
	}
}

func TestComputeEnforcerStats_NoReviews(t *testing.T) {
	users := []*repo.UserSummary{{ID: 1, Nickname: "打手", Role: "enforcer"}}
	orders := []*repo.Order{{Code: "o1", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}}}
	got := computeEnforcerStats(users, orders, map[string]*repo.ReviewSummary{}, map[int64]*repo.EnforcerAccount{}, 300)
	if got[0].GoodRate != nil {
		t.Fatalf("expected nil goodRate, got %v", *got[0].GoodRate)
	}
	if got[0].DepositRequired != 300 || got[0].DepositOwed != 300 {
		t.Fatalf("default deposit wrong: %+v", got[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/handler/ -run TestComputeEnforcerStats`
Expected: FAIL — `undefined: computeEnforcerStats`.

- [ ] **Step 3: Implement `enforcer.go`**

Create `server/internal/handler/enforcer.go`:

```go
package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

type enforcerStatView struct {
	UserID          int64    `json:"userId"`
	Nickname        string   `json:"nickname"`
	Username        string   `json:"username"`
	DepositRequired float64  `json:"depositRequired"`
	DepositPaid     float64  `json:"depositPaid"`
	DepositOwed     float64  `json:"depositOwed"`
	OrderCount      int      `json:"orderCount"`
	GoodCount       int      `json:"goodCount"`
	BadCount        int      `json:"badCount"`
	ReviewedCount   int      `json:"reviewedCount"`
	GoodRate        *float64 `json:"goodRate"`
	PendingFine     float64  `json:"pendingFine"`
}

func round2f(v float64) float64 { return math.Round(v*100) / 100 }

// computeEnforcerStats aggregates per-enforcer order/review stats and account
// fields. Only users with role "enforcer" are included. goodRate is nil when
// the enforcer has no reviewed orders.
func computeEnforcerStats(users []*repo.UserSummary, orders []*repo.Order,
	reviews map[string]*repo.ReviewSummary, accounts map[int64]*repo.EnforcerAccount,
	defaultDeposit float64) []enforcerStatView {
	out := make([]enforcerStatView, 0)
	for _, u := range users {
		if u.Role != "enforcer" {
			continue
		}
		v := enforcerStatView{UserID: u.ID, Nickname: u.Nickname, Username: u.Username}
		for _, o := range orders {
			mine := (o.EnforcerAID.Valid && o.EnforcerAID.Int64 == u.ID) ||
				(o.EnforcerBID.Valid && o.EnforcerBID.Int64 == u.ID)
			if !mine {
				continue
			}
			v.OrderCount++
			if s, ok := reviews[o.Code]; ok {
				v.ReviewedCount++
				total := s.Attitude + s.Technical
				if total >= 8 {
					v.GoodCount++
				} else if total < 6 {
					v.BadCount++
				}
			}
		}
		if v.ReviewedCount > 0 {
			rate := round2f(float64(v.GoodCount) / float64(v.ReviewedCount))
			v.GoodRate = &rate
		}
		required := defaultDeposit
		if acc, ok := accounts[u.ID]; ok {
			if acc.DepositRequired.Valid {
				required = acc.DepositRequired.Float64
			}
			v.DepositPaid = acc.DepositPaid
			v.PendingFine = acc.PendingFine
		}
		v.DepositRequired = required
		owed := required - v.DepositPaid
		if owed < 0 {
			owed = 0
		}
		v.DepositOwed = round2f(owed)
		out = append(out, v)
	}
	return out
}

// parseEnforcerPath splits "/api/enforcers/{id}/{action}".
func parseEnforcerPath(path string) (int64, string) {
	rest := strings.TrimPrefix(path, "/api/enforcers/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, ""
	}
	id, _ := strconv.ParseInt(parts[0], 10, 64)
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action
}

// ListEnforcersHandler handles GET /api/enforcers (owner).
func ListEnforcersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.UserRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		orders, err := ctx.OrderRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		reviews, err := ctx.ReviewRepo.ListSummaries(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		accounts, err := ctx.EnforcerAccountRepo.ListAccounts(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := computeEnforcerStats(users, orders, reviews, accounts, ctx.Config.Payroll.DefaultDeposit)
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

type depositRequest struct {
	DepositRequired *float64 `json:"depositRequired"`
}

// UpdateEnforcerDepositHandler handles PUT /api/enforcers/:id/deposit (owner).
func UpdateEnforcerDepositHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		id, _ := parseEnforcerPath(r.URL.Path)
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		var req depositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		if req.DepositRequired != nil && *req.DepositRequired < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_deposit", nil)
			return
		}
		if err := ctx.EnforcerAccountRepo.UpsertDepositRequired(r.Context(), id, req.DepositRequired); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

type fineRequest struct {
	PendingFine float64 `json:"pendingFine"`
}

// UpdateEnforcerFineHandler handles PUT /api/enforcers/:id/fine (owner).
func UpdateEnforcerFineHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		id, _ := parseEnforcerPath(r.URL.Path)
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		var req fineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PendingFine < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_fine", nil)
			return
		}
		if err := ctx.EnforcerAccountRepo.SetPendingFine(r.Context(), id, req.PendingFine); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}
```

- [ ] **Step 4: Run the pure-fn test to verify it passes**

Run: `cd server && go test ./internal/handler/ -run TestComputeEnforcerStats`
Expected: PASS.

- [ ] **Step 5: Wire the two new repos into `ServiceContext`**

In `server/internal/svc/service_context.go`, add fields to the struct:

```go
	EnforcerAccountRepo *repo.EnforcerAccountRepo
	PayrollRepo         *repo.PayrollRepo
```

and construct them in the returned `&ServiceContext{...}` (they can reuse `conn`):

```go
		EnforcerAccountRepo: repo.NewEnforcerAccountRepo(conn),
		PayrollRepo:         repo.NewPayrollRepo(conn),
```

- [ ] **Step 6: Register the enforcer routes**

In `server/cmd/gameserver/main.go`, inside the `server.AddRoutes([]rest.Route{ ... })` slice, add:

```go
		{Method: http.MethodGet, Path: "/api/enforcers", Handler: handler.ListEnforcersHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/enforcers/:id/deposit", Handler: handler.UpdateEnforcerDepositHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/enforcers/:id/fine", Handler: handler.UpdateEnforcerFineHandler(ctx)},
```

- [ ] **Step 7: Verify the whole backend builds**

Run: `cd server && go build ./... && go test ./internal/handler/ ./internal/repo/ -run 'Compute|ReviewEffects'`
Expected: build succeeds; pure-fn tests PASS.

- [ ] **Step 8: Commit**

```bash
git add server/internal/handler/enforcer.go server/internal/handler/enforcer_stats_test.go server/internal/svc/service_context.go server/cmd/gameserver/main.go
git commit -m "feat(enforcer): add enforcer stats aggregation and owner admin endpoints"
```

---

## Task 5: Payroll handlers + enforcer settle handler + status branching + routes

**Files:**
- Create: `server/internal/handler/payroll.go`
- Modify: `server/internal/handler/order.go`
- Modify: `server/cmd/gameserver/main.go`
- Test: `server/internal/handler/payroll_view_test.go`

**Interfaces:**
- Produces:
  - pure fn `projectPayrollView(item *repo.PayrollItem, order *repo.Order, acc *repo.EnforcerAccount, defaultDeposit float64) payrollView` (pending projection)
  - `payrollView{ id, orderCode, orderTitle, enforcerName, status, gross, depositDeducted, fineDeducted, netPaid, createdAt, paidAt }`
  - handlers `ListPayrollHandler`, `PayPayrollHandler`, `SettleOrderHandler`
- Consumes: `repo.PayrollItem`, `repo.Order.Earnings()`, `repo.ComputeDeduction`, `repo.PayrollRepo`, `repo.EnforcerAccountRepo`, `repo.ErrNotSettleable`, `repo.ErrAlreadyPaid`, `repo.ErrPayrollNotFound`, `ctx.Config.Payroll`, existing `parseOrderPath`, `toOrderView`, `requireAnyRole`, `requireOwner`.

- [ ] **Step 1: Write the failing test**

Create `server/internal/handler/payroll_view_test.go`:

```go
package handler

import (
	"testing"

	"hls.dev/server/internal/repo"
)

func TestProjectPayrollView(t *testing.T) {
	item := &repo.PayrollItem{ID: 7, OrderCode: "o1", EnforcerName: "打手甲", Status: "pending"}
	order := &repo.Order{Code: "o1", Title: "上分单", Price: 800, CommissionRate: 20, BonusRate: 3, PlayerCount: 1}
	acc := &repo.EnforcerAccount{UserID: 1, DepositPaid: 0, PendingFine: 50}
	v := projectPayrollView(item, order, acc, 300)
	// gross = 640 (unqualified), deposit 300, fine 50, net 290
	if v.Gross != 640 || v.DepositDeducted != 300 || v.FineDeducted != 50 || v.NetPaid != 290 {
		t.Fatalf("unexpected projection: %+v", v)
	}
	if v.OrderTitle != "上分单" || v.ID != 7 || v.Status != "pending" {
		t.Fatalf("unexpected meta: %+v", v)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./internal/handler/ -run TestProjectPayrollView`
Expected: FAIL — `undefined: projectPayrollView`.

- [ ] **Step 3: Implement `payroll.go`**

Create `server/internal/handler/payroll.go`:

```go
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

type payrollView struct {
	ID              int64   `json:"id"`
	OrderCode       string  `json:"orderCode"`
	OrderTitle      string  `json:"orderTitle"`
	EnforcerName    string  `json:"enforcerName"`
	Status          string  `json:"status"`
	Gross           float64 `json:"gross"`
	DepositDeducted float64 `json:"depositDeducted"`
	FineDeducted    float64 `json:"fineDeducted"`
	NetPaid         float64 `json:"netPaid"`
	CreatedAt       string  `json:"createdAt"`
	PaidAt          *string `json:"paidAt"`
}

// projectPayrollView builds a pending row's live projection (gross recomputed
// from the order, deductions projected against the account's current state).
func projectPayrollView(item *repo.PayrollItem, order *repo.Order, acc *repo.EnforcerAccount, defaultDeposit float64) payrollView {
	gross := 0.0
	title := ""
	if order != nil {
		gross = order.Earnings()
		title = order.Title
	}
	required := defaultDeposit
	paid := 0.0
	fine := 0.0
	if acc != nil {
		if acc.DepositRequired.Valid {
			required = acc.DepositRequired.Float64
		}
		paid = acc.DepositPaid
		fine = acc.PendingFine
	}
	owed := required - paid
	if owed < 0 {
		owed = 0
	}
	dd, fd, net := repo.ComputeDeduction(gross, owed, fine)
	return payrollView{
		ID: item.ID, OrderCode: item.OrderCode, OrderTitle: title,
		EnforcerName: item.EnforcerName, Status: item.Status,
		Gross: gross, DepositDeducted: dd, FineDeducted: fd, NetPaid: net,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// paidPayrollView builds a paid row from its frozen snapshot.
func paidPayrollView(item *repo.PayrollItem, order *repo.Order) payrollView {
	title := ""
	if order != nil {
		title = order.Title
	}
	v := payrollView{
		ID: item.ID, OrderCode: item.OrderCode, OrderTitle: title,
		EnforcerName: item.EnforcerName, Status: item.Status,
		Gross: item.Gross, DepositDeducted: item.DepositDeducted,
		FineDeducted: item.FineDeducted, NetPaid: item.NetPaid,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if item.PaidAt.Valid {
		s := item.PaidAt.Time.Format("2006-01-02T15:04:05Z07:00")
		v.PaidAt = &s
	}
	return v
}

func ordersByCode(orders []*repo.Order) map[string]*repo.Order {
	m := make(map[string]*repo.Order, len(orders))
	for _, o := range orders {
		m[o.Code] = o
	}
	return m
}

// ListPayrollHandler handles GET /api/payroll?status=pending|paid (owner).
func ListPayrollHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "pending"
		}
		orders, err := ctx.OrderRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		byCode := ordersByCode(orders)
		views := make([]payrollView, 0)
		switch status {
		case "pending":
			items, err := ctx.PayrollRepo.ListPendingDue(r.Context(), ctx.Config.Payroll.SettleDelayDays)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			accounts, err := ctx.EnforcerAccountRepo.ListAccounts(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			for _, it := range items {
				views = append(views, projectPayrollView(it, byCode[it.OrderCode], accounts[it.EnforcerID], ctx.Config.Payroll.DefaultDeposit))
			}
		case "paid":
			items, err := ctx.PayrollRepo.ListPaid(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			for _, it := range items {
				views = append(views, paidPayrollView(it, byCode[it.OrderCode]))
			}
		default:
			writeError(w, r, http.StatusBadRequest, "invalid_status", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// parsePayrollID extracts the {id} from "/api/payroll/{id}/pay".
func parsePayrollID(path string) int64 {
	rest := strings.TrimPrefix(path, "/api/payroll/")
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	id, _ := strconv.ParseInt(rest, 10, 64)
	return id
}

// PayPayrollHandler handles POST /api/payroll/:id/pay (owner).
func PayPayrollHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		id := parsePayrollID(r.URL.Path)
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		item, err := ctx.PayrollRepo.Pay(r.Context(), id, ctx.Config.Payroll.DefaultDeposit)
		if err != nil {
			switch err {
			case repo.ErrPayrollNotFound:
				writeError(w, r, http.StatusNotFound, "payroll_not_found", nil)
			case repo.ErrAlreadyPaid:
				writeError(w, r, http.StatusConflict, "already_paid", nil)
			default:
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
			}
			return
		}
		order, _ := ctx.OrderRepo.GetByCode(r.Context(), item.OrderCode)
		httpx.OkJsonCtx(r.Context(), w, paidPayrollView(item, order))
	})
}
```

- [ ] **Step 4: Add `SettleOrderHandler` + branch `UpdateOrderStatusHandler` in `order.go`**

In `server/internal/handler/order.go`, replace the body of `UpdateOrderStatusHandler`'s success path so that `settled`/`refunded` route through the new logic. Change the block that currently does `ctx.OrderRepo.UpdateStatus(...)` to:

```go
		var err error
		switch req.Status {
		case "settled":
			err = ctx.OrderRepo.SettleWithPayroll(r.Context(), code, "")
		case "refunded":
			if err = ctx.OrderRepo.UpdateStatus(r.Context(), code, "refunded"); err == nil {
				err = ctx.PayrollRepo.DeletePendingByOrder(r.Context(), code)
			}
		default:
			err = ctx.OrderRepo.UpdateStatus(r.Context(), code, req.Status)
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
```

Then add the enforcer-facing handler at the end of `order.go`:

```go
// SettleOrderHandler handles POST /api/orders/:code/settle (enforcer).
// The caller must occupy a slot on the order and the order must be "ongoing".
func SettleOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		isMine := (o.EnforcerAID.Valid && o.EnforcerAID.Int64 == userID) ||
			(o.EnforcerBID.Valid && o.EnforcerBID.Int64 == userID)
		if !isMine {
			writeError(w, r, http.StatusForbidden, "not_your_order", nil)
			return
		}
		if err := ctx.OrderRepo.SettleWithPayroll(r.Context(), code, "ongoing"); err != nil {
			if errors.Is(err, repo.ErrNotSettleable) {
				writeError(w, r, http.StatusConflict, "order_not_settleable", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, err = ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}
```

(`order.go` already imports `errors`, `sqlx`, `repo`, `httpx` — no new imports needed.)

- [ ] **Step 5: Register the payroll + settle routes**

In `server/cmd/gameserver/main.go`, add to the routes slice:

```go
		{Method: http.MethodPost, Path: "/api/orders/:code/settle", Handler: handler.SettleOrderHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/payroll", Handler: handler.ListPayrollHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/payroll/:id/pay", Handler: handler.PayPayrollHandler(ctx)},
```

- [ ] **Step 6: Build + run all pure-fn tests**

Run: `cd server && go build ./... && go test ./internal/handler/ ./internal/repo/ -run 'Project|Compute|Earnings|ReviewEffects'`
Expected: build succeeds; PASS.

- [ ] **Step 7: Commit**

```bash
git add server/internal/handler/payroll.go server/internal/handler/payroll_view_test.go server/internal/handler/order.go server/cmd/gameserver/main.go
git commit -m "feat(payroll): add payroll list/pay endpoints and enforcer settle flow"
```

---

## Task 6: Frontend API clients + types

**Files:**
- Create: `src/api/enforcers.ts`
- Create: `src/api/payroll.ts`
- Modify: `src/api/orders.ts`

**Interfaces:**
- Produces: `listEnforcers()`, `updateEnforcerDeposit(id, depositRequired)`, `updateEnforcerFine(id, pendingFine)`, `EnforcerStat`; `listPayroll(status)`, `payPayroll(id)`, `PayrollRow`; `settleOrder(code)`.
- Consumes: `http` (authed axios) from `src/http.ts`, `OrderView` from `src/shared/orderLabels.ts`.

- [ ] **Step 1: Create `src/api/enforcers.ts`**

```ts
import http from '../http'

export interface EnforcerStat {
  userId: number
  nickname: string
  username: string
  depositRequired: number
  depositPaid: number
  depositOwed: number
  orderCount: number
  goodCount: number
  badCount: number
  reviewedCount: number
  goodRate: number | null
  pendingFine: number
}

export async function listEnforcers(): Promise<EnforcerStat[]> {
  const res = await http.get<EnforcerStat[]>('/enforcers')
  return res.data
}

// depositRequired: number sets an override; null clears it (back to global default).
export async function updateEnforcerDeposit(id: number, depositRequired: number | null): Promise<void> {
  await http.put(`/enforcers/${id}/deposit`, { depositRequired })
}

export async function updateEnforcerFine(id: number, pendingFine: number): Promise<void> {
  await http.put(`/enforcers/${id}/fine`, { pendingFine })
}
```

- [ ] **Step 2: Create `src/api/payroll.ts`**

```ts
import http from '../http'

export type PayrollStatus = 'pending' | 'paid'

export interface PayrollRow {
  id: number
  orderCode: string
  orderTitle: string
  enforcerName: string
  status: PayrollStatus
  gross: number
  depositDeducted: number
  fineDeducted: number
  netPaid: number
  createdAt: string
  paidAt: string | null
}

export async function listPayroll(status: PayrollStatus): Promise<PayrollRow[]> {
  const res = await http.get<PayrollRow[]>('/payroll', { params: { status } })
  return res.data
}

export async function payPayroll(id: number): Promise<PayrollRow> {
  const res = await http.post<PayrollRow>(`/payroll/${id}/pay`)
  return res.data
}
```

- [ ] **Step 3: Add `settleOrder` to `src/api/orders.ts`**

Append this function (mirrors `claimOrder`):

```ts
export async function settleOrder(code: string): Promise<OrderView> {
  const res = await http.post<OrderView>(`/orders/${code}/settle`)
  return res.data
}
```

- [ ] **Step 4: Typecheck**

Run: `npm run typecheck`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add src/api/enforcers.ts src/api/payroll.ts src/api/orders.ts
git commit -m "feat(fe): add enforcers/payroll api clients and settleOrder"
```

---

## Task 7: 打手管理 panel + admin tab

**Files:**
- Create: `src/pages/admin/EnforcersPanel.vue`
- Modify: `src/pages/AdminPage.vue`

**Interfaces:**
- Consumes: `listEnforcers`, `updateEnforcerDeposit`, `updateEnforcerFine`, `EnforcerStat` from Task 6.

- [ ] **Step 1: Create `src/pages/admin/EnforcersPanel.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listEnforcers, updateEnforcerDeposit, updateEnforcerFine, type EnforcerStat } from '../../api/enforcers'

const rows = ref<EnforcerStat[]>([])
const loading = ref(false)
const listError = ref('')

async function load() {
  loading.value = true
  listError.value = ''
  try {
    rows.value = await listEnforcers()
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

function money(v: number) {
  return Number.isInteger(v) ? `${v}` : v.toFixed(2)
}
function rate(v: number | null) {
  return v === null ? '—' : `${(v * 100).toFixed(0)}%`
}

async function editDeposit(row: EnforcerStat) {
  const input = window.prompt(`设置「${row.nickname}」的应缴押金（留空=用全局默认300）`, String(row.depositRequired))
  if (input === null) return
  const trimmed = input.trim()
  const value = trimmed === '' ? null : Number(trimmed)
  if (value !== null && (Number.isNaN(value) || value < 0)) {
    window.alert('请输入 ≥0 的数字，或留空清除')
    return
  }
  try {
    await updateEnforcerDeposit(row.userId, value)
    await load()
  } catch {
    window.alert('保存失败')
  }
}

async function editFine(row: EnforcerStat) {
  const input = window.prompt(`设置「${row.nickname}」的待罚款金额`, String(row.pendingFine))
  if (input === null) return
  const value = Number(input.trim())
  if (Number.isNaN(value) || value < 0) {
    window.alert('请输入 ≥0 的数字')
    return
  }
  try {
    await updateEnforcerFine(row.userId, value)
    await load()
  } catch {
    window.alert('保存失败')
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <h2 class="font-body font-bold text-xl text-hls-gold">打手管理</h2>
      <button class="hls-btn-secondary px-4 py-2.5" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading && rows.length" class="overflow-x-auto">
      <table class="w-full min-w-[56rem] border-collapse font-body text-sm">
        <thead>
          <tr class="text-left text-hls-text-dark/75">
            <th class="px-3 py-2">打手</th>
            <th class="px-3 py-2">押金应缴</th>
            <th class="px-3 py-2">已缴押金</th>
            <th class="px-3 py-2">待缴押金</th>
            <th class="px-3 py-2">接单数</th>
            <th class="px-3 py-2">好评</th>
            <th class="px-3 py-2">差评</th>
            <th class="px-3 py-2">好评率</th>
            <th class="px-3 py-2">待罚款</th>
            <th class="px-3 py-2">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.userId" class="border-t border-hls-border/20">
            <td class="px-3 py-2 font-semibold text-hls-text-dark">{{ row.nickname }}</td>
            <td class="px-3 py-2">{{ money(row.depositRequired) }}</td>
            <td class="px-3 py-2 text-emerald-700">{{ money(row.depositPaid) }}</td>
            <td class="px-3 py-2 text-amber-700">{{ money(row.depositOwed) }}</td>
            <td class="px-3 py-2">{{ row.orderCount }}</td>
            <td class="px-3 py-2 text-emerald-700">{{ row.goodCount }}</td>
            <td class="px-3 py-2 text-rose-700">{{ row.badCount }}</td>
            <td class="px-3 py-2">{{ rate(row.goodRate) }}</td>
            <td class="px-3 py-2 text-rose-700">{{ money(row.pendingFine) }}</td>
            <td class="px-3 py-2 space-x-2 whitespace-nowrap">
              <button class="hls-btn-secondary px-2 py-1 text-xs" @click="editDeposit(row)">改押金</button>
              <button class="hls-btn-secondary px-2 py-1 text-xs" @click="editFine(row)">录罚款</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <p class="font-body text-sm text-hls-text-light">暂无打手</p>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Add the tab in `src/pages/AdminPage.vue`**

Add to the `TabKey` type: `| 'enforcers'`. Import the panel:

```ts
import EnforcersPanel from './admin/EnforcersPanel.vue'
```

Add to `allTabs` (after the `users` entry):

```ts
  { key: 'enforcers', label: '打手管理', roles: ['owner'] },
```

Add to the `<main>` render chain:

```html
      <EnforcersPanel v-else-if="active === 'enforcers'" />
```

- [ ] **Step 3: Typecheck**

Run: `npm run typecheck`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/pages/admin/EnforcersPanel.vue src/pages/AdminPage.vue
git commit -m "feat(fe): add 打手管理 admin panel"
```

---

## Task 8: 工资管理 panel + admin tab

**Files:**
- Create: `src/pages/admin/PayrollPanel.vue`
- Modify: `src/pages/AdminPage.vue`

**Interfaces:**
- Consumes: `listPayroll`, `payPayroll`, `PayrollRow`, `PayrollStatus` from Task 6.

- [ ] **Step 1: Create `src/pages/admin/PayrollPanel.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listPayroll, payPayroll, type PayrollRow, type PayrollStatus } from '../../api/payroll'

const tab = ref<PayrollStatus>('pending')
const rows = ref<PayrollRow[]>([])
const loading = ref(false)
const listError = ref('')
const payingId = ref<number | null>(null)

async function load() {
  loading.value = true
  listError.value = ''
  try {
    rows.value = await listPayroll(tab.value)
  } catch {
    listError.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}
onMounted(load)

function switchTab(t: PayrollStatus) {
  if (tab.value === t) return
  tab.value = t
  load()
}

function money(v: number) {
  return Number.isInteger(v) ? `${v}` : v.toFixed(2)
}
function formatDate(value: string | null) {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  }).format(d)
}
function deductionText(row: PayrollRow) {
  const parts = [`应发 ${money(row.gross)}`]
  if (row.depositDeducted > 0) parts.push(`抵缴押金 ${money(row.depositDeducted)}`)
  if (row.fineDeducted > 0) parts.push(`扣罚款 ${money(row.fineDeducted)}`)
  parts.push(`实发 ${money(row.netPaid)}`)
  return parts.join(' · ')
}

async function pay(row: PayrollRow) {
  if (!window.confirm(`确认给「${row.enforcerName}」发放订单 ${row.orderCode} 的工资？\n${deductionText(row)}`)) return
  payingId.value = row.id
  try {
    await payPayroll(row.id)
    await load()
  } catch {
    window.alert('发放失败（可能已发放）')
  } finally {
    payingId.value = null
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between gap-4 flex-wrap">
      <h2 class="font-body font-bold text-xl text-hls-gold">工资管理</h2>
      <button class="hls-btn-secondary px-4 py-2.5" :disabled="loading" @click="load">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </div>

    <div class="flex gap-2">
      <button
        class="px-3 py-1.5 rounded font-body text-sm"
        :class="tab === 'pending' ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('pending')"
      >待发放</button>
      <button
        class="px-3 py-1.5 rounded font-body text-sm"
        :class="tab === 'paid' ? 'bg-hls-gold/20 text-hls-gold-dark font-medium' : 'text-hls-text-nav hover:bg-hls-border/20'"
        @click="switchTab('paid')"
      >已发放</button>
    </div>

    <p v-if="tab === 'pending'" class="font-body text-xs text-hls-text-light">
      提示：待发放的扣减为预测值，按当前欠缴额计算；同一打手多单时以逐单发放为准。
    </p>

    <p v-if="listError" class="font-body text-sm text-red-400">{{ listError }}</p>
    <p v-if="loading" class="font-body text-sm text-hls-text-light">加载中…</p>

    <div v-if="!loading && rows.length" class="overflow-x-auto">
      <table class="w-full min-w-[60rem] border-collapse font-body text-sm">
        <thead>
          <tr class="text-left text-hls-text-dark/75">
            <th class="px-3 py-2">订单 / 打手</th>
            <th class="px-3 py-2">应发</th>
            <th class="px-3 py-2">抵押金</th>
            <th class="px-3 py-2">扣罚款</th>
            <th class="px-3 py-2">实发</th>
            <th class="px-3 py-2">扣减说明</th>
            <th class="px-3 py-2">{{ tab === 'pending' ? '操作' : '发放时间' }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id" class="border-t border-hls-border/20 align-top">
            <td class="px-3 py-2">
              <p class="font-semibold text-hls-text-dark">{{ row.orderTitle }}</p>
              <p class="text-xs text-hls-text-light">{{ row.orderCode }} · {{ row.enforcerName }}</p>
            </td>
            <td class="px-3 py-2 text-hls-gold-dark">{{ money(row.gross) }}</td>
            <td class="px-3 py-2 text-amber-700">{{ money(row.depositDeducted) }}</td>
            <td class="px-3 py-2 text-rose-700">{{ money(row.fineDeducted) }}</td>
            <td class="px-3 py-2 font-bold text-emerald-700">{{ money(row.netPaid) }}</td>
            <td class="px-3 py-2 text-xs text-hls-text-light max-w-[16rem]">{{ deductionText(row) }}</td>
            <td class="px-3 py-2 whitespace-nowrap">
              <button
                v-if="tab === 'pending'"
                class="hls-btn-primary px-3 py-1 text-xs"
                :disabled="payingId === row.id"
                @click="pay(row)"
              >{{ payingId === row.id ? '发放中…' : '发工资' }}</button>
              <span v-else class="text-xs text-hls-text-light">{{ formatDate(row.paidAt) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="!loading" class="hls-panel px-6 py-12 text-center">
      <p class="font-body text-sm text-hls-text-light">
        {{ tab === 'pending' ? '暂无待发放工资（订单结单满3天后出现）' : '暂无已发放记录' }}
      </p>
    </div>
  </div>
</template>
```

Note: if `hls-btn-primary` is not an existing utility class, reuse `hls-btn-secondary` (grep `src/shared/styles` to confirm before Step 3).

- [ ] **Step 2: Add the tab in `src/pages/AdminPage.vue`**

Add to `TabKey`: `| 'payroll'`. Import:

```ts
import PayrollPanel from './admin/PayrollPanel.vue'
```

Add to `allTabs` (after `enforcers`):

```ts
  { key: 'payroll', label: '工资管理', roles: ['owner'] },
```

Add to the `<main>` chain:

```html
      <PayrollPanel v-else-if="active === 'payroll'" />
```

- [ ] **Step 3: Typecheck (and confirm button class)**

Run: `grep -rn "hls-btn-primary" src/shared/styles src/**/*.vue | head` — if no hits, replace `hls-btn-primary` with `hls-btn-secondary` in the panel. Then:
Run: `npm run typecheck`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/pages/admin/PayrollPanel.vue src/pages/AdminPage.vue
git commit -m "feat(fe): add 工资管理 admin panel"
```

---

## Task 9: 结单 button on both enforcer surfaces

**Files:**
- Modify: `src/pages/admin/MyOrdersPanel.vue`
- Modify: `src/pages/OrderClaimPage.vue`

**Interfaces:**
- Consumes: `settleOrder(code)` from Task 6; `useAuthStore` for role/id.

- [ ] **Step 1: Add settle to `MyOrdersPanel.vue`**

In `<script setup>`, extend the import and add state + handler:

```ts
import { myOrders, settleOrder } from '../../api/orders'
```

```ts
const settlingCode = ref<string | null>(null)
const actionError = ref('')

async function settle(code: string) {
  if (!window.confirm('确认结单？结单后将进入工资结算流程。')) return
  settlingCode.value = code
  actionError.value = ''
  try {
    await settleOrder(code)
    await load()
  } catch {
    actionError.value = '结单失败，请刷新后重试'
  } finally {
    settlingCode.value = null
  }
}
```

In the template, inside each order `<article>` header (after the status badges block), add a settle button shown only for ongoing orders:

```html
            <button
              v-if="o.status === 'ongoing'"
              class="hls-btn-secondary px-3 py-1.5 text-xs"
              :disabled="settlingCode === o.code"
              @click="settle(o.code)"
            >
              {{ settlingCode === o.code ? '结单中…' : '✅ 结单' }}
            </button>
```

And surface `actionError` near the top (after the existing `listError` paragraph):

```html
    <p v-if="actionError" class="font-body text-sm text-red-400">{{ actionError }}</p>
```

- [ ] **Step 2: Add settle to `OrderClaimPage.vue`**

Extend the import:

```ts
import { getOrder, claimOrder, settleOrder } from '../api/orders'
```

Add a computed for visibility and a handler (`order`, `meId`, `auth`, `actionError`, `load` already exist):

```ts
const settling = ref(false)
const canSettle = computed(() =>
  !!order.value && order.value.status === 'ongoing' &&
  auth.user?.role === 'enforcer' &&
  ((order.value.enforcerA?.id === meId.value) || (order.value.enforcerB?.id === meId.value)),
)

async function settle() {
  if (!window.confirm('确认结单？结单后将进入工资结算流程。')) return
  settling.value = true
  actionError.value = ''
  try {
    order.value = await settleOrder(code)
  } catch {
    actionError.value = '结单失败，请刷新后重试'
  } finally {
    settling.value = false
  }
}
```

In the template, near the existing claim action button, add:

```html
        <button
          v-if="canSettle"
          class="hls-btn-secondary w-full py-3"
          :disabled="settling"
          @click="settle"
        >
          {{ settling ? '结单中…' : '✅ 结单' }}
        </button>
```

- [ ] **Step 3: Typecheck**

Run: `npm run typecheck`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/pages/admin/MyOrdersPanel.vue src/pages/OrderClaimPage.vue
git commit -m "feat(fe): add 结单 button on order detail and my-orders"
```

---

## Final Verification

- [ ] **Backend:** `cd server && go build ./... && go vet ./... && go test ./internal/... -run 'Compute|Project|Earnings|ReviewEffects'` (and, with `TEST_DB_DSN` set, the full `go test ./internal/repo/...`).
- [ ] **Frontend:** `npm run build`.
- [ ] **Manual smoke (needs a running DB + migration `004` applied):**
  1. As enforcer, claim an order to `ongoing`, open it, click 结单 → status becomes 结单; a `payroll_items` row exists.
  2. As owner, open 打手管理 → the enforcer shows 接单数≥1; set an override deposit and a fine, confirm they persist.
  3. Temporarily set `SettleDelayDays=0` (or backdate `created_at`) → the settled order appears in 工资管理 待发放 with a projected deduction; click 发工资 → it moves to 已发放 and the enforcer's 已缴押金/待罚款 update accordingly.

---

## Notes for the Implementer

- **Deduction is defined once** in `repo.ComputeDeduction` (Task 1). Both the live projection (`projectPayrollView`, Task 5) and the frozen write (`PayrollRepo.Pay`, Task 3) call it — do not re-derive the arithmetic anywhere else.
- **Settle has one entry point** in the repo (`OrderRepo.SettleWithPayroll`). The enforcer handler passes `expectStatus="ongoing"` (guarded); the owner status-update passes `""` (force). Both create payroll items idempotently.
- **Projection overlap is expected.** Two pending items for the same enforcer both project against the *current* account; actual sequential deductions happen at pay time. The 待发放 hint text communicates this — keep it.
- **Refund cleanup only removes pending items.** Paid items are intentionally retained (money already disbursed); reversing a paid item is out of scope.

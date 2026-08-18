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

func TestRefundWithPayrollCleanup(t *testing.T) {
	conn := payrollTestConn(t)
	orderRepo := repo.NewOrderRepo(conn)
	ctx := context.Background()

	code := "refund" + strconv.FormatInt(time.Now().UnixNano(), 10)
	const uid = 990202
	o := &repo.Order{
		Code: code, Title: "退款单", OrderType: "companion", Price: 500,
		CommissionRate: 20, BonusRate: 3, PlayerCount: 1, DispatcherID: 1, DispatcherName: "客服A",
	}
	if _, err := orderRepo.Create(ctx, o); err != nil {
		t.Fatalf("Create order: %v", err)
	}
	if err := orderRepo.AssignEnforcer(ctx, code, "a", uid, "打手甲"); err != nil {
		t.Fatalf("AssignEnforcer: %v", err)
	}
	_ = orderRepo.UpdateStatus(ctx, code, "ongoing")

	// settle -> creates one pending payroll item
	if err := orderRepo.SettleWithPayroll(ctx, code, "ongoing"); err != nil {
		t.Fatalf("SettleWithPayroll: %v", err)
	}
	var cnt int
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM payroll_items WHERE order_code = ? AND status = 'pending'", code)
	if cnt != 1 {
		t.Fatalf("expected 1 pending payroll item before refund, got %d", cnt)
	}

	if err := orderRepo.RefundWithPayrollCleanup(ctx, code); err != nil {
		t.Fatalf("RefundWithPayrollCleanup: %v", err)
	}

	got, err := orderRepo.GetByCode(ctx, code)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.Status != "refunded" {
		t.Fatalf("expected status 'refunded', got %q", got.Status)
	}
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM payroll_items WHERE order_code = ? AND status = 'pending'", code)
	if cnt != 0 {
		t.Fatalf("expected 0 pending payroll items after refund cleanup, got %d", cnt)
	}

	_, _ = conn.ExecCtx(ctx, "DELETE FROM payroll_items WHERE order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM orders WHERE code = ?", code)
}

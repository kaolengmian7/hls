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

func teachingTestConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func newTeachingOrder(code string, minutes int) *repo.TeachingOrder {
	return &repo.TeachingOrder{
		Code: code, Title: "教学单", StudentID: 700001, StudentName: "学员甲",
		DispatcherID: 1, DispatcherName: "客服A", Price: 1000, CommissionRate: 20, BonusRate: 5,
		TotalMinutes: minutes,
	}
}

func cleanupTeaching(ctx context.Context, conn sqlx.SqlConn, code string) {
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_time_logs WHERE teaching_order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_scorecards WHERE teaching_order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_reviews WHERE teaching_order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM payroll_items WHERE order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_orders WHERE code = ?", code)
}

func TestTeachingCreateAndReads(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXtst" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	created, err := r.Create(ctx, newTeachingOrder(code, 360), repo.Operator{ID: 1, Name: "客服A", Role: "service"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Status != "dispatching" || created.TotalMinutes != 360 || created.RemainingMinutes != 360 {
		t.Fatalf("unexpected created: %+v", created)
	}

	var logCnt int
	_ = conn.QueryRowCtx(ctx, &logCnt, "SELECT COUNT(*) FROM teaching_time_logs WHERE teaching_order_code = ? AND reason='initial'", code)
	if logCnt != 1 {
		t.Fatalf("want 1 initial log, got %d", logCnt)
	}

	all, err := r.ListDispatching(ctx)
	if err != nil {
		t.Fatalf("ListDispatching: %v", err)
	}
	found := false
	for _, o := range all {
		if o.Code == code {
			found = true
		}
	}
	if !found {
		t.Fatalf("created order not in dispatching list")
	}
}

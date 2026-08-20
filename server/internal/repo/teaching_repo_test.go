package repo_test

import (
	"context"
	"database/sql"
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

func TestTeachingClaim(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXclm" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 360), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.Claim(ctx, code, 800123, "打手甲"); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.Status != "ongoing" || !o.TeacherID.Valid || o.TeacherID.Int64 != 800123 {
		t.Fatalf("unexpected after claim: %+v", o)
	}
	// second claim rejected
	if err := r.Claim(ctx, code, 800999, "打手乙"); err != repo.ErrTeacherTaken {
		t.Fatalf("want ErrTeacherTaken, got %v", err)
	}
}

func sqlNullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func TestTeachingAddScorecardAndGuard(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXsc" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 120), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = r.Claim(ctx, code, 800123, "打手甲")

	sc := &repo.TeachingScorecard{
		TeachingOrderCode: code, DurationMinutes: 90,
		Scores:      sqlNullStr(`{"pre_aim":8,"peek":6}`),
		Comment:     sqlNullStr("进步明显"),
		TeacherID:   800123, TeacherName: "打手甲",
	}
	if _, err := r.AddScorecard(ctx, code, sc); err != nil {
		t.Fatalf("AddScorecard: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.RemainingMinutes != 30 {
		t.Fatalf("want remaining 30, got %d", o.RemainingMinutes)
	}
	// second 90 exceeds remaining 30 -> ErrInsufficientTime, remaining unchanged
	sc2 := &repo.TeachingScorecard{TeachingOrderCode: code, DurationMinutes: 90, TeacherID: 800123, TeacherName: "打手甲"}
	if _, err := r.AddScorecard(ctx, code, sc2); err != repo.ErrInsufficientTime {
		t.Fatalf("want ErrInsufficientTime, got %v", err)
	}
	o, _ = r.GetByCode(ctx, code)
	if o.RemainingMinutes != 30 {
		t.Fatalf("remaining must stay 30 after failed add, got %d", o.RemainingMinutes)
	}
	cards, _ := r.ListScorecards(ctx, code)
	if len(cards) != 1 {
		t.Fatalf("want 1 scorecard, got %d", len(cards))
	}
}

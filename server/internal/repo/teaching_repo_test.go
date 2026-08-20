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

func TestTeachingTopUp(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXtop" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 120), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.TopUp(ctx, code, 240, 500, repo.Operator{ID: 2, Name: "店长", Role: "owner"}); err != nil {
		t.Fatalf("TopUp: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.TotalMinutes != 360 || o.RemainingMinutes != 360 || o.Price != 1500 {
		t.Fatalf("unexpected after topup: total=%d remaining=%d price=%v", o.TotalMinutes, o.RemainingMinutes, o.Price)
	}
	logs, _ := r.ListTimeLogs(ctx, code)
	if len(logs) != 2 || logs[0].Reason != "initial" || logs[1].Reason != "topup" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestTeachingSettle(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXset" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 60), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = r.Claim(ctx, code, 800123, "打手甲")

	// remaining not zero -> not settleable
	if err := r.Settle(ctx, code); err != repo.ErrTeachingNotSettleable {
		t.Fatalf("want ErrTeachingNotSettleable, got %v", err)
	}
	// consume all time
	if _, err := r.AddScorecard(ctx, code, &repo.TeachingScorecard{TeachingOrderCode: code, DurationMinutes: 60, TeacherID: 800123, TeacherName: "打手甲"}); err != nil {
		t.Fatalf("AddScorecard: %v", err)
	}
	if err := r.Settle(ctx, code); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	// idempotent-ish: second settle finds status!=ongoing -> not settleable
	if err := r.Settle(ctx, code); err != repo.ErrTeachingNotSettleable {
		t.Fatalf("second settle want ErrTeachingNotSettleable, got %v", err)
	}
	var cnt int
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM payroll_items WHERE order_code=? AND source='teaching'", code)
	if cnt != 1 {
		t.Fatalf("want 1 teaching payroll item, got %d", cnt)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.Status != "settled" {
		t.Fatalf("want settled, got %s", o.Status)
	}
}

func TestTeachingReviewUpsert(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXrev" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 60), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	rv := &repo.TeachingReview{TeachingOrderCode: code, AttitudeScore: 5, TechnicalScore: 4, ReviewerID: 700001, ReviewerName: "学员甲"}
	if err := r.UpsertReviewWithEffects(ctx, rv, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if !o.BonusQualified {
		t.Fatalf("bonus should be qualified")
	}
	// re-submit lower, unqualify
	rv2 := &repo.TeachingReview{TeachingOrderCode: code, AttitudeScore: 2, TechnicalScore: 2, ReviewerID: 700001, ReviewerName: "学员甲"}
	if err := r.UpsertReviewWithEffects(ctx, rv2, false); err != nil {
		t.Fatalf("Upsert2: %v", err)
	}
	o, _ = r.GetByCode(ctx, code)
	if o.BonusQualified {
		t.Fatalf("bonus should be unqualified after re-review")
	}
	got, _ := r.GetReview(ctx, code)
	if got.AttitudeScore != 2 {
		t.Fatalf("review not overwritten: %+v", got)
	}
	var cnt int
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM teaching_reviews WHERE teaching_order_code=?", code)
	if cnt != 1 {
		t.Fatalf("want 1 review row, got %d", cnt)
	}
}

func TestPayTeachingPayroll(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	payRepo := repo.NewPayrollRepo(conn)
	accRepo := repo.NewEnforcerAccountRepo(conn)
	ctx := context.Background()
	code := "JXpay" + strconv.FormatInt(time.Now().UnixNano(), 10)
	const teacher = 800555
	t.Cleanup(func() {
		cleanupTeaching(ctx, conn, code)
		_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", teacher)
	})

	o := newTeachingOrder(code, 60)
	o.Price = 1000
	o.CommissionRate = 20
	o.BonusRate = 5
	if _, err := r.Create(ctx, o, repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = r.Claim(ctx, code, teacher, "打手甲")
	_, _ = r.AddScorecard(ctx, code, &repo.TeachingScorecard{TeachingOrderCode: code, DurationMinutes: 60, TeacherID: teacher, TeacherName: "打手甲"})
	if err := r.Settle(ctx, code); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", teacher)

	var id int64
	_ = conn.QueryRowCtx(ctx, &id, "SELECT id FROM payroll_items WHERE order_code=? AND enforcer_id=?", code, teacher)
	// gross = 1000 * (1-0.2) = 800 (bonus unqualified). deposit default 300 owed, no fine.
	item, err := payRepo.Pay(ctx, id, 300)
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if item.Gross != 800 || item.DepositDeducted != 300 || item.NetPaid != 500 {
		t.Fatalf("unexpected paid teaching item: %+v", item)
	}
	acc, _ := accRepo.GetOrDefault(ctx, teacher)
	if acc.DepositPaid != 300 {
		t.Fatalf("deposit not updated: %+v", acc)
	}
}

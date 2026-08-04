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

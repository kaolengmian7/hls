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

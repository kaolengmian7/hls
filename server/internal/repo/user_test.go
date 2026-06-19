package repo_test

import (
	"context"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/repo"
)

func testConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func TestUpsertByWechat_NewUser(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.UpsertByWechat(ctx, "open_oid_new", "union_uid_new", "TestUser", "https://example.com/avatar.jpg")
	if err != nil {
		t.Fatalf("UpsertByWechat() error = %v", err)
	}
	if u.ID == 0 {
		t.Fatal("ID should be non-zero after insert")
	}
	if u.Nickname != "TestUser" {
		t.Fatalf("Nickname = %q, want TestUser", u.Nickname)
	}

	// cleanup
	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE open_openid = ?", "open_oid_new")
}

func TestUpsertByWechat_ExistingByUnionID(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	// Insert initial
	u1, err := r.UpsertByWechat(ctx, "open_oid_existing", "union_uid_existing", "OldName", "old_avatar")
	if err != nil {
		t.Fatalf("initial insert error = %v", err)
	}

	// Upsert again by same union_id — should update name
	u2, err := r.UpsertByWechat(ctx, "open_oid_existing2", "union_uid_existing", "NewName", "new_avatar")
	if err != nil {
		t.Fatalf("upsert error = %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("ID mismatch: got %d, want %d", u2.ID, u1.ID)
	}
	if u2.Nickname != "NewName" {
		t.Fatalf("Nickname = %q, want NewName", u2.Nickname)
	}

	// cleanup
	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u1.ID)
}

func TestFindByID_NotFound(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	_, err := r.FindByID(context.Background(), 999999999)
	if err == nil {
		t.Fatal("FindByID() with non-existent ID should return error")
	}
}

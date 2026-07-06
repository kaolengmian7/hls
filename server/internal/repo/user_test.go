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

func TestCreateWithPassword_GeneratesUsername(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.CreateWithPassword(ctx, "测试用户", "hashedpw", "customer")
	if err != nil {
		t.Fatalf("CreateWithPassword() error = %v", err)
	}
	if !u.Username.Valid || len(u.Username.String) != 7 {
		t.Fatalf("username = %q, want 7-char hls* string", u.Username.String)
	}
	if u.Username.String[:3] != "hls" {
		t.Fatalf("username prefix = %q, want 'hls'", u.Username.String[:3])
	}
	if u.Nickname != "测试用户" {
		t.Fatalf("nickname = %q, want '测试用户'", u.Nickname)
	}
	if u.Role != "customer" {
		t.Fatalf("role = %q, want 'customer'", u.Role)
	}

	// cleanup
	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestCreateWithPassword_DefaultNickname(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.CreateWithPassword(ctx, "", "hashedpw", "customer")
	if err != nil {
		t.Fatalf("CreateWithPassword() error = %v", err)
	}
	if u.Nickname != "小小花栗鼠" {
		t.Fatalf("nickname = %q, want '小小花栗鼠'", u.Nickname)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestFindByUsername(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.CreateWithPassword(ctx, "查找测试", "hashedpw", "owner")
	if err != nil {
		t.Fatalf("setup error = %v", err)
	}

	found, err := r.FindByUsername(ctx, u.Username.String)
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if found.ID != u.ID {
		t.Fatalf("ID = %d, want %d", found.ID, u.ID)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestUpdatePassword(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, _ := r.CreateWithPassword(ctx, "密码测试", "oldhash", "customer")

	if err := r.UpdatePassword(ctx, u.ID, "newhash"); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	updated, _ := r.FindByID(ctx, u.ID)
	if updated.PasswordHash.String != "newhash" {
		t.Fatalf("password_hash = %q, want 'newhash'", updated.PasswordHash.String)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestUpdateNickname(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, _ := r.CreateWithPassword(ctx, "昵称测试", "hash", "customer")

	if err := r.UpdateNickname(ctx, u.ID, "新昵称"); err != nil {
		t.Fatalf("UpdateNickname() error = %v", err)
	}
	updated, _ := r.FindByID(ctx, u.ID)
	if updated.Nickname != "新昵称" {
		t.Fatalf("nickname = %q, want '新昵称'", updated.Nickname)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestListAll(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, _ := r.CreateWithPassword(ctx, "列表测试", "hash", "service")

	users, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	found := false
	for _, s := range users {
		if s.ID == u.ID {
			found = true
			if s.Role != "service" {
				t.Fatalf("role = %q, want 'service'", s.Role)
			}
		}
	}
	if !found {
		t.Fatal("created user not found in ListAll()")
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

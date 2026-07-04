package repo_test

import (
	"context"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/repo"
)

func testShopConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func TestShopRepo_RoomOps(t *testing.T) {
	conn := testShopConn(t)
	r := repo.NewShopRepo(conn)
	ctx := context.Background()
	code := "T00001"

	if err := r.CreateRoom(ctx, code, "p_host", 100); err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	t.Cleanup(func() { _ = r.CloseRoom(ctx, code) })

	if err := r.UpdateRoomStatus(ctx, code, "playing"); err != nil {
		t.Fatalf("UpdateRoomStatus: %v", err)
	}
	if err := r.UpdateGuaranteedAmount(ctx, code, 150); err != nil {
		t.Fatalf("UpdateGuaranteedAmount: %v", err)
	}

	rows, err := r.LoadActiveRooms(ctx)
	if err != nil {
		t.Fatalf("LoadActiveRooms: %v", err)
	}
	found := false
	for _, row := range rows {
		if row.Code == code {
			found = true
			if row.GuaranteedAmount != 150 {
				t.Errorf("guaranteed_amount got %d, want 150", row.GuaranteedAmount)
			}
		}
	}
	if !found {
		t.Errorf("room %s not in LoadActiveRooms result", code)
	}
}

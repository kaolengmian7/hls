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
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func testShopRepo(t *testing.T) *repo.ShopRepo {
	t.Helper()
	return repo.NewShopRepo(testShopConn(t))
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

func TestShopRepo_PlayerOps(t *testing.T) {
	r := testShopRepo(t)
	ctx := context.Background()
	code := "T00002"
	_ = r.CreateRoom(ctx, code, "p_host", 0)
	t.Cleanup(func() { _ = r.CloseRoom(ctx, code) })

	if err := r.AddPlayer(ctx, code, 1, "p_host", true); err != nil {
		t.Fatalf("AddPlayer host: %v", err)
	}
	if err := r.AddPlayer(ctx, code, 2, "p_guest", false); err != nil {
		t.Fatalf("AddPlayer guest: %v", err)
	}
	if err := r.UpdatePlayerState(ctx, code, "p_guest", "boss", 10, true); err != nil {
		t.Fatalf("UpdatePlayerState: %v", err)
	}

	players, err := r.LoadRoomPlayers(ctx, code)
	if err != nil {
		t.Fatalf("LoadRoomPlayers: %v", err)
	}
	if len(players) != 2 {
		t.Errorf("want 2 players, got %d", len(players))
	}

	got, err := r.FindActiveRoomForUser(ctx, 1)
	if err != nil {
		t.Fatalf("FindActiveRoomForUser: %v", err)
	}
	if got != code {
		t.Errorf("FindActiveRoomForUser got %q, want %q", got, code)
	}

	if err := r.RemovePlayer(ctx, code, "p_guest"); err != nil {
		t.Fatalf("RemovePlayer: %v", err)
	}
	got2, _ := r.FindActiveRoomForUser(ctx, 2)
	if got2 != "" {
		t.Errorf("removed player still active in %q", got2)
	}
}

func TestShopRepo_RecordOps(t *testing.T) {
	r := testShopRepo(t)
	ctx := context.Background()
	code := "T00003"
	_ = r.CreateRoom(ctx, code, "p_host", 0)
	t.Cleanup(func() { _ = r.CloseRoom(ctx, code) })

	recs := []repo.ShopLotteryRow{{
		RoomCode: code, PlayerID: "p_host",
		ItemID: "item_test", ItemName: "测试奖品", Rarity: "R",
		ItemValue: 5, GuaranteeBefore: 0, GuaranteeAfter: 5,
	}}
	if err := r.InsertLotteryRecords(ctx, recs); err != nil {
		t.Fatalf("InsertLotteryRecords: %v", err)
	}
	lottery, err := r.LoadLotteryRecords(ctx, code)
	if err != nil || len(lottery) != 1 || lottery[0].ItemID != "item_test" {
		t.Fatalf("LoadLotteryRecords: err=%v rows=%d", err, len(lottery))
	}

	sr := repo.ShopSettlementRow{
		ID: "settle_test", RoomCode: code, Outcome: "EVACUATED",
		GuaranteeDelta: -10, GuaranteeBefore: 100, GuaranteeAfter: 90,
		PerPlayer: `[]`,
	}
	if err := r.InsertSettlement(ctx, sr); err != nil {
		t.Fatalf("InsertSettlement: %v", err)
	}
	settlements, err := r.LoadSettlements(ctx, code)
	if err != nil || len(settlements) != 1 || settlements[0].ID != "settle_test" {
		t.Fatalf("LoadSettlements: err=%v rows=%d", err, len(settlements))
	}
}

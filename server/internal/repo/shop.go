package repo

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// ShopRoomRow mirrors shop_rooms.
type ShopRoomRow struct {
	Code             string    `db:"code"`
	Status           string    `db:"status"`
	HostPlayerID     string    `db:"host_player_id"`
	GuaranteedAmount int       `db:"guaranteed_amount"`
	CreatedAt        time.Time `db:"created_at"`
}

// ShopRoomPlayerRow mirrors shop_room_players (active rows only).
type ShopRoomPlayerRow struct {
	RoomCode string `db:"room_code"`
	UserID   int64  `db:"user_id"`
	PlayerID string `db:"player_id"`
	Role     string `db:"role"`
	Coin     int    `db:"coin"`
	IsHost   bool   `db:"is_host"`
	Ready    bool   `db:"ready"`
}

// ShopLotteryRow mirrors shop_lottery_records.
type ShopLotteryRow struct {
	RoomCode        string    `db:"room_code"`
	PlayerID        string    `db:"player_id"`
	ItemID          string    `db:"item_id"`
	ItemName        string    `db:"item_name"`
	Rarity          string    `db:"rarity"`
	ItemValue       int       `db:"item_value"`
	GuaranteeBefore int       `db:"guarantee_before"`
	GuaranteeAfter  int       `db:"guarantee_after"`
	CreatedAt       time.Time `db:"created_at"`
}

// ShopSettlementRow mirrors shop_settlement_records.
type ShopSettlementRow struct {
	ID              string    `db:"id"`
	RoomCode        string    `db:"room_code"`
	Outcome         string    `db:"outcome"`
	GuaranteeDelta  int       `db:"guarantee_delta"`
	GuaranteeBefore int       `db:"guarantee_before"`
	GuaranteeAfter  int       `db:"guarantee_after"`
	PerPlayer       string    `db:"per_player"` // JSON
	CreatedAt       time.Time `db:"created_at"`
}

// ShopRepo provides access to all shop_* tables.
type ShopRepo struct {
	db sqlx.SqlConn
}

// NewShopRepo creates a ShopRepo.
func NewShopRepo(db sqlx.SqlConn) *ShopRepo {
	return &ShopRepo{db: db}
}

// CreateRoom creates a new shop room.
func (r *ShopRepo) CreateRoom(ctx context.Context, code, hostPlayerID string, guaranteedAmount int) error {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO shop_rooms (code, host_player_id, guaranteed_amount) VALUES (?, ?, ?)",
		code, hostPlayerID, guaranteedAmount)
	return err
}

// UpdateRoomStatus updates the status of a room.
func (r *ShopRepo) UpdateRoomStatus(ctx context.Context, code, status string) error {
	_, err := r.db.ExecCtx(ctx, "UPDATE shop_rooms SET status=? WHERE code=?", status, code)
	return err
}

// UpdateGuaranteedAmount updates the guaranteed amount for a room.
func (r *ShopRepo) UpdateGuaranteedAmount(ctx context.Context, code string, amount int) error {
	_, err := r.db.ExecCtx(ctx, "UPDATE shop_rooms SET guaranteed_amount=? WHERE code=?", amount, code)
	return err
}

// CloseRoom marks a room closed and stamps left_at for all remaining players.
func (r *ShopRepo) CloseRoom(ctx context.Context, code string) error {
	if _, err := r.db.ExecCtx(ctx,
		"UPDATE shop_rooms SET status='closed' WHERE code=?", code); err != nil {
		return err
	}
	_, err := r.db.ExecCtx(ctx,
		"UPDATE shop_room_players SET left_at=NOW() WHERE room_code=? AND left_at IS NULL", code)
	return err
}

// LoadActiveRooms returns all waiting/playing rooms for restart recovery.
func (r *ShopRepo) LoadActiveRooms(ctx context.Context) ([]ShopRoomRow, error) {
	var rows []ShopRoomRow
	err := r.db.QueryRowsCtx(ctx, &rows,
		"SELECT code, status, host_player_id, guaranteed_amount, created_at FROM shop_rooms WHERE status IN ('waiting','playing')")
	return rows, err
}

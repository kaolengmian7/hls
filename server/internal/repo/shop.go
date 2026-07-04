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

// AddPlayer adds a player to a room.
func (r *ShopRepo) AddPlayer(ctx context.Context, roomCode string, userID int64, playerID string, isHost bool) error {
	host := 0
	if isHost {
		host = 1
	}
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO shop_room_players (room_code, user_id, player_id, is_host) VALUES (?, ?, ?, ?)",
		roomCode, userID, playerID, host)
	return err
}

// UpdatePlayerState updates player role, coin, and ready state.
func (r *ShopRepo) UpdatePlayerState(ctx context.Context, roomCode, playerID, role string, coin int, ready bool) error {
	readyInt := 0
	if ready {
		readyInt = 1
	}
	_, err := r.db.ExecCtx(ctx,
		"UPDATE shop_room_players SET role=?, coin=?, ready=? WHERE room_code=? AND player_id=?",
		role, coin, readyInt, roomCode, playerID)
	return err
}

// RemovePlayer marks a player as left.
func (r *ShopRepo) RemovePlayer(ctx context.Context, roomCode, playerID string) error {
	_, err := r.db.ExecCtx(ctx,
		"UPDATE shop_room_players SET left_at=NOW() WHERE room_code=? AND player_id=? AND left_at IS NULL",
		roomCode, playerID)
	return err
}

// FindActiveRoomForUser returns the room code of the user's current waiting/playing room, or "" if none.
func (r *ShopRepo) FindActiveRoomForUser(ctx context.Context, userID int64) (string, error) {
	var code string
	err := r.db.QueryRowCtx(ctx, &code,
		`SELECT r.code FROM shop_room_players p
		 JOIN shop_rooms r ON r.code = p.room_code
		 WHERE p.user_id = ? AND r.status IN ('waiting','playing') AND p.left_at IS NULL
		 LIMIT 1`, userID)
	if err == sqlx.ErrNotFound {
		return "", nil
	}
	return code, err
}

// LoadRoomPlayers returns all active players in a room.
func (r *ShopRepo) LoadRoomPlayers(ctx context.Context, roomCode string) ([]ShopRoomPlayerRow, error) {
	var rows []ShopRoomPlayerRow
	err := r.db.QueryRowsCtx(ctx, &rows,
		"SELECT room_code, user_id, player_id, role, coin, is_host, ready FROM shop_room_players WHERE room_code=? AND left_at IS NULL",
		roomCode)
	return rows, err
}

// InsertLotteryRecords inserts lottery records for a room.
func (r *ShopRepo) InsertLotteryRecords(ctx context.Context, records []ShopLotteryRow) error {
	for _, rec := range records {
		_, err := r.db.ExecCtx(ctx,
			`INSERT INTO shop_lottery_records
			 (room_code, player_id, item_id, item_name, rarity, item_value, guarantee_before, guarantee_after)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			rec.RoomCode, rec.PlayerID, rec.ItemID, rec.ItemName,
			rec.Rarity, rec.ItemValue, rec.GuaranteeBefore, rec.GuaranteeAfter)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadLotteryRecords loads all lottery records for a room.
func (r *ShopRepo) LoadLotteryRecords(ctx context.Context, roomCode string) ([]ShopLotteryRow, error) {
	var rows []ShopLotteryRow
	err := r.db.QueryRowsCtx(ctx, &rows,
		`SELECT room_code, player_id, item_id, item_name, rarity, item_value,
		        guarantee_before, guarantee_after, created_at
		 FROM shop_lottery_records WHERE room_code=? ORDER BY id DESC`, roomCode)
	return rows, err
}

// InsertSettlement inserts a settlement record.
func (r *ShopRepo) InsertSettlement(ctx context.Context, rec ShopSettlementRow) error {
	_, err := r.db.ExecCtx(ctx,
		`INSERT INTO shop_settlement_records
		 (id, room_code, outcome, guarantee_delta, guarantee_before, guarantee_after, per_player)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rec.ID, rec.RoomCode, rec.Outcome,
		rec.GuaranteeDelta, rec.GuaranteeBefore, rec.GuaranteeAfter, rec.PerPlayer)
	return err
}

// LoadSettlements loads all settlement records for a room.
func (r *ShopRepo) LoadSettlements(ctx context.Context, roomCode string) ([]ShopSettlementRow, error) {
	var rows []ShopSettlementRow
	err := r.db.QueryRowsCtx(ctx, &rows,
		`SELECT id, room_code, outcome, guarantee_delta, guarantee_before, guarantee_after, per_player, created_at
		 FROM shop_settlement_records WHERE room_code=? ORDER BY created_at DESC`, roomCode)
	return rows, err
}

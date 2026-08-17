package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Review mirrors a row in the reviews table (one per order_code).
type Review struct {
	ID               int64          `db:"id"`
	OrderCode        string         `db:"order_code"`
	AttitudeScore    int            `db:"attitude_score"`
	TechnicalScore   int            `db:"technical_score"`
	BlameBoss        bool           `db:"blame_boss"`
	Late             bool           `db:"late"`
	TempUnavailable  bool           `db:"temp_unavailable"`
	SlowReply        bool           `db:"slow_reply"`
	NoBackpack       bool           `db:"no_backpack"`
	NoChannelCard    bool           `db:"no_channel_card"`
	NoSupplyCard     bool           `db:"no_supply_card"`
	NoEmotionalValue bool           `db:"no_emotional_value"`
	GrabContainer    bool           `db:"grab_container"`
	Comment          sql.NullString `db:"comment"`
	ReviewerID       int64          `db:"reviewer_id"`
	ReviewerName     string         `db:"reviewer_name"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

// ReviewSummary is the compact per-order projection used in order lists.
type ReviewSummary struct {
	OrderCode string `db:"order_code"`
	Attitude  int    `db:"attitude_score"`
	Technical int    `db:"technical_score"`
}

const reviewCols = "id, order_code, attitude_score, technical_score, blame_boss, late, " +
	"temp_unavailable, slow_reply, no_backpack, no_channel_card, no_supply_card, " +
	"no_emotional_value, grab_container, comment, reviewer_id, reviewer_name, created_at, updated_at"

// ReviewRepo provides database access for reviews.
type ReviewRepo struct {
	db sqlx.SqlConn
}

// NewReviewRepo creates a ReviewRepo.
func NewReviewRepo(db sqlx.SqlConn) *ReviewRepo {
	return &ReviewRepo{db: db}
}

// UpsertWithEffects inserts or overwrites the single review for rv.OrderCode
// and applies the review-derived bonus_qualified/complaint_status update to
// the corresponding order, all within a single DB transaction so a crash
// between the two writes can never leave the order stale relative to the review.
func (r *ReviewRepo) UpsertWithEffects(ctx context.Context, rv *Review, bonusQualified bool, complaint string) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if _, err := session.ExecCtx(ctx,
			"INSERT INTO reviews (order_code, attitude_score, technical_score, blame_boss, late, "+
				"temp_unavailable, slow_reply, no_backpack, no_channel_card, no_supply_card, "+
				"no_emotional_value, grab_container, comment, reviewer_id, reviewer_name) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE attitude_score=VALUES(attitude_score), "+
				"technical_score=VALUES(technical_score), blame_boss=VALUES(blame_boss), "+
				"late=VALUES(late), temp_unavailable=VALUES(temp_unavailable), "+
				"slow_reply=VALUES(slow_reply), no_backpack=VALUES(no_backpack), "+
				"no_channel_card=VALUES(no_channel_card), no_supply_card=VALUES(no_supply_card), "+
				"no_emotional_value=VALUES(no_emotional_value), grab_container=VALUES(grab_container), "+
				"comment=VALUES(comment), reviewer_id=VALUES(reviewer_id), reviewer_name=VALUES(reviewer_name)",
			rv.OrderCode, rv.AttitudeScore, rv.TechnicalScore, rv.BlameBoss, rv.Late,
			rv.TempUnavailable, rv.SlowReply, rv.NoBackpack, rv.NoChannelCard, rv.NoSupplyCard,
			rv.NoEmotionalValue, rv.GrabContainer, rv.Comment, rv.ReviewerID, rv.ReviewerName); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx,
			"UPDATE orders SET bonus_qualified = ?, complaint_status = ? WHERE code = ?",
			bonusQualified, complaint, rv.OrderCode)
		return err
	})
}

// GetByOrderCode returns the review for code, or sqlx.ErrNotFound.
func (r *ReviewRepo) GetByOrderCode(ctx context.Context, code string) (*Review, error) {
	var rv Review
	err := r.db.QueryRowCtx(ctx, &rv, "SELECT "+reviewCols+" FROM reviews WHERE order_code = ?", code)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// ListSummaries returns compact review projections keyed by order_code.
func (r *ReviewRepo) ListSummaries(ctx context.Context) (map[string]*ReviewSummary, error) {
	var list []*ReviewSummary
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT order_code, attitude_score, technical_score FROM reviews")
	if err != nil {
		return nil, err
	}
	m := make(map[string]*ReviewSummary, len(list))
	for _, s := range list {
		m[s.OrderCode] = s
	}
	return m, nil
}

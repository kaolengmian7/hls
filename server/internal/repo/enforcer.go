package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// EnforcerAccount mirrors a row in enforcer_accounts (sparse: one per enforcer
// that has an override, a fine, or accrued deposit). Missing rows are treated
// as zero-value accounts by GetOrDefault.
type EnforcerAccount struct {
	UserID          int64           `db:"user_id"`
	DepositRequired sql.NullFloat64 `db:"deposit_required"`
	DepositPaid     float64         `db:"deposit_paid"`
	PendingFine     float64         `db:"pending_fine"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

const enforcerAccountCols = "user_id, deposit_required, deposit_paid, pending_fine, created_at, updated_at"

// EnforcerAccountRepo provides database access for enforcer accounts.
type EnforcerAccountRepo struct {
	db sqlx.SqlConn
}

// NewEnforcerAccountRepo creates an EnforcerAccountRepo.
func NewEnforcerAccountRepo(db sqlx.SqlConn) *EnforcerAccountRepo {
	return &EnforcerAccountRepo{db: db}
}

// GetOrDefault returns the account for userID, or a zero-value account (no row)
// without error.
func (r *EnforcerAccountRepo) GetOrDefault(ctx context.Context, userID int64) (*EnforcerAccount, error) {
	var a EnforcerAccount
	err := r.db.QueryRowCtx(ctx, &a,
		"SELECT "+enforcerAccountCols+" FROM enforcer_accounts WHERE user_id = ?", userID)
	if err == sqlx.ErrNotFound {
		return &EnforcerAccount{UserID: userID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAccounts returns all accounts keyed by user_id.
func (r *EnforcerAccountRepo) ListAccounts(ctx context.Context) (map[int64]*EnforcerAccount, error) {
	var list []*EnforcerAccount
	err := r.db.QueryRowsCtx(ctx, &list, "SELECT "+enforcerAccountCols+" FROM enforcer_accounts")
	if err != nil {
		return nil, err
	}
	m := make(map[int64]*EnforcerAccount, len(list))
	for _, a := range list {
		m[a.UserID] = a
	}
	return m, nil
}

// UpsertDepositRequired sets (or clears, when required is nil) the per-enforcer
// deposit override, creating the row if needed.
func (r *EnforcerAccountRepo) UpsertDepositRequired(ctx context.Context, userID int64, required *float64) error {
	var val sql.NullFloat64
	if required != nil {
		val = sql.NullFloat64{Float64: *required, Valid: true}
	}
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO enforcer_accounts (user_id, deposit_required) VALUES (?, ?) "+
			"ON DUPLICATE KEY UPDATE deposit_required = VALUES(deposit_required)",
		userID, val)
	return err
}

// SetPendingFine sets the pending fine (absolute value), creating the row if needed.
func (r *EnforcerAccountRepo) SetPendingFine(ctx context.Context, userID int64, fine float64) error {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO enforcer_accounts (user_id, pending_fine) VALUES (?, ?) "+
			"ON DUPLICATE KEY UPDATE pending_fine = VALUES(pending_fine)",
		userID, fine)
	return err
}

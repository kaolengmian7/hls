package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// AuthState holds a pending or resolved QR login state.
type AuthState struct {
	State     string         `db:"state"`
	JWT       sql.NullString `db:"jwt"`
	ExpiresAt time.Time      `db:"expires_at"`
}

// AuthStateRepo provides database access for QR login states.
type AuthStateRepo struct {
	db sqlx.SqlConn
}

// NewAuthStateRepo creates an AuthStateRepo.
func NewAuthStateRepo(db sqlx.SqlConn) *AuthStateRepo {
	return &AuthStateRepo{db: db}
}

// Create inserts a new pending auth state expiring in 5 minutes.
func (r *AuthStateRepo) Create(ctx context.Context, state string) error {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO auth_states (state, expires_at) VALUES (?, DATE_ADD(NOW(), INTERVAL 5 MINUTE))",
		state)
	return err
}

// SetJWT stores the signed JWT for a state after successful WeChat callback.
func (r *AuthStateRepo) SetJWT(ctx context.Context, state, jwtStr string) error {
	_, err := r.db.ExecCtx(ctx,
		"UPDATE auth_states SET jwt = ? WHERE state = ? AND expires_at > NOW()",
		jwtStr, state)
	return err
}

// Get returns the auth state, cleaning up expired records first.
// Returns sqlx.ErrNotFound if the state does not exist or has expired.
func (r *AuthStateRepo) Get(ctx context.Context, state string) (*AuthState, error) {
	_, _ = r.db.ExecCtx(ctx, "DELETE FROM auth_states WHERE expires_at < NOW()")
	var as AuthState
	err := r.db.QueryRowCtx(ctx, &as,
		"SELECT state, jwt, expires_at FROM auth_states WHERE state = ?", state)
	if err != nil {
		return nil, err
	}
	return &as, nil
}

// Delete removes an auth state after it has been consumed.
func (r *AuthStateRepo) Delete(ctx context.Context, state string) error {
	_, err := r.db.ExecCtx(ctx, "DELETE FROM auth_states WHERE state = ?", state)
	return err
}

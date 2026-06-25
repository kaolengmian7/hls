package repo

import (
	"context"
	"crypto/rand"
	"database/sql"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// User represents a registered user.
type User struct {
	ID         int64          `db:"id"`
	OpenOpenID sql.NullString `db:"open_openid"`
	MpOpenID   sql.NullString `db:"mp_openid"`
	UnionID    sql.NullString `db:"union_id"`
	Phone      sql.NullString `db:"phone"`
	Nickname   string         `db:"nickname"`
	AvatarURL  string         `db:"avatar_url"`
}

const userCols = "id, open_openid, mp_openid, union_id, phone, nickname, avatar_url"

// UserRepo provides database access for users.
type UserRepo struct {
	db sqlx.SqlConn
}

// NewUserRepo creates a UserRepo backed by the given connection.
func NewUserRepo(db sqlx.SqlConn) *UserRepo {
	return &UserRepo{db: db}
}

// UpsertByWechat finds or creates a user by openid/unionid and updates their profile.
func (r *UserRepo) UpsertByWechat(ctx context.Context, openOpenID, unionID, nickname, avatarURL string) (*User, error) {
	if unionID != "" {
		u, err := r.findBy(ctx, "union_id", unionID)
		if err == nil {
			r.updateProfile(ctx, u.ID, nickname, avatarURL)
			u.Nickname = nickname
			u.AvatarURL = avatarURL
			return u, nil
		}
		if err != sqlx.ErrNotFound {
			return nil, err
		}
	}

	u, err := r.findBy(ctx, "open_openid", openOpenID)
	if err == nil {
		r.updateProfile(ctx, u.ID, nickname, avatarURL)
		if unionID != "" {
			_, _ = r.db.ExecCtx(ctx,
				"UPDATE users SET union_id = COALESCE(union_id, ?), updated_at = NOW() WHERE id = ?",
				unionID, u.ID)
		}
		u.Nickname = nickname
		u.AvatarURL = avatarURL
		return u, nil
	}
	if err != sqlx.ErrNotFound {
		return nil, err
	}

	return r.insert(ctx, openOpenID, unionID, nickname, avatarURL)
}

// FindByID returns the user with the given ID.
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	return r.findBy(ctx, "id", id)
}

func (r *UserRepo) findBy(ctx context.Context, col string, val interface{}) (*User, error) {
	var u User
	err := r.db.QueryRowCtx(ctx, &u,
		"SELECT "+userCols+" FROM users WHERE "+col+" = ?", val)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) updateProfile(ctx context.Context, id int64, nickname, avatarURL string) {
	_, _ = r.db.ExecCtx(ctx,
		"UPDATE users SET nickname = ?, avatar_url = ?, updated_at = NOW() WHERE id = ?",
		nickname, avatarURL, id)
}

func (r *UserRepo) insert(ctx context.Context, openOpenID, unionID, nickname, avatarURL string) (*User, error) {
	result, err := r.db.ExecCtx(ctx,
		"INSERT INTO users (open_openid, union_id, nickname, avatar_url) VALUES (?, ?, ?, ?)",
		nullStr(openOpenID), nullStr(unionID), nickname, avatarURL)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// UpsertByPhone finds an existing user by phone or creates a new one.
func (r *UserRepo) UpsertByPhone(ctx context.Context, phone string) (*User, error) {
	u, err := r.findBy(ctx, "phone", phone)
	if err == nil {
		return u, nil
	}
	if err != sqlx.ErrNotFound {
		return nil, err
	}
	nickname := "小小花栗鼠" + randSuffix()
	return r.insertByPhone(ctx, phone, nickname)
}

func (r *UserRepo) insertByPhone(ctx context.Context, phone, nickname string) (*User, error) {
	result, err := r.db.ExecCtx(ctx,
		"INSERT INTO users (phone, nickname, avatar_url) VALUES (?, ?, '')",
		phone, nickname)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func randSuffix() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	out := make([]byte, 4)
	for i, c := range b {
		out[i] = chars[int(c)%len(chars)]
	}
	return string(out)
}

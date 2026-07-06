package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"golang.org/x/crypto/bcrypt"

	"hls.dev/server/internal/svc"
)

type updateNicknameRequest struct {
	Nickname string `json:"nickname"`
}

// UpdateNicknameHandler handles PUT /api/user/nickname.
func UpdateNicknameHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		var req updateNicknameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Nickname = strings.TrimSpace(req.Nickname)
		if req.Nickname == "" {
			writeError(w, r, http.StatusBadRequest, "nickname_required", nil)
			return
		}
		if err := ctx.UserRepo.UpdateNickname(r.Context(), userID, req.Nickname); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	}
}

type updatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// UpdatePasswordHandler handles PUT /api/user/password.
func UpdatePasswordHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		var req updatePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		if req.OldPassword == "" || req.NewPassword == "" {
			writeError(w, r, http.StatusBadRequest, "missing_fields", nil)
			return
		}
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if errors.Is(err, sqlx.ErrNotFound) {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if !user.PasswordHash.Valid {
			writeError(w, r, http.StatusBadRequest, "no_password_set", nil)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(req.OldPassword)); err != nil {
			writeError(w, r, http.StatusUnauthorized, "wrong_password", nil)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "hash_error", err)
			return
		}
		if err := ctx.UserRepo.UpdatePassword(r.Context(), userID, string(hash)); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	}
}

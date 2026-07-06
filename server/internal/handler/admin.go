package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
	"golang.org/x/crypto/bcrypt"

	"hls.dev/server/internal/svc"
)

const defaultPassword = "123456"
const defaultNickname = "小小花栗鼠"

var validRoles = map[string]bool{
	"owner": true, "service": true, "enforcer": true, "customer": true,
}

// ListUsersHandler handles GET /api/admin/users.
func ListUsersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.UserRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, users)
	})
}

type createUserRequest struct {
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// CreateUserHandler handles POST /api/admin/users.
func CreateUserHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Nickname = strings.TrimSpace(req.Nickname)
		if !validRoles[req.Role] {
			writeError(w, r, http.StatusBadRequest, "invalid_role", nil)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "hash_error", err)
			return
		}
		user, err := ctx.UserRepo.CreateWithPassword(r.Context(), req.Nickname, string(hash), req.Role)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"id":       user.ID,
			"username": user.Username.String,
			"nickname": user.Nickname,
			"role":     user.Role,
		})
	})
}

// ResetPasswordHandler handles PUT /api/admin/users/:id/password.
func ResetPasswordHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		idStr := extractCode(r.URL.Path, "/api/admin/users/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "hash_error", err)
			return
		}
		if err := ctx.UserRepo.UpdatePassword(r.Context(), id, string(hash)); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

// ResetNicknameHandler handles PUT /api/admin/users/:id/nickname.
func ResetNicknameHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		idStr := extractCode(r.URL.Path, "/api/admin/users/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		if err := ctx.UserRepo.UpdateNickname(r.Context(), id, defaultNickname); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

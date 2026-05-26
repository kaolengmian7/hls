// Package handler 包含 REST 与 WebSocket 入口。
package handler

import (
	stdrand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/svc"
	"hls.dev/server/pkg/code"
	"hls.dev/server/pkg/errors"
)

// CreateRoomRequest 客户端创建房间的请求体。
//
// GuaranteedAmount 房间保底金额（单位 W），可省略；省略 / 传 0 视为采用服务端默认值
// game.DefaultGuaranteedAmount。该字段是独立展示数值，不参与金币结算。
type CreateRoomRequest struct {
	Nickname         string `json:"nickname"`
	Avatar           string `json:"avatar,omitempty"`
	GuaranteedAmount int    `json:"guaranteedAmount,omitempty"`
}

// CreateRoomResponse 服务端返回。
//
// GuaranteedAmount 是服务端最终落库的保底金额（单位 W），客户端应以此为准展示。
type CreateRoomResponse struct {
	Code             string `json:"code"`
	PlayerID         string `json:"playerId"`
	SessionToken     string `json:"sessionToken"`
	GuaranteedAmount int    `json:"guaranteedAmount"`
}

// CreateRoomHandler 处理 POST /api/rooms。
func CreateRoomHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid body", err)
			return
		}
		req.Nickname = strings.TrimSpace(req.Nickname)
		if req.Nickname == "" {
			writeError(w, r, http.StatusBadRequest, "nickname required", nil)
			return
		}
		if len([]rune(req.Nickname)) > 16 {
			req.Nickname = string([]rune(req.Nickname)[:16])
		}

		hostPlayerID, err := newPlayerID()
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "alloc playerId", err)
			return
		}

		room, tok, err := ctx.Hub.CreateRoom(req.Nickname, req.Avatar, hostPlayerID, req.GuaranteedAmount)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.IsExposable(err) {
				status = http.StatusBadRequest
			}
			writeError(w, r, status, err.Error(), err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, CreateRoomResponse{
			Code:             room.Code(),
			PlayerID:         hostPlayerID,
			SessionToken:     tok,
			GuaranteedAmount: room.GuaranteedAmount(),
		})
	}
}

// QueryRoomHandler 处理 GET /api/rooms/:code。
func QueryRoomHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := extractCode(r.URL.Path, "/api/rooms/")
		if !code.IsValid(c) {
			writeError(w, r, http.StatusBadRequest, "invalid code", nil)
			return
		}
		summary := ctx.Hub.QueryRoom(c)
		if !summary.Exists {
			writeError(w, r, http.StatusNotFound, "room not found", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, summary)
	}
}

// HealthzHandler 处理 GET /api/healthz。
func HealthzHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"ok":    true,
			"rooms": ctx.Hub.Stats(),
		})
	}
}

/* ------------------------------ helpers ------------------------------ */

func extractCode(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

// writeError 返回 JSON 错误并写入日志。
func writeError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	logx.WithContext(r.Context()).Errorw("http error",
		logx.Field("status", status),
		logx.Field("message", message),
		logx.Field("err", err),
	)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      false,
		"message": message,
	})
}

// newPlayerID 生成 14 字符 base64url 玩家 ID。
func newPlayerID() (string, error) {
	var buf [10]byte
	if _, err := stdrand.Read(buf[:]); err != nil {
		return "", errors.WithMessage(err, "rand")
	}
	return "p_" + base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

package handler

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/room"
	"hls.dev/server/internal/svc"
	"hls.dev/server/pkg/code"
)

// defaultUpgrader 默认 WebSocket 升级器；CheckOrigin 在 handler 里按配置注入。
var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// WebSocketHandler 处理 GET /ws/rooms/:code。
//
// 查询参数：
//   - playerId 必填：客户端持久 ID。
//   - token   选填：sessionToken；存在且校验通过 → 视为 reconnect，无需再发 JOIN。
func WebSocketHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := extractCode(r.URL.Path, "/ws/rooms/")
		if !code.IsValid(c) {
			http.Error(w, "invalid code", http.StatusBadRequest)
			return
		}
		rm := ctx.Hub.GetRoom(c)
		if rm == nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}

		query := r.URL.Query()
		playerID := strings.TrimSpace(query.Get("playerId"))
		if playerID == "" {
			http.Error(w, "playerId required", http.StatusBadRequest)
			return
		}
		sessionToken := strings.TrimSpace(query.Get("token"))

		var userID int64
		if jwt := strings.TrimSpace(query.Get("jwt")); jwt != "" {
			if uid, err := ctx.JWTSigner.Verify(jwt); err == nil {
				userID = uid
			}
		}

		upgrader := defaultUpgrader
		upgrader.CheckOrigin = func(req *http.Request) bool {
			return originAllowed(ctx, req)
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("ws upgrade",
				logx.Field("err", err),
				logx.Field("code", c))
			return
		}

		client := room.NewClient(conn, rm, playerID, userID, logx.WithContext(ctx.RootCtx))

		authenticated := false
		if sessionToken != "" {
			if _, err := ctx.Signer.Verify(c, playerID, sessionToken); err != nil {
				logx.WithContext(r.Context()).Infow("session token invalid",
					logx.Field("code", c),
					logx.Field("playerId", playerID),
					logx.Field("err", err))
			} else {
				authenticated = true
			}
		}

		rm.RegisterClient(client, playerID, authenticated)
		go client.Run(ctx.RootCtx)
	}
}

func originAllowed(ctx *svc.ServiceContext, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if len(ctx.Config.Cors.AllowOrigins) == 0 {
		return true
	}
	for _, o := range ctx.Config.Cors.AllowOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/room"
	"hls.dev/server/internal/svc"
	"hls.dev/server/pkg/code"
)

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// WebSocketHandler 处理 GET /ws/rooms/:code。
//
// 查询参数：
//   - jwt 必填：Bearer JWT，用于身份验证并派生 playerID。
//
// 重连判断：服务端通过 rm.HasPlayer(playerID) 检查玩家是否已在房间内；
// 客户端始终发送 JOIN，服务端按 hasJoined 标志幂等处理。
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

		jwt := strings.TrimSpace(r.URL.Query().Get("jwt"))
		userID, err := ctx.JWTSigner.Verify(jwt)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		playerID := strconv.FormatInt(userID, 10)

		upgrader := defaultUpgrader
		upgrader.CheckOrigin = func(req *http.Request) bool {
			return originAllowed(ctx, req)
		}

		conn, wsErr := upgrader.Upgrade(w, r, nil)
		if wsErr != nil {
			logx.WithContext(r.Context()).Errorw("ws upgrade",
				logx.Field("err", wsErr),
				logx.Field("code", c))
			return
		}

		client := room.NewClient(conn, rm, playerID, userID, logx.WithContext(ctx.RootCtx))
		isReturningMember := rm.HasPlayer(playerID)
		rm.RegisterClient(client, playerID, isReturningMember)
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

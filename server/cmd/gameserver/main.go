// Package main 是游戏服务端入口。
//
// 启动流程：
//  1. 解析配置文件（go-zero conf）。
//  2. 构造 ServiceContext（Hub / Generator / Signer）。
//  3. 注册 REST + WebSocket 路由。
//  4. 监听 SIGINT / SIGTERM 做优雅退出。
package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"

	"hls.dev/server/internal/config"
	"hls.dev/server/internal/handler"
	"hls.dev/server/internal/svc"
)

var configFile = flag.String("f", "etc/gameserver.yaml", "config file path")

func main() {
	godotenv.Load() // loads .env if present; silent no-op when not found
	flag.Parse()

	var cfg config.Config
	conf.MustLoad(*configFile, &cfg)
	if err := config.ApplyEnvOverrides(&cfg); err != nil {
		logx.Must(err)
	}

	opts := []rest.RunOption{}
	if len(cfg.Cors.AllowOrigins) > 0 {
		opts = append(opts, rest.WithCors(cfg.Cors.AllowOrigins...))
	} else {
		opts = append(opts, rest.WithCors("*"))
	}
	server := rest.MustNewServer(cfg.RestConf, opts...)
	defer server.Stop()

	ctx := svc.NewServiceContext(cfg)
	defer ctx.Stop()

	server.AddRoutes([]rest.Route{
		{Method: http.MethodPost, Path: "/api/rooms", Handler: handler.CreateRoomHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/rooms/:code", Handler: handler.QueryRoomHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/healthz", Handler: handler.HealthzHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/lottery/pool", Handler: handler.LotteryPoolHandler()},
		{Method: http.MethodGet, Path: "/ws/rooms/:code", Handler: handler.WebSocketHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/auth/wechat/qr/start", Handler: handler.QrStartHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/auth/wechat/qr/callback", Handler: handler.QrCallbackHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/auth/wechat/qr/status", Handler: handler.QrStatusHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/auth/me", Handler: handler.MeHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/auth/sms/send", Handler: handler.SmsSendHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/auth/sms/verify", Handler: handler.SmsVerifyHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/auth/login", Handler: handler.LoginHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/admin/users", Handler: handler.ListUsersHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/admin/users", Handler: handler.CreateUserHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/admin/users/:id/password", Handler: handler.ResetPasswordHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/admin/users/:id/nickname", Handler: handler.ResetNicknameHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/user/password", Handler: handler.UpdatePasswordHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/user/nickname", Handler: handler.UpdateNicknameHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/orders", Handler: handler.CreateOrderHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/orders", Handler: handler.ListOrdersHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/my-orders", Handler: handler.MyOrdersHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/orders/:code", Handler: handler.OrderDetailHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/orders/:code/claim", Handler: handler.ClaimOrderHandler(ctx)},
		{Method: http.MethodDelete, Path: "/api/orders/:code/enforcer/:slot", Handler: handler.RemoveEnforcerHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/orders/:code/status", Handler: handler.UpdateOrderStatusHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/orders/:code/complaint", Handler: handler.UpdateOrderComplaintHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/orders/:code/review", Handler: handler.UpsertReviewHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/orders/:code/review", Handler: handler.OrderReviewHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/public/orders/:code/review", Handler: handler.PublicUpsertReviewHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/public/orders/:code/review", Handler: handler.PublicOrderReviewHandler(ctx)},
	})

	logx.Infof("gameserver starting at %s:%d", cfg.Host, cfg.Port)

	go func() {
		server.Start()
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logx.Infof("gameserver shutting down...")
}

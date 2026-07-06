// Package svc 持有全局共享依赖。
package svc

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/auth"
	"hls.dev/server/internal/config"
	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/room"
	"hls.dev/server/pkg/code"
)

// ServiceContext 是 HTTP / WS handler 共享的依赖容器。
type ServiceContext struct {
	Config     config.Config
	Hub        *room.Hub
	Generator  *code.Generator
	RootCtx    context.Context
	RootCancel context.CancelFunc

	AuthStateRepo *repo.AuthStateRepo
	UserRepo      *repo.UserRepo
	JWTSigner     *auth.JWTSigner
	WechatClient  *auth.WechatClient
	ShopRepo      *repo.ShopRepo

	SmsStore  *SmsStore
	SmsSender auth.SmsSender
}

// NewServiceContext 构建依赖。
func NewServiceContext(cfg config.Config) *ServiceContext {
	rootCtx, cancel := context.WithCancel(context.Background())
	gen := code.NewGenerator()
	conn := sqlx.NewMysql(cfg.DB.DSN)
	shopRepo := repo.NewShopRepo(conn)
	hub := room.NewHub(rootCtx, gen, shopRepo, logx.WithContext(rootCtx), cfg.Shop.VIPUsers, cfg.Shop.VIPHighBoost)
	if err := hub.RestoreFromDB(rootCtx); err != nil {
		logx.WithContext(rootCtx).Errorw("restore rooms from db", logx.Field("err", err))
	}
	smsSender, err := auth.NewTencentSmsSender(cfg.TencentSMS)
	if err != nil {
		logx.Must(err)
	}
	return &ServiceContext{
		Config:        cfg,
		Hub:           hub,
		Generator:     gen,
		RootCtx:       rootCtx,
		RootCancel:    cancel,
		UserRepo:      repo.NewUserRepo(conn),
		AuthStateRepo: repo.NewAuthStateRepo(conn),
		JWTSigner:     auth.NewJWTSigner(cfg.JWT.Secret, cfg.JWT.TTLDays),
		WechatClient:  auth.NewWechatClient(cfg.WechatOpen.AppID, cfg.WechatOpen.AppSecret),
		ShopRepo:      shopRepo,
		SmsStore:      NewSmsStore(),
		SmsSender:     smsSender,
	}
}

// Stop 关闭依赖。
func (s *ServiceContext) Stop() {
	if s.RootCancel != nil {
		s.RootCancel()
	}
}

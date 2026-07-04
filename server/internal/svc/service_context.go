// Package svc 持有全局共享依赖。
package svc

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/auth"
	"hls.dev/server/internal/config"
	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/room"
	"hls.dev/server/pkg/code"
	"hls.dev/server/pkg/token"
)

// ServiceContext 是 HTTP / WS handler 共享的依赖容器。
type ServiceContext struct {
	Config     config.Config
	Hub        *room.Hub
	Generator  *code.Generator
	Signer     *token.Signer
	RootCtx    context.Context
	RootCancel context.CancelFunc

	// Auth fields — wired in Task 7.
	AuthStateRepo *repo.AuthStateRepo
	UserRepo      *repo.UserRepo
	JWTSigner     *auth.JWTSigner
	WechatClient  *auth.WechatClient
	ShopRepo      *repo.ShopRepo

	// SMS fields — wired in Task 5.
	SmsStore  *SmsStore
	SmsSender auth.SmsSender
}

// NewServiceContext 构建依赖。
func NewServiceContext(cfg config.Config) *ServiceContext {
	rootCtx, cancel := context.WithCancel(context.Background())
	gen := code.NewGenerator()
	ttl := time.Duration(cfg.Session.TTLMinutes) * time.Minute
	signer := token.NewSigner(cfg.Session.Secret, ttl)
	conn := sqlx.NewMysql(cfg.DB.DSN)
	shopRepo := repo.NewShopRepo(conn)
	hub := room.NewHub(rootCtx, gen, signer, shopRepo, logx.WithContext(rootCtx))
	smsSender, err := auth.NewTencentSmsSender(cfg.TencentSMS)
	if err != nil {
		logx.Must(err)
	}
	return &ServiceContext{
		Config:        cfg,
		Hub:           hub,
		Generator:     gen,
		Signer:        signer,
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

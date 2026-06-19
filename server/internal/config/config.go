// Package config 包含服务端运行时配置结构。
package config

import "github.com/zeromicro/go-zero/rest"

// Config 顶级配置。
type Config struct {
	rest.RestConf

	Session    SessionConfig
	Cors       CorsConfig
	DB         DBConfig
	JWT        JWTConfig
	WechatOpen WechatOpenConfig
}

// SessionConfig sessionToken 相关（游戏房间断线重连）。
type SessionConfig struct {
	Secret     string `json:",default=change-me"`
	TTLMinutes int    `json:",default=240"`
}

// CorsConfig CORS 允许列表。
type CorsConfig struct {
	AllowOrigins []string `json:",optional"`
}

// DBConfig MySQL 连接配置。
type DBConfig struct {
	DSN string `json:",optional"`
}

// JWTConfig 用户系统 JWT 配置。
type JWTConfig struct {
	Secret  string `json:",optional"`
	TTLDays int    `json:",default=7"`
}

// WechatOpenConfig 微信开放平台（PC 扫码）配置。
type WechatOpenConfig struct {
	AppID          string `json:",optional"`
	AppSecret      string `json:",optional"`
	CallbackDomain string `json:",optional"`
	FrontendURL    string `json:",optional"`
}

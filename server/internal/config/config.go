// Package config 包含服务端运行时配置结构。
package config

import "github.com/zeromicro/go-zero/rest"

// Config 顶级配置。
type Config struct {
	rest.RestConf

	Cors       CorsConfig
	DB         DBConfig
	JWT        JWTConfig
	WechatOpen WechatOpenConfig
	TencentSMS TencentSMSConfig
	Shop       ShopConfig
	Payroll    PayrollConfig
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

// TencentSMSConfig 腾讯云短信配置。
type TencentSMSConfig struct {
	SecretID    string `json:",optional"`
	SecretKey   string `json:",optional"`
	SmsSdkAppID string `json:",optional"`
	SignName    string `json:",optional"`
	TemplateID  string `json:",optional"`
}

// LotteryItemConfig 是奖品池单件的 YAML 配置映射，与 game.LotteryItem 字段一一对应。
type LotteryItemConfig struct {
	ID       string  `json:",optional"`
	Name     string  `json:",optional"`
	Rarity   string  `json:",optional"` // HIGH | MEDIUM | LOW
	Value    int     `json:",optional"` // 万元估值，正整数
	Weight   float64 `json:",optional"` // 抽奖权重（客户端不可见）
	ImageURL string  `json:",optional"`
}

// ShopConfig 神秘商店专项配置。
type ShopConfig struct {
	VIPUsers     []int64             `json:",optional"`
	VIPHighBoost float64             `json:",default=5"`
	LotteryPool  []LotteryItemConfig `json:",optional"`
}

// PayrollConfig 打手工资/押金配置。
type PayrollConfig struct {
	DefaultDeposit  float64 `json:",default=300"`
	SettleDelayDays int     `json:",default=3"`
}

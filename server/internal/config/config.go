// Package config 包含服务端运行时配置结构。
package config

import "github.com/zeromicro/go-zero/rest"

// Config 顶级配置。
type Config struct {
	rest.RestConf

	Session SessionConfig
	Cors    CorsConfig
}

// SessionConfig sessionToken 相关。
type SessionConfig struct {
	// Secret HMAC 密钥。生产环境必须从环境变量注入。
	Secret string `json:",default=change-me"`
	// TTLMinutes token 有效期。默认 240 分钟（4h）。
	TTLMinutes int `json:",default=240"`
}

// CorsConfig CORS 允许列表。
type CorsConfig struct {
	// AllowOrigins 允许的来源；为空表示放行所有（仅开发环境推荐）。
	AllowOrigins []string `json:",optional"`
}

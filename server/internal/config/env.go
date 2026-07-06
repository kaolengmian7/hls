package config

import (
	"os"
	"strconv"
	"strings"

	"hls.dev/server/pkg/errors"
)

const (
	envHost             = "HLS_HOST"
	envPort             = "HLS_PORT"
	envMode             = "HLS_MODE"
	envCorsAllowOrigins = "HLS_CORS_ALLOW_ORIGINS"

	envDBDSN = "HLS_DB_DSN"

	envJWTSecret  = "HLS_JWT_SECRET"
	envJWTTTLDays = "HLS_JWT_TTL_DAYS"

	envWechatOpenAppID          = "HLS_WECHAT_OPEN_APP_ID"
	envWechatOpenAppSecret      = "HLS_WECHAT_OPEN_APP_SECRET"
	envWechatOpenCallbackDomain = "HLS_WECHAT_OPEN_CALLBACK_DOMAIN"
	envWechatOpenFrontendURL    = "HLS_WECHAT_OPEN_FRONTEND_URL"

	envTencentSMSSecretID   = "HLS_TENCENT_SMS_SECRET_ID"
	envTencentSMSSecretKey  = "HLS_TENCENT_SMS_SECRET_KEY"
	envTencentSMSAppID      = "HLS_TENCENT_SMS_APP_ID"
	envTencentSMSSignName   = "HLS_TENCENT_SMS_SIGN_NAME"
	envTencentSMSTemplateID = "HLS_TENCENT_SMS_TEMPLATE_ID"
)

// ApplyEnvOverrides applies deployment-time overrides after the YAML config is loaded.
func ApplyEnvOverrides(cfg *Config) error {
	if cfg == nil {
		return errors.New(errors.CodeInvalidParam, "nil config")
	}

	if host := strings.TrimSpace(os.Getenv(envHost)); host != "" {
		cfg.Host = host
	}
	if mode := strings.TrimSpace(os.Getenv(envMode)); mode != "" {
		cfg.Mode = mode
	}
	if origins := strings.TrimSpace(os.Getenv(envCorsAllowOrigins)); origins != "" {
		cfg.Cors.AllowOrigins = splitCSV(origins)
	}
	if dsn := strings.TrimSpace(os.Getenv(envDBDSN)); dsn != "" {
		cfg.DB.DSN = dsn
	}
	if secret := strings.TrimSpace(os.Getenv(envJWTSecret)); secret != "" {
		cfg.JWT.Secret = secret
	}
	if appID := strings.TrimSpace(os.Getenv(envWechatOpenAppID)); appID != "" {
		cfg.WechatOpen.AppID = appID
	}
	if appSecret := strings.TrimSpace(os.Getenv(envWechatOpenAppSecret)); appSecret != "" {
		cfg.WechatOpen.AppSecret = appSecret
	}
	if domain := strings.TrimSpace(os.Getenv(envWechatOpenCallbackDomain)); domain != "" {
		cfg.WechatOpen.CallbackDomain = domain
	}
	if frontendURL := strings.TrimSpace(os.Getenv(envWechatOpenFrontendURL)); frontendURL != "" {
		cfg.WechatOpen.FrontendURL = frontendURL
	}

	if secretID := strings.TrimSpace(os.Getenv(envTencentSMSSecretID)); secretID != "" {
		cfg.TencentSMS.SecretID = secretID
	}
	if secretKey := strings.TrimSpace(os.Getenv(envTencentSMSSecretKey)); secretKey != "" {
		cfg.TencentSMS.SecretKey = secretKey
	}
	if appID := strings.TrimSpace(os.Getenv(envTencentSMSAppID)); appID != "" {
		cfg.TencentSMS.SmsSdkAppID = appID
	}
	if signName := strings.TrimSpace(os.Getenv(envTencentSMSSignName)); signName != "" {
		cfg.TencentSMS.SignName = signName
	}
	if templateID := strings.TrimSpace(os.Getenv(envTencentSMSTemplateID)); templateID != "" {
		cfg.TencentSMS.TemplateID = templateID
	}

	if port := strings.TrimSpace(os.Getenv(envPort)); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value <= 0 {
			return errors.New(errors.CodeInvalidParam, "%s must be a positive integer", envPort)
		}
		cfg.Port = value
	}
	if ttlDays := strings.TrimSpace(os.Getenv(envJWTTTLDays)); ttlDays != "" {
		value, err := strconv.Atoi(ttlDays)
		if err != nil || value <= 0 {
			return errors.New(errors.CodeInvalidParam, "%s must be a positive integer", envJWTTTLDays)
		}
		cfg.JWT.TTLDays = value
	}

	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

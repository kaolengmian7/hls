package config

import (
	"reflect"
	"testing"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv(envHost, "127.0.0.1")
	t.Setenv(envPort, "9090")
	t.Setenv(envMode, "prod")
	t.Setenv(envCorsAllowOrigins, "https://a.example.com, https://b.example.com")

	cfg := Config{}
	if err := ApplyEnvOverrides(&cfg); err != nil {
		t.Fatalf("ApplyEnvOverrides() error = %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("Host = %q, want %q", cfg.Host, "127.0.0.1")
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d, want %d", cfg.Port, 9090)
	}
	if cfg.Mode != "prod" {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, "prod")
	}

	wantOrigins := []string{"https://a.example.com", "https://b.example.com"}
	if !reflect.DeepEqual(cfg.Cors.AllowOrigins, wantOrigins) {
		t.Fatalf("Cors.AllowOrigins = %#v, want %#v", cfg.Cors.AllowOrigins, wantOrigins)
	}

	t.Setenv(envDBDSN, "root:pass@tcp(localhost:3306)/hls?parseTime=true")
	t.Setenv(envJWTSecret, "jwt-secret")
	t.Setenv(envJWTTTLDays, "14")
	t.Setenv(envWechatOpenAppID, "wx123")
	t.Setenv(envWechatOpenAppSecret, "opensecret")
	t.Setenv(envWechatOpenCallbackDomain, "api.hls-club.com")
	t.Setenv(envWechatOpenFrontendURL, "https://hls-club.com")

	if err := ApplyEnvOverrides(&cfg); err != nil {
		t.Fatalf("ApplyEnvOverrides() error = %v", err)
	}

	if cfg.DB.DSN != "root:pass@tcp(localhost:3306)/hls?parseTime=true" {
		t.Fatalf("DB.DSN = %q", cfg.DB.DSN)
	}
	if cfg.JWT.Secret != "jwt-secret" {
		t.Fatalf("JWT.Secret = %q", cfg.JWT.Secret)
	}
	if cfg.JWT.TTLDays != 14 {
		t.Fatalf("JWT.TTLDays = %d, want 14", cfg.JWT.TTLDays)
	}
	if cfg.WechatOpen.AppID != "wx123" {
		t.Fatalf("WechatOpen.AppID = %q", cfg.WechatOpen.AppID)
	}
	if cfg.WechatOpen.AppSecret != "opensecret" {
		t.Fatalf("WechatOpen.AppSecret = %q", cfg.WechatOpen.AppSecret)
	}
	if cfg.WechatOpen.CallbackDomain != "api.hls-club.com" {
		t.Fatalf("WechatOpen.CallbackDomain = %q", cfg.WechatOpen.CallbackDomain)
	}
	if cfg.WechatOpen.FrontendURL != "https://hls-club.com" {
		t.Fatalf("WechatOpen.FrontendURL = %q", cfg.WechatOpen.FrontendURL)
	}
}

func TestApplyEnvOverridesInvalidIntegers(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "invalid port", key: envPort, value: "bad"},
		{name: "zero port", key: envPort, value: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)

			if err := ApplyEnvOverrides(&Config{}); err == nil {
				t.Fatalf("ApplyEnvOverrides() error = nil, want error")
			}
		})
	}
}

func TestApplyEnvOverridesNilConfig(t *testing.T) {
	if err := ApplyEnvOverrides(nil); err == nil {
		t.Fatalf("ApplyEnvOverrides() error = nil, want error")
	}
}

func TestApplyEnvOverridesTencentSMS(t *testing.T) {
	t.Setenv(envTencentSMSSecretID, "my-secret-id")
	t.Setenv(envTencentSMSSecretKey, "my-secret-key")
	t.Setenv(envTencentSMSAppID, "1400123456")
	t.Setenv(envTencentSMSSignName, "MySign")
	t.Setenv(envTencentSMSTemplateID, "1234567")

	cfg := Config{}
	if err := ApplyEnvOverrides(&cfg); err != nil {
		t.Fatalf("ApplyEnvOverrides() error = %v", err)
	}

	if cfg.TencentSMS.SecretID != "my-secret-id" {
		t.Fatalf("TencentSMS.SecretID = %q, want %q", cfg.TencentSMS.SecretID, "my-secret-id")
	}
	if cfg.TencentSMS.SecretKey != "my-secret-key" {
		t.Fatalf("TencentSMS.SecretKey = %q, want %q", cfg.TencentSMS.SecretKey, "my-secret-key")
	}
	if cfg.TencentSMS.SmsSdkAppID != "1400123456" {
		t.Fatalf("TencentSMS.SmsSdkAppID = %q, want %q", cfg.TencentSMS.SmsSdkAppID, "1400123456")
	}
	if cfg.TencentSMS.SignName != "MySign" {
		t.Fatalf("TencentSMS.SignName = %q, want %q", cfg.TencentSMS.SignName, "MySign")
	}
	if cfg.TencentSMS.TemplateID != "1234567" {
		t.Fatalf("TencentSMS.TemplateID = %q, want %q", cfg.TencentSMS.TemplateID, "1234567")
	}
}

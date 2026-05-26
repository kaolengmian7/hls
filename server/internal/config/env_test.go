package config

import (
	"reflect"
	"testing"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv(envHost, "127.0.0.1")
	t.Setenv(envPort, "9090")
	t.Setenv(envMode, "prod")
	t.Setenv(envSessionSecret, "secret-from-env")
	t.Setenv(envSessionTTL, "60")
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
	if cfg.Session.Secret != "secret-from-env" {
		t.Fatalf("Session.Secret = %q, want %q", cfg.Session.Secret, "secret-from-env")
	}
	if cfg.Session.TTLMinutes != 60 {
		t.Fatalf("Session.TTLMinutes = %d, want %d", cfg.Session.TTLMinutes, 60)
	}

	wantOrigins := []string{"https://a.example.com", "https://b.example.com"}
	if !reflect.DeepEqual(cfg.Cors.AllowOrigins, wantOrigins) {
		t.Fatalf("Cors.AllowOrigins = %#v, want %#v", cfg.Cors.AllowOrigins, wantOrigins)
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
		{name: "invalid ttl", key: envSessionTTL, value: "bad"},
		{name: "zero ttl", key: envSessionTTL, value: "0"},
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

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
	envSessionSecret    = "HLS_SESSION_SECRET"
	envSessionTTL       = "HLS_SESSION_TTL_MINUTES"
	envCorsAllowOrigins = "HLS_CORS_ALLOW_ORIGINS"
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
	if secret := strings.TrimSpace(os.Getenv(envSessionSecret)); secret != "" {
		cfg.Session.Secret = secret
	}
	if origins := strings.TrimSpace(os.Getenv(envCorsAllowOrigins)); origins != "" {
		cfg.Cors.AllowOrigins = splitCSV(origins)
	}

	if port := strings.TrimSpace(os.Getenv(envPort)); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value <= 0 {
			return errors.New(errors.CodeInvalidParam, "%s must be a positive integer", envPort)
		}
		cfg.Port = value
	}
	if ttl := strings.TrimSpace(os.Getenv(envSessionTTL)); ttl != "" {
		value, err := strconv.Atoi(ttl)
		if err != nil || value <= 0 {
			return errors.New(errors.CodeInvalidParam, "%s must be a positive integer", envSessionTTL)
		}
		cfg.Session.TTLMinutes = value
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

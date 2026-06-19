# WeChat Auth User System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a standalone WeChat QR-code login/registration system to the existing go-zero gameserver and Vue SPA, storing users in a self-hosted MySQL instance, with no coupling to the game room logic.

**Architecture:** MySQL 8 runs as a Docker service on the same host as `gameserver`, connected via `localhost`. The gameserver gains 4 new `/api/auth/*` routes. The Vue SPA gains a LoginPage (iframe QR), Pinia auth store, and axios HTTP client. The frontend polls `qr/status` every 2 s until the user scans.

**Tech Stack:** Go 1.25 / go-zero v1.10 / golang-jwt v4 / go-sql-driver/mysql / go-zero sqlx / MySQL 8 / Vue 3 / Pinia / axios / TypeScript

---

## File Map

**Create (backend):**
- `server/db/schema.sql` — CREATE TABLE statements
- `server/internal/auth/jwt.go` — JWT sign/verify
- `server/internal/auth/jwt_test.go`
- `server/internal/auth/wechat.go` — WeChat API client
- `server/internal/auth/wechat_test.go`
- `server/internal/repo/user.go` — UserRepo
- `server/internal/repo/user_test.go` — integration (skipped if no TEST_DB_DSN)
- `server/internal/repo/authstate.go` — AuthStateRepo
- `server/internal/repo/authstate_test.go` — integration
- `server/internal/handler/auth.go` — 4 handlers

**Modify (backend):**
- `server/internal/config/config.go` — add WechatOpenConfig, JWTConfig, DBConfig
- `server/internal/config/env.go` — add env overrides for new fields
- `server/internal/config/env_test.go` — extend existing test
- `server/internal/svc/service_context.go` — add DB, repos, JWTSigner, WechatClient
- `server/cmd/gameserver/main.go` — register 4 new routes
- `server/etc/gameserver.yaml` — add new config blocks
- `server/docker-compose.yml` — add MySQL service
- `server/.env.example` — add new env vars

**Create (frontend):**
- `src/http.ts` — axios instance with auth header + 401 handler
- `src/stores/auth.ts` — Pinia auth store
- `src/composables/useWechatQR.ts` — QR polling
- `src/pages/LoginPage.vue`
- `src/pages/AuthCallbackPage.vue` — future H5 OAuth landing page

**Modify (frontend):**
- `src/router/index.ts` — add /login and /auth/callback routes
- `src/main.ts` — call auth.init() on startup
- `package.json` — add axios

---

## Task 1: Infrastructure — MySQL + Schema

**Files:**
- Create: `server/db/schema.sql`
- Modify: `server/docker-compose.yml`
- Modify: `server/.env.example`

- [ ] **Step 1: Create the SQL schema file**

```sql
-- server/db/schema.sql
CREATE TABLE IF NOT EXISTS users (
  id           BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  mp_openid    VARCHAR(64)  NULL UNIQUE,
  open_openid  VARCHAR(64)  NULL UNIQUE,
  union_id     VARCHAR(64)  NULL UNIQUE,
  nickname     VARCHAR(64)  NOT NULL DEFAULT '',
  avatar_url   VARCHAR(512) NOT NULL DEFAULT '',
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS auth_states (
  state      VARCHAR(64) NOT NULL PRIMARY KEY,
  jwt        TEXT        NULL,
  expires_at DATETIME    NOT NULL,
  INDEX idx_expires_at (expires_at)
);
```

- [ ] **Step 2: Add MySQL service to docker-compose**

Replace the contents of `server/docker-compose.yml` with:

```yaml
services:
  caddy:
    image: caddy:2-alpine
    container_name: hls-caddy
    restart: unless-stopped
    depends_on:
      gameserver:
        condition: service_healthy
    environment:
      HLS_API_DOMAIN: ${HLS_API_DOMAIN:?set HLS_API_DOMAIN in server/.env}
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config

  gameserver:
    build:
      context: .
      dockerfile: Dockerfile
    image: hls-gameserver:local
    container_name: hls-gameserver
    restart: unless-stopped
    init: true
    depends_on:
      mysql:
        condition: service_healthy
    environment:
      HLS_HOST: 0.0.0.0
      HLS_MODE: ${HLS_MODE:-prod}
      HLS_SESSION_SECRET: ${HLS_SESSION_SECRET:?set HLS_SESSION_SECRET in server/.env}
      HLS_SESSION_TTL_MINUTES: ${HLS_SESSION_TTL_MINUTES:-1440}
      HLS_CORS_ALLOW_ORIGINS: ${HLS_CORS_ALLOW_ORIGINS:?set HLS_CORS_ALLOW_ORIGINS in server/.env}
      HLS_DB_DSN: ${HLS_DB_DSN:?set HLS_DB_DSN in server/.env}
      HLS_JWT_SECRET: ${HLS_JWT_SECRET:?set HLS_JWT_SECRET in server/.env}
      HLS_JWT_TTL_DAYS: ${HLS_JWT_TTL_DAYS:-7}
      HLS_WECHAT_OPEN_APP_ID: ${HLS_WECHAT_OPEN_APP_ID:?set HLS_WECHAT_OPEN_APP_ID in server/.env}
      HLS_WECHAT_OPEN_APP_SECRET: ${HLS_WECHAT_OPEN_APP_SECRET:?set HLS_WECHAT_OPEN_APP_SECRET in server/.env}
      HLS_WECHAT_OPEN_CALLBACK_DOMAIN: ${HLS_WECHAT_OPEN_CALLBACK_DOMAIN:?set HLS_WECHAT_OPEN_CALLBACK_DOMAIN in server/.env}
      HLS_WECHAT_OPEN_FRONTEND_URL: ${HLS_WECHAT_OPEN_FRONTEND_URL:?set HLS_WECHAT_OPEN_FRONTEND_URL in server/.env}
    expose:
      - "8088"
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8088/api/healthz >/dev/null || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 10s
    stop_grace_period: 20s

  mysql:
    image: mysql:8.0
    container_name: hls-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:?set MYSQL_ROOT_PASSWORD in server/.env}
      MYSQL_DATABASE: hls
    volumes:
      - mysql_data:/var/lib/mysql
      - ./db/schema.sql:/docker-entrypoint-initdb.d/schema.sql:ro
    ports:
      - "127.0.0.1:3306:3306"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "--password=${MYSQL_ROOT_PASSWORD}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s

volumes:
  caddy_data:
  caddy_config:
  mysql_data:
```

- [ ] **Step 3: Update .env.example**

Replace `server/.env.example` with:

```bash
# Copy this file to .env before running docker compose.

HLS_MODE=prod

# Public API domain that points to the server.
HLS_API_DOMAIN=api.hls-club.com

# Game session token secret.
HLS_SESSION_SECRET=replace-with-a-long-random-secret
HLS_SESSION_TTL_MINUTES=1440

# CORS
HLS_CORS_ALLOW_ORIGINS=https://hls-club.com,https://www.hls-club.com

# MySQL — root password used by the mysql container.
MYSQL_ROOT_PASSWORD=replace-with-a-long-random-password
# DSN used by gameserver to connect. User/pass must match what's configured in MySQL.
# For local dev with Docker: root@tcp(mysql:3306)/hls?parseTime=true
HLS_DB_DSN=root:replace-with-a-long-random-password@tcp(mysql:3306)/hls?parseTime=true

# JWT secret for user auth tokens.
HLS_JWT_SECRET=replace-with-another-long-random-secret
HLS_JWT_TTL_DAYS=7

# WeChat Open Platform credentials.
HLS_WECHAT_OPEN_APP_ID=wxYOUR_APP_ID
HLS_WECHAT_OPEN_APP_SECRET=YOUR_APP_SECRET
# The public API domain used to construct the WeChat callback URL.
HLS_WECHAT_OPEN_CALLBACK_DOMAIN=api.hls-club.com
# The frontend URL WeChat errors redirect to.
HLS_WECHAT_OPEN_FRONTEND_URL=https://hls-club.com
```

- [ ] **Step 4: Commit**

```bash
git add server/db/schema.sql server/docker-compose.yml server/.env.example
git commit -m "feat: add MySQL service and db schema for auth system"
```

---

## Task 2: Go Dependencies + Config

**Files:**
- Modify: `server/internal/config/config.go`
- Modify: `server/internal/config/env.go`
- Modify: `server/internal/config/env_test.go`
- Modify: `server/go.mod` (via `go get`)

- [ ] **Step 1: Add Go dependencies**

Run from `server/`:
```bash
cd server
go get github.com/golang-jwt/jwt/v4@v4.5.2
```

This promotes `golang-jwt/jwt/v4` from indirect to direct in `go.mod`. The MySQL driver (`go-sql-driver/mysql`) does **not** need a direct import — `sqlx.NewMysql` (go-zero) registers the driver internally.

- [ ] **Step 2: Write failing test for new config fields**

Add to `server/internal/config/env_test.go` inside `TestApplyEnvOverrides` (after the existing `t.Setenv` calls):

```go
t.Setenv(envDBDSN, "root:pass@tcp(localhost:3306)/hls?parseTime=true")
t.Setenv(envJWTSecret, "jwt-secret")
t.Setenv(envJWTTTLDays, "14")
t.Setenv(envWechatOpenAppID, "wx123")
t.Setenv(envWechatOpenAppSecret, "opensecret")
t.Setenv(envWechatOpenCallbackDomain, "api.hls-club.com")
t.Setenv(envWechatOpenFrontendURL, "https://hls-club.com")
```

Add assertions after the existing ones:

```go
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
```

- [ ] **Step 3: Run test — expect compile failure**

```bash
cd server && go test ./internal/config/...
```

Expected: compile error — `envDBDSN`, `envJWTSecret`, etc. are undefined.

- [ ] **Step 4: Expand config.go**

Replace `server/internal/config/config.go` with:

```go
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
```

- [ ] **Step 5: Expand env.go**

Replace `server/internal/config/env.go` with:

```go
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

	envDBDSN = "HLS_DB_DSN"

	envJWTSecret  = "HLS_JWT_SECRET"
	envJWTTTLDays = "HLS_JWT_TTL_DAYS"

	envWechatOpenAppID          = "HLS_WECHAT_OPEN_APP_ID"
	envWechatOpenAppSecret      = "HLS_WECHAT_OPEN_APP_SECRET"
	envWechatOpenCallbackDomain = "HLS_WECHAT_OPEN_CALLBACK_DOMAIN"
	envWechatOpenFrontendURL    = "HLS_WECHAT_OPEN_FRONTEND_URL"
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
```

- [ ] **Step 6: Run tests — expect pass**

```bash
cd server && go test ./internal/config/...
```

Expected: `ok  hls.dev/server/internal/config`

- [ ] **Step 7: Commit**

```bash
git add server/go.mod server/go.sum server/internal/config/
git commit -m "feat: expand config with DB, JWT, and WeChat Open Platform fields"
```

---

## Task 3: JWT Package

**Files:**
- Create: `server/internal/auth/jwt.go`
- Create: `server/internal/auth/jwt_test.go`

- [ ] **Step 1: Write failing tests**

Create `server/internal/auth/jwt_test.go`:

```go
package auth_test

import (
	"testing"

	"hls.dev/server/internal/auth"
)

func TestJWTSignAndVerify(t *testing.T) {
	s := auth.NewJWTSigner("test-secret", 7)
	tok, err := s.Sign(42)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	id, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if id != 42 {
		t.Fatalf("userID = %d, want 42", id)
	}
}

func TestJWTVerifyInvalidToken(t *testing.T) {
	s := auth.NewJWTSigner("secret", 7)
	_, err := s.Verify("not.a.valid.token")
	if err == nil {
		t.Fatal("Verify() with invalid token = nil, want error")
	}
}

func TestJWTVerifyWrongSecret(t *testing.T) {
	s1 := auth.NewJWTSigner("secret-a", 7)
	s2 := auth.NewJWTSigner("secret-b", 7)
	tok, _ := s1.Sign(1)
	_, err := s2.Verify(tok)
	if err == nil {
		t.Fatal("Verify() with wrong secret = nil, want error")
	}
}
```

- [ ] **Step 2: Run — expect failure**

```bash
cd server && go test ./internal/auth/...
```

Expected: compile error — package `auth` does not exist.

- [ ] **Step 3: Implement jwt.go**

Create `server/internal/auth/jwt.go`:

```go
// Package auth provides JWT signing/verification for user sessions.
package auth

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"hls.dev/server/pkg/errors"
)

// JWTSigner signs and verifies user session JWTs.
type JWTSigner struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTSigner creates a JWTSigner. ttlDays <= 0 defaults to 7.
func NewJWTSigner(secret string, ttlDays int) *JWTSigner {
	if ttlDays <= 0 {
		ttlDays = 7
	}
	return &JWTSigner{
		secret: []byte(secret),
		ttl:    time.Duration(ttlDays) * 24 * time.Hour,
	}
}

// Sign issues a JWT with sub=userID, valid for ttl.
func (s *JWTSigner) Sign(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", errors.WithMessage(err, "jwt sign")
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning the userID from sub.
func (s *JWTSigner) Verify(tokenStr string) (int64, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.CodeInvalidSession, "unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return 0, errors.New(errors.CodeInvalidSession, "invalid or expired token")
	}
	claims, ok := tok.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return 0, errors.New(errors.CodeInvalidSession, "invalid claims")
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errors.New(errors.CodeInvalidSession, "invalid subject")
	}
	return id, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd server && go test ./internal/auth/...
```

Expected: `ok  hls.dev/server/internal/auth`

- [ ] **Step 5: Commit**

```bash
git add server/internal/auth/jwt.go server/internal/auth/jwt_test.go
git commit -m "feat: add JWT signer for user auth tokens"
```

---

## Task 4: WeChat API Client

**Files:**
- Create: `server/internal/auth/wechat.go`
- Create: `server/internal/auth/wechat_test.go`

- [ ] **Step 1: Write failing tests**

Create `server/internal/auth/wechat_test.go`:

```go
package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hls.dev/server/internal/auth"
)

func newTestWechatClient(srv *httptest.Server) *auth.WechatClient {
	c := auth.NewWechatClient("test-appid", "test-secret")
	c.SetBaseURL(srv.URL)
	return c
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "access_token") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok123",
			"openid":       "oid123",
			"unionid":      "uid123",
		})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	resp, err := c.ExchangeCode("code123")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if resp.AccessToken != "tok123" || resp.OpenID != "oid123" || resp.UnionID != "uid123" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestExchangeCodeWechatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"errcode": 40029, "errmsg": "invalid code"})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	_, err := c.ExchangeCode("bad_code")
	if err == nil {
		t.Fatal("ExchangeCode() with WeChat error should fail")
	}
}

func TestFetchUserInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "userinfo") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"openid":     "oid123",
			"nickname":   "花栗鼠",
			"headimgurl": "https://example.com/avatar.jpg",
			"unionid":    "uid123",
		})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	info, err := c.FetchUserInfo("tok123", "oid123")
	if err != nil {
		t.Fatalf("FetchUserInfo() error = %v", err)
	}
	if info.Nickname != "花栗鼠" {
		t.Fatalf("Nickname = %q, want %q", info.Nickname, "花栗鼠")
	}
}

func TestFetchUserInfoWechatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"errcode": 40001, "errmsg": "invalid access_token"})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	_, err := c.FetchUserInfo("bad_tok", "oid")
	if err == nil {
		t.Fatal("FetchUserInfo() with WeChat error should fail")
	}
}
```

- [ ] **Step 2: Run — expect compile failure**

```bash
cd server && go test ./internal/auth/...
```

Expected: compile error — `SetBaseURL`, `WechatClient` undefined.

- [ ] **Step 3: Implement wechat.go**

Create `server/internal/auth/wechat.go`:

```go
package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const wechatBaseURL = "https://api.weixin.qq.com"

// WechatClient calls WeChat Open Platform APIs to exchange codes and fetch user info.
type WechatClient struct {
	AppID      string
	AppSecret  string
	httpClient *http.Client
	baseURL    string
}

// NewWechatClient creates a WechatClient for the given Open Platform credentials.
func NewWechatClient(appID, appSecret string) *WechatClient {
	return &WechatClient{
		AppID:      appID,
		AppSecret:  appSecret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    wechatBaseURL,
	}
}

// SetBaseURL overrides the base URL — used in tests to point at a mock server.
func (c *WechatClient) SetBaseURL(url string) {
	c.baseURL = url
}

// WechatTokenResp is the response from the access_token endpoint.
type WechatTokenResp struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WechatUserInfo is the response from the userinfo endpoint.
type WechatUserInfo struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// ExchangeCode exchanges an OAuth code for an access token + openid.
func (c *WechatClient) ExchangeCode(code string) (*WechatTokenResp, error) {
	url := fmt.Sprintf(
		"%s/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		c.baseURL, c.AppID, c.AppSecret, code,
	)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("wechat exchange code: %w", err)
	}
	defer resp.Body.Close()
	var result WechatTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("wechat decode token response: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

// FetchUserInfo fetches the user's nickname and avatar using the access token.
func (c *WechatClient) FetchUserInfo(accessToken, openID string) (*WechatUserInfo, error) {
	url := fmt.Sprintf(
		"%s/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		c.baseURL, accessToken, openID,
	)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("wechat fetch userinfo: %w", err)
	}
	defer resp.Body.Close()
	var result WechatUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("wechat decode userinfo: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat userinfo error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd server && go test ./internal/auth/...
```

Expected: `ok  hls.dev/server/internal/auth`

- [ ] **Step 5: Commit**

```bash
git add server/internal/auth/wechat.go server/internal/auth/wechat_test.go
git commit -m "feat: add WeChat Open Platform API client"
```

---

## Task 5: Database Repositories

**Files:**
- Create: `server/internal/repo/user.go`
- Create: `server/internal/repo/user_test.go`
- Create: `server/internal/repo/authstate.go`
- Create: `server/internal/repo/authstate_test.go`

These tests are **integration tests** — they need a running MySQL. They are skipped automatically if the `TEST_DB_DSN` environment variable is not set.

- [ ] **Step 1: Write failing tests for UserRepo**

Create `server/internal/repo/user_test.go`:

```go
package repo_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/repo"
)

func testConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func TestUpsertByWechat_NewUser(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.UpsertByWechat(ctx, "open_oid_new", "union_uid_new", "TestUser", "https://example.com/avatar.jpg")
	if err != nil {
		t.Fatalf("UpsertByWechat() error = %v", err)
	}
	if u.ID == 0 {
		t.Fatal("ID should be non-zero after insert")
	}
	if u.Nickname != "TestUser" {
		t.Fatalf("Nickname = %q, want TestUser", u.Nickname)
	}

	// cleanup
	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE open_openid = ?", "open_oid_new")
}

func TestUpsertByWechat_ExistingByUnionID(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	// Insert initial
	u1, err := r.UpsertByWechat(ctx, "open_oid_existing", "union_uid_existing", "OldName", "old_avatar")
	if err != nil {
		t.Fatalf("initial insert error = %v", err)
	}

	// Upsert again by same union_id — should update name
	u2, err := r.UpsertByWechat(ctx, "open_oid_existing2", "union_uid_existing", "NewName", "new_avatar")
	if err != nil {
		t.Fatalf("upsert error = %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("ID mismatch: got %d, want %d", u2.ID, u1.ID)
	}
	if u2.Nickname != "NewName" {
		t.Fatalf("Nickname = %q, want NewName", u2.Nickname)
	}

	// cleanup
	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u1.ID)
}

func TestFindByID_NotFound(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	_, err := r.FindByID(context.Background(), 999999999)
	if err == nil {
		t.Fatal("FindByID() with non-existent ID should return error")
	}
}
```

- [ ] **Step 2: Write failing tests for AuthStateRepo**

Create `server/internal/repo/authstate_test.go`:

```go
package repo_test

import (
	"context"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestAuthState_CreateAndGet(t *testing.T) {
	conn := testConn(t)
	r := repo.NewAuthStateRepo(conn)
	ctx := context.Background()

	state := "test-state-" + t.Name()
	if err := r.Create(ctx, state); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	as, err := r.Get(ctx, state)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if as.JWT.Valid {
		t.Fatal("JWT should be NULL after Create()")
	}

	// cleanup
	r.Delete(ctx, state)
}

func TestAuthState_SetJWTAndGet(t *testing.T) {
	conn := testConn(t)
	r := repo.NewAuthStateRepo(conn)
	ctx := context.Background()

	state := "test-state-jwt-" + t.Name()
	r.Create(ctx, state)
	if err := r.SetJWT(ctx, state, "mytoken"); err != nil {
		t.Fatalf("SetJWT() error = %v", err)
	}

	as, err := r.Get(ctx, state)
	if err != nil {
		t.Fatalf("Get() after SetJWT error = %v", err)
	}
	if !as.JWT.Valid || as.JWT.String != "mytoken" {
		t.Fatalf("JWT = %v, want {String:mytoken Valid:true}", as.JWT)
	}

	r.Delete(ctx, state)
}

func TestAuthState_GetExpired(t *testing.T) {
	conn := testConn(t)
	r := repo.NewAuthStateRepo(conn)
	ctx := context.Background()

	// Insert a state that is already expired (expires_at in the past)
	db, _ := conn.RawDB()
	state := "test-state-expired-" + t.Name()
	db.ExecContext(ctx,
		"INSERT INTO auth_states (state, expires_at) VALUES (?, DATE_SUB(NOW(), INTERVAL 1 MINUTE))",
		state)

	_, err := r.Get(ctx, state)
	if err == nil {
		t.Fatal("Get() on expired state should return error (not found after cleanup)")
	}
}
```

- [ ] **Step 3: Run — expect compile failure**

```bash
cd server && go test ./internal/repo/...
```

Expected: compile error — package `repo` does not exist.

- [ ] **Step 4: Implement user.go**

Create `server/internal/repo/user.go`:

```go
// Package repo contains database access objects.
package repo

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// User represents a registered user.
type User struct {
	ID         int64          `db:"id"`
	OpenOpenID sql.NullString `db:"open_openid"`
	MpOpenID   sql.NullString `db:"mp_openid"`
	UnionID    sql.NullString `db:"union_id"`
	Nickname   string         `db:"nickname"`
	AvatarURL  string         `db:"avatar_url"`
}

const userCols = "id, open_openid, mp_openid, union_id, nickname, avatar_url"

// UserRepo provides database access for users.
type UserRepo struct {
	db sqlx.SqlConn
}

// NewUserRepo creates a UserRepo backed by the given connection.
func NewUserRepo(db sqlx.SqlConn) *UserRepo {
	return &UserRepo{db: db}
}

// UpsertByWechat finds or creates a user by openid/unionid and updates their profile.
func (r *UserRepo) UpsertByWechat(ctx context.Context, openOpenID, unionID, nickname, avatarURL string) (*User, error) {
	if unionID != "" {
		u, err := r.findBy(ctx, "union_id", unionID)
		if err == nil {
			r.updateProfile(ctx, u.ID, nickname, avatarURL)
			u.Nickname = nickname
			u.AvatarURL = avatarURL
			return u, nil
		}
		if err != sqlx.ErrNotFound {
			return nil, err
		}
	}

	u, err := r.findBy(ctx, "open_openid", openOpenID)
	if err == nil {
		r.updateProfile(ctx, u.ID, nickname, avatarURL)
		if unionID != "" {
			_, _ = r.db.ExecCtx(ctx,
				"UPDATE users SET union_id = COALESCE(union_id, ?), updated_at = NOW() WHERE id = ?",
				unionID, u.ID)
		}
		u.Nickname = nickname
		u.AvatarURL = avatarURL
		return u, nil
	}
	if err != sqlx.ErrNotFound {
		return nil, err
	}

	return r.insert(ctx, openOpenID, unionID, nickname, avatarURL)
}

// FindByID returns the user with the given ID.
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	return r.findBy(ctx, "id", id)
}

func (r *UserRepo) findBy(ctx context.Context, col string, val interface{}) (*User, error) {
	var u User
	err := r.db.QueryRowCtx(ctx, &u,
		"SELECT "+userCols+" FROM users WHERE "+col+" = ?", val)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) updateProfile(ctx context.Context, id int64, nickname, avatarURL string) {
	_, _ = r.db.ExecCtx(ctx,
		"UPDATE users SET nickname = ?, avatar_url = ?, updated_at = NOW() WHERE id = ?",
		nickname, avatarURL, id)
}

func (r *UserRepo) insert(ctx context.Context, openOpenID, unionID, nickname, avatarURL string) (*User, error) {
	result, err := r.db.ExecCtx(ctx,
		"INSERT INTO users (open_openid, union_id, nickname, avatar_url) VALUES (?, ?, ?, ?)",
		nullStr(openOpenID), nullStr(unionID), nickname, avatarURL)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
```

- [ ] **Step 5: Implement authstate.go**

Create `server/internal/repo/authstate.go`:

```go
package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// AuthState holds a pending or resolved QR login state.
type AuthState struct {
	State     string         `db:"state"`
	JWT       sql.NullString `db:"jwt"`
	ExpiresAt time.Time      `db:"expires_at"`
}

// AuthStateRepo provides database access for QR login states.
type AuthStateRepo struct {
	db sqlx.SqlConn
}

// NewAuthStateRepo creates an AuthStateRepo.
func NewAuthStateRepo(db sqlx.SqlConn) *AuthStateRepo {
	return &AuthStateRepo{db: db}
}

// Create inserts a new pending auth state expiring in 5 minutes.
func (r *AuthStateRepo) Create(ctx context.Context, state string) error {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO auth_states (state, expires_at) VALUES (?, DATE_ADD(NOW(), INTERVAL 5 MINUTE))",
		state)
	return err
}

// SetJWT stores the signed JWT for a state after successful WeChat callback.
func (r *AuthStateRepo) SetJWT(ctx context.Context, state, jwtStr string) error {
	_, err := r.db.ExecCtx(ctx,
		"UPDATE auth_states SET jwt = ? WHERE state = ? AND expires_at > NOW()",
		jwtStr, state)
	return err
}

// Get returns the auth state, cleaning up expired records first.
// Returns sqlx.ErrNotFound if the state does not exist or has expired.
func (r *AuthStateRepo) Get(ctx context.Context, state string) (*AuthState, error) {
	_, _ = r.db.ExecCtx(ctx, "DELETE FROM auth_states WHERE expires_at < NOW()")
	var as AuthState
	err := r.db.QueryRowCtx(ctx, &as,
		"SELECT state, jwt, expires_at FROM auth_states WHERE state = ?", state)
	if err != nil {
		return nil, err
	}
	return &as, nil
}

// Delete removes an auth state after it has been consumed.
func (r *AuthStateRepo) Delete(ctx context.Context, state string) error {
	_, err := r.db.ExecCtx(ctx, "DELETE FROM auth_states WHERE state = ?", state)
	return err
}
```

- [ ] **Step 6: Run tests (unit compile check; integration will skip)**

```bash
cd server && go test ./internal/repo/...
```

Expected: `ok  hls.dev/server/internal/repo  (skipping integration tests — no TEST_DB_DSN)`

- [ ] **Step 7: Commit**

```bash
git add server/internal/repo/
git commit -m "feat: add UserRepo and AuthStateRepo"
```

---

## Task 6: Auth HTTP Handlers

**Files:**
- Create: `server/internal/handler/auth.go`

- [ ] **Step 1: Write failing test**

Create `server/internal/handler/auth_test.go`:

```go
package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestQrStatusExpired(t *testing.T) {
	// QrStatusHandler with a nil AuthStateRepo is safe to call with an empty state.
	// We just verify the route returns JSON and not a panic.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/qr/status?state=", nil)
	w := httptest.NewRecorder()

	// Pass a nil svc — handler must check for empty state before accessing deps.
	handler.QrStatusHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["ok"] != false {
		t.Fatalf("body[ok] = %v, want false", body["ok"])
	}
}
```

- [ ] **Step 2: Run — expect compile failure**

```bash
cd server && go test ./internal/handler/...
```

Expected: compile error — `QrStatusHandler` does not exist.

- [ ] **Step 3: Implement auth.go**

Create `server/internal/handler/auth.go`:

```go
package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/svc"
)

// QrStartResponse is returned by GET /api/auth/wechat/qr/start.
type QrStartResponse struct {
	State     string `json:"state"`
	IframeURL string `json:"iframe_url"`
}

// QrStatusResponse is returned by GET /api/auth/wechat/qr/status.
type QrStatusResponse struct {
	Status string `json:"status"` // "pending" | "done" | "expired"
	Token  string `json:"token,omitempty"`
}

// MeResponse is returned by GET /api/auth/me.
type MeResponse struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// QrStartHandler handles GET /api/auth/wechat/qr/start.
func QrStartHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
			return
		}
		state := uuid.New().String()
		if err := ctx.AuthStateRepo.Create(r.Context(), state); err != nil {
			writeError(w, r, http.StatusInternalServerError, "create state", err)
			return
		}
		callbackURL := fmt.Sprintf("https://%s/api/auth/wechat/qr/callback",
			ctx.Config.WechatOpen.CallbackDomain)
		iframeURL := fmt.Sprintf(
			"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s",
			ctx.Config.WechatOpen.AppID,
			url.QueryEscape(callbackURL),
			state,
		)
		httpx.OkJsonCtx(r.Context(), w, QrStartResponse{State: state, IframeURL: iframeURL})
	}
}

// QrCallbackHandler handles GET /api/auth/wechat/qr/callback (WeChat redirects here after scan).
func QrCallbackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		frontendURL := ctx.Config.WechatOpen.FrontendURL

		if code == "" || state == "" {
			http.Redirect(w, r, frontendURL+"/login?error=missing_params", http.StatusFound)
			return
		}

		tokResp, err := ctx.WechatClient.ExchangeCode(code)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("wechat exchange code", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=wechat_code_invalid", http.StatusFound)
			return
		}

		userInfo, err := ctx.WechatClient.FetchUserInfo(tokResp.AccessToken, tokResp.OpenID)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("wechat fetch userinfo", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=wechat_timeout", http.StatusFound)
			return
		}

		user, err := ctx.UserRepo.UpsertByWechat(r.Context(),
			tokResp.OpenID, tokResp.UnionID, userInfo.Nickname, userInfo.HeadImgURL)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("upsert user", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=server_error", http.StatusFound)
			return
		}

		jwtStr, err := ctx.JWTSigner.Sign(user.ID)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("sign jwt", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=server_error", http.StatusFound)
			return
		}

		if err := ctx.AuthStateRepo.SetJWT(r.Context(), state, jwtStr); err != nil {
			logx.WithContext(r.Context()).Warnw("set jwt in state", logx.Field("err", err))
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding-top:60px"><p>✅ 登录成功，页面正在跳转…</p></body></html>`)
	}
}

// QrStatusHandler handles GET /api/auth/wechat/qr/status?state=xxx.
func QrStatusHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if state == "" {
			writeError(w, r, http.StatusBadRequest, "missing state", nil)
			return
		}

		as, err := ctx.AuthStateRepo.Get(r.Context(), state)
		if err == sqlx.ErrNotFound {
			httpx.OkJsonCtx(r.Context(), w, QrStatusResponse{Status: "expired"})
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "get state", err)
			return
		}
		if !as.JWT.Valid || as.JWT.String == "" {
			httpx.OkJsonCtx(r.Context(), w, QrStatusResponse{Status: "pending"})
			return
		}

		tok := as.JWT.String
		_ = ctx.AuthStateRepo.Delete(r.Context(), state)
		httpx.OkJsonCtx(r.Context(), w, QrStatusResponse{Status: "done", Token: tok})
	}
}

// MeHandler handles GET /api/auth/me — requires Authorization: Bearer <jwt>.
func MeHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, r, http.StatusUnauthorized, "missing token", nil)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := ctx.JWTSigner.Verify(tokenStr)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "invalid token", err)
			return
		}

		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err == sqlx.ErrNotFound {
			writeError(w, r, http.StatusUnauthorized, "user not found", nil)
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "find user", err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, MeResponse{
			ID:        user.ID,
			Nickname:  user.Nickname,
			AvatarURL: user.AvatarURL,
		})
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd server && go test ./internal/handler/...
```

Expected: `ok  hls.dev/server/internal/handler`

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/auth.go server/internal/handler/auth_test.go
git commit -m "feat: add WeChat QR auth handlers (qr/start, qr/callback, qr/status, me)"
```

---

## Task 7: Wire ServiceContext + Register Routes

**Files:**
- Modify: `server/internal/svc/service_context.go`
- Modify: `server/cmd/gameserver/main.go`
- Modify: `server/etc/gameserver.yaml`

- [ ] **Step 1: Update service_context.go**

Replace `server/internal/svc/service_context.go` with:

```go
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

// ServiceContext is the shared dependency container for HTTP/WS handlers.
type ServiceContext struct {
	Config     config.Config
	Hub        *room.Hub
	Generator  *code.Generator
	Signer     *token.Signer
	RootCtx    context.Context
	RootCancel context.CancelFunc

	// Auth dependencies
	UserRepo      *repo.UserRepo
	AuthStateRepo *repo.AuthStateRepo
	JWTSigner     *auth.JWTSigner
	WechatClient  *auth.WechatClient
}

// NewServiceContext builds all dependencies.
func NewServiceContext(cfg config.Config) *ServiceContext {
	rootCtx, cancel := context.WithCancel(context.Background())

	// Game room deps (unchanged)
	gen := code.NewGenerator()
	ttl := time.Duration(cfg.Session.TTLMinutes) * time.Minute
	signer := token.NewSigner(cfg.Session.Secret, ttl)
	hub := room.NewHub(rootCtx, gen, signer, logx.WithContext(rootCtx))

	// sqlx.NewMysql registers go-sql-driver/mysql internally — no blank import needed.
	conn := sqlx.NewMysql(cfg.DB.DSN)

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
	}
}

// Stop shuts down dependencies gracefully.
func (s *ServiceContext) Stop() {
	if s.RootCancel != nil {
		s.RootCancel()
	}
}
```

- [ ] **Step 2: Register the 4 new routes in main.go**

In `server/cmd/gameserver/main.go`, add to the `server.AddRoutes` call:

```go
server.AddRoutes([]rest.Route{
    {Method: http.MethodPost, Path: "/api/rooms", Handler: handler.CreateRoomHandler(ctx)},
    {Method: http.MethodGet, Path: "/api/rooms/:code", Handler: handler.QueryRoomHandler(ctx)},
    {Method: http.MethodGet, Path: "/api/healthz", Handler: handler.HealthzHandler(ctx)},
    {Method: http.MethodGet, Path: "/ws/rooms/:code", Handler: handler.WebSocketHandler(ctx)},
    // Auth
    {Method: http.MethodGet, Path: "/api/auth/wechat/qr/start", Handler: handler.QrStartHandler(ctx)},
    {Method: http.MethodGet, Path: "/api/auth/wechat/qr/callback", Handler: handler.QrCallbackHandler(ctx)},
    {Method: http.MethodGet, Path: "/api/auth/wechat/qr/status", Handler: handler.QrStatusHandler(ctx)},
    {Method: http.MethodGet, Path: "/api/auth/me", Handler: handler.MeHandler(ctx)},
})
```

- [ ] **Step 3: Update etc/gameserver.yaml**

Add to `server/etc/gameserver.yaml` (append after the existing Cors block):

```yaml
DB:
  DSN: "root:change-me@tcp(localhost:3306)/hls?parseTime=true"

JWT:
  Secret: please-change-me-in-prod
  TTLDays: 7

WechatOpen:
  AppID: ""
  AppSecret: ""
  CallbackDomain: "localhost:8088"
  FrontendURL: "http://localhost:5173"
```

- [ ] **Step 4: Build to verify it compiles**

```bash
cd server && go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run all Go tests**

```bash
cd server && go test ./...
```

Expected: all existing tests pass; repo integration tests are skipped.

- [ ] **Step 6: Commit**

```bash
git add server/internal/svc/service_context.go server/cmd/gameserver/main.go server/etc/gameserver.yaml
git commit -m "feat: wire auth deps into ServiceContext and register auth routes"
```

---

## Task 8: Frontend — axios + http.ts + auth store

**Files:**
- Modify: `package.json` (via npm install)
- Create: `src/http.ts`
- Create: `src/stores/auth.ts`
- Modify: `src/main.ts`

- [ ] **Step 1: Install axios**

```bash
cd /Users/kratos/hls && npm install axios
```

Expected: axios appears in `package.json` dependencies.

- [ ] **Step 2: Create src/http.ts**

```typescript
import axios from 'axios'

const http = axios.create({
  baseURL: (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '/api',
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('hls_auth_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('hls_auth_token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  },
)

export default http
```

- [ ] **Step 3: Create src/stores/auth.ts**

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import http from '../http'

export interface AuthUser {
  id: number
  nickname: string
  avatar_url: string
}

const TOKEN_KEY = 'hls_auth_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<AuthUser | null>(null)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
  }

  async function init() {
    if (!token.value) return
    try {
      const res = await http.get<AuthUser>('/auth/me')
      user.value = res.data
    } catch {
      token.value = null
      localStorage.removeItem(TOKEN_KEY)
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    window.location.href = '/login'
  }

  return { token, user, setToken, init, logout }
})
```

- [ ] **Step 4: Update src/main.ts to call auth.init() before mount**

Replace `src/main.ts` with:

```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import '@shared/styles/main.css'
import { useAuthStore } from './stores/auth'

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)
app.use(router)

router.isReady().then(async () => {
  const auth = useAuthStore()
  await auth.init()
  app.mount('#app')
})
```

- [ ] **Step 5: Typecheck**

```bash
cd /Users/kratos/hls && npm run typecheck
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add package.json package-lock.json src/http.ts src/stores/auth.ts src/main.ts
git commit -m "feat: add axios http client and Pinia auth store"
```

---

## Task 9: Frontend — QR Composable + Login Page

**Files:**
- Create: `src/composables/useWechatQR.ts`
- Create: `src/pages/LoginPage.vue`

- [ ] **Step 1: Create src/composables/useWechatQR.ts**

```typescript
import { ref, onUnmounted } from 'vue'
import http from '../http'

type QrStatus = 'loading' | 'pending' | 'done' | 'expired' | 'error'

interface QrStartResp {
  state: string
  iframe_url: string
}

interface QrStatusResp {
  status: 'pending' | 'done' | 'expired'
  token?: string
}

export function useWechatQR(onToken: (token: string) => void) {
  const iframeUrl = ref('')
  const status = ref<QrStatus>('loading')
  let timer: ReturnType<typeof setInterval> | null = null

  function stopPolling() {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  async function start() {
    stopPolling()
    status.value = 'loading'
    iframeUrl.value = ''
    try {
      const res = await http.get<QrStartResp>('/auth/wechat/qr/start')
      iframeUrl.value = res.data.iframe_url
      const state = res.data.state
      status.value = 'pending'

      timer = setInterval(async () => {
        try {
          const r = await http.get<QrStatusResp>(`/auth/wechat/qr/status?state=${state}`)
          if (r.data.status === 'done' || r.data.status === 'expired') {
            stopPolling()
            status.value = r.data.status
          }
          if (r.data.status === 'done' && r.data.token) {
            onToken(r.data.token)
          }
        } catch {
          stopPolling()
          status.value = 'error'
        }
      }, 2000)
    } catch {
      status.value = 'error'
    }
  }

  onUnmounted(() => stopPolling())

  return { iframeUrl, status, start }
}
```

- [ ] **Step 2: Create src/pages/LoginPage.vue**

```vue
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useWechatQR } from '../composables/useWechatQR'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const errorMsg = computed(() => {
  const e = route.query.error as string | undefined
  if (e === 'wechat_code_invalid') return '微信授权已过期，请重新扫码'
  if (e === 'wechat_timeout') return '微信服务超时，请稍后重试'
  if (e === 'server_error') return '服务器异常，请稍后重试'
  if (e === 'missing_token') return '登录凭证缺失，请重试'
  return ''
})

const { iframeUrl, status, start } = useWechatQR(async (token) => {
  auth.setToken(token)
  await auth.init()
  router.push('/')
})

onMounted(() => start())
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <p v-if="errorMsg" class="mb-6 text-red-400 text-sm">{{ errorMsg }}</p>

    <template v-if="status === 'expired'">
      <p class="text-parchment-400/60 text-sm mb-4">二维码已过期</p>
      <button class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300" @click="start">
        刷新二维码
      </button>
    </template>

    <template v-else-if="status === 'error'">
      <p class="text-red-400 text-sm mb-4">加载失败，请重试</p>
      <button class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300" @click="start">
        重试
      </button>
    </template>

    <template v-else>
      <div class="border border-white/10 rounded-lg p-3 bg-white">
        <iframe
          v-if="iframeUrl"
          :src="iframeUrl"
          scrolling="no"
          frameborder="0"
          sandbox="allow-scripts allow-top-navigation allow-same-origin"
          style="width: 300px; height: 400px; display: block"
        />
        <div
          v-else
          style="width: 300px; height: 400px"
          class="flex items-center justify-center"
        >
          <span class="text-gray-400 text-sm">加载中…</span>
        </div>
      </div>
      <p class="mt-4 font-body text-xs text-parchment-400/50 tracking-widest">
        使用微信扫码登录 · 花栗鼠俱乐部
      </p>
    </template>
  </div>
</template>
```

- [ ] **Step 3: Typecheck**

```bash
cd /Users/kratos/hls && npm run typecheck
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/composables/useWechatQR.ts src/pages/LoginPage.vue
git commit -m "feat: add QR polling composable and LoginPage"
```

---

## Task 10: Frontend — Router + AuthCallbackPage

**Files:**
- Create: `src/pages/AuthCallbackPage.vue`
- Modify: `src/router/index.ts`

- [ ] **Step 1: Create src/pages/AuthCallbackPage.vue**

This page is for the future H5 WeChat OAuth flow (after domain 备案). It reads a `?token=` query param and stores it.

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

onMounted(async () => {
  const token = route.query.token as string | undefined
  if (token) {
    auth.setToken(token)
    await auth.init()
    router.replace('/')
  } else {
    router.replace('/login?error=missing_token')
  }
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center">
    <p class="text-parchment-400/60 text-sm tracking-wider">正在跳转…</p>
  </div>
</template>
```

- [ ] **Step 2: Update src/router/index.ts**

Replace the file with:

```typescript
import { createRouter, createWebHistory } from 'vue-router'
import shopRoutes from '@/games/shop/router'
import teachRoutes from '@/games/teach/router'
import auditionRoutes from '@/games/audition/router'
import pokerRoutes from '@/games/poker/router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/pages/LandingPage.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
    },
    {
      path: '/auth/callback',
      name: 'auth-callback',
      component: () => import('@/pages/AuthCallbackPage.vue'),
    },
    ...shopRoutes,
    ...teachRoutes,
    ...auditionRoutes,
    ...pokerRoutes,
    { path: '/:catchAll(.*)', redirect: '/' },
  ],
})
```

- [ ] **Step 3: Typecheck**

```bash
cd /Users/kratos/hls && npm run typecheck
```

Expected: no errors.

- [ ] **Step 4: Build to verify**

```bash
cd /Users/kratos/hls && npm run build
```

Expected: build succeeds with no errors.

- [ ] **Step 5: Commit**

```bash
git add src/pages/AuthCallbackPage.vue src/router/index.ts
git commit -m "feat: add auth callback page and login/callback routes"
```

---

## Self-Review Checklist (do not skip)

Before declaring implementation complete, verify:

- [ ] All 4 Go auth routes are registered in `main.go`
- [ ] `docker-compose.yml` `gameserver` `depends_on: mysql: condition: service_healthy`
- [ ] `schema.sql` is mounted as `/docker-entrypoint-initdb.d/schema.sql` so MySQL auto-creates tables on first run
- [ ] `sqlx.NewMysql(cfg.DB.DSN)` is used in `ServiceContext` — no blank import needed, driver is registered internally by go-zero
- [ ] Frontend `http.ts` baseURL uses `/api` by default — configure Vite dev proxy or Caddy to route `/api` to the backend
- [ ] `src/main.ts` mounts the app after `auth.init()` resolves, so there is no flash of unauthenticated state on refresh

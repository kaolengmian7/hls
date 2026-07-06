# Config System Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify dev and production configuration under a single `.env`-based format, with a minimal YAML file retained only for go-zero bootstrapping.

**Architecture:** `etc/gameserver.yaml` is stripped to structural go-zero fields only. All business config (DB, JWT, CORS, WeChat, TencentSMS, Shop) moves entirely to env vars. The server auto-loads a `.env` file on startup via godotenv (silent no-op in production where Docker injects vars directly).

**Tech Stack:** Go, go-zero, github.com/joho/godotenv, Docker Compose

---

## File Map

| File | Change |
|---|---|
| `server/docker-compose.yml` | Remove dead SESSION_* vars; switch gameserver to `env_file: .env` |
| `server/.env.example` | Replace with complete canonical var list |
| `server/internal/config/env.go` | Add TencentSMS + Shop env mappings; add `splitCSVInt64` helper |
| `server/internal/config/env_test.go` | Add tests for new TencentSMS + Shop mappings |
| `server/etc/gameserver.yaml` | Slim to go-zero bootstrap fields only |
| `server/cmd/gameserver/main.go` | Add `godotenv.Load()` before `flag.Parse()` |
| `server/go.mod` / `go.sum` | Add `github.com/joho/godotenv` |
| `server/.gitignore` | Create with `.env` entry |

---

## Task 1: Remove dead SESSION_* variables from docker-compose.yml

**Files:**
- Modify: `server/docker-compose.yml`

- [ ] **Step 1: Remove the two dead environment entries from the gameserver service**

In `server/docker-compose.yml`, find the `gameserver.environment` block and delete these two lines:

```yaml
      HLS_SESSION_SECRET: ${HLS_SESSION_SECRET:?set HLS_SESSION_SECRET in server/.env}
      HLS_SESSION_TTL_MINUTES: ${HLS_SESSION_TTL_MINUTES:-1440}
```

The `gameserver.environment` block should now be:

```yaml
    environment:
      HLS_HOST: 0.0.0.0
      HLS_MODE: ${HLS_MODE:-prod}
      HLS_CORS_ALLOW_ORIGINS: ${HLS_CORS_ALLOW_ORIGINS:?set HLS_CORS_ALLOW_ORIGINS in server/.env}
      HLS_DB_DSN: ${HLS_DB_DSN:?set HLS_DB_DSN in server/.env}
      HLS_JWT_SECRET: ${HLS_JWT_SECRET:?set HLS_JWT_SECRET in server/.env}
      HLS_JWT_TTL_DAYS: ${HLS_JWT_TTL_DAYS:-7}
      HLS_WECHAT_OPEN_APP_ID: ${HLS_WECHAT_OPEN_APP_ID:?set HLS_WECHAT_OPEN_APP_ID in server/.env}
      HLS_WECHAT_OPEN_APP_SECRET: ${HLS_WECHAT_OPEN_APP_SECRET:?set HLS_WECHAT_OPEN_APP_SECRET in server/.env}
      HLS_WECHAT_OPEN_CALLBACK_DOMAIN: ${HLS_WECHAT_OPEN_CALLBACK_DOMAIN:?set HLS_WECHAT_OPEN_CALLBACK_DOMAIN in server/.env}
      HLS_WECHAT_OPEN_FRONTEND_URL: ${HLS_WECHAT_OPEN_FRONTEND_URL:?set HLS_WECHAT_OPEN_FRONTEND_URL in server/.env}
```

- [ ] **Step 2: Commit**

```bash
git add server/docker-compose.yml
git commit -m "chore(config): remove dead SESSION_SECRET and SESSION_TTL_MINUTES from compose"
```

---

## Task 2: Add TencentSMS env var support (TDD)

**Files:**
- Modify: `server/internal/config/env_test.go`
- Modify: `server/internal/config/env.go`

- [ ] **Step 1: Write the failing test**

Append this function to `server/internal/config/env_test.go`:

```go
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
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd server && go test ./internal/config/... -run TestApplyEnvOverridesTencentSMS -v
```

Expected: `FAIL` — `undefined: envTencentSMSSecretID`

- [ ] **Step 3: Add constants and mappings to env.go**

In `server/internal/config/env.go`, add the new constants to the `const` block (after the existing `envWechatOpen*` constants):

```go
	envTencentSMSSecretID   = "HLS_TENCENT_SMS_SECRET_ID"
	envTencentSMSSecretKey  = "HLS_TENCENT_SMS_SECRET_KEY"
	envTencentSMSAppID      = "HLS_TENCENT_SMS_APP_ID"
	envTencentSMSSignName   = "HLS_TENCENT_SMS_SIGN_NAME"
	envTencentSMSTemplateID = "HLS_TENCENT_SMS_TEMPLATE_ID"
```

Then, inside `ApplyEnvOverrides`, add after the existing `WechatOpen` block (before the `return nil`):

```go
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
```

- [ ] **Step 4: Run test to confirm it passes**

```bash
cd server && go test ./internal/config/... -run TestApplyEnvOverridesTencentSMS -v
```

Expected: `PASS`

- [ ] **Step 5: Run all config tests to confirm no regressions**

```bash
cd server && go test ./internal/config/... -v
```

Expected: all tests `PASS`

- [ ] **Step 6: Commit**

```bash
git add server/internal/config/env.go server/internal/config/env_test.go
git commit -m "feat(config): add TencentSMS env var support"
```

---

## Task 3: Add Shop env var support (TDD)

**Files:**
- Modify: `server/internal/config/env_test.go`
- Modify: `server/internal/config/env.go`

- [ ] **Step 1: Write the failing tests**

Append these two functions to `server/internal/config/env_test.go`:

```go
func TestApplyEnvOverridesShop(t *testing.T) {
	t.Setenv(envShopVIPUsers, "900006001, 900006002")
	t.Setenv(envShopVIPHighBoost, "25.5")

	cfg := Config{}
	if err := ApplyEnvOverrides(&cfg); err != nil {
		t.Fatalf("ApplyEnvOverrides() error = %v", err)
	}

	wantUsers := []int64{900006001, 900006002}
	if !reflect.DeepEqual(cfg.Shop.VIPUsers, wantUsers) {
		t.Fatalf("Shop.VIPUsers = %v, want %v", cfg.Shop.VIPUsers, wantUsers)
	}
	if cfg.Shop.VIPHighBoost != 25.5 {
		t.Fatalf("Shop.VIPHighBoost = %v, want 25.5", cfg.Shop.VIPHighBoost)
	}
}

func TestApplyEnvOverridesShopInvalid(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "non-numeric boost", key: envShopVIPHighBoost, value: "not-a-number"},
		{name: "zero boost", key: envShopVIPHighBoost, value: "0"},
		{name: "negative boost", key: envShopVIPHighBoost, value: "-5"},
		{name: "non-numeric vip users", key: envShopVIPUsers, value: "123,abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			if err := ApplyEnvOverrides(&Config{}); err == nil {
				t.Fatalf("ApplyEnvOverrides() error = nil, want error for %s=%s", tt.key, tt.value)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd server && go test ./internal/config/... -run "TestApplyEnvOverridesShop" -v
```

Expected: `FAIL` — `undefined: envShopVIPUsers`

- [ ] **Step 3: Add constants, mappings, and helper to env.go**

In `server/internal/config/env.go`, add constants after the `envTencentSMS*` block:

```go
	envShopVIPUsers     = "HLS_SHOP_VIP_USERS"
	envShopVIPHighBoost = "HLS_SHOP_VIP_HIGH_BOOST"
```

Inside `ApplyEnvOverrides`, add after the TencentSMS block (before `return nil`):

```go
	if vipUsers := strings.TrimSpace(os.Getenv(envShopVIPUsers)); vipUsers != "" {
		users, err := splitCSVInt64(vipUsers)
		if err != nil {
			return errors.New(errors.CodeInvalidParam, "%s must be comma-separated integers", envShopVIPUsers)
		}
		cfg.Shop.VIPUsers = users
	}
	if boost := strings.TrimSpace(os.Getenv(envShopVIPHighBoost)); boost != "" {
		value, err := strconv.ParseFloat(boost, 64)
		if err != nil || value <= 0 {
			return errors.New(errors.CodeInvalidParam, "%s must be a positive number", envShopVIPHighBoost)
		}
		cfg.Shop.VIPHighBoost = value
	}
```

At the bottom of the file, add a new helper after `splitCSV`:

```go
func splitCSVInt64(value string) ([]int64, error) {
	parts := strings.Split(value, ",")
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			v, err := strconv.ParseInt(trimmed, 10, 64)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd server && go test ./internal/config/... -run "TestApplyEnvOverridesShop" -v
```

Expected: all Shop tests `PASS`

- [ ] **Step 5: Run all config tests to confirm no regressions**

```bash
cd server && go test ./internal/config/... -v
```

Expected: all tests `PASS`

- [ ] **Step 6: Commit**

```bash
git add server/internal/config/env.go server/internal/config/env_test.go
git commit -m "feat(config): add Shop env var support (VIPUsers, VIPHighBoost)"
```

---

## Task 4: Replace .env.example with complete canonical version

**Files:**
- Modify: `server/.env.example`

- [ ] **Step 1: Replace the file content**

Overwrite `server/.env.example` with:

```env
# ── Caddy (caddy container only; gameserver does not read this) ───────────────
HLS_API_DOMAIN=api.hls-club.com

# ── MySQL (mysql container only) ─────────────────────────────────────────────
MYSQL_ROOT_PASSWORD=replace-with-a-long-random-password

# ── Server ────────────────────────────────────────────────────────────────────
HLS_MODE=prod
HLS_CORS_ALLOW_ORIGINS=https://hls-club.com,https://www.hls-club.com
HLS_DB_DSN=root:replace-with-a-long-random-password@tcp(mysql:3306)/hls?parseTime=true

HLS_JWT_SECRET=replace-with-a-long-random-secret
HLS_JWT_TTL_DAYS=7

HLS_WECHAT_OPEN_APP_ID=wxYOUR_APP_ID
HLS_WECHAT_OPEN_APP_SECRET=YOUR_APP_SECRET
HLS_WECHAT_OPEN_CALLBACK_DOMAIN=api.hls-club.com
HLS_WECHAT_OPEN_FRONTEND_URL=https://hls-club.com

# Leave empty if Tencent SMS is not in use
HLS_TENCENT_SMS_SECRET_ID=
HLS_TENCENT_SMS_SECRET_KEY=
HLS_TENCENT_SMS_APP_ID=
HLS_TENCENT_SMS_SIGN_NAME=
HLS_TENCENT_SMS_TEMPLATE_ID=

# Comma-separated int64 user IDs; float for boost multiplier
HLS_SHOP_VIP_USERS=900006000
HLS_SHOP_VIP_HIGH_BOOST=30
```

- [ ] **Step 2: Commit**

```bash
git add server/.env.example
git commit -m "chore(config): update .env.example — remove dead SESSION vars, add TencentSMS + Shop"
```

---

## Task 5: Simplify docker-compose.yml to use env_file

**Files:**
- Modify: `server/docker-compose.yml`

- [ ] **Step 1: Replace the gameserver service's environment block**

In `server/docker-compose.yml`, replace the entire `gameserver` service definition's `environment:` block with `env_file` + a minimal `environment:`. The gameserver service should look like this (all other fields unchanged):

```yaml
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
    env_file: .env
    environment:
      HLS_HOST: 0.0.0.0
    expose:
      - "8088"
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:8088/api/healthz >/dev/null || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 3
      start_period: 10s
    stop_grace_period: 20s
```

> **Note:** `HLS_HOST: 0.0.0.0` stays in `environment:` because it is a fixed container-level value, not a per-deployment secret. The `env_file: .env` picks up all the rest from `server/.env` on the host.

- [ ] **Step 2: Verify compose file is valid**

```bash
cd server && docker compose config --quiet
```

Expected: no output (valid), no errors.

- [ ] **Step 3: Commit**

```bash
git add server/docker-compose.yml
git commit -m "chore(config): simplify compose — gameserver uses env_file instead of explicit env block"
```

---

## Task 6: Slim down gameserver.yaml + add godotenv

**Files:**
- Modify: `server/etc/gameserver.yaml`
- Modify: `server/cmd/gameserver/main.go`
- Modify: `server/go.mod`, `server/go.sum`
- Create: `server/.gitignore`

- [ ] **Step 1: Add godotenv dependency**

```bash
cd server && go get github.com/joho/godotenv
```

Expected: `go: added github.com/joho/godotenv v1.x.x`

- [ ] **Step 2: Add godotenv.Load() to main.go**

In `server/cmd/gameserver/main.go`, add the import and the load call.

Add `"github.com/joho/godotenv"` to the import block (in the external imports group):

```go
import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"

	"hls.dev/server/internal/config"
	"hls.dev/server/internal/handler"
	"hls.dev/server/internal/svc"
)
```

Then add `godotenv.Load()` as the first line inside `main()`, before `flag.Parse()`:

```go
func main() {
	godotenv.Load() // loads .env if present; silent no-op when not found
	flag.Parse()
	// ... rest unchanged
```

- [ ] **Step 3: Slim down gameserver.yaml**

Replace the entire content of `server/etc/gameserver.yaml` with:

```yaml
Name: gameserver
Host: 0.0.0.0
Port: 8088
Mode: dev
Timeout: 30000

Log:
  Mode: console
  Encoding: plain
  Level: info
```

All business config fields (`Cors`, `DB`, `JWT`, `WechatOpen`, `TencentSMS`, `Shop`) are removed — they come from env vars only.

- [ ] **Step 4: Create server/.gitignore**

```bash
echo '.env' > server/.gitignore
```

- [ ] **Step 5: Verify the server builds**

```bash
cd server && go build ./cmd/gameserver
```

Expected: exits 0, produces `gameserver` binary.

- [ ] **Step 6: Run all tests**

```bash
cd server && go test ./...
```

Expected: all tests pass.

- [ ] **Step 7: Clean up build artifact**

```bash
rm server/gameserver
```

- [ ] **Step 8: Commit**

```bash
git add server/etc/gameserver.yaml server/cmd/gameserver/main.go server/go.mod server/go.sum server/.gitignore
git commit -m "feat(config): slim YAML to go-zero bootstrap; add godotenv for local .env loading"
```

---

## Local Dev Setup (for reference)

After the refactor, a developer's first-time setup is:

```bash
cp server/.env.example server/.env
# Edit server/.env — set local values:
#   HLS_MODE=dev
#   HLS_CORS_ALLOW_ORIGINS=http://localhost:5173
#   HLS_DB_DSN=root@tcp(localhost:3306)/hls?parseTime=true
#   HLS_JWT_SECRET=any-dev-secret
#   HLS_WECHAT_OPEN_CALLBACK_DOMAIN=localhost:8088
#   HLS_WECHAT_OPEN_FRONTEND_URL=http://localhost:5173
cd server && go run ./cmd/gameserver
```

The server picks up `server/.env` automatically. Production Docker containers set vars via `env_file: .env` in `docker-compose.yml`; `godotenv.Load()` finds no file and is silent.

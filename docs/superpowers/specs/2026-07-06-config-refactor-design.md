# Config System Refactor — Design Spec

**Date:** 2026-07-06  
**Status:** Approved

## Problem

The server currently uses two inconsistent configuration formats:

- **Dev**: `server/etc/gameserver.yaml` — loaded by go-zero's `conf.MustLoad`, contains all fields including sensitive defaults (JWT secret placeholder, DB DSN)
- **Prod**: `server/.env` (from `.env.example`) — env vars injected by Docker Compose

The two formats have drifted:
- `HLS_SESSION_SECRET` and `HLS_SESSION_TTL_MINUTES` exist in `.env.example` and `docker-compose.yml` but are wired to nothing in Go code (dead variables left over from the session token removal)
- `TencentSMS` config has no env var support — cannot be configured in production without editing the YAML
- `Shop` config (`VIPUsers`, `VIPHighBoost`) is YAML-only
- Adding a new config field requires touching three places: `config.go`, `gameserver.yaml`, and `env.go` + `.env.example`

## Goal

A single env-var-based config format used in both environments. Developers work with a local `.env` file; production uses Docker-injected env vars. The format and available variables are identical in both cases.

## Architecture

Configuration is split into two layers:

| Layer | File | Responsibility | In git |
|---|---|---|---|
| Structural | `etc/gameserver.yaml` | go-zero bootstrap only: Name/Host/Port/Mode/Timeout/Log | ✅ |
| Runtime | env vars | All business config: DB, JWT, CORS, WeChat, SMS, Shop | ❌ |
| Template | `server/.env.example` | Documents all available env vars; serves as template | ✅ |
| Local actual | `server/.env` | Developer's local values | ❌ (gitignored) |

**Local dev flow**: copy `.env.example` → `.env`, fill in local values, run `go run ./cmd/gameserver`. The server auto-loads `.env` on startup via `godotenv.Load()`.

**Production flow**: Docker Compose uses `env_file: .env` to inject vars into the container. The container has no `.env` file; `godotenv.Load()` is a no-op. Business config comes entirely from container env vars.

## File Changes

### `etc/gameserver.yaml` — stripped to go-zero bootstrap only

All business config fields (`DB`, `JWT`, `Cors`, `WechatOpen`, `TencentSMS`, `Shop`) are removed. Only the fields needed to initialize `rest.RestConf` remain:

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

### `server/.env.example` — canonical env var reference

Replaces the current file. Covers all variables (including previously missing TencentSMS and Shop), removes dead variables (`HLS_SESSION_SECRET`, `HLS_SESSION_TTL_MINUTES`).

```env
# ── Caddy (caddy container only; gameserver does not read this) ──────────────
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

HLS_TENCENT_SMS_SECRET_ID=
HLS_TENCENT_SMS_SECRET_KEY=
HLS_TENCENT_SMS_APP_ID=
HLS_TENCENT_SMS_SIGN_NAME=
HLS_TENCENT_SMS_TEMPLATE_ID=

# Comma-separated int64 list
HLS_SHOP_VIP_USERS=900006000
HLS_SHOP_VIP_HIGH_BOOST=30
```

### `server/.gitignore` — add `.env`

Append `.env` so local dev values are never committed.

### `cmd/gameserver/main.go` — add godotenv.Load

Add one call before `flag.Parse()`:

```go
import "github.com/joho/godotenv"

godotenv.Load() // loads .env if present; silent no-op in production
```

`godotenv.Load()` with no arguments only loads `.env` if it exists, and does **not** override env vars already set in the process environment. Production containers have no `.env` file, so this is a true no-op.

### `internal/config/env.go` — complete env var coverage

**Add** mappings for all previously uncovered fields:

| Env var | Go field |
|---|---|
| `HLS_TENCENT_SMS_SECRET_ID` | `TencentSMS.SecretID` |
| `HLS_TENCENT_SMS_SECRET_KEY` | `TencentSMS.SecretKey` |
| `HLS_TENCENT_SMS_APP_ID` | `TencentSMS.SmsSdkAppID` |
| `HLS_TENCENT_SMS_SIGN_NAME` | `TencentSMS.SignName` |
| `HLS_TENCENT_SMS_TEMPLATE_ID` | `TencentSMS.TemplateID` |
| `HLS_SHOP_VIP_USERS` | `Shop.VIPUsers` (comma-separated int64) |
| `HLS_SHOP_VIP_HIGH_BOOST` | `Shop.VIPHighBoost` (float64) |

**Remove** dead constants and any references to `HLS_SESSION_SECRET` / `HLS_SESSION_TTL_MINUTES` (these mapped to nothing and the session token concept was already removed).

### `docker-compose.yml` — use env_file, remove duplicate vars

Switch `gameserver` service to use `env_file: .env` instead of listing every variable individually in `environment:`. This means the compose file no longer needs to be updated when env vars are added or renamed.

`HLS_HOST: 0.0.0.0` remains in `environment:` because it is a container-specific fixed value, not a secret or per-deployment setting.

Remove `HLS_SESSION_SECRET` and `HLS_SESSION_TTL_MINUTES` from the compose file.

## Migration Steps

The refactor is broken into three independent, verifiable steps:

1. **Clean up dead variables**: remove `HLS_SESSION_SECRET` and `HLS_SESSION_TTL_MINUTES` from `docker-compose.yml` and `.env.example`.

2. **Complete env var coverage**: add TencentSMS and Shop mappings to `env.go`; update `.env.example` to include them; update `env_test.go`.

3. **Slim down YAML + add godotenv**: remove business config fields from `gameserver.yaml`; add `godotenv.Load()` to `main.go`; add `.env` to `.gitignore`; add `github.com/joho/godotenv` to `go.mod` via `go get`.

Each step leaves the system in a working state and can be reviewed independently.

## Testing

- `env_test.go` is updated to cover the new TencentSMS and Shop env mappings.
- `gameserver.yaml` change is verified by running the server locally with a `.env` file and confirming all config is applied correctly.
- Production path is verified by confirming `godotenv.Load()` is a no-op when no `.env` file is present (standard godotenv behavior).

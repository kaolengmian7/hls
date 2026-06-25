# Phone SMS Login Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the frontend WeChat QR login with a phone number + SMS verification code flow, while keeping all backend WeChat login endpoints intact.

**Architecture:** Backend adds two new endpoints (`/api/auth/sms/send`, `/api/auth/sms/verify`) backed by an in-memory store for codes and rate-limit counters. Frontend replaces `LoginPage.vue` with a two-step form (phone → code) guarded by a pure-frontend slider captcha.

**Tech Stack:** Go 1.25 · go-zero · tencentcloud-sdk-go · Vue 3 · TypeScript · Pinia · Tailwind CSS

---

## File Map

| Operation | Path | Purpose |
|-----------|------|---------|
| Modify | `server/db/schema.sql` | Add `phone` column |
| Modify | `server/internal/repo/user.go` | Add `Phone` field + `UpsertByPhone` |
| Create | `server/internal/auth/sms_sender.go` | `SmsSender` interface + Tencent impl |
| Create | `server/internal/svc/sms_store.go` | In-memory code store + rate limiter |
| Create | `server/internal/svc/sms_store_test.go` | Unit tests for SmsStore |
| Create | `server/internal/handler/sms.go` | Send + Verify handlers |
| Create | `server/internal/handler/sms_test.go` | Handler validation tests |
| Modify | `server/internal/config/config.go` | Add `TencentSMS` config struct |
| Modify | `server/etc/gameserver.yaml` | Add `TencentSMS` block |
| Modify | `server/internal/svc/service_context.go` | Wire `SmsStore` + `SmsSender` |
| Modify | `server/cmd/gameserver/main.go` | Register two new routes |
| Create | `src/shared/components/SliderCaptcha.vue` | Drag-to-pass captcha component |
| Create | `src/composables/usePhoneLogin.ts` | Two-step login logic composable |
| Modify | `src/pages/LoginPage.vue` | Replace WeChat QR with phone form |

---

## Task 1: DB schema + UserRepo phone support

**Files:**
- Modify: `server/db/schema.sql`
- Modify: `server/internal/repo/user.go`

- [ ] **Step 1: Add phone column to schema**

In `server/db/schema.sql`, append after the existing `CREATE TABLE IF NOT EXISTS users` block:

```sql
-- Run this migration on existing databases:
-- ALTER TABLE users ADD COLUMN phone VARCHAR(20) NULL UNIQUE AFTER avatar_url;
```

Then update the `CREATE TABLE` statement to include the phone column so it matches:

```sql
CREATE TABLE IF NOT EXISTS users (
  id           BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  mp_openid    VARCHAR(64)  NULL UNIQUE,
  open_openid  VARCHAR(64)  NULL UNIQUE,
  union_id     VARCHAR(64)  NULL UNIQUE,
  phone        VARCHAR(20)  NULL UNIQUE,
  nickname     VARCHAR(64)  NOT NULL DEFAULT '',
  avatar_url   VARCHAR(512) NOT NULL DEFAULT '',
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

- [ ] **Step 2: Run the migration against the dev database**

```bash
cd server
mysql -u root hls -e "ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20) NULL UNIQUE AFTER avatar_url;"
```

Expected: no error (or "Duplicate column name" if already present — both are fine).

- [ ] **Step 3: Add `Phone` field to User struct and update userCols**

In `server/internal/repo/user.go`, replace:

```go
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
```

With:

```go
// User represents a registered user.
type User struct {
	ID         int64          `db:"id"`
	OpenOpenID sql.NullString `db:"open_openid"`
	MpOpenID   sql.NullString `db:"mp_openid"`
	UnionID    sql.NullString `db:"union_id"`
	Phone      sql.NullString `db:"phone"`
	Nickname   string         `db:"nickname"`
	AvatarURL  string         `db:"avatar_url"`
}

const userCols = "id, open_openid, mp_openid, union_id, phone, nickname, avatar_url"
```

- [ ] **Step 4: Add `randSuffix` helper and `UpsertByPhone` + `insertByPhone` to UserRepo**

At the bottom of `server/internal/repo/user.go`, add:

```go
// UpsertByPhone finds an existing user by phone or creates a new one.
func (r *UserRepo) UpsertByPhone(ctx context.Context, phone string) (*User, error) {
	u, err := r.findBy(ctx, "phone", phone)
	if err == nil {
		return u, nil
	}
	if err != sqlx.ErrNotFound {
		return nil, err
	}
	nickname := "小小花栗鼠" + randSuffix()
	return r.insertByPhone(ctx, phone, nickname)
}

func (r *UserRepo) insertByPhone(ctx context.Context, phone, nickname string) (*User, error) {
	result, err := r.db.ExecCtx(ctx,
		"INSERT INTO users (phone, nickname, avatar_url) VALUES (?, ?, '')",
		phone, nickname)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func randSuffix() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	out := make([]byte, 4)
	for i, c := range b {
		out[i] = chars[int(c)%len(chars)]
	}
	return string(out)
}
```

Also add `"crypto/rand"` to the import block at the top of `user.go`.

- [ ] **Step 5: Build to verify compilation**

```bash
cd server
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
cd server
git add db/schema.sql internal/repo/user.go
git commit -m "feat: add phone column and UpsertByPhone to UserRepo"
```

---

## Task 2: SmsSender interface + Tencent Cloud implementation

**Files:**
- Create: `server/internal/auth/sms_sender.go`
- Modify: `server/go.mod`, `server/go.sum` (via `go get`)

- [ ] **Step 1: Add tencentcloud-sdk-go dependency**

```bash
cd server
go get github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common@latest
go get github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111@latest
```

Expected: go.mod and go.sum updated with tencentcloud entries.

- [ ] **Step 2: Create `server/internal/auth/sms_sender.go`**

```go
package auth

import (
	"context"
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/credentials"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"

	"hls.dev/server/internal/config"
)

// SmsSender sends a single SMS verification code to a phone number.
type SmsSender interface {
	Send(ctx context.Context, phone, code string) error
}

// TencentSmsSender sends SMS via Tencent Cloud SMS service.
type TencentSmsSender struct {
	client      *sms.Client
	sdkAppID    string
	signName    string
	templateID  string
}

// NewTencentSmsSender creates a TencentSmsSender from config.
func NewTencentSmsSender(cfg config.TencentSMSConfig) *TencentSmsSender {
	cred := credentials.NewCredential(cfg.SecretID, cfg.SecretKey)
	cpf := profile.NewClientProfile()
	client, _ := sms.NewClient(cred, "ap-guangzhou", cpf)
	return &TencentSmsSender{
		client:     client,
		sdkAppID:   cfg.SmsSdkAppID,
		signName:   cfg.SignName,
		templateID: cfg.TemplateID,
	}
}

// Send sends a 6-digit code to the given phone number (without country prefix).
func (s *TencentSmsSender) Send(_ context.Context, phone, code string) error {
	req := sms.NewSendSmsRequest()
	req.SmsSdkAppId = common.StringPtr(s.sdkAppID)
	req.SignName = common.StringPtr(s.signName)
	req.TemplateId = common.StringPtr(s.templateID)
	req.TemplateParamSet = common.StringPtrs([]string{code, "10"})
	req.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})

	resp, err := s.client.SendSms(req)
	if err != nil {
		return fmt.Errorf("tencent sms send: %w", err)
	}
	for _, st := range resp.Response.SendStatusSet {
		if st.Code == nil || *st.Code != "Ok" {
			msg := ""
			if st.Message != nil {
				msg = *st.Message
			}
			code := "unknown"
			if st.Code != nil {
				code = *st.Code
			}
			return fmt.Errorf("tencent sms status %s: %s", code, msg)
		}
	}
	return nil
}
```

- [ ] **Step 3: Build to verify compilation**

```bash
cd server
go build ./...
```

Expected: no errors. If tencentcloud imports are missing, re-run the `go get` from Step 1.

- [ ] **Step 4: Commit**

```bash
cd server
git add internal/auth/sms_sender.go go.mod go.sum
git commit -m "feat: add SmsSender interface and TencentSmsSender"
```

---

## Task 3: In-memory SmsStore

**Files:**
- Create: `server/internal/svc/sms_store.go`
- Create: `server/internal/svc/sms_store_test.go`

- [ ] **Step 1: Write the failing tests first**

Create `server/internal/svc/sms_store_test.go`:

```go
package svc_test

import (
	"testing"
	"time"

	"hls.dev/server/internal/svc"
)

func TestSmsStore_CheckAndSend_Success(t *testing.T) {
	s := svc.NewSmsStore()
	code, retryAfter, errKey := s.CheckAndSend("13800138000", "1.2.3.4", 5, 10, 200)
	if errKey != "" {
		t.Fatalf("errKey = %q, want \"\"", errKey)
	}
	if retryAfter != 0 {
		t.Fatalf("retryAfter = %d, want 0", retryAfter)
	}
	if len(code) != 6 {
		t.Fatalf("code len = %d, want 6", len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("code %q contains non-digit", code)
		}
	}
}

func TestSmsStore_CheckAndSend_CooldownImmediately(t *testing.T) {
	s := svc.NewSmsStore()
	s.CheckAndSend("13800138001", "1.2.3.4", 5, 10, 200)
	_, retryAfter, errKey := s.CheckAndSend("13800138001", "1.2.3.4", 5, 10, 200)
	if errKey != "cooldown" {
		t.Fatalf("errKey = %q, want \"cooldown\"", errKey)
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %d, want > 0", retryAfter)
	}
}

func TestSmsStore_CheckAndSend_PhoneLimit(t *testing.T) {
	// Use clock injection: advance 61s between sends to bypass cooldown.
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := svc.NewSmsStoreWithClock(func() time.Time { return now })

	// maxPhone=2: first two sends succeed.
	for i := 0; i < 2; i++ {
		_, _, errKey := s.CheckAndSend("13800138002", "1.2.3.4", 2, 10, 200)
		if errKey != "" {
			t.Fatalf("send %d: errKey = %q, want \"\"", i+1, errKey)
		}
		now = now.Add(61 * time.Second)
	}
	_, _, errKey := s.CheckAndSend("13800138002", "1.2.3.4", 2, 10, 200)
	if errKey != "phone_limit" {
		t.Fatalf("errKey = %q, want \"phone_limit\"", errKey)
	}
}

func TestSmsStore_CheckAndSend_IPLimit(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := svc.NewSmsStoreWithClock(func() time.Time { return now })

	// Different phones, same IP, maxIP=2.
	phones := []string{"13800138010", "13800138011", "13800138012"}
	for i, ph := range phones[:2] {
		_, _, errKey := s.CheckAndSend(ph, "9.9.9.9", 5, 2, 200)
		if errKey != "" {
			t.Fatalf("send %d: errKey = %q, want \"\"", i+1, errKey)
		}
		now = now.Add(61 * time.Second)
	}
	_, _, errKey := s.CheckAndSend(phones[2], "9.9.9.9", 5, 2, 200)
	if errKey != "ip_limit" {
		t.Fatalf("errKey = %q, want \"ip_limit\"", errKey)
	}
}

func TestSmsStore_CheckAndSend_GlobalLimit(t *testing.T) {
	s := svc.NewSmsStore()
	// maxGlobal=1: second request (different phone) is blocked.
	s.CheckAndSend("13800138020", "1.2.3.4", 5, 10, 1)
	_, _, errKey := s.CheckAndSend("13800138021", "1.2.3.5", 5, 10, 1)
	if errKey != "global_limit" {
		t.Fatalf("errKey = %q, want \"global_limit\"", errKey)
	}
}

func TestSmsStore_Verify_Success(t *testing.T) {
	s := svc.NewSmsStore()
	code, _, _ := s.CheckAndSend("13800138030", "1.2.3.4", 5, 10, 200)
	errKey := s.Verify("13800138030", code)
	if errKey != "" {
		t.Fatalf("errKey = %q, want \"\"", errKey)
	}
}

func TestSmsStore_Verify_WrongCode(t *testing.T) {
	s := svc.NewSmsStore()
	s.CheckAndSend("13800138031", "1.2.3.4", 5, 10, 200)
	errKey := s.Verify("13800138031", "000000")
	if errKey != "invalid_code" {
		t.Fatalf("errKey = %q, want \"invalid_code\"", errKey)
	}
}

func TestSmsStore_Verify_MaxFailsInvalidates(t *testing.T) {
	s := svc.NewSmsStore()
	s.CheckAndSend("13800138032", "1.2.3.4", 5, 10, 200)
	for i := 0; i < 5; i++ {
		s.Verify("13800138032", "000000")
	}
	// After 5 failures, code is deleted — subsequent verify returns invalid_code too.
	errKey := s.Verify("13800138032", "000000")
	if errKey != "invalid_code" {
		t.Fatalf("errKey = %q, want \"invalid_code\"", errKey)
	}
}

func TestSmsStore_Verify_Expired(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := svc.NewSmsStoreWithClock(func() time.Time { return now })
	code, _, _ := s.CheckAndSend("13800138033", "1.2.3.4", 5, 10, 200)
	now = now.Add(11 * time.Minute) // past 10-minute TTL
	errKey := s.Verify("13800138033", code)
	if errKey != "code_expired" {
		t.Fatalf("errKey = %q, want \"code_expired\"", errKey)
	}
}

func TestSmsStore_Verify_DeletedAfterSuccess(t *testing.T) {
	s := svc.NewSmsStore()
	code, _, _ := s.CheckAndSend("13800138034", "1.2.3.4", 5, 10, 200)
	s.Verify("13800138034", code)
	// Second verify with same code must fail.
	errKey := s.Verify("13800138034", code)
	if errKey != "invalid_code" {
		t.Fatalf("errKey = %q, want \"invalid_code\"", errKey)
	}
}
```

- [ ] **Step 2: Run tests — expect failure (svc package missing SmsStore)**

```bash
cd server
go test ./internal/svc/... -v -run TestSmsStore
```

Expected: compilation error — `svc.NewSmsStore` undefined.

- [ ] **Step 3: Create `server/internal/svc/sms_store.go`**

```go
package svc

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

type codeEntry struct {
	code      string
	expiresAt time.Time
	sentAt    time.Time
	failCount int
}

// SmsStore is an in-memory store for SMS verification codes and rate-limit counters.
// All state resets on process restart. Counters reset at midnight UTC+8 each day.
type SmsStore struct {
	nowFn      func() time.Time
	mu         sync.Mutex
	codes      map[string]*codeEntry // key: phone
	phoneCount map[string]int         // key: "phone|YYYYMMDD"
	ipCount    map[string]int         // key: "ip|YYYYMMDD"
	globalDay  string
	globalCnt  int
}

// NewSmsStore creates an empty SmsStore using real time.
func NewSmsStore() *SmsStore {
	return &SmsStore{
		nowFn:      time.Now,
		codes:      make(map[string]*codeEntry),
		phoneCount: make(map[string]int),
		ipCount:    make(map[string]int),
	}
}

// NewSmsStoreWithClock creates an SmsStore using the provided clock. Use in tests only.
func NewSmsStoreWithClock(nowFn func() time.Time) *SmsStore {
	s := NewSmsStore()
	s.nowFn = nowFn
	return s
}

func (s *SmsStore) dayKey() string {
	return s.nowFn().In(time.FixedZone("CST", 8*3600)).Format("20060102")
}

// CheckAndSend performs all rate-limit checks and, if they pass, generates and stores
// a 6-digit code. Returns (code, retryAfter, errKey). errKey is "" on success.
func (s *SmsStore) CheckAndSend(phone, ip string, maxPhone, maxIP, maxGlobal int) (code string, retryAfter int, errKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	day := s.dayKey()

	// global daily hard cap
	if s.globalDay != day {
		s.globalCnt = 0
		s.globalDay = day
	}
	if s.globalCnt >= maxGlobal {
		return "", 0, "global_limit"
	}

	// 60-second cooldown per phone
	if e, ok := s.codes[phone]; ok {
		remaining := int(e.sentAt.Add(60 * time.Second).Sub(s.nowFn()).Seconds())
		if remaining > 0 {
			return "", remaining, "cooldown"
		}
	}

	// phone daily limit
	pk := phone + "|" + day
	if s.phoneCount[pk] >= maxPhone {
		return "", 0, "phone_limit"
	}

	// IP daily limit
	ik := ip + "|" + day
	if s.ipCount[ik] >= maxIP {
		return "", 0, "ip_limit"
	}

	newCode, err := genSmsCode()
	if err != nil {
		return "", 0, "internal"
	}

	s.codes[phone] = &codeEntry{
		code:      newCode,
		expiresAt: s.nowFn().Add(10 * time.Minute),
		sentAt:    s.nowFn(),
	}
	s.phoneCount[pk]++
	s.ipCount[ik]++
	s.globalCnt++

	return newCode, 0, ""
}

// Verify checks the provided code for the given phone.
// Returns "" on success, or "invalid_code" / "code_expired" on failure.
func (s *SmsStore) Verify(phone, code string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.codes[phone]
	if !ok {
		return "invalid_code"
	}
	if s.nowFn().After(e.expiresAt) {
		delete(s.codes, phone)
		return "code_expired"
	}
	if e.code != code {
		e.failCount++
		if e.failCount >= 5 {
			delete(s.codes, phone)
		}
		return "invalid_code"
	}
	delete(s.codes, phone)
	return ""
}

func genSmsCode() (string, error) {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
```

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd server
go test ./internal/svc/... -v -run TestSmsStore
```

Expected:
```
--- PASS: TestSmsStore_CheckAndSend_Success (0.00s)
--- PASS: TestSmsStore_CheckAndSend_CooldownImmediately (0.00s)
--- PASS: TestSmsStore_CheckAndSend_PhoneLimit (0.00s)
--- PASS: TestSmsStore_CheckAndSend_IPLimit (0.00s)
--- PASS: TestSmsStore_CheckAndSend_GlobalLimit (0.00s)
--- PASS: TestSmsStore_Verify_Success (0.00s)
--- PASS: TestSmsStore_Verify_WrongCode (0.00s)
--- PASS: TestSmsStore_Verify_MaxFailsInvalidates (0.00s)
--- PASS: TestSmsStore_Verify_Expired (0.00s)
--- PASS: TestSmsStore_Verify_DeletedAfterSuccess (0.00s)
PASS
```

- [ ] **Step 5: Commit**

```bash
cd server
git add internal/svc/sms_store.go internal/svc/sms_store_test.go
git commit -m "feat: add in-memory SmsStore with rate limiting and code verification"
```

---

## Task 4: SMS handlers

**Files:**
- Create: `server/internal/handler/sms.go`
- Create: `server/internal/handler/sms_test.go`

- [ ] **Step 1: Write the failing handler validation tests**

Create `server/internal/handler/sms_test.go`:

```go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
	"hls.dev/server/internal/svc"
)

func minSmsCtx() *svc.ServiceContext {
	return &svc.ServiceContext{
		SmsStore: svc.NewSmsStore(),
	}
}

func TestSmsSendHandler_MissingBody_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/send", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsSendHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSmsSendHandler_InvalidPhone_ReturnsBadRequest(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"phone": "12345", "captcha": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsSendHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_phone" {
		t.Fatalf("error = %v, want \"invalid_phone\"", resp["error"])
	}
}

func TestSmsSendHandler_MissingCaptcha_ReturnsBadRequest(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"phone": "13800138000"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsSendHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSmsVerifyHandler_MissingFields_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/verify", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsVerifyHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
```

- [ ] **Step 2: Run tests — expect compilation failure**

```bash
cd server
go test ./internal/handler/... -v -run TestSms
```

Expected: compilation error — `handler.SmsSendHandler` undefined.

- [ ] **Step 3: Create `server/internal/handler/sms.go`**

```go
package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/svc"
)

var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

const (
	smsMaxPhone  = 5
	smsMaxIP     = 10
	smsMaxGlobal = 200
)

type smsSendRequest struct {
	Phone   string `json:"phone"`
	Captcha string `json:"captcha"`
}

type smsVerifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// SmsSendHandler handles POST /api/auth/sms/send.
func SmsSendHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req smsSendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeSmsError(w, http.StatusBadRequest, "invalid_body", 0)
			return
		}
		req.Phone = strings.TrimSpace(req.Phone)
		if !phoneRe.MatchString(req.Phone) {
			writeSmsError(w, http.StatusBadRequest, "invalid_phone", 0)
			return
		}
		if strings.TrimSpace(req.Captcha) == "" {
			writeSmsError(w, http.StatusBadRequest, "missing_captcha", 0)
			return
		}

		ip := clientIP(r)
		code, retryAfter, errKey := ctx.SmsStore.CheckAndSend(req.Phone, ip, smsMaxPhone, smsMaxIP, smsMaxGlobal)
		if errKey != "" {
			status := http.StatusTooManyRequests
			if errKey == "global_limit" {
				status = http.StatusServiceUnavailable
			}
			writeSmsError(w, status, errKey, retryAfter)
			return
		}

		if err := ctx.SmsSender.Send(r.Context(), req.Phone, code); err != nil {
			logx.WithContext(r.Context()).Errorw("sms send failed", logx.Field("err", err))
			writeSmsError(w, http.StatusInternalServerError, "send_failed", 0)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	}
}

// SmsVerifyHandler handles POST /api/auth/sms/verify.
func SmsVerifyHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req smsVerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeSmsError(w, http.StatusBadRequest, "invalid_body", 0)
			return
		}
		req.Phone = strings.TrimSpace(req.Phone)
		req.Code = strings.TrimSpace(req.Code)
		if !phoneRe.MatchString(req.Phone) || req.Code == "" {
			writeSmsError(w, http.StatusBadRequest, "invalid_body", 0)
			return
		}

		if errKey := ctx.SmsStore.Verify(req.Phone, req.Code); errKey != "" {
			writeSmsError(w, http.StatusBadRequest, errKey, 0)
			return
		}

		user, err := ctx.UserRepo.UpsertByPhone(r.Context(), req.Phone)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("upsert user by phone", logx.Field("err", err))
			writeSmsError(w, http.StatusInternalServerError, "server_error", 0)
			return
		}

		token, err := ctx.JWTSigner.Sign(user.ID)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("sign jwt", logx.Field("err", err))
			writeSmsError(w, http.StatusInternalServerError, "server_error", 0)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, map[string]any{"token": token})
	}
}

func writeSmsError(w http.ResponseWriter, status int, errKey string, retryAfter int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := map[string]any{"error": errKey}
	if retryAfter > 0 {
		resp["retry_after"] = retryAfter
	}
	_ = json.NewEncoder(w).Encode(resp)
}
```

Note: `clientIP` is already defined in `server/internal/handler/http.go` — do **not** redefine it in `sms.go`. If it is not exported, verify it is accessible within the same package (it is, since both files are `package handler`).

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd server
go test ./internal/handler/... -v -run TestSms
```

Expected:
```
--- PASS: TestSmsSendHandler_MissingBody_ReturnsBadRequest (0.00s)
--- PASS: TestSmsSendHandler_InvalidPhone_ReturnsBadRequest (0.00s)
--- PASS: TestSmsSendHandler_MissingCaptcha_ReturnsBadRequest (0.00s)
--- PASS: TestSmsVerifyHandler_MissingFields_ReturnsBadRequest (0.00s)
PASS
```

- [ ] **Step 5: Commit**

```bash
cd server
git add internal/handler/sms.go internal/handler/sms_test.go
git commit -m "feat: add SmsSendHandler and SmsVerifyHandler"
```

---

## Task 5: Config, ServiceContext, and route registration

**Files:**
- Modify: `server/internal/config/config.go`
- Modify: `server/etc/gameserver.yaml`
- Modify: `server/internal/svc/service_context.go`
- Modify: `server/cmd/gameserver/main.go`

- [ ] **Step 1: Add `TencentSMSConfig` to config.go**

In `server/internal/config/config.go`, add the struct and field:

```go
// TencentSMSConfig 腾讯云短信配置。
type TencentSMSConfig struct {
	SecretID    string `json:",optional"`
	SecretKey   string `json:",optional"`
	SmsSdkAppID string `json:",optional"`
	SignName    string `json:",optional"`
	TemplateID  string `json:",optional"`
}
```

And add `TencentSMS TencentSMSConfig` to the `Config` struct:

```go
type Config struct {
	rest.RestConf

	Session    SessionConfig
	Cors       CorsConfig
	DB         DBConfig
	JWT        JWTConfig
	WechatOpen WechatOpenConfig
	TencentSMS TencentSMSConfig
}
```

- [ ] **Step 2: Add TencentSMS block to gameserver.yaml**

In `server/etc/gameserver.yaml`, append:

```yaml
TencentSMS:
  SecretID: ""
  SecretKey: ""
  SmsSdkAppID: ""
  SignName: ""
  TemplateID: ""
```

- [ ] **Step 3: Wire SmsStore and SmsSender into ServiceContext**

In `server/internal/svc/service_context.go`, add the two new fields:

```go
type ServiceContext struct {
	Config     config.Config
	Hub        *room.Hub
	Generator  *code.Generator
	Signer     *token.Signer
	RootCtx    context.Context
	RootCancel context.CancelFunc

	AuthStateRepo *repo.AuthStateRepo
	UserRepo      *repo.UserRepo
	JWTSigner     *auth.JWTSigner
	WechatClient  *auth.WechatClient
	SmsStore      *SmsStore
	SmsSender     auth.SmsSender
}
```

Then in `NewServiceContext`, after the existing wiring, add:

```go
SmsStore:      NewSmsStore(),
SmsSender:     auth.NewTencentSmsSender(cfg.TencentSMS),
```

- [ ] **Step 4: Register new routes in main.go**

In `server/cmd/gameserver/main.go`, add to the `AddRoutes` slice:

```go
{Method: http.MethodPost, Path: "/api/auth/sms/send", Handler: handler.SmsSendHandler(ctx)},
{Method: http.MethodPost, Path: "/api/auth/sms/verify", Handler: handler.SmsVerifyHandler(ctx)},
```

- [ ] **Step 5: Build and run all tests**

```bash
cd server
go build ./...
go test ./... -count=1
```

Expected: build succeeds, all tests pass.

- [ ] **Step 6: Commit**

```bash
cd server
git add internal/config/config.go etc/gameserver.yaml internal/svc/service_context.go cmd/gameserver/main.go
git commit -m "feat: wire SMS config, SmsStore, SmsSender, and register routes"
```

---

## Task 6: SliderCaptcha.vue

**Files:**
- Create: `src/shared/components/SliderCaptcha.vue`

- [ ] **Step 1: Create the component**

Create `src/shared/components/SliderCaptcha.vue`:

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'

const emit = defineEmits<{ (e: 'pass'): void }>()

const TRACK_W = 280
const KNOB_W = 44
const MAX_X = TRACK_W - KNOB_W

const passed = ref(false)
const shaking = ref(false)
const sliderX = ref(0)

const fillWidth = computed(() => sliderX.value + KNOB_W)

function onPointerDown(e: PointerEvent) {
  if (passed.value) return
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  const startClientX = e.clientX - sliderX.value

  function onMove(ev: PointerEvent) {
    const x = Math.min(Math.max(0, ev.clientX - startClientX), MAX_X)
    sliderX.value = x
    if (x / MAX_X >= 0.9) {
      sliderX.value = MAX_X
      passed.value = true
      emit('pass')
    }
  }

  function onUp() {
    if (!passed.value) sliderX.value = 0
    ;(e.currentTarget as HTMLElement)?.removeEventListener('pointermove', onMove)
    ;(e.currentTarget as HTMLElement)?.removeEventListener('pointerup', onUp)
  }

  ;(e.currentTarget as HTMLElement).addEventListener('pointermove', onMove)
  ;(e.currentTarget as HTMLElement).addEventListener('pointerup', onUp)
}

function shake() {
  if (shaking.value) return
  shaking.value = true
  setTimeout(() => { shaking.value = false }, 500)
}

function reset() {
  passed.value = false
  sliderX.value = 0
}

defineExpose({ shake, reset, passed })
</script>

<template>
  <div :class="['relative select-none', { shake: shaking }]" :style="{ width: TRACK_W + 'px' }">
    <div class="relative rounded border border-white/20 bg-white/5 overflow-hidden" style="height: 44px;">
      <!-- fill bar -->
      <div
        class="absolute left-0 top-0 h-full pointer-events-none transition-colors duration-200"
        :class="passed ? 'bg-green-500/30' : 'bg-gold-400/20'"
        :style="{ width: fillWidth + 'px' }"
      />
      <!-- label -->
      <span
        class="absolute inset-0 flex items-center justify-center text-xs font-serif tracking-widest pointer-events-none"
        :class="passed ? 'text-green-400' : 'text-parchment-400/50'"
      >
        {{ passed ? '验证通过 ✓' : '拖动滑块完成验证' }}
      </span>
      <!-- knob -->
      <div
        class="absolute top-0 h-full flex items-center justify-center border-r border-white/20 bg-white/10 cursor-grab active:cursor-grabbing touch-none"
        :style="{ width: KNOB_W + 'px', left: sliderX + 'px' }"
        @pointerdown.prevent="onPointerDown"
      >
        <span class="text-gold-300 text-base select-none">›</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.shake {
  animation: captcha-shake 0.45s ease-in-out;
}
@keyframes captcha-shake {
  0%, 100% { transform: translateX(0); }
  20%       { transform: translateX(-8px); }
  45%       { transform: translateX(8px); }
  65%       { transform: translateX(-5px); }
  85%       { transform: translateX(4px); }
}
</style>
```

- [ ] **Step 2: Build frontend to verify no TypeScript errors**

```bash
cd /Users/kratos/hls
npx vue-tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/shared/components/SliderCaptcha.vue
git commit -m "feat: add SliderCaptcha pure-frontend drag verification component"
```

---

## Task 7: usePhoneLogin composable

**Files:**
- Create: `src/composables/usePhoneLogin.ts`

- [ ] **Step 1: Create the composable**

Create `src/composables/usePhoneLogin.ts`:

```ts
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import http from '@/http'

export type LoginStep = 'phone' | 'code'

interface SendResp {
  ok?: boolean
  error?: string
  retry_after?: number
}

interface VerifyResp {
  token?: string
  error?: string
}

const ERROR_LABELS: Record<string, string> = {
  phone_limit:  '该号码今日发送已达上限',
  ip_limit:     '操作过于频繁，请明日再试',
  global_limit: '服务繁忙，请明日再试',
  invalid_code: '验证码错误，请重新输入',
  code_expired: '验证码已过期，请重新发送',
  send_failed:  '短信发送失败，请稍后重试',
}

export function usePhoneLogin() {
  const router = useRouter()
  const auth = useAuthStore()

  const step = ref<LoginStep>('phone')
  const phone = ref('')
  const code = ref('')
  const cooldown = ref(0)
  const sending = ref(false)
  const verifying = ref(false)
  const error = ref('')

  let cooldownTimer: ReturnType<typeof setInterval> | null = null

  function startCooldown(seconds: number) {
    cooldown.value = seconds
    if (cooldownTimer) clearInterval(cooldownTimer)
    cooldownTimer = setInterval(() => {
      cooldown.value = Math.max(0, cooldown.value - 1)
      if (cooldown.value === 0) {
        clearInterval(cooldownTimer!)
        cooldownTimer = null
      }
    }, 1000)
  }

  async function sendCode(captchaPassed: boolean) {
    error.value = ''
    if (!captchaPassed) return
    sending.value = true
    try {
      await http.post<SendResp>('/auth/sms/send', {
        phone: phone.value,
        captcha: 'passed',
      })
      step.value = 'code'
      startCooldown(60)
    } catch (e: unknown) {
      const data = (e as { response?: { data?: SendResp } }).response?.data
      if (data?.error === 'cooldown' && data.retry_after) {
        startCooldown(data.retry_after)
        error.value = `请 ${data.retry_after} 秒后再试`
      } else {
        error.value = ERROR_LABELS[data?.error ?? ''] ?? '发送失败，请稍后重试'
      }
    } finally {
      sending.value = false
    }
  }

  async function verifyCode() {
    error.value = ''
    verifying.value = true
    try {
      const res = await http.post<VerifyResp>('/auth/sms/verify', {
        phone: phone.value,
        code: code.value,
      })
      if (res.data.token) {
        auth.setToken(res.data.token)
        await auth.init()
        router.push('/')
      }
    } catch (e: unknown) {
      const data = (e as { response?: { data?: VerifyResp } }).response?.data
      error.value = ERROR_LABELS[data?.error ?? ''] ?? '登录失败，请重试'
    } finally {
      verifying.value = false
    }
  }

  function maskedPhone(p: string): string {
    if (p.length !== 11) return p
    return p.slice(0, 3) + '****' + p.slice(7)
  }

  return {
    step,
    phone,
    code,
    cooldown,
    sending,
    verifying,
    error,
    sendCode,
    verifyCode,
    maskedPhone,
  }
}
```

- [ ] **Step 2: Build frontend to verify no TypeScript errors**

```bash
cd /Users/kratos/hls
npx vue-tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/composables/usePhoneLogin.ts
git commit -m "feat: add usePhoneLogin composable for SMS verification flow"
```

---

## Task 8: Replace LoginPage.vue

**Files:**
- Modify: `src/pages/LoginPage.vue`

- [ ] **Step 1: Replace the file content**

Overwrite `src/pages/LoginPage.vue` with:

```vue
<script setup lang="ts">
import { ref } from 'vue'
import SliderCaptcha from '@shared/components/SliderCaptcha.vue'
import { usePhoneLogin } from '../composables/usePhoneLogin'

const captchaRef = ref<InstanceType<typeof SliderCaptcha> | null>(null)
const captchaPassed = ref(false)

const {
  step,
  phone,
  code,
  cooldown,
  sending,
  verifying,
  error,
  sendCode,
  verifyCode,
  maskedPhone,
} = usePhoneLogin()

async function handleSend() {
  if (!captchaPassed.value) {
    captchaRef.value?.shake()
    return
  }
  await sendCode(captchaPassed.value)
  if (step.value === 'code') {
    captchaRef.value?.reset()
    captchaPassed.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <div class="w-full max-w-xs space-y-4">

      <p v-if="error" class="text-red-400 text-sm text-center">{{ error }}</p>

      <!-- Step 1: enter phone -->
      <template v-if="step === 'phone'">
        <SliderCaptcha
          ref="captchaRef"
          class="mx-auto"
          @pass="captchaPassed = true"
        />

        <input
          v-model="phone"
          type="tel"
          maxlength="11"
          placeholder="请输入手机号"
          class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        />

        <button
          class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          :disabled="sending || phone.length !== 11"
          @click="handleSend"
        >
          {{ sending ? '发送中…' : '发送验证码' }}
        </button>
      </template>

      <!-- Step 2: enter code -->
      <template v-else>
        <p class="text-center font-body text-sm text-parchment-400/60">
          验证码已发至 {{ maskedPhone(phone) }}
        </p>

        <input
          v-model="code"
          type="text"
          inputmode="numeric"
          maxlength="6"
          placeholder="请输入6位验证码"
          class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60 tracking-[0.4em] text-center"
        />

        <button
          class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          :disabled="verifying || code.length !== 6"
          @click="verifyCode"
        >
          {{ verifying ? '验证中…' : '登录' }}
        </button>

        <button
          class="w-full font-serif text-xs py-1 transition-colors"
          :class="cooldown > 0 ? 'text-parchment-400/30 cursor-not-allowed' : 'text-parchment-400/50 hover:text-gold-300'"
          :disabled="cooldown > 0"
          @click="step = 'phone'"
        >
          {{ cooldown > 0 ? `重新发送 (${cooldown}s)` : '重新发送' }}
        </button>
      </template>

    </div>
  </div>
</template>
```

- [ ] **Step 2: Build frontend to verify no TypeScript errors**

```bash
cd /Users/kratos/hls
npx vue-tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Start dev server and test the login page manually**

```bash
cd /Users/kratos/hls
npm run dev
```

Open `http://localhost:5173/login`. Verify:
- [ ] Slider renders and responds to drag
- [ ] "发送验证码" button stays disabled when phone < 11 digits
- [ ] Slider shake animation fires when clicking "发送验证码" without completing slider
- [ ] After completing slider + entering phone, button becomes clickable
- [ ] On send success (requires working backend), transitions to code step
- [ ] Masked phone shows correctly (`138****XXXX`)
- [ ] "重新发送" shows countdown, then becomes clickable
- [ ] "登录" is disabled while code < 6 characters

- [ ] **Step 4: Commit**

```bash
git add src/pages/LoginPage.vue
git commit -m "feat: replace WeChat QR login with phone SMS verification flow"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Covered in |
|-----------------|-----------|
| Remove WeChat QR from frontend | Task 8 (LoginPage.vue replaced) |
| Keep backend WeChat endpoints | Not touched — ✓ |
| `POST /api/auth/sms/send` | Tasks 4, 5 |
| `POST /api/auth/sms/verify` | Tasks 4, 5 |
| Tencent Cloud SMS | Task 2 |
| 6-digit code, 10-min TTL, 60-sec cooldown | Task 3 (SmsStore) |
| Per-phone 5/day | Task 3 + Task 4 (constants) |
| Per-IP 10/day | Task 3 + Task 4 (constants) |
| Global 200/day | Task 3 + Task 4 (constants) |
| Max 5 wrong attempts invalidates code | Task 3 (Verify) |
| In-memory storage | Task 3 |
| Auto-nickname "小小花栗鼠XXXX" | Task 1 (randSuffix) |
| phone column in DB | Task 1 |
| SliderCaptcha (pure frontend) | Task 6 |
| usePhoneLogin composable | Task 7 |
| Two-step form UI | Task 8 |
| Error messages per spec | Tasks 7+8 |

All requirements covered. No gaps found.

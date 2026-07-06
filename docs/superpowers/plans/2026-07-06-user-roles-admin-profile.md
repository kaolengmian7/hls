# User Roles, Admin Panel & Profile Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four user roles (owner/service/enforcer/customer), replace phone login with username+password login (username auto-generated as `hls`+4 random chars), add `/admin` user management page (owner only), and add `/profile` personal settings page.

**Architecture:** Extend the existing `users` table with `username`, `password_hash`, and `role` columns. New Go handlers cover login, admin CRUD, and user self-service. Vue frontend gets a rewritten login page plus two new pages, all guarded by role checks in the router.

**Tech Stack:** Go/go-zero (backend), `golang.org/x/crypto/bcrypt` (new dep), Vue 3 + Pinia + Tailwind (frontend), MySQL

---

## File Map

**Create:**
- `server/migrations/001_add_username_password_role.sql` — DB migration
- `server/internal/handler/admin.go` — ListUsers, CreateUser, ResetPassword, ResetNickname handlers
- `server/internal/handler/admin_test.go` — handler unit tests
- `server/internal/handler/user.go` — UpdateNickname, UpdatePassword handlers
- `server/internal/handler/user_test.go` — handler unit tests
- `src/pages/ProfilePage.vue` — personal settings page
- `src/pages/AdminPage.vue` — admin user management page

**Modify:**
- `server/internal/repo/user.go` — extend User struct, add UserSummary, new repo methods, genUsername
- `server/internal/repo/user_test.go` — add integration tests for new methods
- `server/internal/handler/auth.go` — add LoginHandler, update MeHandler response
- `server/internal/handler/auth_test.go` — add LoginHandler unit tests
- `server/internal/handler/http.go` — add requireOwner helper
- `server/cmd/gameserver/main.go` — register new routes
- `src/stores/auth.ts` — add `username`/`role` to AuthUser, add `login()` method
- `src/pages/LoginPage.vue` — rewrite to username+password form
- `src/router/index.ts` — add `/profile`, `/admin` routes + guards
- `src/shared/components/AuthWidget.vue` — add profile link

---

## Task 1: DB Migration SQL

**Files:**
- Create: `server/migrations/001_add_username_password_role.sql`

- [ ] **Step 1: Create the migration file**

```sql
-- server/migrations/001_add_username_password_role.sql
ALTER TABLE users
  ADD COLUMN username      VARCHAR(7)   UNIQUE NULL,
  ADD COLUMN password_hash VARCHAR(255) NULL,
  ADD COLUMN role          ENUM('owner','service','enforcer','customer')
                           NOT NULL DEFAULT 'customer';
```

- [ ] **Step 2: Run against your local DB**

```bash
mysql -u root hls < server/migrations/001_add_username_password_role.sql
```

Expected: no error. Verify with:
```bash
mysql -u root hls -e "DESCRIBE users;" | grep -E "username|password_hash|role"
```

Expected output includes three new rows.

- [ ] **Step 3: Commit**

```bash
git add server/migrations/001_add_username_password_role.sql
git commit -m "feat(db): add username, password_hash, role columns to users"
```

---

## Task 2: Add bcrypt dependency

**Files:**
- Modify: `server/go.mod`, `server/go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd server && go get golang.org/x/crypto@latest
```

- [ ] **Step 2: Tidy**

```bash
go mod tidy
```

- [ ] **Step 3: Verify**

```bash
grep "golang.org/x/crypto" go.mod
```

Expected: a line like `golang.org/x/crypto v0.x.x`

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add golang.org/x/crypto for bcrypt"
```

---

## Task 3: Extend User struct and add genUsername

**Files:**
- Modify: `server/internal/repo/user.go`

- [ ] **Step 1: Update User struct and userCols**

In `server/internal/repo/user.go`, replace the `User` struct and `userCols` constant:

```go
// User represents a registered user.
type User struct {
	ID           int64          `db:"id"`
	OpenOpenID   sql.NullString `db:"open_openid"`
	MpOpenID     sql.NullString `db:"mp_openid"`
	UnionID      sql.NullString `db:"union_id"`
	Phone        sql.NullString `db:"phone"`
	Nickname     string         `db:"nickname"`
	AvatarURL    string         `db:"avatar_url"`
	Username     sql.NullString `db:"username"`
	PasswordHash sql.NullString `db:"password_hash"`
	Role         string         `db:"role"`
}

// UserSummary is a lightweight projection used in admin listings.
type UserSummary struct {
	ID       int64  `db:"id"       json:"id"`
	Username string `db:"username" json:"username"`
	Nickname string `db:"nickname" json:"nickname"`
	Role     string `db:"role"     json:"role"`
}

const userCols = "id, open_openid, mp_openid, union_id, phone, nickname, avatar_url, username, password_hash, role"
```

- [ ] **Step 2: Add genUsername helper**

Add these to the bottom of `server/internal/repo/user.go` (before the closing of the file):

```go
const usernameChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func genUsername() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, 4)
	for i, c := range b {
		out[i] = usernameChars[int(c)%len(usernameChars)]
	}
	return "hls" + string(out), nil
}

func isDuplicateKey(err error) bool {
	return strings.Contains(err.Error(), "Duplicate entry")
}
```

Also add `"strings"` to the import block (it's not currently imported in user.go).

- [ ] **Step 3: Build to verify no compile errors**

```bash
cd server && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add server/internal/repo/user.go
git commit -m "feat(repo): extend User struct with username/password_hash/role; add genUsername"
```

---

## Task 4: UserRepo — FindByUsername, CreateWithPassword

**Files:**
- Modify: `server/internal/repo/user.go`
- Modify: `server/internal/repo/user_test.go`

- [ ] **Step 1: Write failing integration tests**

Add to `server/internal/repo/user_test.go`:

```go
func TestCreateWithPassword_GeneratesUsername(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.CreateWithPassword(ctx, "测试用户", "hashedpw", "customer")
	if err != nil {
		t.Fatalf("CreateWithPassword() error = %v", err)
	}
	if !u.Username.Valid || len(u.Username.String) != 7 {
		t.Fatalf("username = %q, want 7-char hls* string", u.Username.String)
	}
	if u.Username.String[:3] != "hls" {
		t.Fatalf("username prefix = %q, want 'hls'", u.Username.String[:3])
	}
	if u.Nickname != "测试用户" {
		t.Fatalf("nickname = %q, want '测试用户'", u.Nickname)
	}
	if u.Role != "customer" {
		t.Fatalf("role = %q, want 'customer'", u.Role)
	}

	// cleanup
	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestCreateWithPassword_DefaultNickname(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.CreateWithPassword(ctx, "", "hashedpw", "customer")
	if err != nil {
		t.Fatalf("CreateWithPassword() error = %v", err)
	}
	if u.Nickname != "小小花栗鼠" {
		t.Fatalf("nickname = %q, want '小小花栗鼠'", u.Nickname)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestFindByUsername(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, err := r.CreateWithPassword(ctx, "查找测试", "hashedpw", "owner")
	if err != nil {
		t.Fatalf("setup error = %v", err)
	}

	found, err := r.FindByUsername(ctx, u.Username.String)
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if found.ID != u.ID {
		t.Fatalf("ID = %d, want %d", found.ID, u.ID)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}
```

- [ ] **Step 2: Run tests — expect FAIL (methods not defined)**

```bash
cd server && go test ./internal/repo/... -run "TestCreateWithPassword|TestFindByUsername" -v
```

Expected: compile error "r.CreateWithPassword undefined" (or skip if TEST_DB_DSN not set).

- [ ] **Step 3: Implement FindByUsername and CreateWithPassword**

Add to `server/internal/repo/user.go`:

```go
import "fmt" // add to existing import block

// FindByUsername returns the user with the given username.
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*User, error) {
	return r.findBy(ctx, "username", username)
}

// CreateWithPassword inserts a new user with an auto-generated username and hashed password.
// nickname defaults to "小小花栗鼠" if empty.
func (r *UserRepo) CreateWithPassword(ctx context.Context, nickname, passwordHash, role string) (*User, error) {
	if nickname == "" {
		nickname = "小小花栗鼠"
	}
	for i := 0; i < 5; i++ {
		username, err := genUsername()
		if err != nil {
			return nil, err
		}
		result, err := r.db.ExecCtx(ctx,
			"INSERT INTO users (username, password_hash, role, nickname, avatar_url) VALUES (?, ?, ?, ?, '')",
			username, passwordHash, role, nickname)
		if err != nil {
			if isDuplicateKey(err) {
				continue
			}
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		return r.FindByID(ctx, id)
	}
	return nil, fmt.Errorf("failed to generate unique username after 5 attempts")
}
```

- [ ] **Step 4: Run tests — expect PASS (or SKIP without TEST_DB_DSN)**

```bash
cd server && go test ./internal/repo/... -run "TestCreateWithPassword|TestFindByUsername" -v
```

Expected: PASS or `--- SKIP` with message "TEST_DB_DSN not set".

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/user.go server/internal/repo/user_test.go
git commit -m "feat(repo): add FindByUsername, CreateWithPassword"
```

---

## Task 5: UserRepo — UpdatePassword, UpdateNickname, ListAll

**Files:**
- Modify: `server/internal/repo/user.go`
- Modify: `server/internal/repo/user_test.go`

- [ ] **Step 1: Write failing integration tests**

Add to `server/internal/repo/user_test.go`:

```go
func TestUpdatePassword(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, _ := r.CreateWithPassword(ctx, "密码测试", "oldhash", "customer")

	if err := r.UpdatePassword(ctx, u.ID, "newhash"); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	updated, _ := r.FindByID(ctx, u.ID)
	if updated.PasswordHash.String != "newhash" {
		t.Fatalf("password_hash = %q, want 'newhash'", updated.PasswordHash.String)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestUpdateNickname(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, _ := r.CreateWithPassword(ctx, "昵称测试", "hash", "customer")

	if err := r.UpdateNickname(ctx, u.ID, "新昵称"); err != nil {
		t.Fatalf("UpdateNickname() error = %v", err)
	}
	updated, _ := r.FindByID(ctx, u.ID)
	if updated.Nickname != "新昵称" {
		t.Fatalf("nickname = %q, want '新昵称'", updated.Nickname)
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}

func TestListAll(t *testing.T) {
	conn := testConn(t)
	r := repo.NewUserRepo(conn)
	ctx := context.Background()

	u, _ := r.CreateWithPassword(ctx, "列表测试", "hash", "service")

	users, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	found := false
	for _, s := range users {
		if s.ID == u.ID {
			found = true
			if s.Role != "service" {
				t.Fatalf("role = %q, want 'service'", s.Role)
			}
		}
	}
	if !found {
		t.Fatal("created user not found in ListAll()")
	}

	db, _ := conn.RawDB()
	db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", u.ID)
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./internal/repo/... -run "TestUpdatePassword|TestUpdateNickname|TestListAll" -v
```

Expected: compile error (methods undefined).

- [ ] **Step 3: Implement UpdatePassword, UpdateNickname, ListAll**

Add to `server/internal/repo/user.go`:

```go
// UpdatePassword sets a new password hash for the given user.
func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := r.db.ExecCtx(ctx,
		"UPDATE users SET password_hash = ?, updated_at = NOW() WHERE id = ?",
		passwordHash, id)
	return err
}

// UpdateNickname sets a new nickname for the given user.
func (r *UserRepo) UpdateNickname(ctx context.Context, id int64, nickname string) error {
	_, err := r.db.ExecCtx(ctx,
		"UPDATE users SET nickname = ?, updated_at = NOW() WHERE id = ?",
		nickname, id)
	return err
}

// ListAll returns a summary of all users ordered by id.
func (r *UserRepo) ListAll(ctx context.Context) ([]*UserSummary, error) {
	var users []*UserSummary
	err := r.db.QueryRowsCtx(ctx, &users,
		"SELECT id, COALESCE(username, '') AS username, nickname, role FROM users ORDER BY id")
	if err != nil {
		return nil, err
	}
	return users, nil
}
```

- [ ] **Step 4: Run — expect PASS/SKIP**

```bash
cd server && go test ./internal/repo/... -v
```

Expected: all PASS or SKIP.

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/user.go server/internal/repo/user_test.go
git commit -m "feat(repo): add UpdatePassword, UpdateNickname, ListAll"
```

---

## Task 6: requireOwner helper

**Files:**
- Modify: `server/internal/handler/http.go`

- [ ] **Step 1: Add requireOwner to http.go**

Add this function to the bottom of `server/internal/handler/http.go`:

```go
// requireOwner wraps a handler and rejects non-owner users with 401/403.
func requireOwner(ctx *svc.ServiceContext, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		if user.Role != "owner" {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		next(w, r)
	}
}
```

- [ ] **Step 2: Build to verify**

```bash
cd server && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add server/internal/handler/http.go
git commit -m "feat(handler): add requireOwner middleware helper"
```

---

## Task 7: LoginHandler + update MeHandler

**Files:**
- Modify: `server/internal/handler/auth.go`
- Modify: `server/internal/handler/auth_test.go`

- [ ] **Step 1: Write failing handler unit tests**

Add to `server/internal/handler/auth_test.go`:

```go
import (
	"bytes"
	// existing imports...
)

func TestLoginHandler_EmptyBody_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()

	handler.LoginHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestLoginHandler_MissingFields_ReturnsBadRequest(t *testing.T) {
	body := `{"username":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.LoginHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./internal/handler/... -run "TestLoginHandler" -v
```

Expected: compile error "LoginHandler undefined".

- [ ] **Step 3: Update MeResponse and add LoginHandler to auth.go**

In `server/internal/handler/auth.go`, replace the `MeResponse` struct:

```go
type MeResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}
```

Update `MeHandler` response body (find the `httpx.OkJsonCtx` call in `MeHandler`):

```go
		httpx.OkJsonCtx(r.Context(), w, MeResponse{
			ID:        user.ID,
			Username:  user.Username.String,
			Nickname:  user.Nickname,
			AvatarURL: user.AvatarURL,
			Role:      user.Role,
		})
```

Add the import for bcrypt and the `LoginHandler` function. Add to the import block:

```go
	"golang.org/x/crypto/bcrypt"
```

Add `LoginHandler` to `server/internal/handler/auth.go`:

```go
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginHandler handles POST /api/auth/login.
func LoginHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if req.Username == "" || req.Password == "" {
			writeError(w, r, http.StatusBadRequest, "missing_fields", nil)
			return
		}
		user, err := ctx.UserRepo.FindByUsername(r.Context(), req.Username)
		if errors.Is(err, sqlx.ErrNotFound) {
			writeError(w, r, http.StatusUnauthorized, "invalid_credentials", nil)
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if !user.PasswordHash.Valid {
			writeError(w, r, http.StatusUnauthorized, "invalid_credentials", nil)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(req.Password)); err != nil {
			writeError(w, r, http.StatusUnauthorized, "invalid_credentials", nil)
			return
		}
		token, err := ctx.JWTSigner.Sign(user.ID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "sign_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"token": token})
	}
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd server && go test ./internal/handler/... -run "TestLoginHandler" -v
```

Expected: both tests PASS.

- [ ] **Step 5: Build full project**

```bash
cd server && go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add server/internal/handler/auth.go server/internal/handler/auth_test.go
git commit -m "feat(handler): add LoginHandler; update MeHandler to return username and role"
```

---

## Task 8: Admin handlers

**Files:**
- Create: `server/internal/handler/admin.go`
- Create: `server/internal/handler/admin_test.go`

- [ ] **Step 1: Write failing unit tests**

Create `server/internal/handler/admin_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestListUsersHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()

	handler.ListUsersHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestCreateUserHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", nil)
	w := httptest.NewRecorder()

	handler.CreateUserHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestResetPasswordHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/password", nil)
	w := httptest.NewRecorder()

	handler.ResetPasswordHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestResetNicknameHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/nickname", nil)
	w := httptest.NewRecorder()

	handler.ResetNicknameHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./internal/handler/... -run "TestListUsersHandler|TestCreateUserHandler|TestResetPasswordHandler|TestResetNicknameHandler" -v
```

Expected: compile error (handlers undefined).

- [ ] **Step 3: Create admin.go**

Create `server/internal/handler/admin.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
	"golang.org/x/crypto/bcrypt"

	"hls.dev/server/internal/svc"
)

const defaultPassword = "123456"
const defaultNickname = "小小花栗鼠"

var validRoles = map[string]bool{
	"owner": true, "service": true, "enforcer": true, "customer": true,
}

// ListUsersHandler handles GET /api/admin/users.
func ListUsersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.UserRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, users)
	})
}

type createUserRequest struct {
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// CreateUserHandler handles POST /api/admin/users.
func CreateUserHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		var req createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Nickname = strings.TrimSpace(req.Nickname)
		if !validRoles[req.Role] {
			writeError(w, r, http.StatusBadRequest, "invalid_role", nil)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "hash_error", err)
			return
		}
		user, err := ctx.UserRepo.CreateWithPassword(r.Context(), req.Nickname, string(hash), req.Role)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{
			"id":       user.ID,
			"username": user.Username.String,
			"nickname": user.Nickname,
			"role":     user.Role,
		})
	})
}

// ResetPasswordHandler handles PUT /api/admin/users/:id/password.
func ResetPasswordHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		idStr := extractCode(r.URL.Path, "/api/admin/users/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "hash_error", err)
			return
		}
		if err := ctx.UserRepo.UpdatePassword(r.Context(), id, string(hash)); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

// ResetNicknameHandler handles PUT /api/admin/users/:id/nickname.
func ResetNicknameHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		idStr := extractCode(r.URL.Path, "/api/admin/users/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		if err := ctx.UserRepo.UpdateNickname(r.Context(), id, defaultNickname); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd server && go test ./internal/handler/... -run "TestListUsersHandler|TestCreateUserHandler|TestResetPasswordHandler|TestResetNicknameHandler" -v
```

Expected: all 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/admin.go server/internal/handler/admin_test.go
git commit -m "feat(handler): add admin user management handlers"
```

---

## Task 9: User self-service handlers

**Files:**
- Create: `server/internal/handler/user.go`
- Create: `server/internal/handler/user_test.go`

- [ ] **Step 1: Write failing unit tests**

Create `server/internal/handler/user_test.go`:

```go
package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestUpdateNicknameHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/user/nickname", bytes.NewBufferString(`{"nickname":"new"}`))
	w := httptest.NewRecorder()

	handler.UpdateNicknameHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUpdatePasswordHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/user/password",
		bytes.NewBufferString(`{"old_password":"a","new_password":"b"}`))
	w := httptest.NewRecorder()

	handler.UpdatePasswordHandler(nil)(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./internal/handler/... -run "TestUpdateNicknameHandler|TestUpdatePasswordHandler" -v
```

Expected: compile error (handlers undefined).

- [ ] **Step 3: Create user.go**

Create `server/internal/handler/user.go`:

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"golang.org/x/crypto/bcrypt"

	"hls.dev/server/internal/svc"
)

type updateNicknameRequest struct {
	Nickname string `json:"nickname"`
}

// UpdateNicknameHandler handles PUT /api/user/nickname.
func UpdateNicknameHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		var req updateNicknameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Nickname = strings.TrimSpace(req.Nickname)
		if req.Nickname == "" {
			writeError(w, r, http.StatusBadRequest, "nickname_required", nil)
			return
		}
		if err := ctx.UserRepo.UpdateNickname(r.Context(), userID, req.Nickname); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	}
}

type updatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// UpdatePasswordHandler handles PUT /api/user/password.
func UpdatePasswordHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		var req updatePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		if req.OldPassword == "" || req.NewPassword == "" {
			writeError(w, r, http.StatusBadRequest, "missing_fields", nil)
			return
		}
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if errors.Is(err, sqlx.ErrNotFound) {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if !user.PasswordHash.Valid {
			writeError(w, r, http.StatusBadRequest, "no_password_set", nil)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(req.OldPassword)); err != nil {
			writeError(w, r, http.StatusUnauthorized, "wrong_password", nil)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "hash_error", err)
			return
		}
		if err := ctx.UserRepo.UpdatePassword(r.Context(), userID, string(hash)); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	}
}
```

- [ ] **Step 4: Run all handler tests — expect PASS**

```bash
cd server && go test ./internal/handler/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/user.go server/internal/handler/user_test.go
git commit -m "feat(handler): add UpdateNickname and UpdatePassword user handlers"
```

---

## Task 10: Register new routes in main.go

**Files:**
- Modify: `server/cmd/gameserver/main.go`

- [ ] **Step 1: Add new routes to server.AddRoutes**

In `server/cmd/gameserver/main.go`, add these routes to the `server.AddRoutes` call (after the existing routes):

```go
		{Method: http.MethodPost, Path: "/api/auth/login", Handler: handler.LoginHandler(ctx)},
		{Method: http.MethodGet, Path: "/api/admin/users", Handler: handler.ListUsersHandler(ctx)},
		{Method: http.MethodPost, Path: "/api/admin/users", Handler: handler.CreateUserHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/admin/users/:id/password", Handler: handler.ResetPasswordHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/admin/users/:id/nickname", Handler: handler.ResetNicknameHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/user/password", Handler: handler.UpdatePasswordHandler(ctx)},
		{Method: http.MethodPut, Path: "/api/user/nickname", Handler: handler.UpdateNicknameHandler(ctx)},
```

- [ ] **Step 2: Build**

```bash
cd server && go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run all tests**

```bash
cd server && go test ./...
```

Expected: all PASS or SKIP.

- [ ] **Step 4: Commit**

```bash
git add server/cmd/gameserver/main.go
git commit -m "feat(server): register login, admin, and user profile routes"
```

---

## Task 11: Update AuthStore (frontend)

**Files:**
- Modify: `src/stores/auth.ts`

- [ ] **Step 1: Replace the entire contents of src/stores/auth.ts**

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import http from '../http'

export interface AuthUser {
  id: number
  username: string
  nickname: string
  avatar_url: string
  role: 'owner' | 'service' | 'enforcer' | 'customer'
}

const TOKEN_KEY = 'hls_auth_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<AuthUser | null>(null)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
  }

  async function login(username: string, password: string) {
    const res = await http.post<{ token: string }>('/auth/login', { username, password })
    setToken(res.data.token)
    await init()
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

  return { token, user, setToken, login, init, logout }
})
```

- [ ] **Step 2: Build check**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: no errors (or only pre-existing errors unrelated to auth.ts).

- [ ] **Step 3: Commit**

```bash
git add src/stores/auth.ts
git commit -m "feat(store): add username/role to AuthUser; add login() method"
```

---

## Task 12: Rewrite LoginPage.vue

**Files:**
- Modify: `src/pages/LoginPage.vue`

- [ ] **Step 1: Replace the entire contents of src/pages/LoginPage.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import axios from 'axios'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value.trim(), password.value)
    const redirect = (route.query.redirect as string | undefined) || '/'
    router.push(redirect)
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      error.value = '账号或密码错误'
    } else {
      error.value = '登录失败，请重试'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center px-4">
    <h1 class="rpg-title text-2xl mb-8 tracking-widest">花栗鼠俱乐部</h1>

    <div class="w-full max-w-xs space-y-4">
      <p v-if="error" class="text-red-400 text-sm text-center">{{ error }}</p>

      <input
        v-model="username"
        type="text"
        autocomplete="username"
        placeholder="账号（hlsXXXX）"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        @keydown.enter="!loading && handleLogin()"
      />

      <input
        v-model="password"
        type="password"
        autocomplete="current-password"
        placeholder="密码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
        @keydown.enter="!loading && handleLogin()"
      />

      <button
        class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        :disabled="loading || !username || !password"
        @click="handleLogin"
      >
        {{ loading ? '登录中…' : '登录' }}
      </button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Build check**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/pages/LoginPage.vue
git commit -m "feat(login): replace phone/SMS login with username+password form"
```

---

## Task 13: Create ProfilePage.vue

**Files:**
- Create: `src/pages/ProfilePage.vue`

- [ ] **Step 1: Create src/pages/ProfilePage.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import http from '../http'
import axios from 'axios'

const auth = useAuthStore()
const router = useRouter()

const nickname = ref(auth.user?.nickname ?? '')
const nicknameLoading = ref(false)
const nicknameError = ref('')
const nicknameSuccess = ref(false)

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordLoading = ref(false)
const passwordError = ref('')
const passwordSuccess = ref(false)

async function updateNickname() {
  nicknameError.value = ''
  nicknameSuccess.value = false
  nicknameLoading.value = true
  try {
    await http.put('/user/nickname', { nickname: nickname.value.trim() })
    if (auth.user) auth.user.nickname = nickname.value.trim()
    nicknameSuccess.value = true
  } catch {
    nicknameError.value = '修改失败，请重试'
  } finally {
    nicknameLoading.value = false
  }
}

async function updatePassword() {
  passwordError.value = ''
  passwordSuccess.value = false
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = '两次输入的新密码不一致'
    return
  }
  if (newPassword.value.length < 6) {
    passwordError.value = '新密码至少6位'
    return
  }
  passwordLoading.value = true
  try {
    await http.put('/user/password', {
      old_password: oldPassword.value,
      new_password: newPassword.value,
    })
    oldPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordSuccess.value = true
  } catch (e: unknown) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      passwordError.value = '旧密码错误'
    } else {
      passwordError.value = '修改失败，请重试'
    }
  } finally {
    passwordLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen px-4 py-8 max-w-md mx-auto space-y-8">
    <h1 class="rpg-title text-xl tracking-widest">个人管理</h1>

    <div v-if="auth.user" class="text-xs text-parchment-400/60 font-serif">
      账号：{{ auth.user.username }}
    </div>

    <!-- 修改昵称 -->
    <div class="card border border-white/10 rounded-lg p-6 space-y-4">
      <h2 class="font-serif text-sm text-gold-300 tracking-wide">修改昵称</h2>
      <p v-if="nicknameError" class="text-red-400 text-xs">{{ nicknameError }}</p>
      <p v-if="nicknameSuccess" class="text-green-400 text-xs">修改成功</p>
      <input
        v-model="nickname"
        type="text"
        placeholder="昵称"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <button
        class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 transition-colors"
        :disabled="nicknameLoading || !nickname.trim()"
        @click="updateNickname"
      >
        {{ nicknameLoading ? '保存中…' : '保存昵称' }}
      </button>
    </div>

    <!-- 修改密码 -->
    <div class="card border border-white/10 rounded-lg p-6 space-y-4">
      <h2 class="font-serif text-sm text-gold-300 tracking-wide">修改密码</h2>
      <p v-if="passwordError" class="text-red-400 text-xs">{{ passwordError }}</p>
      <p v-if="passwordSuccess" class="text-green-400 text-xs">密码修改成功</p>
      <input
        v-model="oldPassword"
        type="password"
        placeholder="旧密码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <input
        v-model="newPassword"
        type="password"
        placeholder="新密码（至少6位）"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <input
        v-model="confirmPassword"
        type="password"
        placeholder="确认新密码"
        class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
      />
      <button
        class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 transition-colors"
        :disabled="passwordLoading || !oldPassword || !newPassword || !confirmPassword"
        @click="updatePassword"
      >
        {{ passwordLoading ? '修改中…' : '修改密码' }}
      </button>
    </div>

    <!-- 店长专属 -->
    <div v-if="auth.user?.role === 'owner'" class="card border border-gold-400/20 rounded-lg p-6">
      <button
        class="w-full font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 transition-colors"
        @click="router.push('/admin')"
      >
        进入后台管理
      </button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Build check**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/pages/ProfilePage.vue
git commit -m "feat(profile): add user profile page with nickname/password management"
```

---

## Task 14: Create AdminPage.vue

**Files:**
- Create: `src/pages/AdminPage.vue`

- [ ] **Step 1: Create src/pages/AdminPage.vue**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import http from '../http'
import ConfirmDialog from '../shared/components/ConfirmDialog.vue'

interface AdminUser {
  id: number
  username: string
  nickname: string
  role: 'owner' | 'service' | 'enforcer' | 'customer'
}

interface CreatedUser {
  id: number
  username: string
  nickname: string
  role: string
}

const ROLE_LABELS: Record<string, string> = {
  owner: '店长',
  service: '客服',
  enforcer: '打手',
  customer: '客户',
}

const users = ref<AdminUser[]>([])
const loadError = ref('')

const showCreateModal = ref(false)
const createNickname = ref('')
const createRole = ref<string>('customer')
const createLoading = ref(false)
const createError = ref('')
const createdUser = ref<CreatedUser | null>(null)

const confirmTitle = ref('')
const confirmMsg = ref('')
const confirmAction = ref<() => Promise<void>>(() => Promise.resolve())
const showConfirm = ref(false)

async function loadUsers() {
  loadError.value = ''
  try {
    const res = await http.get<AdminUser[]>('/admin/users')
    users.value = res.data
  } catch {
    loadError.value = '加载用户列表失败'
  }
}

onMounted(loadUsers)

function openCreateModal() {
  createNickname.value = ''
  createRole.value = 'customer'
  createError.value = ''
  createdUser.value = null
  showCreateModal.value = true
}

async function submitCreate() {
  createError.value = ''
  createLoading.value = true
  try {
    const res = await http.post<CreatedUser>('/admin/users', {
      nickname: createNickname.value.trim() || undefined,
      role: createRole.value,
    })
    createdUser.value = res.data
    await loadUsers()
  } catch {
    createError.value = '创建失败，请重试'
  } finally {
    createLoading.value = false
  }
}

function promptResetPassword(user: AdminUser) {
  confirmTitle.value = '重置密码'
  confirmMsg.value = `将「${user.nickname}」(${user.username}) 的密码重置为 123456？`
  confirmAction.value = async () => {
    await http.put(`/admin/users/${user.id}/password`, {})
  }
  showConfirm.value = true
}

function promptResetNickname(user: AdminUser) {
  confirmTitle.value = '重置昵称'
  confirmMsg.value = `将「${user.nickname}」(${user.username}) 的昵称重置为「小小花栗鼠」？`
  confirmAction.value = async () => {
    await http.put(`/admin/users/${user.id}/nickname`, {})
    await loadUsers()
  }
  showConfirm.value = true
}

async function handleConfirm() {
  await confirmAction.value()
  showConfirm.value = false
}
</script>

<template>
  <div class="min-h-screen px-4 py-8 max-w-3xl mx-auto space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="rpg-title text-xl tracking-widest">后台管理 · 用户管理</h1>
      <button
        class="font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 transition-colors"
        @click="openCreateModal"
      >
        + 创建用户
      </button>
    </div>

    <p v-if="loadError" class="text-red-400 text-sm">{{ loadError }}</p>

    <!-- 用户列表 -->
    <div class="overflow-x-auto">
      <table class="w-full text-sm font-serif">
        <thead>
          <tr class="text-parchment-400/60 border-b border-white/10">
            <th class="text-left py-2 pr-4">ID</th>
            <th class="text-left py-2 pr-4">账号</th>
            <th class="text-left py-2 pr-4">昵称</th>
            <th class="text-left py-2 pr-4">角色</th>
            <th class="text-left py-2">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="u in users"
            :key="u.id"
            class="border-b border-white/5 text-parchment-200"
          >
            <td class="py-2 pr-4 text-parchment-400/60">{{ u.id }}</td>
            <td class="py-2 pr-4">{{ u.username || '—' }}</td>
            <td class="py-2 pr-4">{{ u.nickname }}</td>
            <td class="py-2 pr-4">
              <span class="text-xs border border-white/20 rounded px-1.5 py-0.5">
                {{ ROLE_LABELS[u.role] ?? u.role }}
              </span>
            </td>
            <td class="py-2 space-x-2">
              <button
                class="text-xs text-parchment-300/70 hover:text-gold-300 transition-colors"
                @click="promptResetPassword(u)"
              >
                重置密码
              </button>
              <button
                class="text-xs text-parchment-300/70 hover:text-gold-300 transition-colors"
                @click="promptResetNickname(u)"
              >
                重置昵称
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 创建用户 Modal -->
    <div
      v-if="showCreateModal"
      class="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      @click.self="showCreateModal = false"
    >
      <div class="card border border-white/10 rounded-lg p-6 w-full max-w-sm space-y-4">
        <h2 class="font-serif text-sm text-gold-300 tracking-wide">创建用户</h2>

        <!-- 创建成功：显示账号 -->
        <div v-if="createdUser" class="space-y-3">
          <p class="text-parchment-200 text-sm font-serif">创建成功！请记录账号信息：</p>
          <div class="bg-white/5 border border-white/10 rounded p-3 space-y-1 text-sm font-serif">
            <p>账号：<span class="text-gold-300 tracking-widest">{{ createdUser.username }}</span></p>
            <p>昵称：{{ createdUser.nickname }}</p>
            <p>角色：{{ ROLE_LABELS[createdUser.role] ?? createdUser.role }}</p>
            <p>初始密码：<span class="text-gold-300">123456</span></p>
          </div>
          <button
            class="w-full font-serif text-sm text-parchment-300 border border-white/20 px-4 py-2 rounded hover:border-white/40 transition-colors"
            @click="showCreateModal = false"
          >
            关闭
          </button>
        </div>

        <!-- 创建表单 -->
        <template v-else>
          <p v-if="createError" class="text-red-400 text-xs">{{ createError }}</p>
          <input
            v-model="createNickname"
            type="text"
            placeholder="昵称（选填，默认小小花栗鼠）"
            class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 placeholder:text-parchment-400/40 focus:outline-none focus:border-gold-400/60"
          />
          <select
            v-model="createRole"
            class="w-full bg-white/5 border border-white/20 rounded px-3 py-2 text-sm font-serif text-parchment-200 focus:outline-none focus:border-gold-400/60"
          >
            <option value="customer">客户</option>
            <option value="service">客服</option>
            <option value="enforcer">打手</option>
            <option value="owner">店长</option>
          </select>
          <div class="flex gap-2">
            <button
              class="flex-1 font-serif text-sm text-parchment-300 border border-white/20 px-4 py-2 rounded hover:border-white/40 transition-colors"
              @click="showCreateModal = false"
            >
              取消
            </button>
            <button
              class="flex-1 font-serif text-sm text-gold-300 border border-gold-400/40 px-4 py-2 rounded hover:border-gold-300 disabled:opacity-40 transition-colors"
              :disabled="createLoading"
              @click="submitCreate"
            >
              {{ createLoading ? '创建中…' : '创建' }}
            </button>
          </div>
        </template>
      </div>
    </div>

    <!-- 确认对话框 -->
    <ConfirmDialog
      v-if="showConfirm"
      :title="confirmTitle"
      :message="confirmMsg"
      @confirm="handleConfirm"
      @cancel="showConfirm = false"
    />
  </div>
</template>
```

- [ ] **Step 2: Check ConfirmDialog props**

The existing `ConfirmDialog.vue` needs to support `title`, `message`, `@confirm`, and `@cancel`. Read `src/shared/components/ConfirmDialog.vue` and verify. If its props differ, adjust the `AdminPage.vue` usage to match the actual API.

- [ ] **Step 3: Build check**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/pages/AdminPage.vue
git commit -m "feat(admin): add admin user management page"
```

---

## Task 15: Update router — add routes and guards

**Files:**
- Modify: `src/router/index.ts`

- [ ] **Step 1: Replace the entire contents of src/router/index.ts**

```typescript
import { createRouter, createWebHistory } from 'vue-router';
import shopRoutes from '@/games/shop/router';
import teachRoutes from '@/games/teach/router';
import auditionRoutes from '@/games/audition/router';
import pokerRoutes from '@/games/poker/router';
import { useAuthStore } from '@/stores/auth';

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
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/pages/ProfilePage.vue'),
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/pages/AdminPage.vue'),
    },
    ...shopRoutes,
    ...teachRoutes,
    ...auditionRoutes,
    ...pokerRoutes,
    { path: '/:catchAll(.*)', redirect: '/' },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore()

  if (to.path.startsWith('/shop') && !auth.token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.path === '/profile' && !auth.token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.path === '/admin') {
    if (!auth.token) return { name: 'login', query: { redirect: to.fullPath } }
    if (auth.user && auth.user.role !== 'owner') return { name: 'home' }
  }
});
```

- [ ] **Step 2: Build check**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/router/index.ts
git commit -m "feat(router): add /profile and /admin routes with auth/role guards"
```

---

## Task 16: Update AuthWidget — add profile link

**Files:**
- Modify: `src/shared/components/AuthWidget.vue`

- [ ] **Step 1: Add profile link to the dropdown menu**

In `src/shared/components/AuthWidget.vue`, add a "个人管理" button to the dropdown (before the existing logout button):

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Avatar from '@shared/components/Avatar.vue'

const auth = useAuthStore()
const router = useRouter()
const open = ref(false)
const wrapperRef = ref<HTMLElement | null>(null)

function handleOutsideClick(e: MouseEvent) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', handleOutsideClick))
onUnmounted(() => document.removeEventListener('click', handleOutsideClick))
</script>

<template>
  <div ref="wrapperRef" class="relative">
    <button
      v-if="!auth.user"
      class="font-serif text-xs tracking-widest text-gold-300 border border-gold-400/40 px-3 py-1.5 rounded hover:border-gold-300 transition-colors"
      @click="router.push('/login')"
    >
      登录
    </button>

    <div v-else>
      <button
        class="flex items-center gap-2 hover:opacity-80 transition-opacity"
        @click="open = !open"
      >
        <Avatar :seed="String(auth.user.id)" :name="auth.user.nickname" size="sm" />
        <span class="font-serif text-xs text-parchment-300 tracking-wide max-w-[80px] truncate">
          {{ auth.user.nickname || '用户' }}
        </span>
      </button>

      <div
        v-if="open"
        class="absolute right-0 top-full mt-2 min-w-[100px] card border border-white/10 rounded-lg py-1 z-50"
      >
        <button
          class="w-full text-left font-serif text-xs text-parchment-300/80 hover:text-gold-300 px-4 py-2 transition-colors"
          @click="router.push('/profile'); open = false"
        >
          个人管理
        </button>
        <button
          class="w-full text-left font-serif text-xs text-parchment-300/80 hover:text-gold-300 px-4 py-2 transition-colors"
          @click="auth.logout()"
        >
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Build check**

```bash
cd /Users/kratos/hls && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add src/shared/components/AuthWidget.vue
git commit -m "feat(auth-widget): add profile link to user dropdown"
```

---

## Task 17: Smoke test

- [ ] **Step 1: Start the dev server**

```bash
cd /Users/kratos/hls && npm run dev
```

- [ ] **Step 2: Test login flow**

Navigate to `http://localhost:5173/login`. Verify:
- Page shows 账号 + 密码 inputs (no phone/SMS fields)
- Submitting wrong credentials shows "账号或密码错误"
- Submitting correct credentials (use the manually-inserted owner account) logs in and redirects

- [ ] **Step 3: Test profile page**

Navigate to `http://localhost:5173/profile`. Verify:
- Shows current account (username) and nickname
- Nickname can be changed and saved
- Wrong old password shows "旧密码错误"
- Owner sees "进入后台管理" button

- [ ] **Step 4: Test admin page**

Navigate to `http://localhost:5173/admin` with owner account. Verify:
- User list loads
- "创建用户" opens modal; after submit shows generated account + initial password
- "重置密码" prompts confirmation then resets
- "重置昵称" prompts confirmation then resets
- Non-owner visiting `/admin` is redirected to home

- [ ] **Step 5: Final commit if any fixups were needed**

```bash
git add -A
git commit -m "fix: smoke test fixups"
```

# 订单评价系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给订单系统增加"每单一条"的评价（态度分、技术分、9 个扣分项标记、评语），并据评价联动鸡腿奖励（影响收益）与客诉状态。

**Architecture:** 后端 go-zero + 手写 SQL：新增 `reviews` 表与 `ReviewRepo`，评分/客诉/鸡腿的联动规则抽成一个纯函数 `reviewEffects` 做单一事实来源并 TDD 覆盖；`Order.Earnings()` 改为鸡腿有条件计入（新增 `bonus_qualified` 列）。前端 Vue3+Tailwind 纯自研组件：在订单管理页卡片上展示评分、用弹窗录入/查看评价，复用现有"创建订单弹窗"范式。评价保存后前端整表刷新。

**Tech Stack:** Go 1.x + go-zero `sqlx`（MySQL，手写 SQL，无 ORM/迁移框架）；Vue 3 `<script setup lang="ts">` + Vite + TailwindCSS + axios；无前端测试框架（前端以 `npm run typecheck` + `npm run build` + 人工验证）。

**Spec:** `docs/superpowers/specs/2026-08-17-order-review-system-design.md`

## Global Constraints

- 评分粒度：**按订单一条**（`reviews.order_code` UNIQUE），非按打手。
- 计分规则集中在后端纯函数 `reviewEffects`：`total = attitude + technical`；`total >= 8` → 鸡腿达标；`total < 6` → 客诉 `complained`；`total >= 6` 时仅当当前为 `complained` 才回退 `none`（不动 `resolved`）。
- 收益：`Order.Earnings()` 中鸡腿部分**仅在 `bonus_qualified=true` 时计入**。
- 权限：评价录入与查询均为 `requireAnyRole(ctx, []string{"owner","service"}, ...)`。
- 数据库风格：手写 SQL、`db:"..."` tag、`errors.Is(err, sqlx.ErrNotFound)` 判空、错误经 `writeError(w, r, status, key, err)` 返回短 key；沿用现有非事务写法（如 `OrderRepo.Create` 一样顺序执行多条 SQL）。
- 迁移文件手动执行，无 runner；文件放 `server/migrations/`，`utf8mb4`。
- 态度、技术评分取值 1–5；评语最多 500 字（rune 计），可空。
- 9 个扣分 bool **不自动扣数字分**，仅作定性记录。

---

### Task 1: 数据库迁移（reviews 表 + orders 加列）

**Files:**
- Create: `server/migrations/003_create_reviews.sql`

**Interfaces:**
- Produces: `reviews` 表（列名见下，供 Task 4 的 `db:"..."` tag 对齐）；`orders.bonus_qualified TINYINT(1)` 列（供 Task 3/4 使用）。

- [ ] **Step 1: 写迁移文件**

```sql
-- server/migrations/003_create_reviews.sql
CREATE TABLE reviews (
  id                 BIGINT       AUTO_INCREMENT PRIMARY KEY,
  order_code         VARCHAR(20)  NOT NULL UNIQUE,
  attitude_score     TINYINT      NOT NULL,
  technical_score    TINYINT      NOT NULL,
  blame_boss         TINYINT(1)   NOT NULL DEFAULT 0,
  late               TINYINT(1)   NOT NULL DEFAULT 0,
  temp_unavailable   TINYINT(1)   NOT NULL DEFAULT 0,
  slow_reply         TINYINT(1)   NOT NULL DEFAULT 0,
  no_backpack        TINYINT(1)   NOT NULL DEFAULT 0,
  no_channel_card    TINYINT(1)   NOT NULL DEFAULT 0,
  no_supply_card     TINYINT(1)   NOT NULL DEFAULT 0,
  no_emotional_value TINYINT(1)   NOT NULL DEFAULT 0,
  grab_container     TINYINT(1)   NOT NULL DEFAULT 0,
  comment            TEXT         NULL,
  reviewer_id        BIGINT       NOT NULL,
  reviewer_name      VARCHAR(64)  NOT NULL,
  created_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

ALTER TABLE orders ADD COLUMN bonus_qualified TINYINT(1) NOT NULL DEFAULT 0 AFTER complaint_status;
```

- [ ] **Step 2: 校验 SQL 语法（不需真实库，dry check）**

Run: `grep -c "CREATE TABLE reviews" server/migrations/003_create_reviews.sql`
Expected: 输出 `1`；肉眼确认列名与 spec 一致、`order_code` 有 UNIQUE、末尾 ALTER orders 存在。

- [ ] **Step 3: Commit**

```bash
git add server/migrations/003_create_reviews.sql
git commit -m "feat(db): add reviews table and orders.bonus_qualified column"
```

---

### Task 2: 计分联动纯函数 `reviewEffects`（TDD）

**Files:**
- Create: `server/internal/handler/review.go`
- Test: `server/internal/handler/review_effects_test.go`

**Interfaces:**
- Produces: `func reviewEffects(attitude, technical int, currentComplaint string) (total int, bonusQualified bool, newComplaint string)` — Task 5 的 handler 调用它。

- [ ] **Step 1: 写失败测试**（内部测试包 `package handler`，可访问未导出函数）

```go
// server/internal/handler/review_effects_test.go
package handler

import "testing"

func TestReviewEffects(t *testing.T) {
	cases := []struct {
		name          string
		attitude      int
		technical     int
		current       string
		wantTotal     int
		wantQualified bool
		wantComplaint string
	}{
		{"perfect keeps none", 5, 5, "none", 10, true, "none"},
		{"exactly 8 qualifies", 4, 4, "none", 8, true, "none"},
		{"total 7 neither", 4, 3, "none", 7, false, "none"},
		{"total 6 boundary no complaint", 3, 3, "none", 6, false, "none"},
		{"below 6 sets complained", 2, 3, "none", 5, false, "complained"},
		{"recovery reverts complained", 4, 3, "complained", 7, false, "none"},
		{"recovery does not touch resolved", 4, 4, "resolved", 8, true, "resolved"},
		{"low score overrides resolved", 2, 2, "resolved", 4, false, "complained"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			total, qualified, complaint := reviewEffects(c.attitude, c.technical, c.current)
			if total != c.wantTotal || qualified != c.wantQualified || complaint != c.wantComplaint {
				t.Fatalf("reviewEffects(%d,%d,%q) = (%d,%v,%q), want (%d,%v,%q)",
					c.attitude, c.technical, c.current, total, qualified, complaint,
					c.wantTotal, c.wantQualified, c.wantComplaint)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd server && go test ./internal/handler/ -run TestReviewEffects`
Expected: 编译失败 `undefined: reviewEffects`。

- [ ] **Step 3: 写最小实现**

```go
// server/internal/handler/review.go
package handler

// reviewEffects derives the score total, chicken-leg (bonus) qualification,
// and the resulting order complaint status from a review's two scores.
// Rule: total = attitude + technical; total>=8 qualifies for bonus;
// total<6 marks the order complained; total>=6 only reverts an
// auto-set "complained" back to "none" and never touches "resolved".
func reviewEffects(attitude, technical int, currentComplaint string) (total int, bonusQualified bool, newComplaint string) {
	total = attitude + technical
	bonusQualified = total >= 8
	switch {
	case total < 6:
		newComplaint = "complained"
	case currentComplaint == "complained":
		newComplaint = "none"
	default:
		newComplaint = currentComplaint
	}
	return
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd server && go test ./internal/handler/ -run TestReviewEffects -v`
Expected: PASS（8 个子用例全绿）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/review.go server/internal/handler/review_effects_test.go
git commit -m "feat(order): add reviewEffects scoring/complaint/bonus rule"
```

---

### Task 3: `Order.BonusQualified` + 收益公式条件化（TDD）

**Files:**
- Modify: `server/internal/repo/order.go:14-46,51-53`
- Test: `server/internal/repo/order_earnings_test.go`

**Interfaces:**
- Consumes: `orders.bonus_qualified` 列（Task 1）。
- Produces: `Order` struct 新增 `BonusQualified bool \`db:"bonus_qualified"\``；`Earnings()` 仅在其为 true 时计入鸡腿。

- [ ] **Step 1: 先改测试为新语义（会失败）**

替换 `server/internal/repo/order_earnings_test.go` 全文：

```go
package repo_test

import (
	"math"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestOrderEarnings(t *testing.T) {
	cases := []struct {
		name       string
		price      float64
		commission float64
		bonus      float64
		players    int
		qualified  bool
		want       float64
	}{
		{"qualified two players default rates", 800, 20, 3, 2, true, 332},
		{"qualified one player default rates", 800, 20, 3, 1, true, 664},
		{"unqualified two players drops bonus", 800, 20, 3, 2, false, 320},
		{"unqualified one player drops bonus", 800, 20, 3, 1, false, 640},
		{"two players no fees", 100, 0, 0, 2, true, 50},
		{"one player no fees unqualified", 100, 0, 0, 1, false, 100},
		{"qualified rounds to two decimals", 333, 20, 3, 2, true, 138.2},
		{"zero players guarded", 800, 20, 3, 0, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := &repo.Order{
				Price:          c.price,
				CommissionRate: c.commission,
				BonusRate:      c.bonus,
				PlayerCount:    c.players,
				BonusQualified: c.qualified,
			}
			got := o.Earnings()
			if math.Abs(got-c.want) > 0.001 {
				t.Fatalf("Earnings() = %v, want %v", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestOrderEarnings`
Expected: 编译失败 `unknown field 'BonusQualified'`。

- [ ] **Step 3: 加字段并改公式**

在 `server/internal/repo/order.go` 的 `Order` struct 里，`ComplaintStatus` 行之后加一行：

```go
	BonusQualified  bool           `db:"bonus_qualified"  json:"bonusQualified"`
```

把 `Earnings()` 改为：

```go
func (o *Order) Earnings() float64 {
	if o.PlayerCount <= 0 {
		return 0
	}
	net := o.Price * (1 - o.CommissionRate/100)
	bonus := 0.0
	if o.BonusQualified {
		bonus = o.Price * o.BonusRate / 100
	}
	per := (net + bonus) / float64(o.PlayerCount)
	return math.Round(per*100) / 100
}
```

把 `orderCols` 常量加入新列（放在 `complaint_status,` 之后）：

```go
const orderCols = "id, code, title, order_type, price, commission_rate, bonus_rate, " +
	"player_count, dispatcher_id, dispatcher_name, enforcer_a_id, enforcer_a_name, " +
	"enforcer_b_id, enforcer_b_name, remark, status, complaint_status, bonus_qualified, created_at, updated_at"
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/repo/ -run TestOrderEarnings -v`
Expected: PASS（8 个子用例全绿）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/order.go server/internal/repo/order_earnings_test.go
git commit -m "feat(order): gate chicken-leg bonus on bonus_qualified in Earnings"
```

---

### Task 4: `ReviewRepo` + `OrderRepo.ApplyReviewEffects` + 依赖注入

**Files:**
- Create: `server/internal/repo/review.go`
- Modify: `server/internal/repo/order.go`（追加 `ApplyReviewEffects` 方法）
- Modify: `server/internal/svc/service_context.go:19-35,53-67`

**Interfaces:**
- Consumes: `reviews` 表（Task 1）、`orders.bonus_qualified`（Task 1/3）。
- Produces:
  - `repo.Review` struct（字段见下）
  - `func NewReviewRepo(db sqlx.SqlConn) *ReviewRepo`
  - `func (r *ReviewRepo) Upsert(ctx, rv *Review) error`
  - `func (r *ReviewRepo) GetByOrderCode(ctx, code string) (*Review, error)`
  - `func (r *ReviewRepo) ListSummaries(ctx) (map[string]*ReviewSummary, error)`；`ReviewSummary{OrderCode string; Attitude int; Technical int}`
  - `func (r *OrderRepo) ApplyReviewEffects(ctx, code string, bonusQualified bool, complaint string) error`
  - `ctx.ReviewRepo *repo.ReviewRepo`

- [ ] **Step 1: 写 `review.go`**

```go
// server/internal/repo/review.go
package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Review mirrors a row in the reviews table (one per order_code).
type Review struct {
	ID               int64          `db:"id"`
	OrderCode        string         `db:"order_code"`
	AttitudeScore    int            `db:"attitude_score"`
	TechnicalScore   int            `db:"technical_score"`
	BlameBoss        bool           `db:"blame_boss"`
	Late             bool           `db:"late"`
	TempUnavailable  bool           `db:"temp_unavailable"`
	SlowReply        bool           `db:"slow_reply"`
	NoBackpack       bool           `db:"no_backpack"`
	NoChannelCard    bool           `db:"no_channel_card"`
	NoSupplyCard     bool           `db:"no_supply_card"`
	NoEmotionalValue bool           `db:"no_emotional_value"`
	GrabContainer    bool           `db:"grab_container"`
	Comment          sql.NullString `db:"comment"`
	ReviewerID       int64          `db:"reviewer_id"`
	ReviewerName     string         `db:"reviewer_name"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

// ReviewSummary is the compact per-order projection used in order lists.
type ReviewSummary struct {
	OrderCode string `db:"order_code"`
	Attitude  int    `db:"attitude_score"`
	Technical int    `db:"technical_score"`
}

const reviewCols = "id, order_code, attitude_score, technical_score, blame_boss, late, " +
	"temp_unavailable, slow_reply, no_backpack, no_channel_card, no_supply_card, " +
	"no_emotional_value, grab_container, comment, reviewer_id, reviewer_name, created_at, updated_at"

// ReviewRepo provides database access for reviews.
type ReviewRepo struct {
	db sqlx.SqlConn
}

// NewReviewRepo creates a ReviewRepo.
func NewReviewRepo(db sqlx.SqlConn) *ReviewRepo {
	return &ReviewRepo{db: db}
}

// Upsert inserts or overwrites the single review for rv.OrderCode.
func (r *ReviewRepo) Upsert(ctx context.Context, rv *Review) error {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO reviews (order_code, attitude_score, technical_score, blame_boss, late, "+
			"temp_unavailable, slow_reply, no_backpack, no_channel_card, no_supply_card, "+
			"no_emotional_value, grab_container, comment, reviewer_id, reviewer_name) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE attitude_score=VALUES(attitude_score), "+
			"technical_score=VALUES(technical_score), blame_boss=VALUES(blame_boss), "+
			"late=VALUES(late), temp_unavailable=VALUES(temp_unavailable), "+
			"slow_reply=VALUES(slow_reply), no_backpack=VALUES(no_backpack), "+
			"no_channel_card=VALUES(no_channel_card), no_supply_card=VALUES(no_supply_card), "+
			"no_emotional_value=VALUES(no_emotional_value), grab_container=VALUES(grab_container), "+
			"comment=VALUES(comment), reviewer_id=VALUES(reviewer_id), reviewer_name=VALUES(reviewer_name)",
		rv.OrderCode, rv.AttitudeScore, rv.TechnicalScore, rv.BlameBoss, rv.Late,
		rv.TempUnavailable, rv.SlowReply, rv.NoBackpack, rv.NoChannelCard, rv.NoSupplyCard,
		rv.NoEmotionalValue, rv.GrabContainer, rv.Comment, rv.ReviewerID, rv.ReviewerName)
	return err
}

// GetByOrderCode returns the review for code, or sqlx.ErrNotFound.
func (r *ReviewRepo) GetByOrderCode(ctx context.Context, code string) (*Review, error) {
	var rv Review
	err := r.db.QueryRowCtx(ctx, &rv, "SELECT "+reviewCols+" FROM reviews WHERE order_code = ?", code)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// ListSummaries returns compact review projections keyed by order_code.
func (r *ReviewRepo) ListSummaries(ctx context.Context) (map[string]*ReviewSummary, error) {
	var list []*ReviewSummary
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT order_code, attitude_score, technical_score FROM reviews")
	if err != nil {
		return nil, err
	}
	m := make(map[string]*ReviewSummary, len(list))
	for _, s := range list {
		m[s.OrderCode] = s
	}
	return m, nil
}
```

- [ ] **Step 2: 给 `OrderRepo` 加 `ApplyReviewEffects`**

在 `server/internal/repo/order.go` 末尾追加：

```go
// ApplyReviewEffects updates the review-derived fields on an order:
// chicken-leg qualification and the (possibly changed) complaint status.
func (r *OrderRepo) ApplyReviewEffects(ctx context.Context, code string, bonusQualified bool, complaint string) error {
	_, err := r.db.ExecCtx(ctx,
		"UPDATE orders SET bonus_qualified = ?, complaint_status = ? WHERE code = ?",
		bonusQualified, complaint, code)
	return err
}
```

- [ ] **Step 3: 注入 `ReviewRepo`**

在 `server/internal/svc/service_context.go` 的 `ServiceContext` struct 里 `OrderRepo *repo.OrderRepo` 之后加一行：

```go
	ReviewRepo    *repo.ReviewRepo
```

在 `NewServiceContext` 返回的 `&ServiceContext{...}` 里 `OrderRepo: repo.NewOrderRepo(conn),` 之后加一行：

```go
		ReviewRepo:    repo.NewReviewRepo(conn),
```

- [ ] **Step 4: 编译**

Run: `cd server && go build ./...`
Expected: 无错误。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/review.go server/internal/repo/order.go server/internal/svc/service_context.go
git commit -m "feat(order): add ReviewRepo and order review-effects update"
```

---

### Task 5: 评价 handler（upsert / get）+ 路由

**Files:**
- Modify: `server/internal/handler/review.go`（在 Task 2 的文件上追加）
- Modify: `server/cmd/gameserver/main.go:77`（追加两条路由）

**Interfaces:**
- Consumes: `reviewEffects`（Task 2）、`repo.Review`/`ReviewRepo`（Task 4）、`OrderRepo.ApplyReviewEffects`（Task 4）、现有 `requireAnyRole`/`parseOrderPath`/`extractUserID`/`writeError`（`order.go`/`http.go`）。
- Produces: `UpsertReviewHandler(ctx) http.HandlerFunc`、`OrderReviewHandler(ctx) http.HandlerFunc`、`toReviewView(*repo.Review) map[string]any`（后者亦供 Task 6 参考）。

- [ ] **Step 1: 追加 handler 代码到 `review.go`**

在 `server/internal/handler/review.go` 顶部补 import 并追加以下内容（`reviewEffects` 已在同文件）：

```go
import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

type reviewRequest struct {
	AttitudeScore    int    `json:"attitudeScore"`
	TechnicalScore   int    `json:"technicalScore"`
	BlameBoss        bool   `json:"blameBoss"`
	Late             bool   `json:"late"`
	TempUnavailable  bool   `json:"tempUnavailable"`
	SlowReply        bool   `json:"slowReply"`
	NoBackpack       bool   `json:"noBackpack"`
	NoChannelCard    bool   `json:"noChannelCard"`
	NoSupplyCard     bool   `json:"noSupplyCard"`
	NoEmotionalValue bool   `json:"noEmotionalValue"`
	GrabContainer    bool   `json:"grabContainer"`
	Comment          string `json:"comment"`
}

// toReviewView builds the canonical JSON view of a review.
func toReviewView(rv *repo.Review) map[string]any {
	total := rv.AttitudeScore + rv.TechnicalScore
	return map[string]any{
		"orderCode":        rv.OrderCode,
		"attitudeScore":    rv.AttitudeScore,
		"technicalScore":   rv.TechnicalScore,
		"total":            total,
		"bonusQualified":   total >= 8,
		"blameBoss":        rv.BlameBoss,
		"late":             rv.Late,
		"tempUnavailable":  rv.TempUnavailable,
		"slowReply":        rv.SlowReply,
		"noBackpack":       rv.NoBackpack,
		"noChannelCard":    rv.NoChannelCard,
		"noSupplyCard":     rv.NoSupplyCard,
		"noEmotionalValue": rv.NoEmotionalValue,
		"grabContainer":    rv.GrabContainer,
		"comment":          rv.Comment.String,
		"reviewerName":     rv.ReviewerName,
		"createdAt":        rv.CreatedAt,
	}
}

// UpsertReviewHandler handles PUT /api/orders/:code/review (owner/service).
// Writes the single review for the order, then updates the order's
// bonus_qualified and complaint_status per reviewEffects.
func UpsertReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		code, _ := parseOrderPath(r.URL.Path)
		var req reviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Comment = strings.TrimSpace(req.Comment)
		if req.AttitudeScore < 1 || req.AttitudeScore > 5 ||
			req.TechnicalScore < 1 || req.TechnicalScore > 5 ||
			len([]rune(req.Comment)) > 500 {
			writeError(w, r, http.StatusBadRequest, "invalid_review", nil)
			return
		}
		order, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		_, bonusQualified, newComplaint := reviewEffects(req.AttitudeScore, req.TechnicalScore, order.ComplaintStatus)
		rv := &repo.Review{
			OrderCode:        code,
			AttitudeScore:    req.AttitudeScore,
			TechnicalScore:   req.TechnicalScore,
			BlameBoss:        req.BlameBoss,
			Late:             req.Late,
			TempUnavailable:  req.TempUnavailable,
			SlowReply:        req.SlowReply,
			NoBackpack:       req.NoBackpack,
			NoChannelCard:    req.NoChannelCard,
			NoSupplyCard:     req.NoSupplyCard,
			NoEmotionalValue: req.NoEmotionalValue,
			GrabContainer:    req.GrabContainer,
			Comment:          sql.NullString{String: req.Comment, Valid: req.Comment != ""},
			ReviewerID:       userID,
			ReviewerName:     user.Nickname,
		}
		if err := ctx.ReviewRepo.Upsert(r.Context(), rv); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if err := ctx.OrderRepo.ApplyReviewEffects(r.Context(), code, bonusQualified, newComplaint); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		saved, err := ctx.ReviewRepo.GetByOrderCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toReviewView(saved))
	})
}

// OrderReviewHandler handles GET /api/orders/:code/review (owner/service).
func OrderReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		rv, err := ctx.ReviewRepo.GetByOrderCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "review_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toReviewView(rv))
	})
}
```

> 注意：`parseOrderPath("/api/orders/123/review")` 返回 `("123", ["review"])`，取 `code` 正确。

- [ ] **Step 2: 注册路由**

在 `server/cmd/gameserver/main.go` 的 `AddRoutes` 列表里，`UpdateOrderComplaintHandler` 那条之后追加：

```go
			{Method: http.MethodPut, Path: "/api/orders/:code/review", Handler: handler.UpsertReviewHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/orders/:code/review", Handler: handler.OrderReviewHandler(ctx)},
```

- [ ] **Step 3: 编译 + 全量测试**

Run: `cd server && go build ./... && go test ./...`
Expected: 编译无错误；`go test ./...` 现有用例 + `TestReviewEffects` + `TestOrderEarnings` 全绿。

- [ ] **Step 4: Commit**

```bash
git add server/internal/handler/review.go server/cmd/gameserver/main.go
git commit -m "feat(order): add review upsert/get handlers and routes"
```

---

### Task 6: 订单列表附带评分摘要

**Files:**
- Modify: `server/internal/handler/order.go:78-96,157-170`

**Interfaces:**
- Consumes: `ReviewRepo.ListSummaries`（Task 4）、`Order.BonusQualified`（Task 3）。
- Produces: `GET /api/orders` 每个订单视图新增 `"review"` 字段（对象或 `null`）与 `"bonusQualified"` 字段。

- [ ] **Step 1: `toOrderView` 加 `bonusQualified`**

在 `server/internal/handler/order.go` 的 `toOrderView` 返回 map 里，`"complaintStatus": o.ComplaintStatus,` 之后加一行：

```go
		"bonusQualified":  o.BonusQualified,
```

- [ ] **Step 2: `ListOrdersHandler` 合入评分摘要**

把 `ListOrdersHandler` 的函数体替换为：

```go
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		orders, err := ctx.OrderRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		summaries, err := ctx.ReviewRepo.ListSummaries(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(orders))
		for _, o := range orders {
			v := toOrderView(o)
			if s, ok := summaries[o.Code]; ok {
				total := s.Attitude + s.Technical
				v["review"] = map[string]any{
					"attitudeScore":  s.Attitude,
					"technicalScore": s.Technical,
					"total":          total,
					"bonusQualified": total >= 8,
				}
			} else {
				v["review"] = nil
			}
			views = append(views, v)
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
```

- [ ] **Step 3: 编译 + 测试**

Run: `cd server && go build ./... && go test ./...`
Expected: 全绿。

- [ ] **Step 4: Commit**

```bash
git add server/internal/handler/order.go
git commit -m "feat(order): embed review summary in order list view"
```

---

### Task 7: 前端类型与标签（orderLabels.ts）

**Files:**
- Modify: `src/shared/orderLabels.ts`

**Interfaces:**
- Produces: `OrderReview`、`ReviewSummary`、`ReviewFlagKey`、`reviewFlagLabels` 类型/常量；`OrderView` 追加 `review?` 与 `bonusQualified?`（供 Task 8/9/10/11）。

- [ ] **Step 1: 追加类型与标签**

在 `src/shared/orderLabels.ts` 的 `OrderView` 接口内，`complaintStatus` 行之后加：

```ts
  bonusQualified?: boolean
  review?: ReviewSummary | null
```

在文件末尾追加：

```ts
export interface ReviewSummary {
  attitudeScore: number
  technicalScore: number
  total: number
  bonusQualified: boolean
}

export type ReviewFlagKey =
  | 'blameBoss' | 'late' | 'tempUnavailable' | 'slowReply'
  | 'noBackpack' | 'noChannelCard' | 'noSupplyCard'
  | 'noEmotionalValue' | 'grabContainer'

export interface OrderReview {
  orderCode: string
  attitudeScore: number
  technicalScore: number
  total: number
  bonusQualified: boolean
  blameBoss: boolean
  late: boolean
  tempUnavailable: boolean
  slowReply: boolean
  noBackpack: boolean
  noChannelCard: boolean
  noSupplyCard: boolean
  noEmotionalValue: boolean
  grabContainer: boolean
  comment: string
  reviewerName: string
  createdAt: string
}

export const reviewFlagLabels: Record<ReviewFlagKey, string> = {
  blameBoss: '压力指责老板',
  late: '迟到',
  tempUnavailable: '约好时间临时有事',
  slowReply: '群里长时间不回复老板消息',
  noBackpack: '未携带背包',
  noChannelCard: '未携带通道卡',
  noSupplyCard: '未携带物资卡',
  noEmotionalValue: '未提供情绪价值',
  grabContainer: '抢老板的容器',
}
```

- [ ] **Step 2: 类型检查**

Run: `npm run typecheck`
Expected: 无错误。

- [ ] **Step 3: Commit**

```bash
git add src/shared/orderLabels.ts
git commit -m "feat(fe): add review types and flag labels"
```

---

### Task 8: 前端评价 API（api/orders.ts）

**Files:**
- Modify: `src/api/orders.ts:1-2`（import）末尾追加函数

**Interfaces:**
- Consumes: `OrderReview`（Task 7）。
- Produces: `ReviewPayload` 接口、`getOrderReview(code)`、`upsertOrderReview(code, payload)`（供 Task 10/11）。

- [ ] **Step 1: 追加 import 与函数**

把 `src/api/orders.ts` 第 2 行的类型 import 改为包含 `OrderReview`：

```ts
import type { OrderView, OrderType, OrderStatus, ComplaintStatus, OrderReview } from '../shared/orderLabels'
```

在文件末尾追加：

```ts
export interface ReviewPayload {
  attitudeScore: number
  technicalScore: number
  blameBoss: boolean
  late: boolean
  tempUnavailable: boolean
  slowReply: boolean
  noBackpack: boolean
  noChannelCard: boolean
  noSupplyCard: boolean
  noEmotionalValue: boolean
  grabContainer: boolean
  comment: string
}

export async function getOrderReview(code: string): Promise<OrderReview> {
  const res = await http.get<OrderReview>(`/orders/${code}/review`)
  return res.data
}

export async function upsertOrderReview(code: string, payload: ReviewPayload): Promise<OrderReview> {
  const res = await http.put<OrderReview>(`/orders/${code}/review`, payload)
  return res.data
}
```

- [ ] **Step 2: 类型检查**

Run: `npm run typecheck`
Expected: 无错误。

- [ ] **Step 3: Commit**

```bash
git add src/api/orders.ts
git commit -m "feat(fe): add getOrderReview and upsertOrderReview api"
```

---

### Task 9: 订单卡片展示评分 + 入口按钮

**Files:**
- Modify: `src/pages/admin/OrdersPanel.vue`（script imports、helper、模板左栏）

**Interfaces:**
- Consumes: `OrderView.review`（Task 7）、后续 Task 10/11 的 `openReviewForm`/`openReviewDetail`（本任务模板会引用，函数在 Task 10/11 补全 —— 为避免中间态编译报错，本任务同时先声明这两个函数的空壳，Task 10/11 再填实现）。

- [ ] **Step 1: 更新 import**

把 `src/pages/admin/OrdersPanel.vue` 顶部两段 import 改为：

```ts
import {
  listOrders, createOrder, removeEnforcer, updateOrderStatus, updateComplaint,
  getOrderReview, upsertOrderReview,
  type CreateOrderPayload, type ReviewPayload,
} from '../../api/orders'
import {
  orderTypeLabels, orderStatusLabels, complaintLabels, reviewFlagLabels,
  type OrderView, type OrderType, type OrderStatus, type ComplaintStatus,
  type OrderReview, type ReviewFlagKey,
} from '../../shared/orderLabels'
```

- [ ] **Step 2: 加评分色阶 helper + 函数空壳**

在 `getFillCount` 函数之后追加：

```ts
function getRatingClass(total: number) {
  if (total >= 8) return 'bg-emerald-100 text-emerald-900 border-emerald-300'
  if (total < 6) return 'bg-rose-100 text-rose-900 border-rose-300'
  return 'bg-stone-100 text-stone-800 border-stone-300'
}

// filled in by later tasks
async function openReviewForm(_o: OrderView) {}
async function openReviewDetail(_code: string) {}
```

- [ ] **Step 3: 卡片左栏加评价区块**

在模板左栏 `<section class="space-y-4">` 内、"订单备注"那个 `<div>`（以 `订单备注` 文案结尾）之后、`</section>` 之前，插入：

```html
            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <div class="flex items-center justify-between gap-3 flex-wrap">
                <p class="font-body text-sm font-semibold text-hls-text-dark">订单评价</p>
                <div class="flex gap-2">
                  <button class="hls-btn-secondary text-xs px-3 py-1.5" @click="openReviewForm(o)">
                    {{ o.review ? '修改评价' : '录入评价' }}
                  </button>
                  <button v-if="o.review" class="hls-btn-secondary text-xs px-3 py-1.5" @click="openReviewDetail(o.code)">
                    评价详情
                  </button>
                </div>
              </div>
              <div v-if="o.review" class="mt-3 flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-3 py-1 font-body text-xs text-hls-text-dark">态度 {{ o.review.attitudeScore }}/5</span>
                <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-3 py-1 font-body text-xs text-hls-text-dark">技术 {{ o.review.technicalScore }}/5</span>
                <span class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs" :class="getRatingClass(o.review.total)">总分 {{ o.review.total }}/10</span>
                <span v-if="o.review.bonusQualified" class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">🍗 鸡腿达标</span>
              </div>
              <p v-else class="mt-3 font-body text-sm text-hls-text-light">暂无评价</p>
            </div>
```

- [ ] **Step 4: 类型检查 + 构建**

Run: `npm run typecheck && npm run build`
Expected: 无错误（`openReviewForm`/`openReviewDetail` 空壳可编译；卡片评分区可渲染）。

- [ ] **Step 5: Commit**

```bash
git add src/pages/admin/OrdersPanel.vue
git commit -m "feat(fe): show order rating and review entry buttons on cards"
```

---

### Task 10: 评价录入/编辑弹窗

**Files:**
- Modify: `src/pages/admin/OrdersPanel.vue`（script 状态与函数、模板加弹窗）

**Interfaces:**
- Consumes: `getOrderReview`/`upsertOrderReview`/`ReviewPayload`（Task 8）、`reviewFlagLabels`/`ReviewFlagKey`（Task 7）。
- Produces: 可用的 `openReviewForm(o)`（替换 Task 9 空壳）、`submitReview()`。

- [ ] **Step 1: 加状态与函数（替换 Task 9 的 `openReviewForm` 空壳）**

把 Task 9 加的 `async function openReviewForm(_o: OrderView) {}` 那一行删除，并在同处插入：

```ts
// ---- review form ----
const emptyReview = (): ReviewPayload => ({
  attitudeScore: 5, technicalScore: 5,
  blameBoss: false, late: false, tempUnavailable: false, slowReply: false,
  noBackpack: false, noChannelCard: false, noSupplyCard: false,
  noEmotionalValue: false, grabContainer: false, comment: '',
})
const showReviewForm = ref(false)
const reviewForm = ref<ReviewPayload>(emptyReview())
const reviewCode = ref('')
const reviewError = ref('')
const savingReview = ref(false)
const reviewFlagOptions = Object.entries(reviewFlagLabels) as [ReviewFlagKey, string][]
const reviewTotal = computed(() => reviewForm.value.attitudeScore + reviewForm.value.technicalScore)

async function openReviewForm(o: OrderView) {
  reviewCode.value = o.code
  reviewError.value = ''
  if (o.review) {
    try {
      const r = await getOrderReview(o.code)
      reviewForm.value = {
        attitudeScore: r.attitudeScore, technicalScore: r.technicalScore,
        blameBoss: r.blameBoss, late: r.late, tempUnavailable: r.tempUnavailable,
        slowReply: r.slowReply, noBackpack: r.noBackpack, noChannelCard: r.noChannelCard,
        noSupplyCard: r.noSupplyCard, noEmotionalValue: r.noEmotionalValue,
        grabContainer: r.grabContainer, comment: r.comment,
      }
    } catch {
      reviewForm.value = emptyReview()
    }
  } else {
    reviewForm.value = emptyReview()
  }
  showReviewForm.value = true
}

async function submitReview() {
  reviewError.value = ''
  savingReview.value = true
  try {
    await upsertOrderReview(reviewCode.value, { ...reviewForm.value, comment: reviewForm.value.comment.trim() })
    showReviewForm.value = false
    await load()
  } catch {
    reviewError.value = '保存失败，请重试'
  } finally {
    savingReview.value = false
  }
}
```

- [ ] **Step 2: 加录入弹窗模板**

在模板 `<!-- Create Modal -->` 的 `</transition>` 之后插入：

```html
    <!-- Review Form Modal -->
    <transition name="fade">
      <div v-if="showReviewForm" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4" @click.self="showReviewForm = false">
        <div class="hls-panel w-[94%] max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">订单评价</div>
          <div class="p-6 space-y-5">
            <p v-if="reviewError" class="font-body text-xs text-red-400 text-center">{{ reviewError }}</p>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">态度评分</span>
                <select v-model.number="reviewForm.attitudeScore" class="hls-input mt-2 w-full">
                  <option v-for="n in 5" :key="n" :value="n">{{ n }} 分</option>
                </select>
              </label>
              <label class="block">
                <span class="font-body text-xs text-hls-text-dark/75">技术评分</span>
                <select v-model.number="reviewForm.technicalScore" class="hls-input mt-2 w-full">
                  <option v-for="n in 5" :key="n" :value="n">{{ n }} 分</option>
                </select>
              </label>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-3">
              <p class="font-body text-xs text-hls-text-dark/75">总分</p>
              <p class="mt-1 font-body text-lg font-bold" :class="reviewTotal >= 8 ? 'text-emerald-700' : reviewTotal < 6 ? 'text-rose-700' : 'text-hls-text-dark'">
                {{ reviewTotal }}/10
                <span v-if="reviewTotal >= 8" class="font-body text-xs text-amber-700">· 🍗 鸡腿达标</span>
                <span v-else-if="reviewTotal < 6" class="font-body text-xs text-rose-600">· 低于 6 分将标记为客诉</span>
              </p>
            </div>

            <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
              <p class="font-body text-sm font-semibold text-hls-text-dark">扣分项标记</p>
              <div class="mt-3 grid gap-2 sm:grid-cols-2">
                <label v-for="[key, label] in reviewFlagOptions" :key="key" class="flex items-center gap-2 font-body text-sm text-hls-text-dark">
                  <input type="checkbox" v-model="reviewForm[key]" class="h-4 w-4" />
                  {{ label }}
                </label>
              </div>
            </div>

            <label class="block">
              <span class="font-body text-xs text-hls-text-dark/75">评语</span>
              <textarea v-model="reviewForm.comment" placeholder="补充评价内容（选填，最多 500 字）" class="hls-input mt-2 w-full" rows="3" />
            </label>

            <div class="flex gap-3 pt-1">
              <button class="flex-1 hls-btn-secondary py-2" @click="showReviewForm = false">取消</button>
              <button class="flex-1 hls-btn py-2 disabled:opacity-40" :disabled="savingReview" @click="submitReview">
                {{ savingReview ? '保存中…' : '保存评价' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>
```

- [ ] **Step 3: 类型检查 + 构建**

Run: `npm run typecheck && npm run build`
Expected: 无错误。

- [ ] **Step 4: Commit**

```bash
git add src/pages/admin/OrdersPanel.vue
git commit -m "feat(fe): add review create/edit modal"
```

---

### Task 11: 评价详情弹窗

**Files:**
- Modify: `src/pages/admin/OrdersPanel.vue`（script 状态与函数、模板加弹窗）

**Interfaces:**
- Consumes: `getOrderReview`（Task 8）、`reviewFlagOptions`（Task 10）、`OrderReview`（Task 7）、`getRatingClass`/`formatDate`（已有）。
- Produces: 可用的 `openReviewDetail(code)`（替换 Task 9 空壳）。

- [ ] **Step 1: 加状态与函数（替换 Task 9 的 `openReviewDetail` 空壳）**

把 Task 9 加的 `async function openReviewDetail(_code: string) {}` 那一行删除，并在同处插入：

```ts
// ---- review detail ----
const showReviewDetail = ref(false)
const reviewDetail = ref<OrderReview | null>(null)
const reviewDetailFlags = computed(() =>
  reviewDetail.value
    ? reviewFlagOptions.filter(([key]) => reviewDetail.value![key])
    : [],
)

async function openReviewDetail(code: string) {
  reviewDetail.value = null
  showReviewDetail.value = true
  try {
    reviewDetail.value = await getOrderReview(code)
  } catch {
    reviewDetail.value = null
  }
}
```

- [ ] **Step 2: 加详情弹窗模板**

在 Task 10 的 `<!-- Review Form Modal -->` 的 `</transition>` 之后插入：

```html
    <!-- Review Detail Modal -->
    <transition name="fade">
      <div v-if="showReviewDetail" class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4" @click.self="showReviewDetail = false">
        <div class="hls-panel w-[94%] max-w-2xl max-h-[90vh] overflow-y-auto">
          <div class="hls-panel-title text-center">评价详情</div>
          <div class="p-6 space-y-5">
            <template v-if="reviewDetail">
              <div class="flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-3 py-1 font-body text-xs text-hls-text-dark">态度 {{ reviewDetail.attitudeScore }}/5</span>
                <span class="inline-flex items-center rounded-full border border-hls-border/25 bg-black/10 px-3 py-1 font-body text-xs text-hls-text-dark">技术 {{ reviewDetail.technicalScore }}/5</span>
                <span class="inline-flex items-center rounded-full border px-3 py-1 font-body text-xs" :class="getRatingClass(reviewDetail.total)">总分 {{ reviewDetail.total }}/10</span>
                <span v-if="reviewDetail.bonusQualified" class="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-3 py-1 font-body text-xs text-amber-900">🍗 鸡腿达标</span>
              </div>

              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
                <p class="font-body text-sm font-semibold text-hls-text-dark">扣分项</p>
                <div v-if="reviewDetailFlags.length" class="mt-3 flex flex-wrap gap-2">
                  <span v-for="[key, label] in reviewDetailFlags" :key="key" class="inline-flex items-center rounded-full border border-rose-300 bg-rose-100 px-3 py-1 font-body text-xs text-rose-900">{{ label }}</span>
                </div>
                <p v-else class="mt-3 font-body text-sm text-hls-text-light">无扣分项</p>
              </div>

              <div class="rounded-lg border border-hls-border/20 bg-hls-bg/35 px-4 py-4">
                <p class="font-body text-sm font-semibold text-hls-text-dark">评语</p>
                <p class="mt-2 font-body text-sm leading-6 text-hls-text-light whitespace-pre-wrap break-words">{{ reviewDetail.comment || '暂无评语' }}</p>
              </div>

              <p class="font-body text-xs text-hls-text-light">代录人：{{ reviewDetail.reviewerName }} · {{ formatDate(reviewDetail.createdAt) }}</p>
            </template>
            <p v-else class="font-body text-sm text-hls-text-light text-center">暂无评价</p>

            <button class="hls-btn-secondary py-2 w-full" @click="showReviewDetail = false">关闭</button>
          </div>
        </div>
      </div>
    </transition>
```

- [ ] **Step 3: 类型检查 + 构建**

Run: `npm run typecheck && npm run build`
Expected: 无错误。

- [ ] **Step 4: 人工验证（端到端）**

先应用迁移：`mysql <db> < server/migrations/003_create_reviews.sql`；启动后端与 `npm run dev`。以 owner/service 登录，进入订单管理页：
- 对一个订单点"录入评价"，态度 4 + 技术 4（总分 8）→ 保存后卡片出现 `🍗 鸡腿达标`，人均收益含鸡腿。
- 改为态度 2 + 技术 2（总分 4）→ 保存后该订单客诉状态变为"被投诉"，收益不含鸡腿。
- 再改回总分 ≥6 → 客诉自动回到"无客诉"。
- 点"评价详情"→ 弹窗展示评分、勾选的扣分项、评语、代录人。

- [ ] **Step 5: Commit**

```bash
git add src/pages/admin/OrdersPanel.vue
git commit -m "feat(fe): add review detail modal"
```

---

## Self-Review 记录

- **Spec 覆盖**：reviews 表(Task1) / 态度·技术评分(Task1,5,10) / 9 个 bool(Task1,5,7,10,11) / 评语支持汉字(Task1 utf8mb4, Task5 rune 校验) / 鸡腿≥8 影响收益(Task3) / <6 客诉联动(Task2,5) / 后台卡片评分展示(Task9) / 评价详情子页面-modal(Task11) / 每单一条可覆盖(Task1 UNIQUE + Task4 Upsert) —— 全部有对应任务。
- **占位符**：无 TBD/TODO；每个代码步骤均为可直接落地的完整代码。
- **类型一致性**：`reviewEffects` 签名在 Task2 定义、Task5 使用一致；`ReviewPayload`/`OrderReview`/`ReviewFlagKey` 在 Task7/8 定义、Task9/10/11 使用一致；`getRatingClass`/`reviewFlagOptions`/`openReviewForm`/`openReviewDetail` 命名跨任务一致（Task9 先空壳、Task10/11 替换实现，避免中间态编译失败）。
- **已知取舍**：评价 upsert 与订单联动更新用两条顺序 SQL（非显式事务），与现有 `OrderRepo.Create` 的非事务风格一致；操作幂等，重试安全。历史订单迁移后 `bonus_qualified=0`，其展示收益会降掉鸡腿直到被评价——符合新规则，不做数据回填。

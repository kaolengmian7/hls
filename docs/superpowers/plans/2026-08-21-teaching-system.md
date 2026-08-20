# 教学系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增长周期教学子系统：打手(老师)/客户(学员)，按固定大纲逐项打分的成绩单、带续时的课时台账、结单接回现有工资流、学员评价驱动奉金。

**Architecture:** 独立 `teaching_orders` 实体 + 成绩单/时长台账/学员评价三张附表，配套 `TeachingRepo`（go-zero sqlx，事务 + 条件式 UPDATE 并发保护）与 `handler/teaching.go`。工资**复用现有 `payroll_items` 流**——在 `payroll_items` 加 `source` 列，`PayrollRepo.Pay()` 与工资面板按 source 分支读订单或教学单，earnings 用抽取出的共享纯函数 `ComputeEarnings` 计算。前端在单路由 `/admin` 按角色新增 3 个 tab。

**Tech Stack:** Go 1.25.3 + go-zero（rest + core/stores/sqlx）+ MySQL；Vue 3 + TypeScript + Pinia + vue-router + Tailwind。

**Spec:** `docs/superpowers/specs/2026-08-21-teaching-system-design.md`

## Global Constraints

- 后端分层 `handler → repo → db`；repo 用 `const xCols` 复用列清单；多步写用 `db.TransactCtx` + `sqlx.Session`；并发保护用条件式 `UPDATE ... WHERE ...` 判 `RowsAffected()==0` 返回哨兵错误。
- MySQL 迁移是 `server/migrations/` 下**手工执行的编号 SQL 文件**（无 runner）；改表后同步 `server/db/schema.sql`。**无外键约束、无软删除**，跨表按业务键 `code` 或 id 关联。
- 约定：`id BIGINT AUTO_INCREMENT PK`；金额 `DECIMAL(10,2)`、比率 `DECIMAL(5,2)`、名字 `VARCHAR(50)`、状态 `ENUM`；每表 `created_at`（附 `updated_at` 于可更新表）。
- **课时一律以分钟（int）存库**；前端按小时输入/显示（0.5 步进）换算。
- **教学单 code 带前缀 `JX`**（如 `JX1734000000000`），与订单 code 命名空间隔离，避免 `payroll_items(order_code, enforcer_id)` 唯一键跨来源碰撞。
- 分数值域 **0–10**；大纲写死在 `server/etc/gameserver.yaml` 的 `TeachingSyllabus`，全局共用，经 `GET /api/teaching/syllabus` 下发，前端不另维护。
- 密码哈希用 `golang.org/x/crypto/bcrypt`（与现有一致）。
- 鉴权：JWT bearer + `requireOwner` / `requireAnyRole(ctx, roles, next)` + 单据内归属校验（照搬 `SettleOrderHandler`）。
- repo 测试是 **`TEST_DB_DSN` 环境变量门控的集成测试**（未设置则 `t.Skip`）；纯函数测试无需 DB。
- 后端验证命令：`cd server && go build ./...` 与 `cd server && go test ./...`；集成测试：`cd server && TEST_DB_DSN="user:pass@tcp(127.0.0.1:3306)/hls_test?parseTime=true&charset=utf8mb4" go test ./internal/repo/ -run <name> -v`。
- 前端验证命令：`npm run typecheck` 与 `npm run build`。
- 分支：`feat/enforcer-payroll-management`（沿用当前分支）。

---

## 文件结构

**新建：**
- `server/migrations/005_create_teaching.sql` — 4 张教学表
- `server/migrations/006_payroll_add_source.sql` — `payroll_items.source` 列
- `server/internal/repo/teaching.go` — `TeachingRepo` 与教学实体、事务
- `server/internal/repo/teaching_repo_test.go` — 集成测试
- `server/internal/repo/earnings.go` — 共享纯函数 `ComputeEarnings`（+ 单测同文件 `earnings_test.go`）
- `server/internal/handler/teaching.go` — 教学 REST handler 与 view 投影
- `server/internal/handler/teaching_test.go` — 路径/校验纯函数单测
- `src/api/teaching.ts` — 前端 API 客户端与类型
- `src/stores/syllabus.ts` — 大纲 Pinia store
- `src/pages/admin/TeachingPanel.vue` — 教学管理（owner/service）
- `src/pages/admin/MyTeachingTeacherPanel.vue` — 我的教学·老师（enforcer）
- `src/pages/admin/MyTeachingStudentPanel.vue` — 我的教学·学员（customer）

**修改：**
- `server/internal/config/config.go` — 加 `TeachingSyllabus` 类型/字段 + `SyllabusItemKeys` 助手
- `server/etc/gameserver.yaml` — 填 `TeachingSyllabus`（16 项）
- `server/internal/repo/order.go` — `Order.Earnings()` 改调 `ComputeEarnings`
- `server/internal/repo/payroll.go` — `PayrollItem.Source` 字段 + `Pay()` 按 source 分支
- `server/internal/handler/payroll.go` — 面板按合并 source 投影
- `server/internal/handler/review.go` — 抽 `BonusQualifies(attitude, technical int) bool`，`reviewEffects` 改调它
- `server/internal/svc/service_context.go` — 挂 `TeachingRepo`
- `server/cmd/gameserver/main.go` — 注册教学路由
- `server/db/schema.sql` — 同步新表与新列
- `src/router/index.ts` — `/admin` guard 放行 `customer`
- `src/pages/AdminPage.vue` — 加 3 个 tab + 组件

---

## Task 1: 数据库迁移与 schema 同步

**Files:**
- Create: `server/migrations/005_create_teaching.sql`
- Create: `server/migrations/006_payroll_add_source.sql`
- Modify: `server/db/schema.sql`

**Interfaces:**
- Produces: 表 `teaching_orders`, `teaching_scorecards`, `teaching_time_logs`, `teaching_reviews`；`payroll_items.source ENUM('order','teaching') NOT NULL DEFAULT 'order'`。后续所有 repo 集成测试依赖这些表存在于 `TEST_DB_DSN` 指向的库。

- [ ] **Step 1: 写 `005_create_teaching.sql`**

```sql
-- 005_create_teaching.sql — teaching subsystem tables
CREATE TABLE teaching_orders (
  id                BIGINT AUTO_INCREMENT PRIMARY KEY,
  code              VARCHAR(20) NOT NULL,
  title             VARCHAR(50) NOT NULL,
  student_id        BIGINT NOT NULL,
  student_name      VARCHAR(50) NOT NULL,
  dispatcher_id     BIGINT NOT NULL,
  dispatcher_name   VARCHAR(50) NOT NULL,
  teacher_id        BIGINT NULL,
  teacher_name      VARCHAR(50) NULL,
  price             DECIMAL(10,2) NOT NULL DEFAULT 0,
  commission_rate   DECIMAL(5,2)  NOT NULL DEFAULT 0,
  bonus_rate        DECIMAL(5,2)  NOT NULL DEFAULT 0,
  bonus_qualified   TINYINT(1)    NOT NULL DEFAULT 0,
  total_minutes     INT NOT NULL DEFAULT 0,
  remaining_minutes INT NOT NULL DEFAULT 0,
  status            ENUM('dispatching','ongoing','settled') NOT NULL DEFAULT 'dispatching',
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_code (code),
  INDEX idx_status (status),
  INDEX idx_teacher (teacher_id),
  INDEX idx_student (student_id)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE teaching_scorecards (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  duration_minutes    INT NOT NULL,
  scores              JSON NULL,
  comment             TEXT NULL,
  teacher_id          BIGINT NOT NULL,
  teacher_name        VARCHAR(50) NOT NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE teaching_time_logs (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  delta_minutes       INT NOT NULL,
  reason              ENUM('initial','topup','lesson') NOT NULL,
  ref_scorecard_id    BIGINT NULL,
  price_delta         DECIMAL(10,2) NOT NULL DEFAULT 0,
  operator_id         BIGINT NOT NULL,
  operator_name       VARCHAR(50) NOT NULL,
  operator_role       VARCHAR(20) NOT NULL,
  note                VARCHAR(255) NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE teaching_reviews (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  attitude_score      INT NOT NULL,
  technical_score     INT NOT NULL,
  comment             TEXT NULL,
  reviewer_id         BIGINT NOT NULL,
  reviewer_name       VARCHAR(50) NOT NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;
```

- [ ] **Step 2: 写 `006_payroll_add_source.sql`**

```sql
-- 006_payroll_add_source.sql — distinguish order vs teaching payroll
ALTER TABLE payroll_items
  ADD COLUMN source ENUM('order','teaching') NOT NULL DEFAULT 'order';
```

- [ ] **Step 3: 同步 `server/db/schema.sql`**

把上面两段 DDL 追加到 `server/db/schema.sql`（`payroll_items` 的建表语句里加上 `source` 列，与现有快照风格一致）。

- [ ] **Step 4: 应用到测试库并验证**

Run: `mysql "$TEST_DB_URL" < server/migrations/005_create_teaching.sql && mysql "$TEST_DB_URL" < server/migrations/006_payroll_add_source.sql && mysql "$TEST_DB_URL" -e "SHOW TABLES LIKE 'teaching_%'; SHOW COLUMNS FROM payroll_items LIKE 'source';"`
（`$TEST_DB_URL` 为你的测试库连接串）
Expected: 列出 4 张 `teaching_*` 表，且 `payroll_items` 出现 `source` 列。若无本地库，跳过此 Step，后续集成测试会覆盖。

- [ ] **Step 5: Commit**

```bash
git add server/migrations/005_create_teaching.sql server/migrations/006_payroll_add_source.sql server/db/schema.sql
git commit -m "feat(teaching): add migrations for teaching tables and payroll source column"
```

---

## Task 2: 大纲配置

**Files:**
- Modify: `server/internal/config/config.go`
- Modify: `server/etc/gameserver.yaml`
- Test: `server/internal/config/config_test.go` (create)

**Interfaces:**
- Produces:
  - `config.SyllabusItemConfig{ Key, Label, Desc string }`
  - `config.SyllabusCategoryConfig{ Key, Label string; Items []SyllabusItemConfig }`
  - `Config.TeachingSyllabus []SyllabusCategoryConfig`
  - `config.SyllabusItemKeys(cats []SyllabusCategoryConfig) map[string]bool` — 扁平化所有 item.key 的集合，供打分校验用。

- [ ] **Step 1: 写失败测试** `server/internal/config/config_test.go`

```go
package config

import "testing"

func TestSyllabusItemKeys(t *testing.T) {
	cats := []SyllabusCategoryConfig{
		{Key: "gunplay", Label: "枪法与身法", Items: []SyllabusItemConfig{
			{Key: "pre_aim", Label: "预瞄"},
			{Key: "peek", Label: "闪身枪"},
		}},
		{Key: "awareness", Label: "意识", Items: []SyllabusItemConfig{
			{Key: "sound_sense", Label: "听声辨位"},
		}},
	}
	keys := SyllabusItemKeys(cats)
	if len(keys) != 3 || !keys["pre_aim"] || !keys["peek"] || !keys["sound_sense"] {
		t.Fatalf("unexpected keys: %v", keys)
	}
	if keys["nope"] {
		t.Fatalf("should not contain unknown key")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/config/ -run TestSyllabusItemKeys -v`
Expected: 编译失败（`SyllabusItemConfig` / `SyllabusItemKeys` undefined）。

- [ ] **Step 3: 实现 config 类型与助手** — 在 `config.go` 末尾追加：

```go
// SyllabusItemConfig 是教学大纲单个评分项。
type SyllabusItemConfig struct {
	Key   string `json:",optional"`
	Label string `json:",optional"`
	Desc  string `json:",optional"`
}

// SyllabusCategoryConfig 是教学大纲的一个分类。
type SyllabusCategoryConfig struct {
	Key   string               `json:",optional"`
	Label string               `json:",optional"`
	Items []SyllabusItemConfig `json:",optional"`
}

// SyllabusItemKeys 返回所有大纲评分项 key 的集合，供成绩单打分校验。
func SyllabusItemKeys(cats []SyllabusCategoryConfig) map[string]bool {
	m := make(map[string]bool)
	for _, c := range cats {
		for _, it := range c.Items {
			m[it.Key] = true
		}
	}
	return m
}
```

并在 `Config` 结构体加字段（放在 `Payroll` 之后）：

```go
	Payroll          PayrollConfig
	TeachingSyllabus []SyllabusCategoryConfig `json:",optional"`
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/config/ -run TestSyllabusItemKeys -v`
Expected: PASS

- [ ] **Step 5: 填 `server/etc/gameserver.yaml` 的大纲**（追加到文件末尾，YAML 顶层键）

```yaml
TeachingSyllabus:
  - Key: gunplay
    Label: 枪法与身法
    Items:
      - { Key: pre_aim,         Label: 预瞄,               Desc: "移动时准星始终保持在敌方头部高度与常见拉出点，做到出掩体即开枪。" }
      - { Key: peek,            Label: 闪身枪与小身位Peek,  Desc: "利用掩体极快速拉出并收回身位，分为探头Peek与提速闪身枪。" }
      - { Key: jump_slide_peek, Label: 跳拉与铲拉,          Desc: "利用跳跃或滑铲大身位快速拉出掩体，打破敌方架枪的爆头线与预瞄节奏。" }
      - { Key: prone_dodge,     Label: 拜佛,               Desc: "近距离贴脸对枪时突然卧倒，躲避对方第一波爆头线。" }
      - { Key: bunny_hop,       Label: 兔子跳,             Desc: "跳拉出去后连续跳跃，躲避敌方枪线，增加敌方跟枪难度。" }
      - { Key: recoil_control,  Label: 控枪,               Desc: "熟练掌握主流枪械的前20发后坐力规律。" }
      - { Key: tracking,        Label: 跟枪,               Desc: "瞄准移动靶的能力。" }
      - { Key: spray_transfer,  Label: 扫射转移,           Desc: "同时面对多个敌人时的枪械控制能力。" }
  - Key: awareness
    Label: 意识与信息收集
    Items:
      - { Key: map_headlines,   Label: 地图地形与爆头线,     Desc: "熟记各地图高频交战点、优势掩体、高地位以及撤离/复活路线。" }
      - { Key: sound_sense,     Label: 听声辨位与细节感知,   Desc: "通过音效判断脚声、换弹声、修甲声及技能音效的距离与空间感知。" }
      - { Key: info_integration, Label: 地图与信息流整合,     Desc: "收集地图信息，推算敌方小队的人数与包抄路径。" }
      - { Key: rotate_bait,     Label: 转场与拉扯,          Desc: "利用道具或地形转场，创造侧面偷袭机会。" }
  - Key: utility_teamwork
    Label: 道具与干员协同
    Items:
      - { Key: utility_clear,   Label: 道具清点,           Desc: "熟练运用道具将死角架枪的敌人逼出掩体。" }
      - { Key: oneway_skill,    Label: 单向视线与技能组合,   Desc: "运用道具制造单向输出环境，发挥不同兵种的功能定位。" }
      - { Key: crossfire,       Label: 交叉火力与补枪,      Desc: "与队友形成不同角度的交叉架枪线，第一时间拉出补枪。" }
      - { Key: callout,         Label: 高效战术报点,        Desc: "遵循方向+参照物+状态原则，拒绝无效废话。" }
```

- [ ] **Step 6: 确认服务能加载配置**

Run: `cd server && go build ./... && go vet ./internal/config/`
Expected: 编译通过。

- [ ] **Step 7: Commit**

```bash
git add server/internal/config/config.go server/internal/config/config_test.go server/etc/gameserver.yaml
git commit -m "feat(teaching): add teaching syllabus config"
```

---

## Task 3: 抽取共享工资/奉金纯函数

**Files:**
- Create: `server/internal/repo/earnings.go`
- Create: `server/internal/repo/earnings_test.go`
- Modify: `server/internal/repo/order.go:39-50`（`Order.Earnings()` 改调共享函数）
- Modify: `server/internal/handler/review.go:22-34`（抽 `BonusQualifies`）
- Create: `server/internal/handler/review_test.go`

**Interfaces:**
- Produces:
  - `repo.ComputeEarnings(price, commissionRate, bonusRate float64, bonusQualified bool, playerCount int) float64` — 公式 `(price*(1-commission/100) + (bonusQualified ? price*bonus/100 : 0)) / playerCount`，四舍五入两位；`playerCount<=0` 返回 0。
  - `handler.BonusQualifies(attitude, technical int) bool` — `attitude+technical >= 8`。

- [ ] **Step 1: 写失败测试** `server/internal/repo/earnings_test.go`

```go
package repo_test

import (
	"testing"

	"hls.dev/server/internal/repo"
)

func TestComputeEarnings(t *testing.T) {
	// 800 @ 20% commission, bonus 3% not qualified, 1 player = 640
	if got := repo.ComputeEarnings(800, 20, 3, false, 1); got != 640 {
		t.Fatalf("want 640, got %v", got)
	}
	// bonus qualified adds 800*3% = 24 -> 664
	if got := repo.ComputeEarnings(800, 20, 3, true, 1); got != 664 {
		t.Fatalf("want 664, got %v", got)
	}
	// 2 players split
	if got := repo.ComputeEarnings(800, 20, 3, false, 2); got != 320 {
		t.Fatalf("want 320, got %v", got)
	}
	if got := repo.ComputeEarnings(800, 20, 3, false, 0); got != 0 {
		t.Fatalf("want 0 for zero players, got %v", got)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestComputeEarnings -v`
Expected: 编译失败（`ComputeEarnings` undefined）。

- [ ] **Step 3: 实现 `earnings.go`**

```go
package repo

import "math"

// ComputeEarnings 计算单人应发工资，四舍五入两位。
// 公式：(price*(1-commission%/100) + (qualified ? price*bonus%/100 : 0)) / playerCount。
func ComputeEarnings(price, commissionRate, bonusRate float64, bonusQualified bool, playerCount int) float64 {
	if playerCount <= 0 {
		return 0
	}
	net := price * (1 - commissionRate/100)
	bonus := 0.0
	if bonusQualified {
		bonus = price * bonusRate / 100
	}
	per := (net + bonus) / float64(playerCount)
	return math.Round(per*100) / 100
}
```

- [ ] **Step 4: 改 `Order.Earnings()` 调用共享函数** — 替换 `order.go:39-50` 的函数体：

```go
// Earnings returns per-player earnings, rounded to two decimals.
func (o *Order) Earnings() float64 {
	return ComputeEarnings(o.Price, o.CommissionRate, o.BonusRate, o.BonusQualified, o.PlayerCount)
}
```

（`order.go` 若因此不再直接用 `math`，删除其 `math` import 以免编译错误。）

- [ ] **Step 5: 运行确认通过**

Run: `cd server && go test ./internal/repo/ -run TestComputeEarnings -v && go build ./...`
Expected: PASS 且编译通过。

- [ ] **Step 6: 写 `BonusQualifies` 失败测试** `server/internal/handler/review_test.go`

```go
package handler

import "testing"

func TestBonusQualifies(t *testing.T) {
	if !BonusQualifies(5, 3) { // total 8
		t.Fatal("8 should qualify")
	}
	if BonusQualifies(4, 3) { // total 7
		t.Fatal("7 should not qualify")
	}
}
```

- [ ] **Step 7: 运行确认失败**

Run: `cd server && go test ./internal/handler/ -run TestBonusQualifies -v`
Expected: 编译失败（`BonusQualifies` undefined）。

- [ ] **Step 8: 抽 `BonusQualifies`，`reviewEffects` 改调它** — 在 `review.go` 加函数，并把 `reviewEffects` 里 `bonusQualified = total >= 8` 改为 `bonusQualified = BonusQualifies(attitude, technical)`：

```go
// BonusQualifies 判定评价是否达到奖金标准：两项分数之和 >= 8。
func BonusQualifies(attitude, technical int) bool {
	return attitude+technical >= 8
}
```

- [ ] **Step 9: 运行确认通过 + 全量回归**

Run: `cd server && go test ./internal/handler/ ./internal/repo/ -run 'TestBonusQualifies|TestComputeEarnings' -v && go build ./...`
Expected: PASS。

- [ ] **Step 10: Commit**

```bash
git add server/internal/repo/earnings.go server/internal/repo/earnings_test.go server/internal/repo/order.go server/internal/handler/review.go server/internal/handler/review_test.go
git commit -m "refactor(payroll): extract ComputeEarnings and BonusQualifies for reuse"
```

---

## Task 4: TeachingRepo 实体、Create、读取查询

**Files:**
- Create: `server/internal/repo/teaching.go`
- Create: `server/internal/repo/teaching_repo_test.go`

**Interfaces:**
- Produces:
  - `repo.TeachingOrder`（字段见下）、`repo.TeachingScorecard`、`repo.TeachingTimeLog`、`repo.TeachingReview`、`repo.Operator{ID int64; Name, Role string}`
  - `repo.NewTeachingRepo(db sqlx.SqlConn) *TeachingRepo`
  - `(*TeachingRepo).Create(ctx, o *TeachingOrder, op Operator) (*TeachingOrder, error)` — 事务：插 order（`status='dispatching'`，`total_minutes=remaining_minutes=o.TotalMinutes`）+ 插 `initial` 台账（delta=+o.TotalMinutes）。要求 `o.Code`、`o.Title`、`o.StudentID/Name`、`o.DispatcherID/Name`、`o.Price/CommissionRate/BonusRate`、`o.TotalMinutes>0` 已由调用方填好。
  - `(*TeachingRepo).GetByCode(ctx, code) (*TeachingOrder, error)`
  - `(*TeachingRepo).ListAll(ctx) ([]*TeachingOrder, error)`
  - `(*TeachingRepo).ListDispatching(ctx) ([]*TeachingOrder, error)`
  - `(*TeachingRepo).ListByTeacher(ctx, teacherID int64) ([]*TeachingOrder, error)`
  - `(*TeachingRepo).ListByStudent(ctx, studentID int64) ([]*TeachingOrder, error)`

- [ ] **Step 1: 写 `teaching.go` 的实体与 Create/读取（先让测试可编译）**

```go
package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TeachingOrder 对应 teaching_orders 一行。
type TeachingOrder struct {
	ID               int64          `db:"id"`
	Code             string         `db:"code"`
	Title            string         `db:"title"`
	StudentID        int64          `db:"student_id"`
	StudentName      string         `db:"student_name"`
	DispatcherID     int64          `db:"dispatcher_id"`
	DispatcherName   string         `db:"dispatcher_name"`
	TeacherID        sql.NullInt64  `db:"teacher_id"`
	TeacherName      sql.NullString `db:"teacher_name"`
	Price            float64        `db:"price"`
	CommissionRate   float64        `db:"commission_rate"`
	BonusRate        float64        `db:"bonus_rate"`
	BonusQualified   bool           `db:"bonus_qualified"`
	TotalMinutes     int            `db:"total_minutes"`
	RemainingMinutes int            `db:"remaining_minutes"`
	Status           string         `db:"status"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

// TeachingScorecard 对应 teaching_scorecards 一行。Scores 是 JSON 字符串。
type TeachingScorecard struct {
	ID                int64          `db:"id"`
	TeachingOrderCode string         `db:"teaching_order_code"`
	DurationMinutes   int            `db:"duration_minutes"`
	Scores            sql.NullString `db:"scores"`
	Comment           sql.NullString `db:"comment"`
	TeacherID         int64          `db:"teacher_id"`
	TeacherName       string         `db:"teacher_name"`
	CreatedAt         time.Time      `db:"created_at"`
}

// TeachingTimeLog 对应 teaching_time_logs 一行。
type TeachingTimeLog struct {
	ID                int64          `db:"id"`
	TeachingOrderCode string         `db:"teaching_order_code"`
	DeltaMinutes      int            `db:"delta_minutes"`
	Reason            string         `db:"reason"`
	RefScorecardID    sql.NullInt64  `db:"ref_scorecard_id"`
	PriceDelta        float64        `db:"price_delta"`
	OperatorID        int64          `db:"operator_id"`
	OperatorName      string         `db:"operator_name"`
	OperatorRole      string         `db:"operator_role"`
	Note              sql.NullString `db:"note"`
	CreatedAt         time.Time      `db:"created_at"`
}

// TeachingReview 对应 teaching_reviews 一行。
type TeachingReview struct {
	ID                int64          `db:"id"`
	TeachingOrderCode string         `db:"teaching_order_code"`
	AttitudeScore     int            `db:"attitude_score"`
	TechnicalScore    int            `db:"technical_score"`
	Comment           sql.NullString `db:"comment"`
	ReviewerID        int64          `db:"reviewer_id"`
	ReviewerName      string         `db:"reviewer_name"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
}

// Operator 是写台账时的操作者身份。
type Operator struct {
	ID   int64
	Name string
	Role string
}

// 教学子系统哨兵错误。
var (
	ErrTeacherTaken          = errors.New("teacher slot taken")
	ErrInsufficientTime      = errors.New("insufficient remaining time")
	ErrTeachingNotSettleable = errors.New("teaching order not settleable")
)

const teachingOrderCols = "id, code, title, student_id, student_name, dispatcher_id, dispatcher_name, " +
	"teacher_id, teacher_name, price, commission_rate, bonus_rate, bonus_qualified, " +
	"total_minutes, remaining_minutes, status, created_at, updated_at"

// TeachingRepo 提供教学子系统的数据库访问。
type TeachingRepo struct {
	db sqlx.SqlConn
}

// NewTeachingRepo 创建 TeachingRepo。
func NewTeachingRepo(db sqlx.SqlConn) *TeachingRepo {
	return &TeachingRepo{db: db}
}

// Create 插入教学单并写一条 initial 时长台账，单事务。
func (r *TeachingRepo) Create(ctx context.Context, o *TeachingOrder, op Operator) (*TeachingOrder, error) {
	err := r.db.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := s.ExecCtx(ctx,
			"INSERT INTO teaching_orders (code, title, student_id, student_name, dispatcher_id, "+
				"dispatcher_name, price, commission_rate, bonus_rate, total_minutes, remaining_minutes, status) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'dispatching')",
			o.Code, o.Title, o.StudentID, o.StudentName, o.DispatcherID, o.DispatcherName,
			o.Price, o.CommissionRate, o.BonusRate, o.TotalMinutes, o.TotalMinutes); err != nil {
			return err
		}
		_, err := s.ExecCtx(ctx,
			"INSERT INTO teaching_time_logs (teaching_order_code, delta_minutes, reason, price_delta, "+
				"operator_id, operator_name, operator_role, note) VALUES (?, ?, 'initial', ?, ?, ?, ?, '建单初始课时')",
			o.Code, o.TotalMinutes, o.Price, op.ID, op.Name, op.Role)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r.GetByCode(ctx, o.Code)
}

// GetByCode 按 code 返回教学单，或 sqlx.ErrNotFound。
func (r *TeachingRepo) GetByCode(ctx context.Context, code string) (*TeachingOrder, error) {
	var o TeachingOrder
	err := r.db.QueryRowCtx(ctx, &o, "SELECT "+teachingOrderCols+" FROM teaching_orders WHERE code = ?", code)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListAll 返回全部教学单，最新在前。
func (r *TeachingRepo) ListAll(ctx context.Context) ([]*TeachingOrder, error) {
	var list []*TeachingOrder
	err := r.db.QueryRowsCtx(ctx, &list, "SELECT "+teachingOrderCols+" FROM teaching_orders ORDER BY id DESC")
	return list, err
}

// ListDispatching 返回可领（未派老师）的教学单。
func (r *TeachingRepo) ListDispatching(ctx context.Context) ([]*TeachingOrder, error) {
	var list []*TeachingOrder
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+teachingOrderCols+" FROM teaching_orders WHERE status='dispatching' ORDER BY id DESC")
	return list, err
}

// ListByTeacher 返回某老师教的教学单。
func (r *TeachingRepo) ListByTeacher(ctx context.Context, teacherID int64) ([]*TeachingOrder, error) {
	var list []*TeachingOrder
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+teachingOrderCols+" FROM teaching_orders WHERE teacher_id = ? ORDER BY id DESC", teacherID)
	return list, err
}

// ListByStudent 返回某学员学的教学单。
func (r *TeachingRepo) ListByStudent(ctx context.Context, studentID int64) ([]*TeachingOrder, error) {
	var list []*TeachingOrder
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+teachingOrderCols+" FROM teaching_orders WHERE student_id = ? ORDER BY id DESC", studentID)
	return list, err
}
```

- [ ] **Step 2: 写集成测试** `server/internal/repo/teaching_repo_test.go`

```go
package repo_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"hls.dev/server/internal/repo"
)

func teachingTestConn(t *testing.T) sqlx.SqlConn {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
	}
	return sqlx.NewMysql(dsn)
}

func newTeachingOrder(code string, minutes int) *repo.TeachingOrder {
	return &repo.TeachingOrder{
		Code: code, Title: "教学单", StudentID: 700001, StudentName: "学员甲",
		DispatcherID: 1, DispatcherName: "客服A", Price: 1000, CommissionRate: 20, BonusRate: 5,
		TotalMinutes: minutes,
	}
}

func cleanupTeaching(ctx context.Context, conn sqlx.SqlConn, code string) {
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_time_logs WHERE teaching_order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_scorecards WHERE teaching_order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_reviews WHERE teaching_order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM payroll_items WHERE order_code = ?", code)
	_, _ = conn.ExecCtx(ctx, "DELETE FROM teaching_orders WHERE code = ?", code)
}

func TestTeachingCreateAndReads(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXtst" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	created, err := r.Create(ctx, newTeachingOrder(code, 360), repo.Operator{ID: 1, Name: "客服A", Role: "service"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Status != "dispatching" || created.TotalMinutes != 360 || created.RemainingMinutes != 360 {
		t.Fatalf("unexpected created: %+v", created)
	}

	var logCnt int
	_ = conn.QueryRowCtx(ctx, &logCnt, "SELECT COUNT(*) FROM teaching_time_logs WHERE teaching_order_code = ? AND reason='initial'", code)
	if logCnt != 1 {
		t.Fatalf("want 1 initial log, got %d", logCnt)
	}

	all, err := r.ListDispatching(ctx)
	if err != nil {
		t.Fatalf("ListDispatching: %v", err)
	}
	found := false
	for _, o := range all {
		if o.Code == code {
			found = true
		}
	}
	if !found {
		t.Fatalf("created order not in dispatching list")
	}
}
```

- [ ] **Step 3: 运行集成测试**

Run: `cd server && go build ./... && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestTeachingCreateAndReads -v`
Expected: PASS（无 `TEST_DB_DSN` 时 SKIP — 至少要 `go build ./...` 通过）。

- [ ] **Step 4: Commit**

```bash
git add server/internal/repo/teaching.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(teaching): add TeachingRepo entities, Create and read queries"
```

---

## Task 5: 自领老师（Claim）

**Files:**
- Modify: `server/internal/repo/teaching.go`
- Modify: `server/internal/repo/teaching_repo_test.go`

**Interfaces:**
- Consumes: `TeachingRepo`, `ErrTeacherTaken`（Task 4）
- Produces: `(*TeachingRepo).Claim(ctx, code string, teacherID int64, teacherName string) error` — 条件式 `UPDATE ... WHERE teacher_id IS NULL AND status='dispatching'`；`RowsAffected==0` → `ErrTeacherTaken`；成功后 `status='ongoing'`。

- [ ] **Step 1: 写失败测试**（追加到 `teaching_repo_test.go`）

```go
func TestTeachingClaim(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXclm" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 360), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.Claim(ctx, code, 800123, "打手甲"); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.Status != "ongoing" || !o.TeacherID.Valid || o.TeacherID.Int64 != 800123 {
		t.Fatalf("unexpected after claim: %+v", o)
	}
	// second claim rejected
	if err := r.Claim(ctx, code, 800999, "打手乙"); err != repo.ErrTeacherTaken {
		t.Fatalf("want ErrTeacherTaken, got %v", err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestTeachingClaim -v`
Expected: 编译失败（`Claim` undefined）。

- [ ] **Step 3: 实现 Claim**（追加到 `teaching.go`）

```go
// Claim 把 code 的老师槽从空占为 teacherID，状态转 ongoing；已被占则 ErrTeacherTaken。
func (r *TeachingRepo) Claim(ctx context.Context, code string, teacherID int64, teacherName string) error {
	res, err := r.db.ExecCtx(ctx,
		"UPDATE teaching_orders SET teacher_id=?, teacher_name=?, status='ongoing' "+
			"WHERE code=? AND teacher_id IS NULL AND status='dispatching'",
		teacherID, teacherName, code)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTeacherTaken
	}
	return nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestTeachingClaim -v`
Expected: PASS（或 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/teaching.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(teaching): add teacher self-claim with concurrency guard"
```

---

## Task 6: 录成绩单 + 扣课时（并发保护）

**Files:**
- Modify: `server/internal/repo/teaching.go`
- Modify: `server/internal/repo/teaching_repo_test.go`

**Interfaces:**
- Consumes: `TeachingRepo`, `ErrInsufficientTime`
- Produces:
  - `(*TeachingRepo).AddScorecard(ctx, code string, sc *TeachingScorecard) (*TeachingScorecard, error)` — 事务：插 scorecard（`sc.DurationMinutes`, `sc.Scores`(JSON字符串或空), `sc.Comment`, `sc.TeacherID/Name`）→ 插 `lesson` 台账（delta=`-sc.DurationMinutes`, `ref_scorecard_id`=新id）→ 条件式 `UPDATE ... SET remaining_minutes=remaining_minutes-? WHERE code=? AND remaining_minutes>=?`；不足则 `ErrInsufficientTime` 并回滚。调用方保证 `sc.DurationMinutes>0`、`sc.Scores` 已是合法 JSON（或 `Valid=false`）。
  - `(*TeachingRepo).ListScorecards(ctx, code) ([]*TeachingScorecard, error)` — 最新在前。

- [ ] **Step 1: 写失败测试**（追加）

```go
func TestTeachingAddScorecardAndGuard(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXsc" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 120), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = r.Claim(ctx, code, 800123, "打手甲")

	sc := &repo.TeachingScorecard{
		TeachingOrderCode: code, DurationMinutes: 90,
		Scores:      sqlNullStr(`{"pre_aim":8,"peek":6}`),
		Comment:     sqlNullStr("进步明显"),
		TeacherID:   800123, TeacherName: "打手甲",
	}
	if _, err := r.AddScorecard(ctx, code, sc); err != nil {
		t.Fatalf("AddScorecard: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.RemainingMinutes != 30 {
		t.Fatalf("want remaining 30, got %d", o.RemainingMinutes)
	}
	// second 90 exceeds remaining 30 -> ErrInsufficientTime, remaining unchanged
	sc2 := &repo.TeachingScorecard{TeachingOrderCode: code, DurationMinutes: 90, TeacherID: 800123, TeacherName: "打手甲"}
	if _, err := r.AddScorecard(ctx, code, sc2); err != repo.ErrInsufficientTime {
		t.Fatalf("want ErrInsufficientTime, got %v", err)
	}
	o, _ = r.GetByCode(ctx, code)
	if o.RemainingMinutes != 30 {
		t.Fatalf("remaining must stay 30 after failed add, got %d", o.RemainingMinutes)
	}
	cards, _ := r.ListScorecards(ctx, code)
	if len(cards) != 1 {
		t.Fatalf("want 1 scorecard, got %d", len(cards))
	}
}
```

在测试文件加一个小助手（若尚无）：

```go
func sqlNullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
```

并确保测试文件 import `"database/sql"`。

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestTeachingAddScorecardAndGuard -v`
Expected: 编译失败（`AddScorecard`/`ListScorecards` undefined）。

- [ ] **Step 3: 实现 AddScorecard + ListScorecards**（追加到 `teaching.go`）

```go
const scorecardCols = "id, teaching_order_code, duration_minutes, scores, comment, teacher_id, teacher_name, created_at"

// AddScorecard 录一节课的成绩单并扣减剩余课时，单事务；剩余不足则 ErrInsufficientTime。
func (r *TeachingRepo) AddScorecard(ctx context.Context, code string, sc *TeachingScorecard) (*TeachingScorecard, error) {
	var newID int64
	err := r.db.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		res, err := s.ExecCtx(ctx,
			"INSERT INTO teaching_scorecards (teaching_order_code, duration_minutes, scores, comment, teacher_id, teacher_name) "+
				"VALUES (?, ?, ?, ?, ?, ?)",
			code, sc.DurationMinutes, sc.Scores, sc.Comment, sc.TeacherID, sc.TeacherName)
		if err != nil {
			return err
		}
		newID, err = res.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := s.ExecCtx(ctx,
			"INSERT INTO teaching_time_logs (teaching_order_code, delta_minutes, reason, ref_scorecard_id, "+
				"operator_id, operator_name, operator_role, note) VALUES (?, ?, 'lesson', ?, ?, ?, 'enforcer', '上课扣减')",
			code, -sc.DurationMinutes, newID, sc.TeacherID, sc.TeacherName); err != nil {
			return err
		}
		upd, err := s.ExecCtx(ctx,
			"UPDATE teaching_orders SET remaining_minutes = remaining_minutes - ? WHERE code = ? AND remaining_minutes >= ?",
			sc.DurationMinutes, code, sc.DurationMinutes)
		if err != nil {
			return err
		}
		if n, _ := upd.RowsAffected(); n == 0 {
			return ErrInsufficientTime
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out TeachingScorecard
	if err := r.db.QueryRowCtx(ctx, &out, "SELECT "+scorecardCols+" FROM teaching_scorecards WHERE id = ?", newID); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListScorecards 返回某教学单的成绩单，最新在前。
func (r *TeachingRepo) ListScorecards(ctx context.Context, code string) ([]*TeachingScorecard, error) {
	var list []*TeachingScorecard
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+scorecardCols+" FROM teaching_scorecards WHERE teaching_order_code = ? ORDER BY id DESC", code)
	return list, err
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestTeachingAddScorecardAndGuard -v`
Expected: PASS（或 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/teaching.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(teaching): add scorecard recording with time-deduction guard"
```

---

## Task 7: 续时长（TopUp）与时长台账读取

**Files:**
- Modify: `server/internal/repo/teaching.go`
- Modify: `server/internal/repo/teaching_repo_test.go`

**Interfaces:**
- Produces:
  - `(*TeachingRepo).TopUp(ctx, code string, addMinutes int, addPrice float64, op Operator) error` — 事务：插 `topup` 台账（delta=+addMinutes, price_delta=+addPrice）→ `UPDATE ... SET total_minutes+=?, remaining_minutes+=?, price=price+?`。调用方保证 `addMinutes>0`、`addPrice>=0`。
  - `(*TeachingRepo).ListTimeLogs(ctx, code) ([]*TeachingTimeLog, error)` — 按时间正序。

- [ ] **Step 1: 写失败测试**（追加）

```go
func TestTeachingTopUp(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXtop" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 120), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.TopUp(ctx, code, 240, 500, repo.Operator{ID: 2, Name: "店长", Role: "owner"}); err != nil {
		t.Fatalf("TopUp: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.TotalMinutes != 360 || o.RemainingMinutes != 360 || o.Price != 1500 {
		t.Fatalf("unexpected after topup: total=%d remaining=%d price=%v", o.TotalMinutes, o.RemainingMinutes, o.Price)
	}
	logs, _ := r.ListTimeLogs(ctx, code)
	if len(logs) != 2 || logs[0].Reason != "initial" || logs[1].Reason != "topup" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestTeachingTopUp -v`
Expected: 编译失败（`TopUp`/`ListTimeLogs` undefined）。

- [ ] **Step 3: 实现 TopUp + ListTimeLogs**（追加到 `teaching.go`）

```go
const timeLogCols = "id, teaching_order_code, delta_minutes, reason, ref_scorecard_id, price_delta, " +
	"operator_id, operator_name, operator_role, note, created_at"

// TopUp 为教学单增加课时与对应价格，单事务，并写 topup 台账。
func (r *TeachingRepo) TopUp(ctx context.Context, code string, addMinutes int, addPrice float64, op Operator) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := s.ExecCtx(ctx,
			"INSERT INTO teaching_time_logs (teaching_order_code, delta_minutes, reason, price_delta, "+
				"operator_id, operator_name, operator_role, note) VALUES (?, ?, 'topup', ?, ?, ?, ?, '店长续时长')",
			code, addMinutes, addPrice, op.ID, op.Name, op.Role); err != nil {
			return err
		}
		_, err := s.ExecCtx(ctx,
			"UPDATE teaching_orders SET total_minutes=total_minutes+?, remaining_minutes=remaining_minutes+?, price=price+? WHERE code=?",
			addMinutes, addMinutes, addPrice, code)
		return err
	})
}

// ListTimeLogs 返回某教学单的时长台账，按时间正序。
func (r *TeachingRepo) ListTimeLogs(ctx context.Context, code string) ([]*TeachingTimeLog, error) {
	var list []*TeachingTimeLog
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+timeLogCols+" FROM teaching_time_logs WHERE teaching_order_code = ? ORDER BY id ASC", code)
	return list, err
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestTeachingTopUp -v`
Expected: PASS（或 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/teaching.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(teaching): add top-up (minutes+price) with time ledger"
```

---

## Task 8: 结单生成工资条（source=teaching）

**Files:**
- Modify: `server/internal/repo/teaching.go`
- Modify: `server/internal/repo/teaching_repo_test.go`

**Interfaces:**
- Consumes: `ErrTeachingNotSettleable`
- Produces: `(*TeachingRepo).Settle(ctx, code string) error` — 事务：`UPDATE teaching_orders SET status='settled' WHERE code=? AND status='ongoing' AND remaining_minutes=0`，`RowsAffected==0` → `ErrTeachingNotSettleable`；同事务读回老师并 `INSERT IGNORE INTO payroll_items (order_code, enforcer_id, enforcer_name, source, status) VALUES (?, ?, ?, 'teaching', 'pending')`。（老师身份校验在 handler 层做。）

- [ ] **Step 1: 写失败测试**（追加）

```go
func TestTeachingSettle(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXset" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 60), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = r.Claim(ctx, code, 800123, "打手甲")

	// remaining not zero -> not settleable
	if err := r.Settle(ctx, code); err != repo.ErrTeachingNotSettleable {
		t.Fatalf("want ErrTeachingNotSettleable, got %v", err)
	}
	// consume all time
	if _, err := r.AddScorecard(ctx, code, &repo.TeachingScorecard{TeachingOrderCode: code, DurationMinutes: 60, TeacherID: 800123, TeacherName: "打手甲"}); err != nil {
		t.Fatalf("AddScorecard: %v", err)
	}
	if err := r.Settle(ctx, code); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	// idempotent-ish: second settle finds status!=ongoing -> not settleable
	if err := r.Settle(ctx, code); err != repo.ErrTeachingNotSettleable {
		t.Fatalf("second settle want ErrTeachingNotSettleable, got %v", err)
	}
	var cnt int
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM payroll_items WHERE order_code=? AND source='teaching'", code)
	if cnt != 1 {
		t.Fatalf("want 1 teaching payroll item, got %d", cnt)
	}
	o, _ := r.GetByCode(ctx, code)
	if o.Status != "settled" {
		t.Fatalf("want settled, got %s", o.Status)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestTeachingSettle -v`
Expected: 编译失败（`Settle` undefined）。

- [ ] **Step 3: 实现 Settle**（追加到 `teaching.go`）

```go
// Settle 在剩余课时为 0 且状态 ongoing 时结单，并生成一条 teaching 来源的待发工资条。
func (r *TeachingRepo) Settle(ctx context.Context, code string) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		res, err := s.ExecCtx(ctx,
			"UPDATE teaching_orders SET status='settled' WHERE code=? AND status='ongoing' AND remaining_minutes=0", code)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrTeachingNotSettleable
		}
		var o TeachingOrder
		if err := s.QueryRowCtx(ctx, &o, "SELECT "+teachingOrderCols+" FROM teaching_orders WHERE code=?", code); err != nil {
			return err
		}
		if !o.TeacherID.Valid {
			return ErrTeachingNotSettleable
		}
		_, err = s.ExecCtx(ctx,
			"INSERT IGNORE INTO payroll_items (order_code, enforcer_id, enforcer_name, source, status) "+
				"VALUES (?, ?, ?, 'teaching', 'pending')",
			code, o.TeacherID.Int64, o.TeacherName.String)
		return err
	})
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestTeachingSettle -v`
Expected: PASS（或 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/teaching.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(teaching): settle order and generate teaching payroll item"
```

---

## Task 9: 学员评价回写奉金

**Files:**
- Modify: `server/internal/repo/teaching.go`
- Modify: `server/internal/repo/teaching_repo_test.go`

**Interfaces:**
- Produces:
  - `(*TeachingRepo).UpsertReviewWithEffects(ctx, rv *TeachingReview, bonusQualified bool) error` — 事务：`INSERT ... ON DUPLICATE KEY UPDATE` 写 `teaching_reviews`（按 `teaching_order_code` 唯一）+ `UPDATE teaching_orders SET bonus_qualified=? WHERE code=?`。`bonusQualified` 由 handler 用 `BonusQualifies` 算好传入。
  - `(*TeachingRepo).GetReview(ctx, code) (*TeachingReview, error)`

- [ ] **Step 1: 写失败测试**（追加）

```go
func TestTeachingReviewUpsert(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	ctx := context.Background()
	code := "JXrev" + strconv.FormatInt(time.Now().UnixNano(), 10)
	t.Cleanup(func() { cleanupTeaching(ctx, conn, code) })

	if _, err := r.Create(ctx, newTeachingOrder(code, 60), repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	rv := &repo.TeachingReview{TeachingOrderCode: code, AttitudeScore: 5, TechnicalScore: 4, ReviewerID: 700001, ReviewerName: "学员甲"}
	if err := r.UpsertReviewWithEffects(ctx, rv, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	o, _ := r.GetByCode(ctx, code)
	if !o.BonusQualified {
		t.Fatalf("bonus should be qualified")
	}
	// re-submit lower, unqualify
	rv2 := &repo.TeachingReview{TeachingOrderCode: code, AttitudeScore: 2, TechnicalScore: 2, ReviewerID: 700001, ReviewerName: "学员甲"}
	if err := r.UpsertReviewWithEffects(ctx, rv2, false); err != nil {
		t.Fatalf("Upsert2: %v", err)
	}
	o, _ = r.GetByCode(ctx, code)
	if o.BonusQualified {
		t.Fatalf("bonus should be unqualified after re-review")
	}
	got, _ := r.GetReview(ctx, code)
	if got.AttitudeScore != 2 {
		t.Fatalf("review not overwritten: %+v", got)
	}
	var cnt int
	_ = conn.QueryRowCtx(ctx, &cnt, "SELECT COUNT(*) FROM teaching_reviews WHERE teaching_order_code=?", code)
	if cnt != 1 {
		t.Fatalf("want 1 review row, got %d", cnt)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestTeachingReviewUpsert -v`
Expected: 编译失败。

- [ ] **Step 3: 实现 UpsertReviewWithEffects + GetReview**（追加到 `teaching.go`）

```go
const teachingReviewCols = "id, teaching_order_code, attitude_score, technical_score, comment, " +
	"reviewer_id, reviewer_name, created_at, updated_at"

// UpsertReviewWithEffects 写/覆盖学员评价，并在同事务回写教学单的 bonus_qualified。
func (r *TeachingRepo) UpsertReviewWithEffects(ctx context.Context, rv *TeachingReview, bonusQualified bool) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := s.ExecCtx(ctx,
			"INSERT INTO teaching_reviews (teaching_order_code, attitude_score, technical_score, comment, reviewer_id, reviewer_name) "+
				"VALUES (?, ?, ?, ?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE attitude_score=VALUES(attitude_score), technical_score=VALUES(technical_score), "+
				"comment=VALUES(comment), reviewer_id=VALUES(reviewer_id), reviewer_name=VALUES(reviewer_name)",
			rv.TeachingOrderCode, rv.AttitudeScore, rv.TechnicalScore, rv.Comment, rv.ReviewerID, rv.ReviewerName); err != nil {
			return err
		}
		_, err := s.ExecCtx(ctx,
			"UPDATE teaching_orders SET bonus_qualified=? WHERE code=?", bonusQualified, rv.TeachingOrderCode)
		return err
	})
}

// GetReview 返回某教学单的学员评价，或 sqlx.ErrNotFound。
func (r *TeachingRepo) GetReview(ctx context.Context, code string) (*TeachingReview, error) {
	var rv TeachingReview
	err := r.db.QueryRowCtx(ctx, &rv, "SELECT "+teachingReviewCols+" FROM teaching_reviews WHERE teaching_order_code = ?", code)
	if err != nil {
		return nil, err
	}
	return &rv, nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run TestTeachingReviewUpsert -v`
Expected: PASS（或 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/repo/teaching.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(teaching): student review upsert with bonus_qualified writeback"
```

---

## Task 10: 工资流接入 —— `Pay()` 按 source 分支

**Files:**
- Modify: `server/internal/repo/payroll.go`（`PayrollItem` 加 `Source`；`payrollCols`；`Pay()` 分支）
- Modify: `server/internal/repo/teaching_repo_test.go`（加 Pay 集成测试）

**Interfaces:**
- Consumes: `repo.ComputeEarnings`（Task 3）、`teachingOrderCols` / `TeachingOrder`（Task 4）
- Produces: `PayrollItem.Source string`（db:"source"）；`Pay()` 对 `source='teaching'` 从 `teaching_orders` 读单并用 `ComputeEarnings(price, commission, bonus, bonus_qualified, 1)` 算 gross，押金/罚款扣减逻辑不变。

- [ ] **Step 1: 写失败测试**（追加到 `teaching_repo_test.go`）

```go
func TestPayTeachingPayroll(t *testing.T) {
	conn := teachingTestConn(t)
	r := repo.NewTeachingRepo(conn)
	payRepo := repo.NewPayrollRepo(conn)
	accRepo := repo.NewEnforcerAccountRepo(conn)
	ctx := context.Background()
	code := "JXpay" + strconv.FormatInt(time.Now().UnixNano(), 10)
	const teacher = 800555
	t.Cleanup(func() {
		cleanupTeaching(ctx, conn, code)
		_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", teacher)
	})

	o := newTeachingOrder(code, 60)
	o.Price = 1000
	o.CommissionRate = 20
	o.BonusRate = 5
	if _, err := r.Create(ctx, o, repo.Operator{ID: 1, Name: "客服A", Role: "service"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = r.Claim(ctx, code, teacher, "打手甲")
	_, _ = r.AddScorecard(ctx, code, &repo.TeachingScorecard{TeachingOrderCode: code, DurationMinutes: 60, TeacherID: teacher, TeacherName: "打手甲"})
	if err := r.Settle(ctx, code); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	_, _ = conn.ExecCtx(ctx, "DELETE FROM enforcer_accounts WHERE user_id = ?", teacher)

	var id int64
	_ = conn.QueryRowCtx(ctx, &id, "SELECT id FROM payroll_items WHERE order_code=? AND enforcer_id=?", code, teacher)
	// gross = 1000 * (1-0.2) = 800 (bonus unqualified). deposit default 300 owed, no fine.
	item, err := payRepo.Pay(ctx, id, 300)
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if item.Gross != 800 || item.DepositDeducted != 300 || item.NetPaid != 500 {
		t.Fatalf("unexpected paid teaching item: %+v", item)
	}
	acc, _ := accRepo.GetOrDefault(ctx, teacher)
	if acc.DepositPaid != 300 {
		t.Fatalf("deposit not updated: %+v", acc)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/repo/ -run TestPayTeachingPayroll -v`
Expected: FAIL（当前 `Pay()` 会去 `orders` 表找教学 code，报 ErrNotFound / 或 Source 字段不存在编译失败）。

- [ ] **Step 3: `PayrollItem` 加 Source 字段与列**

在 `payroll.go` 的 `PayrollItem` 结构体末尾加：

```go
	Source          string       `db:"source"`
```

并把 `payrollCols` 改为（加 `source`）：

```go
const payrollCols = "id, order_code, enforcer_id, enforcer_name, status, gross, " +
	"deposit_deducted, fine_deducted, net_paid, created_at, paid_at, source"
```

- [ ] **Step 4: `Pay()` 按 source 分支算 gross**

把 `payroll.go` 中读取订单并算 gross 的段（当前 `var o Order ... gross := o.Earnings()` 与 refunded 检查）替换为：

```go
		var gross float64
		refunded := false
		if item.Source == "teaching" {
			var t TeachingOrder
			if err := session.QueryRowCtx(ctx, &t,
				"SELECT "+teachingOrderCols+" FROM teaching_orders WHERE code = ?", item.OrderCode); err != nil {
				return err
			}
			gross = ComputeEarnings(t.Price, t.CommissionRate, t.BonusRate, t.BonusQualified, 1)
			// 教学 v1 无退款，refunded 恒 false
		} else {
			var o Order
			if err := session.QueryRowCtx(ctx, &o,
				"SELECT "+orderCols+" FROM orders WHERE code = ?", item.OrderCode); err != nil {
				return err
			}
			gross = o.Earnings()
			refunded = o.Status == "refunded"
		}
		if refunded {
			return ErrOrderRefunded
		}
```

（其余押金/罚款计算、UPDATE payroll_items、更新 enforcer_accounts 全部保持不变。）

- [ ] **Step 5: 运行通过 + 订单工资回归**

Run: `cd server && go build ./... && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run 'TestPayTeachingPayroll|TestSettleWithPayroll_And_Pay' -v`
Expected: 两个测试都 PASS（或 SKIP）——证明教学工资可发放、订单工资无回归。

- [ ] **Step 6: Commit**

```bash
git add server/internal/repo/payroll.go server/internal/repo/teaching_repo_test.go
git commit -m "feat(payroll): pay teaching payroll items via source branch"
```

---

## Task 11: 工资面板按合并 source 投影

**Files:**
- Modify: `server/internal/handler/payroll.go`
- Modify: `server/internal/svc/service_context.go`（挂 `TeachingRepo`，本任务需要）

**Interfaces:**
- Consumes: `ctx.OrderRepo.ListAll`、`ctx.TeachingRepo.ListAll`
- Produces:
  - `svc.ServiceContext.TeachingRepo *repo.TeachingRepo`
  - `handler` 内 `payrollSource{ Title string; Gross float64; Refunded bool }` 与 `buildPayrollSources(orders []*repo.Order, teach []*repo.TeachingOrder) map[string]payrollSource`
  - `projectPayrollView(item *repo.PayrollItem, src *payrollSource, acc *repo.EnforcerAccount, defaultDeposit float64) payrollView`
  - `paidPayrollView(item *repo.PayrollItem, src *payrollSource) payrollView`

- [ ] **Step 1: 挂 TeachingRepo 到 ServiceContext**

在 `service_context.go` 结构体 `PayrollRepo` 附近加字段：

```go
	PayrollRepo         *repo.PayrollRepo
	TeachingRepo        *repo.TeachingRepo
```

在 `NewServiceContext` 的返回构造里加：

```go
		PayrollRepo:         repo.NewPayrollRepo(conn),
		TeachingRepo:        repo.NewTeachingRepo(conn),
```

- [ ] **Step 2: 改 `payroll.go` 的投影签名与来源合并**

把 `ordersByCode` 删除，新增：

```go
type payrollSource struct {
	Title    string
	Gross    float64
	Refunded bool
}

// buildPayrollSources 合并订单与教学单为 code -> 来源投影，供工资面板统一取用。
func buildPayrollSources(orders []*repo.Order, teach []*repo.TeachingOrder) map[string]payrollSource {
	m := make(map[string]payrollSource, len(orders)+len(teach))
	for _, o := range orders {
		m[o.Code] = payrollSource{Title: o.Title, Gross: o.Earnings(), Refunded: o.Status == "refunded"}
	}
	for _, t := range teach {
		m[t.Code] = payrollSource{
			Title: t.Title,
			Gross: repo.ComputeEarnings(t.Price, t.CommissionRate, t.BonusRate, t.BonusQualified, 1),
			// 教学无退款
		}
	}
	return m
}
```

把 `projectPayrollView` 与 `paidPayrollView` 的第二参数从 `order *repo.Order` 改成 `src *payrollSource`，函数体用 `src.Gross` / `src.Title`（`src==nil` 时 gross=0、title=""）。例如 `projectPayrollView` 开头：

```go
func projectPayrollView(item *repo.PayrollItem, src *payrollSource, acc *repo.EnforcerAccount, defaultDeposit float64) payrollView {
	gross := 0.0
	title := ""
	if src != nil {
		gross = src.Gross
		title = src.Title
	}
	// ... 其余不变（deduction 计算沿用 gross）...
```

`paidPayrollView` 同理用 `src.Title`（gross 走 item 冻结快照，不受来源影响）。

- [ ] **Step 3: 改 `ListPayrollHandler` 与 `PayPayrollHandler` 取来源**

`ListPayrollHandler` 里把 `orders, err := ctx.OrderRepo.ListAll(...)` 后的 `byCode := ordersByCode(orders)` 改成：

```go
		teach, err := ctx.TeachingRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		sources := buildPayrollSources(orders, teach)
```

把所有 `byCode[it.OrderCode]` 替换为 `srcPtr(sources, it.OrderCode)`，其中新增助手：

```go
func srcPtr(m map[string]payrollSource, code string) *payrollSource {
	if s, ok := m[code]; ok {
		return &s
	}
	return nil
}
```

`PayPayrollHandler` 末尾把 `order, _ := ctx.OrderRepo.GetByCode(...)` + `paidPayrollView(item, order)` 改成按来源取标题：

```go
		var src *payrollSource
		if item.Source == "teaching" {
			if t, e := ctx.TeachingRepo.GetByCode(r.Context(), item.OrderCode); e == nil {
				src = &payrollSource{Title: t.Title}
			}
		} else {
			if o, e := ctx.OrderRepo.GetByCode(r.Context(), item.OrderCode); e == nil {
				src = &payrollSource{Title: o.Title}
			}
		}
		httpx.OkJsonCtx(r.Context(), w, paidPayrollView(item, src))
```

- [ ] **Step 4: 编译 + 回归**

Run: `cd server && go build ./... && go test ./internal/handler/ ./internal/repo/ -run 'Payroll|Earnings|Bonus' -v`
Expected: 编译通过；相关测试 PASS（或 SKIP）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/payroll.go server/internal/svc/service_context.go
git commit -m "feat(payroll): merge order+teaching sources in payroll panel"
```

---

## Task 12: 教学 handler —— 路径解析与请求校验（纯函数 TDD）

**Files:**
- Create: `server/internal/handler/teaching.go`
- Create: `server/internal/handler/teaching_test.go`

**Interfaces:**
- Produces:
  - `parseTeachingPath(path string) (code string, rest []string)` — 从 `/api/teaching-orders/{code}/a/b` 解析。
  - `validateCreateTeaching(req createTeachingRequest) (errKey string)` — 校验标题非空、`totalHours>0`、`commissionRate∈[0,100]`、`(studentId>0 或 newStudentNickname 非空)`。
  - `validateScores(scores map[string]int, validKeys map[string]bool) (errKey string)` — 校验每个 key 合法、值 ∈ [0,10]。
  - 请求结构体 `createTeachingRequest`、`topUpRequest`、`scorecardRequest`、`teachingReviewRequest`。
  - 单位换算 `hoursToMinutes(h float64) int`（`round(h*60)`）、`minutesToHours(m int) float64`。

- [ ] **Step 1: 写失败测试** `server/internal/handler/teaching_test.go`

```go
package handler

import "testing"

func TestParseTeachingPath(t *testing.T) {
	code, rest := parseTeachingPath("/api/teaching-orders/JX123/scorecards")
	if code != "JX123" || len(rest) != 1 || rest[0] != "scorecards" {
		t.Fatalf("unexpected: %q %v", code, rest)
	}
	code, rest = parseTeachingPath("/api/teaching-orders/JX9")
	if code != "JX9" || len(rest) != 0 {
		t.Fatalf("unexpected: %q %v", code, rest)
	}
}

func TestValidateScores(t *testing.T) {
	valid := map[string]bool{"pre_aim": true, "peek": true}
	if validateScores(map[string]int{"pre_aim": 8}, valid) != "" {
		t.Fatal("valid scores should pass")
	}
	if validateScores(map[string]int{"pre_aim": 11}, valid) == "" {
		t.Fatal("score 11 should fail")
	}
	if validateScores(map[string]int{"nope": 5}, valid) == "" {
		t.Fatal("unknown key should fail")
	}
}

func TestHoursToMinutes(t *testing.T) {
	if hoursToMinutes(1.5) != 90 || hoursToMinutes(6) != 360 {
		t.Fatalf("bad conversion")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `cd server && go test ./internal/handler/ -run 'TestParseTeachingPath|TestValidateScores|TestHoursToMinutes' -v`
Expected: 编译失败。

- [ ] **Step 3: 实现 `teaching.go` 的纯函数骨架 + 请求结构体**

```go
package handler

import (
	"math"
	"strings"
)

type createTeachingRequest struct {
	Title              string  `json:"title"`
	StudentID          int64   `json:"studentId"`
	NewStudentNickname string  `json:"newStudentNickname"`
	TotalHours         float64 `json:"totalHours"`
	Price              float64 `json:"price"`
	CommissionRate     float64 `json:"commissionRate"`
	BonusRate          float64 `json:"bonusRate"`
}

type topUpRequest struct {
	AddHours float64 `json:"addHours"`
	AddPrice float64 `json:"addPrice"`
}

type scorecardRequest struct {
	DurationHours float64        `json:"durationHours"`
	Scores        map[string]int `json:"scores"`
	Comment       string         `json:"comment"`
}

type teachingReviewRequest struct {
	AttitudeScore  int    `json:"attitudeScore"`
	TechnicalScore int    `json:"technicalScore"`
	Comment        string `json:"comment"`
}

func parseTeachingPath(path string) (string, []string) {
	rest := strings.TrimPrefix(path, "/api/teaching-orders/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil
	}
	return parts[0], parts[1:]
}

func validateCreateTeaching(req createTeachingRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "invalid_title"
	}
	if req.TotalHours <= 0 {
		return "invalid_hours"
	}
	if req.CommissionRate < 0 || req.CommissionRate > 100 {
		return "invalid_commission"
	}
	if req.StudentID <= 0 && strings.TrimSpace(req.NewStudentNickname) == "" {
		return "missing_student"
	}
	return ""
}

func validateScores(scores map[string]int, validKeys map[string]bool) string {
	for k, v := range scores {
		if !validKeys[k] {
			return "invalid_score_key"
		}
		if v < 0 || v > 10 {
			return "invalid_score_value"
		}
	}
	return ""
}

func hoursToMinutes(h float64) int { return int(math.Round(h * 60)) }

func minutesToHours(m int) float64 { return math.Round(float64(m)/60*100) / 100 }
```

- [ ] **Step 4: 运行确认通过**

Run: `cd server && go test ./internal/handler/ -run 'TestParseTeachingPath|TestValidateScores|TestHoursToMinutes' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/teaching.go server/internal/handler/teaching_test.go
git commit -m "feat(teaching): add handler path parsing and request validation"
```

---

## Task 13: 教学 handler —— view 投影与建单/详情/列表/我的教学/大纲/学员选择

**Files:**
- Modify: `server/internal/handler/teaching.go`

**Interfaces:**
- Consumes: `ctx.TeachingRepo`、`ctx.UserRepo.CreateWithPassword`、`ctx.UserRepo.ListAll`、`ctx.Config.TeachingSyllabus`、`config.SyllabusItemKeys`、`requireAnyRole`、`extractUserID`、`bcrypt`
- Produces（供 Task 14 路由注册）：
  - `toTeachingOrderView(o *repo.TeachingOrder) map[string]any`（含 `remainingHours`/`totalHours` = `minutesToHours`）
  - `CreateTeachingOrderHandler`, `ListTeachingOrdersHandler`, `TeachingOrderDetailHandler`, `MyTeachingHandler`, `SyllabusHandler`, `ListCustomersHandler`

- [ ] **Step 1: 实现 view 投影 + 建单/详情/列表/我的教学/大纲/客户** —— 追加到 `teaching.go`：

```go
import 需补充:
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"golang.org/x/crypto/bcrypt"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
```

```go
func teacherView(o *repo.TeachingOrder) any {
	if !o.TeacherID.Valid {
		return nil
	}
	return map[string]any{"id": o.TeacherID.Int64, "name": o.TeacherName.String}
}

func toTeachingOrderView(o *repo.TeachingOrder) map[string]any {
	return map[string]any{
		"code":            o.Code,
		"title":           o.Title,
		"student":         map[string]any{"id": o.StudentID, "name": o.StudentName},
		"dispatcherName":  o.DispatcherName,
		"teacher":         teacherView(o),
		"price":           o.Price,
		"commissionRate":  o.CommissionRate,
		"bonusRate":       o.BonusRate,
		"bonusQualified":  o.BonusQualified,
		"totalHours":      minutesToHours(o.TotalMinutes),
		"remainingHours":  minutesToHours(o.RemainingMinutes),
		"totalMinutes":    o.TotalMinutes,
		"remainingMinutes": o.RemainingMinutes,
		"status":          o.Status,
		"earnings":        repo.ComputeEarnings(o.Price, o.CommissionRate, o.BonusRate, o.BonusQualified, 1),
		"createdAt":       o.CreatedAt,
	}
}

// CreateTeachingOrderHandler POST /api/teaching-orders (owner, service).
func CreateTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, err := ctx.UserRepo.FindByID(r.Context(), uid)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		var req createTeachingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		if errKey := validateCreateTeaching(req); errKey != "" {
			writeError(w, r, http.StatusBadRequest, errKey, nil)
			return
		}

		// resolve student: existing id, or create a new customer
		studentID, studentName := req.StudentID, ""
		var newCreds map[string]any
		if studentID > 0 {
			su, err := ctx.UserRepo.FindByID(r.Context(), studentID)
			if err != nil || su.Role != "customer" {
				writeError(w, r, http.StatusBadRequest, "invalid_student", nil)
				return
			}
			studentName = su.Nickname
		} else {
			// create a customer with a random password; return creds so owner can hand them over
			pw, _ := genUsername() // reuse repo generator via handler? -> use local random below
			hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "hash_error", err)
				return
			}
			su, err := ctx.UserRepo.CreateWithPassword(r.Context(), strings.TrimSpace(req.NewStudentNickname), string(hash), "customer")
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			studentID, studentName = su.ID, su.Nickname
			newCreds = map[string]any{"username": su.Username.String, "password": pw}
		}

		o := &repo.TeachingOrder{
			Title: strings.TrimSpace(req.Title), StudentID: studentID, StudentName: studentName,
			DispatcherID: uid, DispatcherName: me.Nickname,
			Price: req.Price, CommissionRate: req.CommissionRate, BonusRate: req.BonusRate,
			TotalMinutes: hoursToMinutes(req.TotalHours),
		}
		op := repo.Operator{ID: uid, Name: me.Nickname, Role: me.Role}
		var created *repo.TeachingOrder
		for i := 0; i < 5; i++ {
			o.Code = genTeachingCode()
			created, err = ctx.TeachingRepo.Create(r.Context(), o, op)
			if err == nil {
				break
			}
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		view := toTeachingOrderView(created)
		if newCreds != nil {
			view["newStudentCredentials"] = newCreds
		}
		httpx.OkJsonCtx(r.Context(), w, view)
	})
}

// ListTeachingOrdersHandler GET /api/teaching-orders (owner, service).
func ListTeachingOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		list, err := ctx.TeachingRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(list))
		for _, o := range list {
			views = append(views, toTeachingOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// TeachingOrderDetailHandler GET /api/teaching-orders/:code (owner, service, its teacher, its student).
func TeachingOrderDetailHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, errKey := extractUserID(r, ctx)
		if errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		me, err := ctx.UserRepo.FindByID(r.Context(), uid)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		code, _ := parseTeachingPath(r.URL.Path)
		o, err := ctx.TeachingRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "teaching_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if !canViewTeaching(me, o) {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(o))
	}
}

// canViewTeaching: staff, the assigned teacher, or the student.
func canViewTeaching(u *repo.User, o *repo.TeachingOrder) bool {
	if u.Role == "owner" || u.Role == "service" {
		return true
	}
	if u.Role == "enforcer" && o.TeacherID.Valid && o.TeacherID.Int64 == u.ID {
		return true
	}
	if u.Role == "customer" && o.StudentID == u.ID {
		return true
	}
	return false
}

// MyTeachingHandler GET /api/my-teaching (enforcer: taught; customer: learned).
func MyTeachingHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer", "customer"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, _ := ctx.UserRepo.FindByID(r.Context(), uid)
		var list []*repo.TeachingOrder
		var err error
		if me.Role == "enforcer" {
			list, err = ctx.TeachingRepo.ListByTeacher(r.Context(), uid)
		} else {
			list, err = ctx.TeachingRepo.ListByStudent(r.Context(), uid)
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(list))
		for _, o := range list {
			views = append(views, toTeachingOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// SyllabusHandler GET /api/teaching/syllabus (any authed).
func SyllabusHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, errKey := extractUserID(r, ctx); errKey != "" {
			writeError(w, r, http.StatusUnauthorized, errKey, nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, ctx.Config.TeachingSyllabus)
	}
}

// ListCustomersHandler GET /api/customers (owner, service) — for the student picker.
func ListCustomersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.UserRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		out := make([]map[string]any, 0)
		for _, u := range users {
			if u.Role == "customer" {
				out = append(out, map[string]any{"id": u.ID, "nickname": u.Nickname, "username": u.Username})
			}
		}
		httpx.OkJsonCtx(r.Context(), w, out)
	})
}

// genTeachingCode returns a prefixed timestamp code, isolated from order codes.
func genTeachingCode() string {
	return "JX" + genOrderCode()
}
```

注：`genOrderCode` 已在 `order.go` 定义（同 package 可用）。`genUsername` 是 `repo` 包私有；本文件用它生成学员随机密码不可行 —— 改为在 handler 里内联一个随机密码生成器。在 `teaching.go` 加：

```go
func genStudentPassword() string {
	const chars = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	out := make([]byte, 8)
	for i, c := range b {
		out[i] = chars[int(c)%len(chars)]
	}
	return string(out)
}
```

并把上面 `pw, _ := genUsername()` 改为 `pw := genStudentPassword()`，import 里加 `crand "crypto/rand"`。

- [ ] **Step 2: 编译**

Run: `cd server && go build ./...`
Expected: 通过（handler 未注册路由也能编译）。

- [ ] **Step 3: 运行 handler 纯函数测试确保无回归**

Run: `cd server && go test ./internal/handler/ -v`
Expected: PASS（含 Task 12 的纯函数测试；集成部分不在此）。

- [ ] **Step 4: Commit**

```bash
git add server/internal/handler/teaching.go
git commit -m "feat(teaching): create/list/detail/my-teaching/syllabus/customers handlers"
```

---

## Task 14: 教学 handler —— claim/topup/scorecards/settle/timelogs/review + 路由注册

**Files:**
- Modify: `server/internal/handler/teaching.go`
- Modify: `server/cmd/gameserver/main.go`

**Interfaces:**
- Consumes: `ctx.TeachingRepo` 的 Claim/AddScorecard/ListScorecards/TopUp/ListTimeLogs/Settle/UpsertReviewWithEffects/GetReview；`config.SyllabusItemKeys`；`BonusQualifies`
- Produces（路由处理器）：`ClaimTeachingOrderHandler`, `TopUpTeachingOrderHandler`, `CreateScorecardHandler`, `ListScorecardsHandler`, `SettleTeachingOrderHandler`, `ListTimeLogsHandler`, `UpsertTeachingReviewHandler`, `TeachingReviewHandler`

- [ ] **Step 1: 实现剩余 handler** —— 追加到 `teaching.go`：

```go
func toScorecardView(sc *repo.TeachingScorecard) map[string]any {
	scores := map[string]int{}
	if sc.Scores.Valid && sc.Scores.String != "" {
		_ = json.Unmarshal([]byte(sc.Scores.String), &scores)
	}
	return map[string]any{
		"id":            sc.ID,
		"durationHours": minutesToHours(sc.DurationMinutes),
		"scores":        scores,
		"comment":       sc.Comment.String,
		"teacherName":   sc.TeacherName,
		"createdAt":     sc.CreatedAt,
	}
}

func toTimeLogView(l *repo.TeachingTimeLog) map[string]any {
	return map[string]any{
		"deltaHours":   minutesToHours(l.DeltaMinutes),
		"reason":       l.Reason,
		"priceDelta":   l.PriceDelta,
		"operatorName": l.OperatorName,
		"operatorRole": l.OperatorRole,
		"note":         l.Note.String,
		"createdAt":    l.CreatedAt,
	}
}

func toTeachingReviewView(rv *repo.TeachingReview) map[string]any {
	return map[string]any{
		"attitudeScore":  rv.AttitudeScore,
		"technicalScore": rv.TechnicalScore,
		"total":          rv.AttitudeScore + rv.TechnicalScore,
		"bonusQualified": BonusQualifies(rv.AttitudeScore, rv.TechnicalScore),
		"comment":        rv.Comment.String,
		"reviewerName":   rv.ReviewerName,
		"createdAt":      rv.CreatedAt,
	}
}

// loadTeachingForActor loads the order and the acting user, returning them or writing an error.
func loadTeachingForActor(ctx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) (*repo.User, *repo.TeachingOrder, bool) {
	uid, errKey := extractUserID(r, ctx)
	if errKey != "" {
		writeError(w, r, http.StatusUnauthorized, errKey, nil)
		return nil, nil, false
	}
	me, err := ctx.UserRepo.FindByID(r.Context(), uid)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
		return nil, nil, false
	}
	code, _ := parseTeachingPath(r.URL.Path)
	o, err := ctx.TeachingRepo.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, sqlx.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "teaching_not_found", nil)
			return nil, nil, false
		}
		writeError(w, r, http.StatusInternalServerError, "db_error", err)
		return nil, nil, false
	}
	return me, o, true
}

func isTeacherOf(u *repo.User, o *repo.TeachingOrder) bool {
	return o.TeacherID.Valid && o.TeacherID.Int64 == u.ID
}

// ClaimTeachingOrderHandler POST /api/teaching-orders/:code/claim (enforcer).
func ClaimTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, _ := ctx.UserRepo.FindByID(r.Context(), uid)
		code, _ := parseTeachingPath(r.URL.Path)
		if err := ctx.TeachingRepo.Claim(r.Context(), code, uid, me.Nickname); err != nil {
			if errors.Is(err, repo.ErrTeacherTaken) {
				writeError(w, r, http.StatusConflict, "teacher_taken", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, _ := ctx.TeachingRepo.GetByCode(r.Context(), code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(o))
	})
}

// TopUpTeachingOrderHandler POST /api/teaching-orders/:code/topup (owner).
func TopUpTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner"}, func(w http.ResponseWriter, r *http.Request) {
		uid, _ := extractUserID(r, ctx)
		me, _ := ctx.UserRepo.FindByID(r.Context(), uid)
		code, _ := parseTeachingPath(r.URL.Path)
		var req topUpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AddHours <= 0 || req.AddPrice < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_topup", nil)
			return
		}
		op := repo.Operator{ID: uid, Name: me.Nickname, Role: me.Role}
		if err := ctx.TeachingRepo.TopUp(r.Context(), code, hoursToMinutes(req.AddHours), req.AddPrice, op); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, _ := ctx.TeachingRepo.GetByCode(r.Context(), code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(o))
	})
}

// CreateScorecardHandler POST /api/teaching-orders/:code/scorecards (enforcer = teacher).
func CreateScorecardHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !isTeacherOf(me, o) {
			writeError(w, r, http.StatusForbidden, "not_your_teaching", nil)
			return
		}
		if o.Status != "ongoing" {
			writeError(w, r, http.StatusConflict, "teaching_not_ongoing", nil)
			return
		}
		var req scorecardRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DurationHours <= 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_scorecard", nil)
			return
		}
		if errKey := validateScores(req.Scores, config.SyllabusItemKeys(ctx.Config.TeachingSyllabus)); errKey != "" {
			writeError(w, r, http.StatusBadRequest, errKey, nil)
			return
		}
		scoresJSON := sql.NullString{}
		if len(req.Scores) > 0 {
			b, _ := json.Marshal(req.Scores)
			scoresJSON = sql.NullString{String: string(b), Valid: true}
		}
		sc := &repo.TeachingScorecard{
			TeachingOrderCode: o.Code, DurationMinutes: hoursToMinutes(req.DurationHours),
			Scores: scoresJSON, Comment: sql.NullString{String: strings.TrimSpace(req.Comment), Valid: req.Comment != ""},
			TeacherID: me.ID, TeacherName: me.Nickname,
		}
		saved, err := ctx.TeachingRepo.AddScorecard(r.Context(), o.Code, sc)
		if err != nil {
			if errors.Is(err, repo.ErrInsufficientTime) {
				writeError(w, r, http.StatusConflict, "insufficient_time", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toScorecardView(saved))
	})
}

// ListScorecardsHandler GET /api/teaching-orders/:code/scorecards (owner, teacher, student).
func ListScorecardsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !canViewTeaching(me, o) {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		cards, err := ctx.TeachingRepo.ListScorecards(r.Context(), o.Code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(cards))
		for _, c := range cards {
			views = append(views, toScorecardView(c))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	}
}

// SettleTeachingOrderHandler POST /api/teaching-orders/:code/settle (enforcer = teacher).
func SettleTeachingOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !isTeacherOf(me, o) {
			writeError(w, r, http.StatusForbidden, "not_your_teaching", nil)
			return
		}
		if err := ctx.TeachingRepo.Settle(r.Context(), o.Code); err != nil {
			if errors.Is(err, repo.ErrTeachingNotSettleable) {
				writeError(w, r, http.StatusConflict, "not_settleable", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		updated, _ := ctx.TeachingRepo.GetByCode(r.Context(), o.Code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingOrderView(updated))
	})
}

// ListTimeLogsHandler GET /api/teaching-orders/:code/timelogs (owner).
func ListTimeLogsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseTeachingPath(r.URL.Path)
		logs, err := ctx.TeachingRepo.ListTimeLogs(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(logs))
		for _, l := range logs {
			views = append(views, toTimeLogView(l))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// UpsertTeachingReviewHandler PUT /api/teaching-orders/:code/review (customer = student).
func UpsertTeachingReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"customer"}, func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if o.StudentID != me.ID {
			writeError(w, r, http.StatusForbidden, "not_your_teaching", nil)
			return
		}
		var req teachingReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
			req.AttitudeScore < 0 || req.TechnicalScore < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_review", nil)
			return
		}
		rv := &repo.TeachingReview{
			TeachingOrderCode: o.Code, AttitudeScore: req.AttitudeScore, TechnicalScore: req.TechnicalScore,
			Comment:    sql.NullString{String: strings.TrimSpace(req.Comment), Valid: req.Comment != ""},
			ReviewerID: me.ID, ReviewerName: me.Nickname,
		}
		if err := ctx.TeachingRepo.UpsertReviewWithEffects(r.Context(), rv, BonusQualifies(req.AttitudeScore, req.TechnicalScore)); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		saved, _ := ctx.TeachingRepo.GetReview(r.Context(), o.Code)
		httpx.OkJsonCtx(r.Context(), w, toTeachingReviewView(saved))
	})
}

// TeachingReviewHandler GET /api/teaching-orders/:code/review (owner, teacher, student).
func TeachingReviewHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		me, o, ok := loadTeachingForActor(ctx, w, r)
		if !ok {
			return
		}
		if !canViewTeaching(me, o) {
			writeError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		rv, err := ctx.TeachingRepo.GetReview(r.Context(), o.Code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "review_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toTeachingReviewView(rv))
	}
}
```

在 `teaching.go` 的 import 补上 `"hls.dev/server/internal/config"`。

- [ ] **Step 2: 注册路由** —— 在 `main.go` 的 `AddRoutes` 切片里（`/api/payroll` 之后）追加：

```go
			{Method: http.MethodPost, Path: "/api/teaching-orders", Handler: handler.CreateTeachingOrderHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/teaching-orders", Handler: handler.ListTeachingOrdersHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/teaching-orders/:code", Handler: handler.TeachingOrderDetailHandler(ctx)},
			{Method: http.MethodPost, Path: "/api/teaching-orders/:code/claim", Handler: handler.ClaimTeachingOrderHandler(ctx)},
			{Method: http.MethodPost, Path: "/api/teaching-orders/:code/topup", Handler: handler.TopUpTeachingOrderHandler(ctx)},
			{Method: http.MethodPost, Path: "/api/teaching-orders/:code/settle", Handler: handler.SettleTeachingOrderHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/teaching-orders/:code/scorecards", Handler: handler.ListScorecardsHandler(ctx)},
			{Method: http.MethodPost, Path: "/api/teaching-orders/:code/scorecards", Handler: handler.CreateScorecardHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/teaching-orders/:code/timelogs", Handler: handler.ListTimeLogsHandler(ctx)},
			{Method: http.MethodPut, Path: "/api/teaching-orders/:code/review", Handler: handler.UpsertTeachingReviewHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/teaching-orders/:code/review", Handler: handler.TeachingReviewHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/my-teaching", Handler: handler.MyTeachingHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/teaching/syllabus", Handler: handler.SyllabusHandler(ctx)},
			{Method: http.MethodGet, Path: "/api/customers", Handler: handler.ListCustomersHandler(ctx)},
```

> 注意 go-zero 路由匹配：`/api/teaching-orders/:code` 与 `/api/teaching/syllabus` 前缀不同（`teaching-orders` vs `teaching`），不冲突。

- [ ] **Step 3: 编译 + 全量后端测试**

Run: `cd server && go build ./... && go test ./...`
Expected: 编译通过；纯函数测试 PASS；集成测试在无 `TEST_DB_DSN` 时 SKIP。

- [ ] **Step 4: 冒烟（可选，有本地库时）**

启动服务 `cd server && go run ./cmd/gameserver -f etc/gameserver.yaml`，用店长 token：
Run: `curl -s -H "Authorization: Bearer $OWNER_JWT" localhost:8888/api/teaching/syllabus | head -c 200`
Expected: 返回大纲 JSON（3 个分类）。

- [ ] **Step 5: Commit**

```bash
git add server/internal/handler/teaching.go server/cmd/gameserver/main.go
git commit -m "feat(teaching): claim/topup/scorecard/settle/timelogs/review handlers and routes"
```

---

## Task 15: 前端 API 客户端与大纲 store

**Files:**
- Create: `src/api/teaching.ts`
- Create: `src/stores/syllabus.ts`

**Interfaces:**
- Produces（供 Panel 用）：类型 `TeachingOrderView`, `ScorecardView`, `TimeLogView`, `TeachingReviewView`, `SyllabusCategory`, `CustomerOption`；函数 `listTeachingOrders`, `getTeachingOrder`, `createTeachingOrder`, `claimTeachingOrder`, `topUpTeachingOrder`, `settleTeachingOrder`, `listScorecards`, `createScorecard`, `listTimeLogs`, `getTeachingReview`, `upsertTeachingReview`, `listMyTeaching`, `getSyllabus`, `listCustomers`；`useSyllabusStore`。

- [ ] **Step 1: 写 `src/api/teaching.ts`**

```ts
import http from '../http'

export interface TeachingOrderView {
  code: string
  title: string
  student: { id: number; name: string }
  dispatcherName: string
  teacher: { id: number; name: string } | null
  price: number
  commissionRate: number
  bonusRate: number
  bonusQualified: boolean
  totalHours: number
  remainingHours: number
  totalMinutes: number
  remainingMinutes: number
  status: 'dispatching' | 'ongoing' | 'settled'
  earnings: number
  createdAt: string
  newStudentCredentials?: { username: string; password: string }
}

export interface ScorecardView {
  id: number
  durationHours: number
  scores: Record<string, number>
  comment: string
  teacherName: string
  createdAt: string
}

export interface TimeLogView {
  deltaHours: number
  reason: 'initial' | 'topup' | 'lesson'
  priceDelta: number
  operatorName: string
  operatorRole: string
  note: string
  createdAt: string
}

export interface TeachingReviewView {
  attitudeScore: number
  technicalScore: number
  total: number
  bonusQualified: boolean
  comment: string
  reviewerName: string
  createdAt: string
}

export interface SyllabusItem { key: string; label: string; desc: string }
export interface SyllabusCategory { key: string; label: string; items: SyllabusItem[] }
export interface CustomerOption { id: number; nickname: string; username: string }

export interface CreateTeachingPayload {
  title: string
  studentId?: number
  newStudentNickname?: string
  totalHours: number
  price: number
  commissionRate: number
  bonusRate: number
}

export async function listTeachingOrders(): Promise<TeachingOrderView[]> {
  return (await http.get<TeachingOrderView[]>('/teaching-orders')).data
}
export async function getTeachingOrder(code: string): Promise<TeachingOrderView> {
  return (await http.get<TeachingOrderView>(`/teaching-orders/${code}`)).data
}
export async function createTeachingOrder(p: CreateTeachingPayload): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>('/teaching-orders', p)).data
}
export async function claimTeachingOrder(code: string): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>(`/teaching-orders/${code}/claim`)).data
}
export async function topUpTeachingOrder(code: string, addHours: number, addPrice: number): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>(`/teaching-orders/${code}/topup`, { addHours, addPrice })).data
}
export async function settleTeachingOrder(code: string): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>(`/teaching-orders/${code}/settle`)).data
}
export async function listScorecards(code: string): Promise<ScorecardView[]> {
  return (await http.get<ScorecardView[]>(`/teaching-orders/${code}/scorecards`)).data
}
export async function createScorecard(code: string, durationHours: number, scores: Record<string, number>, comment: string): Promise<ScorecardView> {
  return (await http.post<ScorecardView>(`/teaching-orders/${code}/scorecards`, { durationHours, scores, comment })).data
}
export async function listTimeLogs(code: string): Promise<TimeLogView[]> {
  return (await http.get<TimeLogView[]>(`/teaching-orders/${code}/timelogs`)).data
}
export async function getTeachingReview(code: string): Promise<TeachingReviewView | null> {
  try {
    return (await http.get<TeachingReviewView>(`/teaching-orders/${code}/review`)).data
  } catch {
    return null
  }
}
export async function upsertTeachingReview(code: string, attitudeScore: number, technicalScore: number, comment: string): Promise<TeachingReviewView> {
  return (await http.put<TeachingReviewView>(`/teaching-orders/${code}/review`, { attitudeScore, technicalScore, comment })).data
}
export async function listMyTeaching(): Promise<TeachingOrderView[]> {
  return (await http.get<TeachingOrderView[]>('/my-teaching')).data
}
export async function getSyllabus(): Promise<SyllabusCategory[]> {
  return (await http.get<SyllabusCategory[]>('/teaching/syllabus')).data
}
export async function listCustomers(): Promise<CustomerOption[]> {
  return (await http.get<CustomerOption[]>('/customers')).data
}
```

- [ ] **Step 2: 写 `src/stores/syllabus.ts`**

```ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getSyllabus, type SyllabusCategory } from '../api/teaching'

export const useSyllabusStore = defineStore('syllabus', () => {
  const categories = ref<SyllabusCategory[]>([])
  const loaded = ref(false)

  async function ensureLoaded() {
    if (loaded.value) return
    categories.value = await getSyllabus()
    loaded.value = true
  }

  return { categories, loaded, ensureLoaded }
})
```

- [ ] **Step 3: typecheck**

Run: `npm run typecheck`
Expected: 通过（新文件类型自洽；未被引用不报错）。

- [ ] **Step 4: Commit**

```bash
git add src/api/teaching.ts src/stores/syllabus.ts
git commit -m "feat(fe): add teaching api client and syllabus store"
```

---

## Task 16: 前端路由放行 customer + AdminPage 三个 tab

**Files:**
- Modify: `src/router/index.ts:71`
- Modify: `src/pages/AdminPage.vue`

**Interfaces:**
- Consumes: 三个 Panel 组件（Task 17 创建；本任务先建空壳以便挂载）
- Produces: `/admin` 放行 `customer`；tab keys `'teaching' | 'my-teaching-teacher' | 'my-teaching-student'`

- [ ] **Step 1: 路由 guard 放行 customer** —— `src/router/index.ts` 把 `const allowed = ['owner', 'service', 'enforcer']` 改为：

```ts
    const allowed = ['owner', 'service', 'enforcer', 'customer']
```

- [ ] **Step 2: 建三个 Panel 空壳**（Task 17 会填充；先让 AdminPage 能编译）

`src/pages/admin/TeachingPanel.vue`、`MyTeachingTeacherPanel.vue`、`MyTeachingStudentPanel.vue` 各写占位：

```vue
<script setup lang="ts"></script>
<template>
  <div class="font-body text-sm text-hls-text-light">教学面板占位</div>
</template>
```

- [ ] **Step 3: AdminPage 挂 tab** —— 在 `src/pages/AdminPage.vue`：

import 追加：
```ts
import TeachingPanel from './admin/TeachingPanel.vue'
import MyTeachingTeacherPanel from './admin/MyTeachingTeacherPanel.vue'
import MyTeachingStudentPanel from './admin/MyTeachingStudentPanel.vue'
```

`TabKey` 类型改为：
```ts
type TabKey = 'orders' | 'users' | 'my-orders' | 'enforcers' | 'payroll' | 'teaching' | 'my-teaching-teacher' | 'my-teaching-student'
```

`allTabs` 追加：
```ts
  { key: 'teaching', label: '教学管理', roles: ['owner', 'service'] },
  { key: 'my-teaching-teacher', label: '我的教学', roles: ['enforcer'] },
  { key: 'my-teaching-student', label: '我的教学', roles: ['customer'] },
```

`<main>` 里追加渲染：
```html
      <TeachingPanel v-else-if="active === 'teaching'" />
      <MyTeachingTeacherPanel v-else-if="active === 'my-teaching-teacher'" />
      <MyTeachingStudentPanel v-else-if="active === 'my-teaching-student'" />
```

- [ ] **Step 4: typecheck + build**

Run: `npm run typecheck && npm run build`
Expected: 通过。

- [ ] **Step 5: Commit**

```bash
git add src/router/index.ts src/pages/AdminPage.vue src/pages/admin/TeachingPanel.vue src/pages/admin/MyTeachingTeacherPanel.vue src/pages/admin/MyTeachingStudentPanel.vue
git commit -m "feat(fe): route customer into /admin and add teaching tabs"
```

---

## Task 17: 三个教学 Panel 实现

> 每个 Panel 复用现有 `PayrollPanel.vue` / `EnforcersPanel.vue` 的排版风格（`hls-panel` / `hls-stat` / `hls-btn-secondary` / 卡片 + 表格）。下方给出**完整可运行的最小组件**；样式类沿用现有 token。

**Files:**
- Modify: `src/pages/admin/TeachingPanel.vue`
- Modify: `src/pages/admin/MyTeachingTeacherPanel.vue`
- Modify: `src/pages/admin/MyTeachingStudentPanel.vue`

**Interfaces:**
- Consumes: `src/api/teaching.ts` 全部函数、`useSyllabusStore`

- [ ] **Step 1: `TeachingPanel.vue`（owner/service：建单/列表/续时/成绩单/台账）**

```vue
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '../../stores/auth'
import {
  listTeachingOrders, createTeachingOrder, topUpTeachingOrder,
  listScorecards, listTimeLogs, listCustomers,
  type TeachingOrderView, type ScorecardView, type TimeLogView, type CustomerOption,
} from '../../api/teaching'

const auth = useAuthStore()
const isOwner = computed(() => auth.user?.role === 'owner')

const rows = ref<TeachingOrderView[]>([])
const customers = ref<CustomerOption[]>([])
const loading = ref(false)
const err = ref('')

// create form
const form = ref({ title: '', studentId: 0, newStudentNickname: '', totalHours: 6, price: 0, commissionRate: 20, bonusRate: 0, mode: 'existing' as 'existing' | 'new' })
const creating = ref(false)
const createdCreds = ref<{ username: string; password: string } | null>(null)

// detail drawer
const selected = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])
const logs = ref<TimeLogView[]>([])
const topUp = ref({ addHours: 1, addPrice: 0 })

async function load() {
  loading.value = true; err.value = ''
  try {
    rows.value = await listTeachingOrders()
    customers.value = await listCustomers()
  } catch { err.value = '加载失败' } finally { loading.value = false }
}
onMounted(load)

async function submitCreate() {
  creating.value = true; createdCreds.value = null
  try {
    const payload = {
      title: form.value.title, totalHours: form.value.totalHours, price: form.value.price,
      commissionRate: form.value.commissionRate, bonusRate: form.value.bonusRate,
      ...(form.value.mode === 'existing' ? { studentId: form.value.studentId } : { newStudentNickname: form.value.newStudentNickname }),
    }
    const created = await createTeachingOrder(payload)
    if (created.newStudentCredentials) createdCreds.value = created.newStudentCredentials
    form.value.title = ''; form.value.newStudentNickname = ''; form.value.price = 0
    await load()
  } catch { window.alert('建单失败，请检查输入') } finally { creating.value = false }
}

async function openDetail(o: TeachingOrderView) {
  selected.value = o
  cards.value = await listScorecards(o.code)
  logs.value = isOwner.value ? await listTimeLogs(o.code) : []
}

async function doTopUp() {
  if (!selected.value) return
  try {
    await topUpTeachingOrder(selected.value.code, topUp.value.addHours, topUp.value.addPrice)
    await load()
    const fresh = rows.value.find(r => r.code === selected.value!.code)
    if (fresh) await openDetail(fresh)
  } catch { window.alert('续时长失败') }
}

function statusLabel(s: string) { return s === 'dispatching' ? '待接单' : s === 'ongoing' ? '教学中' : '已结单' }
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">教学管理</h2>
      <button class="hls-btn-secondary px-4 py-2" :disabled="loading" @click="load">{{ loading ? '刷新中…' : '刷新' }}</button>
    </div>
    <p v-if="err" class="text-sm text-red-400">{{ err }}</p>

    <!-- create form -->
    <div class="hls-panel p-4 space-y-3">
      <div class="hls-panel-title">新建教学单</div>
      <input v-model="form.title" placeholder="标题" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
      <div class="flex gap-2 text-sm">
        <label><input type="radio" value="existing" v-model="form.mode" /> 选已有学员</label>
        <label><input type="radio" value="new" v-model="form.mode" /> 新建学员</label>
      </div>
      <select v-if="form.mode === 'existing'" v-model.number="form.studentId" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm">
        <option :value="0" disabled>选择学员</option>
        <option v-for="c in customers" :key="c.id" :value="c.id">{{ c.nickname }}（{{ c.username }}）</option>
      </select>
      <input v-else v-model="form.newStudentNickname" placeholder="新学员昵称" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <label class="text-xs text-hls-text-muted">总课时(小时)<input type="number" step="0.5" v-model.number="form.totalHours" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
        <label class="text-xs text-hls-text-muted">价格<input type="number" v-model.number="form.price" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
        <label class="text-xs text-hls-text-muted">分成率%<input type="number" v-model.number="form.commissionRate" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
        <label class="text-xs text-hls-text-muted">奉金率%<input type="number" v-model.number="form.bonusRate" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" /></label>
      </div>
      <button class="hls-btn-secondary px-4 py-2 text-sm" :disabled="creating" @click="submitCreate">{{ creating ? '创建中…' : '创建' }}</button>
      <p v-if="createdCreds" class="text-sm text-emerald-700">新学员登录：账号 {{ createdCreds.username }} / 密码 {{ createdCreds.password }}（请转交学员）</p>
    </div>

    <!-- list -->
    <div class="space-y-2">
      <article v-for="o in rows" :key="o.code" class="hls-panel p-4 space-y-1 cursor-pointer" @click="openDetail(o)">
        <div class="flex justify-between">
          <p class="font-bold text-hls-text-dark">{{ o.title }}</p>
          <span class="hls-badge hls-badge-gold">{{ statusLabel(o.status) }}</span>
        </div>
        <p class="text-xs text-hls-text-light">{{ o.code }} · 学员 {{ o.student.name }} · 老师 {{ o.teacher?.name ?? '未接单' }}</p>
        <p class="text-xs text-hls-text-light">剩余 {{ o.remainingHours }}h / 共 {{ o.totalHours }}h · 应发 {{ o.earnings }}</p>
      </article>
      <p v-if="!loading && !rows.length" class="text-sm text-hls-text-light">暂无教学单</p>
    </div>

    <!-- detail -->
    <div v-if="selected" class="hls-panel p-4 space-y-4">
      <div class="flex justify-between">
        <div class="hls-panel-title">教学单详情 · {{ selected.title }}</div>
        <button class="text-sm text-hls-text-nav" @click="selected = null">关闭 ✕</button>
      </div>
      <div v-if="isOwner" class="rounded border border-hls-border/40 p-3 space-y-2">
        <p class="text-sm font-semibold">续时长</p>
        <div class="flex gap-2 items-end">
          <label class="text-xs">加课时(小时)<input type="number" step="0.5" v-model.number="topUp.addHours" class="w-full rounded border px-2 py-1 text-sm" /></label>
          <label class="text-xs">加价格<input type="number" v-model.number="topUp.addPrice" class="w-full rounded border px-2 py-1 text-sm" /></label>
          <button class="hls-btn-secondary px-3 py-1.5 text-sm" @click="doTopUp">续时</button>
        </div>
      </div>
      <div>
        <p class="text-sm font-semibold mb-2">成绩单（{{ cards.length }}）</p>
        <div v-for="c in cards" :key="c.id" class="rounded border border-hls-border/30 p-2 mb-2 text-sm">
          <p class="text-xs text-hls-text-light">{{ c.teacherName }} · {{ c.durationHours }}h · {{ new Date(c.createdAt).toLocaleString('zh-CN') }}</p>
          <p v-if="c.comment" class="text-hls-text-dark">{{ c.comment }}</p>
          <p class="text-xs">{{ Object.entries(c.scores).map(([k, v]) => `${k}:${v}`).join('  ') }}</p>
        </div>
      </div>
      <div v-if="isOwner && logs.length">
        <p class="text-sm font-semibold mb-2">时长台账</p>
        <p v-for="(l, i) in logs" :key="i" class="text-xs text-hls-text-light">
          {{ new Date(l.createdAt).toLocaleString('zh-CN') }} · {{ l.reason }} · {{ l.deltaHours }}h · {{ l.operatorName }} · {{ l.note }}
        </p>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: `MyTeachingTeacherPanel.vue`（enforcer：可领/我教的、自领、录成绩单、结单）**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listMyTeaching, listTeachingOrders, claimTeachingOrder, createScorecard, settleTeachingOrder, listScorecards,
  type TeachingOrderView, type ScorecardView,
} from '../../api/teaching'

const syllabus = useSyllabusStore()
const mine = ref<TeachingOrderView[]>([])
const claimable = ref<TeachingOrderView[]>([])
const loading = ref(false)
const selected = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])

const form = ref<{ durationHours: number; scores: Record<string, number>; comment: string }>({ durationHours: 1, scores: {}, comment: '' })
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    await syllabus.ensureLoaded()
    mine.value = await listMyTeaching()
    // claimable list comes from the staff list endpoint is owner/service-only; teachers see claimable via a dedicated view:
    // reuse my-teaching for taught; claimable requires status dispatching — fetch via list is staff-only,
    // so teachers claim by code shared out-of-band. Here we surface dispatching ones they can see through detail links only.
    claimable.value = []
  } catch { /* ignore */ } finally { loading.value = false }
}
onMounted(load)

async function open(o: TeachingOrderView) {
  selected.value = o
  form.value = { durationHours: 1, scores: {}, comment: '' }
  cards.value = await listScorecards(o.code)
}

async function submitCard() {
  if (!selected.value) return
  saving.value = true
  try {
    await createScorecard(selected.value.code, form.value.durationHours, form.value.scores, form.value.comment)
    mine.value = await listMyTeaching()
    const fresh = mine.value.find(r => r.code === selected.value!.code)
    if (fresh) await open(fresh)
  } catch (e) { window.alert('录入失败：课时可能不足') } finally { saving.value = false }
}

async function settle(o: TeachingOrderView) {
  if (!window.confirm('确认结单发工资？需剩余课时为 0。')) return
  try { await settleTeachingOrder(o.code); mine.value = await listMyTeaching(); selected.value = null }
  catch { window.alert('结单失败：剩余课时未清零或状态不符') }
}

async function claimByCode() {
  const code = window.prompt('输入教学单编号自领为老师')
  if (!code) return
  try { await claimTeachingOrder(code.trim()); mine.value = await listMyTeaching() }
  catch { window.alert('自领失败：已被接单或不存在') }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">我的教学</h2>
      <div class="flex gap-2">
        <button class="hls-btn-secondary px-3 py-2 text-sm" @click="claimByCode">按编号接单</button>
        <button class="hls-btn-secondary px-3 py-2 text-sm" :disabled="loading" @click="load">刷新</button>
      </div>
    </div>

    <article v-for="o in mine" :key="o.code" class="hls-panel p-4 space-y-1">
      <div class="flex justify-between">
        <p class="font-bold text-hls-text-dark">{{ o.title }}</p>
        <span class="hls-badge hls-badge-gold">{{ o.status === 'ongoing' ? '教学中' : o.status === 'settled' ? '已结单' : '待接单' }}</span>
      </div>
      <p class="text-xs text-hls-text-light">学员 {{ o.student.name }} · 剩余 {{ o.remainingHours }}h / 共 {{ o.totalHours }}h</p>
      <div class="flex gap-2 pt-1">
        <button class="hls-btn-secondary px-3 py-1.5 text-xs" @click="open(o)">录成绩单 / 查看</button>
        <button v-if="o.status === 'ongoing' && o.remainingMinutes === 0" class="hls-btn-secondary px-3 py-1.5 text-xs" @click="settle(o)">结单发工资</button>
      </div>
    </article>
    <p v-if="!loading && !mine.length" class="text-sm text-hls-text-light">暂无教学单（可用「按编号接单」自领）</p>

    <!-- scorecard editor -->
    <div v-if="selected" class="hls-panel p-4 space-y-4">
      <div class="flex justify-between">
        <div class="hls-panel-title">录成绩单 · {{ selected.title }}（剩余 {{ selected.remainingHours }}h）</div>
        <button class="text-sm text-hls-text-nav" @click="selected = null">关闭 ✕</button>
      </div>
      <label class="text-xs text-hls-text-muted block">本节课时长(小时)
        <input type="number" step="0.5" v-model.number="form.durationHours" class="w-full rounded border border-hls-border/40 px-2 py-1.5 text-sm" />
      </label>
      <div v-for="cat in syllabus.categories" :key="cat.key" class="space-y-2">
        <p class="text-sm font-semibold text-hls-gold-dark">{{ cat.label }}</p>
        <div v-for="it in cat.items" :key="it.key" class="flex items-center justify-between gap-2">
          <span class="text-sm" :title="it.desc">{{ it.label }}</span>
          <input type="number" min="0" max="10" placeholder="—" v-model.number="form.scores[it.key]" class="w-16 rounded border border-hls-border/40 px-2 py-1 text-sm" />
        </div>
      </div>
      <textarea v-model="form.comment" placeholder="评语（可选）" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
      <button class="hls-btn-secondary px-4 py-2 text-sm" :disabled="saving" @click="submitCard">{{ saving ? '保存中…' : '保存成绩单' }}</button>

      <div class="pt-2">
        <p class="text-sm font-semibold mb-2">历史成绩单（{{ cards.length }}）</p>
        <div v-for="c in cards" :key="c.id" class="rounded border border-hls-border/30 p-2 mb-2 text-xs">
          <p class="text-hls-text-light">{{ c.durationHours }}h · {{ new Date(c.createdAt).toLocaleString('zh-CN') }}</p>
          <p v-if="c.comment">{{ c.comment }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
```

> 注：`v-model.number` 对留空项会得到 `undefined`，`createScorecard` 里 `scores` 只提交已填项。为避免把空值发成 `null`，提交前在 `submitCard` 里过滤：把下面这行加到 `submitCard` 组装前 —— `const scores = Object.fromEntries(Object.entries(form.value.scores).filter(([, v]) => typeof v === 'number' && !Number.isNaN(v)))`，并用 `scores` 替换 `form.value.scores` 传入。

- [ ] **Step 3: `MyTeachingStudentPanel.vue`（customer：列表/剩余/成绩单只读/评价老师）**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSyllabusStore } from '../../stores/syllabus'
import {
  listMyTeaching, listScorecards, getTeachingReview, upsertTeachingReview,
  type TeachingOrderView, type ScorecardView, type TeachingReviewView,
} from '../../api/teaching'

const syllabus = useSyllabusStore()
const rows = ref<TeachingOrderView[]>([])
const loading = ref(false)
const selected = ref<TeachingOrderView | null>(null)
const cards = ref<ScorecardView[]>([])
const review = ref<TeachingReviewView | null>(null)
const reviewForm = ref({ attitudeScore: 5, technicalScore: 5, comment: '' })

function labelOf(key: string): string {
  for (const c of syllabus.categories) for (const it of c.items) if (it.key === key) return it.label
  return key
}

async function load() {
  loading.value = true
  try { await syllabus.ensureLoaded(); rows.value = await listMyTeaching() }
  finally { loading.value = false }
}
onMounted(load)

async function open(o: TeachingOrderView) {
  selected.value = o
  cards.value = await listScorecards(o.code)
  review.value = await getTeachingReview(o.code)
  if (review.value) reviewForm.value = { attitudeScore: review.value.attitudeScore, technicalScore: review.value.technicalScore, comment: review.value.comment }
}

async function submitReview() {
  if (!selected.value) return
  try {
    review.value = await upsertTeachingReview(selected.value.code, reviewForm.value.attitudeScore, reviewForm.value.technicalScore, reviewForm.value.comment)
    window.alert('评价已提交')
  } catch { window.alert('评价失败') }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="font-body font-bold text-xl text-hls-gold">我的教学</h2>
      <button class="hls-btn-secondary px-3 py-2 text-sm" :disabled="loading" @click="load">刷新</button>
    </div>

    <article v-for="o in rows" :key="o.code" class="hls-panel p-4 space-y-1 cursor-pointer" @click="open(o)">
      <p class="font-bold text-hls-text-dark">{{ o.title }}</p>
      <p class="text-xs text-hls-text-light">老师 {{ o.teacher?.name ?? '未接单' }} · 剩余 {{ o.remainingHours }}h / 共 {{ o.totalHours }}h</p>
    </article>
    <p v-if="!loading && !rows.length" class="text-sm text-hls-text-light">暂无教学单</p>

    <div v-if="selected" class="hls-panel p-4 space-y-4">
      <div class="flex justify-between">
        <div class="hls-panel-title">{{ selected.title }} · 剩余 {{ selected.remainingHours }}h</div>
        <button class="text-sm text-hls-text-nav" @click="selected = null">关闭 ✕</button>
      </div>

      <div>
        <p class="text-sm font-semibold mb-2">成绩单（{{ cards.length }}）</p>
        <div v-for="c in cards" :key="c.id" class="rounded border border-hls-border/30 p-2 mb-2 text-sm">
          <p class="text-xs text-hls-text-light">{{ c.teacherName }} · {{ c.durationHours }}h · {{ new Date(c.createdAt).toLocaleString('zh-CN') }}</p>
          <p v-if="c.comment" class="text-hls-text-dark">{{ c.comment }}</p>
          <p class="text-xs">{{ Object.entries(c.scores).map(([k, v]) => `${labelOf(k)} ${v}`).join(' · ') }}</p>
        </div>
      </div>

      <div class="rounded border border-hls-border/40 p-3 space-y-2">
        <p class="text-sm font-semibold">评价老师</p>
        <label class="text-xs block">态度分 <input type="number" min="0" max="5" v-model.number="reviewForm.attitudeScore" class="w-16 rounded border px-2 py-1 text-sm" /></label>
        <label class="text-xs block">教学分 <input type="number" min="0" max="5" v-model.number="reviewForm.technicalScore" class="w-16 rounded border px-2 py-1 text-sm" /></label>
        <textarea v-model="reviewForm.comment" placeholder="评语（可选）" class="w-full rounded border border-hls-border/40 px-3 py-2 text-sm" />
        <button class="hls-btn-secondary px-4 py-2 text-sm" @click="submitReview">提交评价</button>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 4: 应用 Teacher 面板的过滤修正** —— 按 Step 2 注记，把 `MyTeachingTeacherPanel.vue` 的 `submitCard` 改为提交过滤后的 scores：

```ts
    const scores = Object.fromEntries(
      Object.entries(form.value.scores).filter(([, v]) => typeof v === 'number' && !Number.isNaN(v)),
    ) as Record<string, number>
    await createScorecard(selected.value.code, form.value.durationHours, scores, form.value.comment)
```

- [ ] **Step 5: typecheck + build**

Run: `npm run typecheck && npm run build`
Expected: 通过。

- [ ] **Step 6: Commit**

```bash
git add src/pages/admin/TeachingPanel.vue src/pages/admin/MyTeachingTeacherPanel.vue src/pages/admin/MyTeachingStudentPanel.vue
git commit -m "feat(fe): implement teaching admin/teacher/student panels"
```

---

## Task 18: 端到端验证与收尾

**Files:** 无新增（验证 + 可能的小修）

- [ ] **Step 1: 后端全量**

Run: `cd server && go build ./... && go vet ./... && go test ./...`
Expected: 编译/vet 通过；纯函数测试 PASS；集成测试 SKIP（无库）或 PASS（有库）。

- [ ] **Step 2: 集成测试（有本地库时）**

Run: `cd server && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run Teaching -v && TEST_DB_DSN="$TEST_DB_DSN" go test ./internal/repo/ -run 'TestPayTeachingPayroll|TestSettleWithPayroll' -v`
Expected: 全 PASS。

- [ ] **Step 3: 前端全量**

Run: `npm run typecheck && npm run build`
Expected: 通过。

- [ ] **Step 4: 手工冒烟（有本地环境时）** 走一遍主流程：
  1. 店长/客服 建教学单（新建学员，记下返回的账号密码）。
  2. 打手用「按编号接单」自领。
  3. 打手录成绩单（填时长 + 部分评分 + 评语），剩余课时下降。
  4. 续时长（店长）→ 剩余与总时长、价格增加，台账新增 topup 行。
  5. 打手把剩余课时录到 0 → 结单 → 「工资管理」待发（结单冷静期后）出现该教学工资条（标题=教学单标题）。
  6. 学员登录 → 我的教学 → 看到成绩单；提交评价（态度+技术≥8）。
  7. 店长在工资管理发放该教学工资，金额 = 价格×(1−分成%) + 达标奉金，经押金/罚款扣减。

- [ ] **Step 5: 最终提交（若有冒烟修复）**

```bash
git add -A
git commit -m "chore(teaching): end-to-end verification fixes"
```

---

## Self-Review（已核对）

- **Spec 覆盖**：4 张表 + payroll source 列 → Task 1；大纲配置 → Task 2；ComputeEarnings/BonusQualifies 抽取 → Task 3；建单/自领/录课扣时/续时/结单/评价 repo → Task 4–9；工资流接入（Pay 分支 + 面板合并） → Task 10–11；handler + 路由 → Task 12–14；前端 api/store/路由/tab/三面板 → Task 15–17；端到端 → Task 18。全部 spec 小节均有对应任务。
- **Placeholder 扫描**：无 TBD/TODO；每个代码步骤给出完整可运行代码。
- **类型一致性**：`TeachingOrder`/`TeachingScorecard`/`Operator` 字段在 repo 各任务间一致；`ComputeEarnings` 五参签名在 Task 3/10/11/13 一致；`payrollSource`/`projectPayrollView` 新签名在 Task 11 内自洽；前端 `TeachingOrderView` 等类型在 Task 15 定义、Task 17 消费一致。
- **已知取舍（非阻塞）**：打手端「可领教学单」列表接口为 staff-only（`GET /api/teaching-orders`），故 Teacher 面板用「按编号接单」自领（编号由店长线下转交），符合"打手自领"语义且不给打手开放全量列表权限；若后续要打手浏览可领单，可加 `GET /api/teaching-orders/claimable`（enforcer 可见 dispatching 列表）——留作增量。

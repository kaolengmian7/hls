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

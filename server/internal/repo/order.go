package repo

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Order mirrors a row in the orders table.
type Order struct {
	ID              int64          `db:"id"               json:"id"`
	Code            string         `db:"code"             json:"code"`
	Title           string         `db:"title"            json:"title"`
	OrderType       string         `db:"order_type"       json:"orderType"`
	Price           float64        `db:"price"            json:"price"`
	CommissionRate  float64        `db:"commission_rate"  json:"commissionRate"`
	BonusRate       float64        `db:"bonus_rate"       json:"bonusRate"`
	PlayerCount     int            `db:"player_count"     json:"playerCount"`
	DispatcherID    int64          `db:"dispatcher_id"    json:"dispatcherId"`
	DispatcherName  string         `db:"dispatcher_name"  json:"dispatcherName"`
	EnforcerAID     sql.NullInt64  `db:"enforcer_a_id"    json:"-"`
	EnforcerAName   sql.NullString `db:"enforcer_a_name"  json:"-"`
	EnforcerBID     sql.NullInt64  `db:"enforcer_b_id"    json:"-"`
	EnforcerBName   sql.NullString `db:"enforcer_b_name"  json:"-"`
	Remark          sql.NullString `db:"remark"           json:"-"`
	Status          string         `db:"status"           json:"status"`
	ComplaintStatus string         `db:"complaint_status" json:"complaintStatus"`
	BonusQualified  bool           `db:"bonus_qualified"  json:"bonusQualified"`
	CreatedAt       time.Time      `db:"created_at"       json:"createdAt"`
	UpdatedAt       time.Time      `db:"updated_at"       json:"updatedAt"`
}

// Earnings returns per-player earnings, rounded to two decimals.
// Formula: (price*(1-commission%/100) + price*bonus%/100) / playerCount.
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

// ErrSlotTaken is returned when assigning to an already-filled enforcer slot.
var ErrSlotTaken = errors.New("slot taken")

// ErrNotSettleable is returned when a guarded settle finds no matching order/status.
var ErrNotSettleable = errors.New("not settleable")

const orderCols = "id, code, title, order_type, price, commission_rate, bonus_rate, " +
	"player_count, dispatcher_id, dispatcher_name, enforcer_a_id, enforcer_a_name, " +
	"enforcer_b_id, enforcer_b_name, remark, status, complaint_status, bonus_qualified, created_at, updated_at"

// OrderRepo provides database access for orders.
type OrderRepo struct {
	db sqlx.SqlConn
}

// NewOrderRepo creates an OrderRepo.
func NewOrderRepo(db sqlx.SqlConn) *OrderRepo {
	return &OrderRepo{db: db}
}

// Create inserts a new order (by o.Code) and returns the stored row.
func (r *OrderRepo) Create(ctx context.Context, o *Order) (*Order, error) {
	_, err := r.db.ExecCtx(ctx,
		"INSERT INTO orders (code, title, order_type, price, commission_rate, bonus_rate, "+
			"player_count, dispatcher_id, dispatcher_name, remark, status) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'dispatching')",
		o.Code, o.Title, o.OrderType, o.Price, o.CommissionRate, o.BonusRate,
		o.PlayerCount, o.DispatcherID, o.DispatcherName, o.Remark)
	if err != nil {
		return nil, err
	}
	return r.GetByCode(ctx, o.Code)
}

// GetByCode returns the order with the given code, or sqlx.ErrNotFound.
func (r *OrderRepo) GetByCode(ctx context.Context, code string) (*Order, error) {
	var o Order
	err := r.db.QueryRowCtx(ctx, &o, "SELECT "+orderCols+" FROM orders WHERE code = ?", code)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListAll returns all orders, newest first.
func (r *OrderRepo) ListAll(ctx context.Context) ([]*Order, error) {
	var list []*Order
	err := r.db.QueryRowsCtx(ctx, &list, "SELECT "+orderCols+" FROM orders ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListByEnforcer returns orders where the user occupies slot A or B, newest first.
func (r *OrderRepo) ListByEnforcer(ctx context.Context, userID int64) ([]*Order, error) {
	var list []*Order
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+orderCols+" FROM orders WHERE enforcer_a_id = ? OR enforcer_b_id = ? ORDER BY id DESC",
		userID, userID)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// AssignEnforcer fills slot "a" or "b" only if currently NULL; else ErrSlotTaken.
func (r *OrderRepo) AssignEnforcer(ctx context.Context, code, slot string, uid int64, name string) error {
	idCol, nameCol := "enforcer_a_id", "enforcer_a_name"
	if slot == "b" {
		idCol, nameCol = "enforcer_b_id", "enforcer_b_name"
	}
	res, err := r.db.ExecCtx(ctx,
		"UPDATE orders SET "+idCol+" = ?, "+nameCol+" = ? WHERE code = ? AND "+idCol+" IS NULL",
		uid, name, code)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSlotTaken
	}
	return nil
}

// RemoveEnforcer clears slot "a" or "b".
func (r *OrderRepo) RemoveEnforcer(ctx context.Context, code, slot string) error {
	idCol, nameCol := "enforcer_a_id", "enforcer_a_name"
	if slot == "b" {
		idCol, nameCol = "enforcer_b_id", "enforcer_b_name"
	}
	_, err := r.db.ExecCtx(ctx,
		"UPDATE orders SET "+idCol+" = NULL, "+nameCol+" = NULL WHERE code = ?", code)
	return err
}

// UpdateStatus sets the order status.
func (r *OrderRepo) UpdateStatus(ctx context.Context, code, status string) error {
	_, err := r.db.ExecCtx(ctx, "UPDATE orders SET status = ? WHERE code = ?", status, code)
	return err
}

// UpdateComplaint sets the complaint status.
func (r *OrderRepo) UpdateComplaint(ctx context.Context, code, complaint string) error {
	_, err := r.db.ExecCtx(ctx, "UPDATE orders SET complaint_status = ? WHERE code = ?", complaint, code)
	return err
}

// RefundWithPayrollCleanup sets the order to "refunded" and removes any
// still-pending payroll items for it, in a single transaction, so a partial
// failure can never leave a payable pending item behind for a refunded order.
func (r *OrderRepo) RefundWithPayrollCleanup(ctx context.Context, code string) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if _, err := session.ExecCtx(ctx, "UPDATE orders SET status = 'refunded' WHERE code = ?", code); err != nil {
			return err
		}
		if _, err := session.ExecCtx(ctx,
			"DELETE FROM payroll_items WHERE order_code = ? AND status = 'pending'", code); err != nil {
			return err
		}
		return nil
	})
}

// SettleWithPayroll sets the order to "settled" and inserts one pending
// payroll item per filled enforcer slot, in a single transaction. When
// expectStatus is non-empty the status update is guarded on it and returns
// ErrNotSettleable if no row matches. INSERT IGNORE keeps item creation
// idempotent across repeated settles.
func (r *OrderRepo) SettleWithPayroll(ctx context.Context, code, expectStatus string) error {
	return r.db.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		q := "UPDATE orders SET status = 'settled' WHERE code = ?"
		args := []any{code}
		if expectStatus != "" {
			q += " AND status = ?"
			args = append(args, expectStatus)
		}
		res, err := session.ExecCtx(ctx, q, args...)
		if err != nil {
			return err
		}
		if expectStatus != "" {
			if n, _ := res.RowsAffected(); n == 0 {
				return ErrNotSettleable
			}
		}
		var o Order
		if err := session.QueryRowCtx(ctx, &o, "SELECT "+orderCols+" FROM orders WHERE code = ?", code); err != nil {
			return err
		}
		insert := func(id int64, name string) error {
			_, err := session.ExecCtx(ctx,
				"INSERT IGNORE INTO payroll_items (order_code, enforcer_id, enforcer_name, status) "+
					"VALUES (?, ?, ?, 'pending')", code, id, name)
			return err
		}
		if o.EnforcerAID.Valid {
			if err := insert(o.EnforcerAID.Int64, o.EnforcerAName.String); err != nil {
				return err
			}
		}
		if o.EnforcerBID.Valid {
			if err := insert(o.EnforcerBID.Int64, o.EnforcerBName.String); err != nil {
				return err
			}
		}
		return nil
	})
}

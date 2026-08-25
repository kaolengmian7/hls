package repo

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// round2 rounds a monetary value to two decimals.
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ComputeDeduction splits a gross salary into deposit repayment then fine
// repayment, flooring the net at zero. owed = outstanding deposit,
// fine = pending fine. Deposit is repaid before fine.
func ComputeDeduction(gross, owed, fine float64) (depositDeduct, fineDeduct, net float64) {
	if gross < 0 {
		gross = 0
	}
	if owed < 0 {
		owed = 0
	}
	if fine < 0 {
		fine = 0
	}
	depositDeduct = math.Min(gross, owed)
	remaining := gross - depositDeduct
	fineDeduct = math.Min(remaining, fine)
	net = remaining - fineDeduct
	return round2(depositDeduct), round2(fineDeduct), round2(net)
}

// ErrPayrollNotFound is returned when a payroll item id does not exist.
var ErrPayrollNotFound = errors.New("payroll item not found")

// ErrAlreadyPaid is returned when paying an item that is not pending.
var ErrAlreadyPaid = errors.New("payroll item already paid")

// ErrOrderRefunded is returned when paying an item whose order has been
// refunded (defense-in-depth alongside the atomic refund cleanup).
var ErrOrderRefunded = errors.New("order refunded")

// PayrollItem mirrors a row in payroll_items (one per enforcer per order).
type PayrollItem struct {
	ID              int64        `db:"id"`
	OrderCode       string       `db:"order_code"`
	EnforcerID      int64        `db:"enforcer_id"`
	EnforcerName    string       `db:"enforcer_name"`
	Status          string       `db:"status"`
	Gross           float64      `db:"gross"`
	DepositDeducted float64      `db:"deposit_deducted"`
	FineDeducted    float64      `db:"fine_deducted"`
	NetPaid         float64      `db:"net_paid"`
	CreatedAt       time.Time    `db:"created_at"`
	PaidAt          sql.NullTime `db:"paid_at"`
	Source          string       `db:"source"`
}

const payrollCols = "id, order_code, enforcer_id, enforcer_name, status, gross, " +
	"deposit_deducted, fine_deducted, net_paid, created_at, paid_at, source"

// PayrollRepo provides database access for payroll items.
type PayrollRepo struct {
	db sqlx.SqlConn
}

// NewPayrollRepo creates a PayrollRepo.
func NewPayrollRepo(db sqlx.SqlConn) *PayrollRepo {
	return &PayrollRepo{db: db}
}

// ListPendingDue returns pending items whose created_at is at least delayDays old.
func (r *PayrollRepo) ListPendingDue(ctx context.Context, delayDays int) ([]*PayrollItem, error) {
	var list []*PayrollItem
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+payrollCols+" FROM payroll_items WHERE status = 'pending' "+
			"AND created_at <= NOW() - INTERVAL ? DAY ORDER BY created_at", delayDays)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListPendingFrozen returns pending items still within the freeze window
// (created less than delayDays ago), newest first.
func (r *PayrollRepo) ListPendingFrozen(ctx context.Context, delayDays int) ([]*PayrollItem, error) {
	var list []*PayrollItem
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+payrollCols+" FROM payroll_items WHERE status = 'pending' "+
			"AND created_at > NOW() - INTERVAL ? DAY ORDER BY created_at DESC", delayDays)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListPaid returns paid items, newest first.
func (r *PayrollRepo) ListPaid(ctx context.Context) ([]*PayrollItem, error) {
	var list []*PayrollItem
	err := r.db.QueryRowsCtx(ctx, &list,
		"SELECT "+payrollCols+" FROM payroll_items WHERE status = 'paid' ORDER BY paid_at DESC")
	if err != nil {
		return nil, err
	}
	return list, nil
}

// DeletePendingByOrder removes unpaid payroll items for an order (used on refund).
func (r *PayrollRepo) DeletePendingByOrder(ctx context.Context, code string) error {
	_, err := r.db.ExecCtx(ctx,
		"DELETE FROM payroll_items WHERE order_code = ? AND status = 'pending'", code)
	return err
}

// Pay freezes the deduction for a pending item and updates the enforcer account,
// all in one transaction. gross is recomputed live from the order (so late
// reviews affecting the bonus are reflected). defaultDeposit is the global
// fallback when the enforcer has no deposit override.
func (r *PayrollRepo) Pay(ctx context.Context, id int64, defaultDeposit float64) (*PayrollItem, error) {
	var out *PayrollItem
	err := r.db.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var item PayrollItem
		if err := session.QueryRowCtx(ctx, &item, "SELECT "+payrollCols+" FROM payroll_items WHERE id = ?", id); err != nil {
			if err == sqlx.ErrNotFound {
				return ErrPayrollNotFound
			}
			return err
		}
		if item.Status != "pending" {
			return ErrAlreadyPaid
		}
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

		var acc EnforcerAccount
		required := defaultDeposit
		err := session.QueryRowCtx(ctx, &acc,
			"SELECT "+enforcerAccountCols+" FROM enforcer_accounts WHERE user_id = ? FOR UPDATE", item.EnforcerID)
		if err != nil && err != sqlx.ErrNotFound {
			return err
		}
		if err == nil && acc.DepositRequired.Valid {
			required = acc.DepositRequired.Float64
		}
		owed := required - acc.DepositPaid
		if owed < 0 {
			owed = 0
		}
		depositDeduct, fineDeduct, net := ComputeDeduction(gross, owed, acc.PendingFine)

		res, err := session.ExecCtx(ctx,
			"UPDATE payroll_items SET status='paid', gross=?, deposit_deducted=?, fine_deducted=?, "+
				"net_paid=?, paid_at=NOW() WHERE id = ? AND status = 'pending'",
			gross, depositDeduct, fineDeduct, net, id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrAlreadyPaid
		}
		if _, err := session.ExecCtx(ctx,
			"INSERT INTO enforcer_accounts (user_id, deposit_paid, pending_fine) VALUES (?, ?, ?) "+
				"ON DUPLICATE KEY UPDATE deposit_paid = deposit_paid + ?, pending_fine = pending_fine - ?",
			item.EnforcerID, depositDeduct, round2(acc.PendingFine-fineDeduct), depositDeduct, fineDeduct); err != nil {
			return err
		}
		item.Status = "paid"
		item.Gross = gross
		item.DepositDeducted = depositDeduct
		item.FineDeducted = fineDeduct
		item.NetPaid = net
		out = &item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

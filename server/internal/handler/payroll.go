package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

type payrollView struct {
	ID              int64   `json:"id"`
	OrderCode       string  `json:"orderCode"`
	OrderTitle      string  `json:"orderTitle"`
	EnforcerName    string  `json:"enforcerName"`
	Status          string  `json:"status"`
	Gross           float64 `json:"gross"`
	DepositDeducted float64 `json:"depositDeducted"`
	FineDeducted    float64 `json:"fineDeducted"`
	NetPaid         float64 `json:"netPaid"`
	CreatedAt       string  `json:"createdAt"`
	PaidAt          *string `json:"paidAt"`
	UnlockAt        *string `json:"unlockAt"`
}

// projectPayrollView builds a pending row's live projection (gross recomputed
// from the order, deductions projected against the account's current state).
func projectPayrollView(item *repo.PayrollItem, order *repo.Order, acc *repo.EnforcerAccount, defaultDeposit float64) payrollView {
	gross := 0.0
	title := ""
	if order != nil {
		gross = order.Earnings()
		title = order.Title
	}
	required := defaultDeposit
	paid := 0.0
	fine := 0.0
	if acc != nil {
		if acc.DepositRequired.Valid {
			required = acc.DepositRequired.Float64
		}
		paid = acc.DepositPaid
		fine = acc.PendingFine
	}
	owed := required - paid
	if owed < 0 {
		owed = 0
	}
	dd, fd, net := repo.ComputeDeduction(gross, owed, fine)
	return payrollView{
		ID: item.ID, OrderCode: item.OrderCode, OrderTitle: title,
		EnforcerName: item.EnforcerName, Status: item.Status,
		Gross: gross, DepositDeducted: dd, FineDeducted: fd, NetPaid: net,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// paidPayrollView builds a paid row from its frozen snapshot.
func paidPayrollView(item *repo.PayrollItem, order *repo.Order) payrollView {
	title := ""
	if order != nil {
		title = order.Title
	}
	v := payrollView{
		ID: item.ID, OrderCode: item.OrderCode, OrderTitle: title,
		EnforcerName: item.EnforcerName, Status: item.Status,
		Gross: item.Gross, DepositDeducted: item.DepositDeducted,
		FineDeducted: item.FineDeducted, NetPaid: item.NetPaid,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if item.PaidAt.Valid {
		s := item.PaidAt.Time.Format("2006-01-02T15:04:05Z07:00")
		v.PaidAt = &s
	}
	return v
}

func ordersByCode(orders []*repo.Order) map[string]*repo.Order {
	m := make(map[string]*repo.Order, len(orders))
	for _, o := range orders {
		m[o.Code] = o
	}
	return m
}

// ListPayrollHandler handles GET /api/payroll?status=pending|paid (owner).
func ListPayrollHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "pending"
		}
		orders, err := ctx.OrderRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		byCode := ordersByCode(orders)
		views := make([]payrollView, 0)
		switch status {
		case "pending":
			items, err := ctx.PayrollRepo.ListPendingDue(r.Context(), ctx.Config.Payroll.SettleDelayDays)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			accounts, err := ctx.EnforcerAccountRepo.ListAccounts(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			for _, it := range items {
				views = append(views, projectPayrollView(it, byCode[it.OrderCode], accounts[it.EnforcerID], ctx.Config.Payroll.DefaultDeposit))
			}
		case "frozen":
			items, err := ctx.PayrollRepo.ListPendingFrozen(r.Context(), ctx.Config.Payroll.SettleDelayDays)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			accounts, err := ctx.EnforcerAccountRepo.ListAccounts(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			for _, it := range items {
				v := projectPayrollView(it, byCode[it.OrderCode], accounts[it.EnforcerID], ctx.Config.Payroll.DefaultDeposit)
				v.Status = "frozen"
				unlock := it.CreatedAt.AddDate(0, 0, ctx.Config.Payroll.SettleDelayDays).Format("2006-01-02T15:04:05Z07:00")
				v.UnlockAt = &unlock
				views = append(views, v)
			}
		case "paid":
			items, err := ctx.PayrollRepo.ListPaid(r.Context())
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			for _, it := range items {
				views = append(views, paidPayrollView(it, byCode[it.OrderCode]))
			}
		default:
			writeError(w, r, http.StatusBadRequest, "invalid_status", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// parsePayrollID extracts the {id} from "/api/payroll/{id}/pay".
func parsePayrollID(path string) int64 {
	rest := strings.TrimPrefix(path, "/api/payroll/")
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	id, _ := strconv.ParseInt(rest, 10, 64)
	return id
}

// PayPayrollHandler handles POST /api/payroll/:id/pay (owner).
func PayPayrollHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		id := parsePayrollID(r.URL.Path)
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		item, err := ctx.PayrollRepo.Pay(r.Context(), id, ctx.Config.Payroll.DefaultDeposit)
		if err != nil {
			switch err {
			case repo.ErrPayrollNotFound:
				writeError(w, r, http.StatusNotFound, "payroll_not_found", nil)
			case repo.ErrAlreadyPaid:
				writeError(w, r, http.StatusConflict, "already_paid", nil)
			case repo.ErrOrderRefunded:
				writeError(w, r, http.StatusConflict, "order_refunded", nil)
			default:
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
			}
			return
		}
		order, _ := ctx.OrderRepo.GetByCode(r.Context(), item.OrderCode)
		httpx.OkJsonCtx(r.Context(), w, paidPayrollView(item, order))
	})
}

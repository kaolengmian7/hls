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
// from the source, deductions projected against the account's current state).
func projectPayrollView(item *repo.PayrollItem, src *payrollSource, acc *repo.EnforcerAccount, defaultDeposit float64) payrollView {
	gross := 0.0
	title := ""
	if src != nil {
		gross = src.Gross
		title = src.Title
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
func paidPayrollView(item *repo.PayrollItem, src *payrollSource) payrollView {
	title := ""
	if src != nil {
		title = src.Title
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

// srcPtr looks up a merged payroll source by code, returning nil when absent.
func srcPtr(m map[string]payrollSource, code string) *payrollSource {
	if s, ok := m[code]; ok {
		return &s
	}
	return nil
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
		teach, err := ctx.TeachingRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		sources := buildPayrollSources(orders, teach)
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
				views = append(views, projectPayrollView(it, srcPtr(sources, it.OrderCode), accounts[it.EnforcerID], ctx.Config.Payroll.DefaultDeposit))
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
				v := projectPayrollView(it, srcPtr(sources, it.OrderCode), accounts[it.EnforcerID], ctx.Config.Payroll.DefaultDeposit)
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
				views = append(views, paidPayrollView(it, srcPtr(sources, it.OrderCode)))
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
	})
}

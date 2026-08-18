package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

type enforcerStatView struct {
	UserID          int64    `json:"userId"`
	Nickname        string   `json:"nickname"`
	Username        string   `json:"username"`
	DepositRequired float64  `json:"depositRequired"`
	DepositPaid     float64  `json:"depositPaid"`
	DepositOwed     float64  `json:"depositOwed"`
	OrderCount      int      `json:"orderCount"`
	GoodCount       int      `json:"goodCount"`
	BadCount        int      `json:"badCount"`
	ReviewedCount   int      `json:"reviewedCount"`
	GoodRate        *float64 `json:"goodRate"`
	PendingFine     float64  `json:"pendingFine"`
}

func round2f(v float64) float64 { return math.Round(v*100) / 100 }

// computeEnforcerStats aggregates per-enforcer order/review stats and account
// fields. Only users with role "enforcer" are included. goodRate is nil when
// the enforcer has no reviewed orders.
func computeEnforcerStats(users []*repo.UserSummary, orders []*repo.Order,
	reviews map[string]*repo.ReviewSummary, accounts map[int64]*repo.EnforcerAccount,
	defaultDeposit float64) []enforcerStatView {
	out := make([]enforcerStatView, 0)
	for _, u := range users {
		if u.Role != "enforcer" {
			continue
		}
		v := enforcerStatView{UserID: u.ID, Nickname: u.Nickname, Username: u.Username}
		for _, o := range orders {
			mine := (o.EnforcerAID.Valid && o.EnforcerAID.Int64 == u.ID) ||
				(o.EnforcerBID.Valid && o.EnforcerBID.Int64 == u.ID)
			if !mine {
				continue
			}
			v.OrderCount++
			if s, ok := reviews[o.Code]; ok {
				v.ReviewedCount++
				total := s.Attitude + s.Technical
				if total >= 8 {
					v.GoodCount++
				} else if total < 6 {
					v.BadCount++
				}
			}
		}
		if v.ReviewedCount > 0 {
			rate := round2f(float64(v.GoodCount) / float64(v.ReviewedCount))
			v.GoodRate = &rate
		}
		required := defaultDeposit
		if acc, ok := accounts[u.ID]; ok {
			if acc.DepositRequired.Valid {
				required = acc.DepositRequired.Float64
			}
			v.DepositPaid = acc.DepositPaid
			v.PendingFine = acc.PendingFine
		}
		v.DepositRequired = required
		owed := required - v.DepositPaid
		if owed < 0 {
			owed = 0
		}
		v.DepositOwed = round2f(owed)
		out = append(out, v)
	}
	return out
}

// parseEnforcerPath splits "/api/enforcers/{id}/{action}".
func parseEnforcerPath(path string) (int64, string) {
	rest := strings.TrimPrefix(path, "/api/enforcers/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, ""
	}
	id, _ := strconv.ParseInt(parts[0], 10, 64)
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action
}

// ListEnforcersHandler handles GET /api/enforcers (owner).
func ListEnforcersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.UserRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		orders, err := ctx.OrderRepo.ListAll(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		reviews, err := ctx.ReviewRepo.ListSummaries(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		accounts, err := ctx.EnforcerAccountRepo.ListAccounts(r.Context())
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := computeEnforcerStats(users, orders, reviews, accounts, ctx.Config.Payroll.DefaultDeposit)
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

type depositRequest struct {
	DepositRequired *float64 `json:"depositRequired"`
}

// UpdateEnforcerDepositHandler handles PUT /api/enforcers/:id/deposit (owner).
func UpdateEnforcerDepositHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		id, _ := parseEnforcerPath(r.URL.Path)
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		var req depositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		if req.DepositRequired != nil && *req.DepositRequired < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_deposit", nil)
			return
		}
		if err := ctx.EnforcerAccountRepo.UpsertDepositRequired(r.Context(), id, req.DepositRequired); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

type fineRequest struct {
	PendingFine float64 `json:"pendingFine"`
}

// UpdateEnforcerFineHandler handles PUT /api/enforcers/:id/fine (owner).
func UpdateEnforcerFineHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireOwner(ctx, func(w http.ResponseWriter, r *http.Request) {
		id, _ := parseEnforcerPath(r.URL.Path)
		if id == 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_id", nil)
			return
		}
		var req fineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PendingFine < 0 {
			writeError(w, r, http.StatusBadRequest, "invalid_fine", nil)
			return
		}
		if err := ctx.EnforcerAccountRepo.SetPendingFine(r.Context(), id, req.PendingFine); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/repo"
	"hls.dev/server/internal/svc"
)

var validOrderTypes = map[string]bool{
	"guarantee": true, "fun": true, "companion": true, "teaching": true,
}
var validOrderStatuses = map[string]bool{
	"dispatching": true, "ongoing": true, "settled": true, "refunded": true,
}
var validComplaintStatuses = map[string]bool{
	"none": true, "complained": true, "resolved": true,
}

// genOrderCode returns a timestamp-based unique order code.
func genOrderCode() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// parseOrderPath splits "/api/orders/{code}/a/b" into ("{code}", ["a","b"]).
func parseOrderPath(path string) (string, []string) {
	rest := strings.TrimPrefix(path, "/api/orders/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil
	}
	return parts[0], parts[1:]
}

// ParseOrderPathForTest exposes parseOrderPath to the external test package.
func ParseOrderPathForTest(path string) (string, []string) { return parseOrderPath(path) }

// requireAnyRole rejects users whose role is not in roles.
func requireAnyRole(ctx *svc.ServiceContext, roles []string, next http.HandlerFunc) http.HandlerFunc {
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
		for _, role := range roles {
			if user.Role == role {
				next(w, r)
				return
			}
		}
		writeError(w, r, http.StatusForbidden, "forbidden", nil)
	}
}

func nullEnforcer(id sql.NullInt64, name sql.NullString) any {
	if !id.Valid {
		return nil
	}
	return map[string]any{"id": id.Int64, "name": name.String}
}

// toOrderView builds the canonical JSON view of an order.
func toOrderView(o *repo.Order) map[string]any {
	return map[string]any{
		"code":            o.Code,
		"title":           o.Title,
		"orderType":       o.OrderType,
		"price":           o.Price,
		"commissionRate":  o.CommissionRate,
		"bonusRate":       o.BonusRate,
		"playerCount":     o.PlayerCount,
		"dispatcherName":  o.DispatcherName,
		"enforcerA":       nullEnforcer(o.EnforcerAID, o.EnforcerAName),
		"enforcerB":       nullEnforcer(o.EnforcerBID, o.EnforcerBName),
		"remark":          o.Remark.String,
		"status":          o.Status,
		"complaintStatus": o.ComplaintStatus,
		"bonusQualified":  o.BonusQualified,
		"earnings":        o.Earnings(),
		"createdAt":       o.CreatedAt,
	}
}

type createOrderRequest struct {
	Title          string  `json:"title"`
	OrderType      string  `json:"orderType"`
	Price          float64 `json:"price"`
	CommissionRate float64 `json:"commissionRate"`
	BonusRate      float64 `json:"bonusRate"`
	PlayerCount    int     `json:"playerCount"`
	Remark         string  `json:"remark"`
}

// CreateOrderHandler handles POST /api/orders.
func CreateOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_body", err)
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		if req.Title == "" || !validOrderTypes[req.OrderType] || req.Price <= 0 ||
			(req.PlayerCount != 1 && req.PlayerCount != 2) {
			writeError(w, r, http.StatusBadRequest, "invalid_order", nil)
			return
		}
		o := &repo.Order{
			Title:          req.Title,
			OrderType:      req.OrderType,
			Price:          req.Price,
			CommissionRate: req.CommissionRate,
			BonusRate:      req.BonusRate,
			PlayerCount:    req.PlayerCount,
			DispatcherID:   userID,
			DispatcherName: user.Nickname,
			Remark:         sql.NullString{String: strings.TrimSpace(req.Remark), Valid: req.Remark != ""},
		}
		// retry a few times on code collision
		var created *repo.Order
		for i := 0; i < 5; i++ {
			o.Code = genOrderCode()
			created, err = ctx.OrderRepo.Create(r.Context(), o)
			if err == nil {
				break
			}
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(created))
	})
}

// ListOrdersHandler handles GET /api/orders.
func ListOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
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
}

// MyOrdersHandler handles GET /api/my-orders.
func MyOrdersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		orders, err := ctx.OrderRepo.ListByEnforcer(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		views := make([]map[string]any, 0, len(orders))
		for _, o := range orders {
			views = append(views, toOrderView(o))
		}
		httpx.OkJsonCtx(r.Context(), w, views)
	})
}

// OrderDetailHandler handles GET /api/orders/:code (staff+enforcer only).
func OrderDetailHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service", "enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

// ClaimOrderHandler handles POST /api/orders/:code/claim (enforcer).
// Fills the first open slot (A then B); rejects if the order is not dispatching,
// is already full, or the user already occupies a slot. When all required slots
// are filled, transitions the order to "ongoing".
func ClaimOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "user_not_found", nil)
			return
		}
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if o.Status != "dispatching" {
			writeError(w, r, http.StatusConflict, "order_not_claimable", nil)
			return
		}
		if (o.EnforcerAID.Valid && o.EnforcerAID.Int64 == userID) ||
			(o.EnforcerBID.Valid && o.EnforcerBID.Int64 == userID) {
			writeError(w, r, http.StatusConflict, "already_claimed", nil)
			return
		}
		// choose first open slot within required player_count
		slot := ""
		if !o.EnforcerAID.Valid {
			slot = "a"
		} else if o.PlayerCount == 2 && !o.EnforcerBID.Valid {
			slot = "b"
		}
		if slot == "" {
			writeError(w, r, http.StatusConflict, "order_full", nil)
			return
		}
		if err := ctx.OrderRepo.AssignEnforcer(r.Context(), code, slot, userID, user.Nickname); err != nil {
			if err == repo.ErrSlotTaken {
				writeError(w, r, http.StatusConflict, "slot_taken", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		// re-read to decide status transition
		o, err = ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if isFullyStaffed(o) {
			if err := ctx.OrderRepo.UpdateStatus(r.Context(), code, "ongoing"); err != nil {
				writeError(w, r, http.StatusInternalServerError, "db_error", err)
				return
			}
			o.Status = "ongoing"
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

func isFullyStaffed(o *repo.Order) bool {
	if o.PlayerCount == 1 {
		return o.EnforcerAID.Valid
	}
	return o.EnforcerAID.Valid && o.EnforcerBID.Valid
}

// RemoveEnforcerHandler handles DELETE /api/orders/:code/enforcer/:slot (owner/service).
// Clears the slot; if the order is no longer fully staffed, reverts to "dispatching".
func RemoveEnforcerHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, rest := parseOrderPath(r.URL.Path)
		if len(rest) != 2 || rest[0] != "enforcer" || (rest[1] != "a" && rest[1] != "b") {
			writeError(w, r, http.StatusBadRequest, "invalid_slot", nil)
			return
		}
		if err := ctx.OrderRepo.RemoveEnforcer(r.Context(), code, rest[1]); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		if !isFullyStaffed(o) && (o.Status == "ongoing") {
			_ = ctx.OrderRepo.UpdateStatus(r.Context(), code, "dispatching")
			o.Status = "dispatching"
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

type statusRequest struct {
	Status string `json:"status"`
}

// UpdateOrderStatusHandler handles PUT /api/orders/:code/status (owner/service).
func UpdateOrderStatusHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		var req statusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validOrderStatuses[req.Status] {
			writeError(w, r, http.StatusBadRequest, "invalid_status", nil)
			return
		}
		var err error
		switch req.Status {
		case "settled":
			err = ctx.OrderRepo.SettleWithPayroll(r.Context(), code, "")
		case "refunded":
			if err = ctx.OrderRepo.UpdateStatus(r.Context(), code, "refunded"); err == nil {
				err = ctx.PayrollRepo.DeletePendingByOrder(r.Context(), code)
			}
		default:
			err = ctx.OrderRepo.UpdateStatus(r.Context(), code, req.Status)
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

type complaintRequest struct {
	ComplaintStatus string `json:"complaintStatus"`
}

// UpdateOrderComplaintHandler handles PUT /api/orders/:code/complaint (owner/service).
func UpdateOrderComplaintHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		var req complaintRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !validComplaintStatuses[req.ComplaintStatus] {
			writeError(w, r, http.StatusBadRequest, "invalid_complaint", nil)
			return
		}
		if err := ctx.OrderRepo.UpdateComplaint(r.Context(), code, req.ComplaintStatus); err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	})
}

// SettleOrderHandler handles POST /api/orders/:code/settle (enforcer).
// The caller must occupy a slot on the order and the order must be "ongoing".
func SettleOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		userID, _ := extractUserID(r, ctx)
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				writeError(w, r, http.StatusNotFound, "order_not_found", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		isMine := (o.EnforcerAID.Valid && o.EnforcerAID.Int64 == userID) ||
			(o.EnforcerBID.Valid && o.EnforcerBID.Int64 == userID)
		if !isMine {
			writeError(w, r, http.StatusForbidden, "not_your_order", nil)
			return
		}
		if err := ctx.OrderRepo.SettleWithPayroll(r.Context(), code, "ongoing"); err != nil {
			if errors.Is(err, repo.ErrNotSettleable) {
				writeError(w, r, http.StatusConflict, "order_not_settleable", nil)
				return
			}
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		o, err = ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "db_error", err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

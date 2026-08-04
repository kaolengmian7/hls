package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		views := make([]map[string]any, 0, len(orders))
		for _, o := range orders {
			views = append(views, toOrderView(o))
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

// OrderDetailHandler handles GET /api/orders/:code (any logged-in role).
func OrderDetailHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"owner", "service", "enforcer", "customer"}, func(w http.ResponseWriter, r *http.Request) {
		code, _ := parseOrderPath(r.URL.Path)
		o, err := ctx.OrderRepo.GetByCode(r.Context(), code)
		if err != nil {
			writeError(w, r, http.StatusNotFound, "order_not_found", nil)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, toOrderView(o))
	})
}

// ClaimOrderHandler is implemented in Task 7; placeholder keeps the package compiling.
func ClaimOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotImplemented, "not_implemented", nil)
	})
}

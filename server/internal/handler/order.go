package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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

// ClaimOrderHandler is implemented in Task 7; placeholder keeps the package compiling.
func ClaimOrderHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return requireAnyRole(ctx, []string{"enforcer"}, func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotImplemented, "not_implemented", nil)
	})
}

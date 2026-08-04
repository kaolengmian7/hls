package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestParseOrderPath(t *testing.T) {
	code, rest := handler.ParseOrderPathForTest("/api/orders/1699999999/enforcer/a")
	if code != "1699999999" {
		t.Fatalf("code = %q, want 1699999999", code)
	}
	if len(rest) != 2 || rest[0] != "enforcer" || rest[1] != "a" {
		t.Fatalf("rest = %v, want [enforcer a]", rest)
	}
}

func TestClaimHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/orders/123/claim", nil)
	w := httptest.NewRecorder()
	handler.ClaimOrderHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

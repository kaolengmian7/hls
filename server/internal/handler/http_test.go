package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestLotteryPoolHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/lottery/pool", nil)
	w := httptest.NewRecorder()

	handler.LotteryPoolHandler()(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
	var body []map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty pool")
	}
	first := body[0]
	for _, field := range []string{"id", "name", "rarity", "value"} {
		if _, ok := first[field]; !ok {
			t.Errorf("missing field %q in first item", field)
		}
	}
	if _, ok := first["weight"]; ok {
		t.Error("weight must not be exposed to client")
	}
}

func TestLotteryPoolHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/lottery/pool", nil)
	w := httptest.NewRecorder()

	handler.LotteryPoolHandler()(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want 405", w.Code)
	}
}

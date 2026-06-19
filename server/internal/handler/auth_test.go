package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestQrStatusExpired(t *testing.T) {
	// QrStatusHandler with a nil ServiceContext returns 400 for empty state.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/wechat/qr/status?state=", nil)
	w := httptest.NewRecorder()

	handler.QrStatusHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["ok"] != false {
		t.Fatalf("body[ok] = %v, want false", body["ok"])
	}
}

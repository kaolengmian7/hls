package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestLoginHandler_EmptyBody_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()

	handler.LoginHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestLoginHandler_MissingFields_ReturnsBadRequest(t *testing.T) {
	body := `{"username":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.LoginHandler(nil)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestQrStatusHandler_EmptyState_ReturnsBadRequest(t *testing.T) {
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

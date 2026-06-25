package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
	"hls.dev/server/internal/svc"
)

func minSmsCtx() *svc.ServiceContext {
	return &svc.ServiceContext{
		SmsStore: svc.NewSmsStore(),
	}
}

func TestSmsSendHandler_MissingBody_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/send", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsSendHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSmsSendHandler_InvalidPhone_ReturnsBadRequest(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"phone": "12345", "captcha": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsSendHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "invalid_phone" {
		t.Fatalf("error = %v, want \"invalid_phone\"", resp["error"])
	}
}

func TestSmsSendHandler_MissingCaptcha_ReturnsBadRequest(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"phone": "13800138000"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsSendHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSmsVerifyHandler_MissingFields_ReturnsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sms/verify", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.SmsVerifyHandler(minSmsCtx())(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

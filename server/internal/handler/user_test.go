package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestUpdateNicknameHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/user/nickname", bytes.NewBufferString(`{"nickname":"new"}`))
	w := httptest.NewRecorder()
	handler.UpdateNicknameHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUpdatePasswordHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/user/password",
		bytes.NewBufferString(`{"old_password":"a","new_password":"b"}`))
	w := httptest.NewRecorder()
	handler.UpdatePasswordHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

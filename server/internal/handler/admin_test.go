package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hls.dev/server/internal/handler"
)

func TestListUsersHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()
	handler.ListUsersHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestCreateUserHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", nil)
	w := httptest.NewRecorder()
	handler.CreateUserHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestResetPasswordHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/password", nil)
	w := httptest.NewRecorder()
	handler.ResetPasswordHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestResetNicknameHandler_NoAuth_ReturnsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/1/nickname", nil)
	w := httptest.NewRecorder()
	handler.ResetNicknameHandler(nil)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

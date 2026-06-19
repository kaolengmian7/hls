package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hls.dev/server/internal/auth"
)

func newTestWechatClient(srv *httptest.Server) *auth.WechatClient {
	c := auth.NewWechatClient("test-appid", "test-secret")
	c.SetBaseURL(srv.URL)
	return c
}

func TestExchangeCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "access_token") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok123",
			"openid":       "oid123",
			"unionid":      "uid123",
		})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	resp, err := c.ExchangeCode("code123")
	if err != nil {
		t.Fatalf("ExchangeCode() error = %v", err)
	}
	if resp.AccessToken != "tok123" || resp.OpenID != "oid123" || resp.UnionID != "uid123" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestExchangeCodeWechatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"errcode": 40029, "errmsg": "invalid code"})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	_, err := c.ExchangeCode("bad_code")
	if err == nil {
		t.Fatal("ExchangeCode() with WeChat error should fail")
	}
}

func TestFetchUserInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "userinfo") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"openid":     "oid123",
			"nickname":   "花栗鼠",
			"headimgurl": "https://example.com/avatar.jpg",
			"unionid":    "uid123",
		})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	info, err := c.FetchUserInfo("tok123", "oid123")
	if err != nil {
		t.Fatalf("FetchUserInfo() error = %v", err)
	}
	if info.Nickname != "花栗鼠" {
		t.Fatalf("Nickname = %q, want %q", info.Nickname, "花栗鼠")
	}
}

func TestFetchUserInfoWechatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"errcode": 40001, "errmsg": "invalid access_token"})
	}))
	defer srv.Close()

	c := newTestWechatClient(srv)
	_, err := c.FetchUserInfo("bad_tok", "oid")
	if err == nil {
		t.Fatal("FetchUserInfo() with WeChat error should fail")
	}
}

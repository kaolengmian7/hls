package svc_test

import (
	"testing"
	"time"

	"hls.dev/server/internal/svc"
)

func TestSmsStore_CheckAndSend_Success(t *testing.T) {
	s := svc.NewSmsStore()
	code, retryAfter, errKey := s.CheckAndSend("13800138000", "1.2.3.4", 5, 10, 200)
	if errKey != "" {
		t.Fatalf("errKey = %q, want \"\"", errKey)
	}
	if retryAfter != 0 {
		t.Fatalf("retryAfter = %d, want 0", retryAfter)
	}
	if len(code) != 6 {
		t.Fatalf("code len = %d, want 6", len(code))
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("code %q contains non-digit", code)
		}
	}
}

func TestSmsStore_CheckAndSend_CooldownImmediately(t *testing.T) {
	s := svc.NewSmsStore()
	s.CheckAndSend("13800138001", "1.2.3.4", 5, 10, 200)
	_, retryAfter, errKey := s.CheckAndSend("13800138001", "1.2.3.4", 5, 10, 200)
	if errKey != "cooldown" {
		t.Fatalf("errKey = %q, want \"cooldown\"", errKey)
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %d, want > 0", retryAfter)
	}
}

func TestSmsStore_CheckAndSend_PhoneLimit(t *testing.T) {
	// Use clock injection: advance 61s between sends to bypass cooldown.
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := svc.NewSmsStoreWithClock(func() time.Time { return now })

	// maxPhone=2: first two sends succeed.
	for i := 0; i < 2; i++ {
		_, _, errKey := s.CheckAndSend("13800138002", "1.2.3.4", 2, 10, 200)
		if errKey != "" {
			t.Fatalf("send %d: errKey = %q, want \"\"", i+1, errKey)
		}
		now = now.Add(61 * time.Second)
	}
	_, _, errKey := s.CheckAndSend("13800138002", "1.2.3.4", 2, 10, 200)
	if errKey != "phone_limit" {
		t.Fatalf("errKey = %q, want \"phone_limit\"", errKey)
	}
}

func TestSmsStore_CheckAndSend_IPLimit(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := svc.NewSmsStoreWithClock(func() time.Time { return now })

	// Different phones, same IP, maxIP=2.
	phones := []string{"13800138010", "13800138011", "13800138012"}
	for i, ph := range phones[:2] {
		_, _, errKey := s.CheckAndSend(ph, "9.9.9.9", 5, 2, 200)
		if errKey != "" {
			t.Fatalf("send %d: errKey = %q, want \"\"", i+1, errKey)
		}
		now = now.Add(61 * time.Second)
	}
	_, _, errKey := s.CheckAndSend(phones[2], "9.9.9.9", 5, 2, 200)
	if errKey != "ip_limit" {
		t.Fatalf("errKey = %q, want \"ip_limit\"", errKey)
	}
}

func TestSmsStore_CheckAndSend_GlobalLimit(t *testing.T) {
	s := svc.NewSmsStore()
	// maxGlobal=1: second request (different phone) is blocked.
	s.CheckAndSend("13800138020", "1.2.3.4", 5, 10, 1)
	_, _, errKey := s.CheckAndSend("13800138021", "1.2.3.5", 5, 10, 1)
	if errKey != "global_limit" {
		t.Fatalf("errKey = %q, want \"global_limit\"", errKey)
	}
}

func TestSmsStore_Verify_Success(t *testing.T) {
	s := svc.NewSmsStore()
	code, _, _ := s.CheckAndSend("13800138030", "1.2.3.4", 5, 10, 200)
	errKey := s.Verify("13800138030", code)
	if errKey != "" {
		t.Fatalf("errKey = %q, want \"\"", errKey)
	}
}

func TestSmsStore_Verify_WrongCode(t *testing.T) {
	s := svc.NewSmsStore()
	s.CheckAndSend("13800138031", "1.2.3.4", 5, 10, 200)
	errKey := s.Verify("13800138031", "XXXXXX")
	if errKey != "invalid_code" {
		t.Fatalf("errKey = %q, want \"invalid_code\"", errKey)
	}
}

func TestSmsStore_Verify_MaxFailsInvalidates(t *testing.T) {
	s := svc.NewSmsStore()
	s.CheckAndSend("13800138032", "1.2.3.4", 5, 10, 200)
	for i := 0; i < 5; i++ {
		s.Verify("13800138032", "XXXXXX")
	}
	// After 5 failures, code is deleted — subsequent verify returns invalid_code too.
	errKey := s.Verify("13800138032", "XXXXXX")
	if errKey != "invalid_code" {
		t.Fatalf("errKey = %q, want \"invalid_code\"", errKey)
	}
}

func TestSmsStore_Verify_Expired(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	s := svc.NewSmsStoreWithClock(func() time.Time { return now })
	code, _, _ := s.CheckAndSend("13800138033", "1.2.3.4", 5, 10, 200)
	now = now.Add(11 * time.Minute) // past 10-minute TTL
	errKey := s.Verify("13800138033", code)
	if errKey != "code_expired" {
		t.Fatalf("errKey = %q, want \"code_expired\"", errKey)
	}
}

func TestSmsStore_Verify_DeletedAfterSuccess(t *testing.T) {
	s := svc.NewSmsStore()
	code, _, _ := s.CheckAndSend("13800138034", "1.2.3.4", 5, 10, 200)
	s.Verify("13800138034", code)
	// Second verify with same code must fail.
	errKey := s.Verify("13800138034", code)
	if errKey != "invalid_code" {
		t.Fatalf("errKey = %q, want \"invalid_code\"", errKey)
	}
}

func TestSmsStore_CheckAndSend_CooldownAfterVerify(t *testing.T) {
	s := svc.NewSmsStore()
	code, _, _ := s.CheckAndSend("13800138040", "1.2.3.4", 5, 10, 200)
	s.Verify("13800138040", code) // successful verify clears the code
	// New send within 60s should still be blocked by cooldown
	_, _, errKey := s.CheckAndSend("13800138040", "1.2.3.4", 5, 10, 200)
	if errKey != "cooldown" {
		t.Fatalf("errKey = %q, want \"cooldown\"", errKey)
	}
}

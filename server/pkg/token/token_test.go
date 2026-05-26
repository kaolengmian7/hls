package token

import (
	"testing"
	"time"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	s := NewSigner("secret-key", time.Hour)
	tok := s.Sign("123456", "p_abc")
	if tok == "" {
		t.Fatal("empty token")
	}
	if _, err := s.Verify("123456", "p_abc", tok); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyMismatchRoom(t *testing.T) {
	s := NewSigner("secret-key", time.Hour)
	tok := s.Sign("123456", "p_abc")
	if _, err := s.Verify("654321", "p_abc", tok); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestVerifyMismatchPlayer(t *testing.T) {
	s := NewSigner("secret-key", time.Hour)
	tok := s.Sign("123456", "p_abc")
	if _, err := s.Verify("123456", "p_other", tok); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestVerifyExpired(t *testing.T) {
	s := NewSigner("secret-key", time.Hour)
	// 直接构造一个过去 1 小时的 token：使用 signWithExp 内部方法避免 ttl 默认值兜底。
	pastExp := time.Now().Add(-time.Hour).Unix()
	tok := s.signWithExp("123456", "p_abc", pastExp)
	if _, err := s.Verify("123456", "p_abc", tok); err == nil {
		t.Fatal("expected expired error")
	}
}

func TestVerifyMalformed(t *testing.T) {
	s := NewSigner("secret-key", time.Hour)
	if _, err := s.Verify("123456", "p_abc", "garbage"); err == nil {
		t.Fatal("expected format error")
	}
}

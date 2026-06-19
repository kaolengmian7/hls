package auth_test

import (
	"testing"

	"hls.dev/server/internal/auth"
)

func TestJWTSignAndVerify(t *testing.T) {
	s := auth.NewJWTSigner("test-secret", 7)
	tok, err := s.Sign(42)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	id, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if id != 42 {
		t.Fatalf("userID = %d, want 42", id)
	}
}

func TestJWTVerifyInvalidToken(t *testing.T) {
	s := auth.NewJWTSigner("secret", 7)
	_, err := s.Verify("not.a.valid.token")
	if err == nil {
		t.Fatal("Verify() with invalid token = nil, want error")
	}
}

func TestJWTVerifyWrongSecret(t *testing.T) {
	s1 := auth.NewJWTSigner("secret-a", 7)
	s2 := auth.NewJWTSigner("secret-b", 7)
	tok, _ := s1.Sign(1)
	_, err := s2.Verify(tok)
	if err == nil {
		t.Fatal("Verify() with wrong secret = nil, want error")
	}
}

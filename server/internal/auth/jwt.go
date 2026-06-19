// Package auth provides JWT signing/verification for user sessions.
package auth

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"hls.dev/server/pkg/errors"
)

// JWTSigner signs and verifies user session JWTs.
type JWTSigner struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTSigner creates a JWTSigner. ttlDays <= 0 defaults to 7.
func NewJWTSigner(secret string, ttlDays int) *JWTSigner {
	if ttlDays <= 0 {
		ttlDays = 7
	}
	return &JWTSigner{
		secret: []byte(secret),
		ttl:    time.Duration(ttlDays) * 24 * time.Hour,
	}
}

// Sign issues a JWT with sub=userID, valid for ttl.
func (s *JWTSigner) Sign(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", errors.WithMessage(err, "jwt sign")
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning the userID from sub.
func (s *JWTSigner) Verify(tokenStr string) (int64, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.CodeInvalidSession, "unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return 0, errors.New(errors.CodeInvalidSession, "invalid or expired token")
	}
	claims, ok := tok.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return 0, errors.New(errors.CodeInvalidSession, "invalid claims")
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errors.New(errors.CodeInvalidSession, "invalid subject")
	}
	return id, nil
}

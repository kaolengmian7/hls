// Package token 负责签发/校验玩家断线重连时使用的 sessionToken。
//
// 设计：HMAC-SHA256 over "<roomCode>|<playerId>|<exp>"，
// 输出形如 "<exp_hex>.<sig_base64url>"，
// 与房间生命周期一致；超过 TTL 视为失效。
package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"hls.dev/server/pkg/errors"
)

// Signer 持有签名密钥与默认 TTL。
type Signer struct {
	secret []byte
	ttl    time.Duration
}

// NewSigner 创建 Signer。secret 不能为空；ttl<=0 时使用默认 4h。
func NewSigner(secret string, ttl time.Duration) *Signer {
	if ttl <= 0 {
		ttl = 4 * time.Hour
	}
	return &Signer{secret: []byte(secret), ttl: ttl}
}

// Sign 为指定 (roomCode, playerId) 签发 token。
func (s *Signer) Sign(roomCode, playerID string) string {
	exp := time.Now().Add(s.ttl).Unix()
	return s.signWithExp(roomCode, playerID, exp)
}

// Verify 校验 token 是否仍然有效，并返回过期时间戳。
func (s *Signer) Verify(roomCode, playerID, token string) (time.Time, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return time.Time{}, errors.New(errors.CodeInvalidSession, "token format")
	}
	exp, err := strconv.ParseInt(parts[0], 16, 64)
	if err != nil {
		return time.Time{}, errors.New(errors.CodeInvalidSession, "token exp parse")
	}
	expect := s.signWithExp(roomCode, playerID, exp)
	if !hmac.Equal([]byte(expect), []byte(token)) {
		return time.Time{}, errors.New(errors.CodeInvalidSession, "token signature mismatch")
	}
	expireAt := time.Unix(exp, 0)
	if time.Now().After(expireAt) {
		return expireAt, errors.New(errors.CodeInvalidSession, "token expired")
	}
	return expireAt, nil
}

func (s *Signer) signWithExp(roomCode, playerID string, exp int64) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(roomCode))
	mac.Write([]byte("|"))
	mac.Write([]byte(playerID))
	mac.Write([]byte("|"))
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return strconv.FormatInt(exp, 16) + "." + sig
}

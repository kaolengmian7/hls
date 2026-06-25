package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/svc"
)

var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

const (
	smsMaxPhone  = 5
	smsMaxIP     = 10
	smsMaxGlobal = 200
)

type smsSendRequest struct {
	Phone   string `json:"phone"`
	Captcha string `json:"captcha"`
}

type smsVerifyRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// SmsSendHandler handles POST /api/auth/sms/send.
func SmsSendHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req smsSendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeSmsError(w, http.StatusBadRequest, "invalid_body", 0)
			return
		}
		req.Phone = strings.TrimSpace(req.Phone)
		if !phoneRe.MatchString(req.Phone) {
			writeSmsError(w, http.StatusBadRequest, "invalid_phone", 0)
			return
		}
		// Frontend-only captcha: we only verify the field is non-empty.
		// Server-side rate limits (CheckAndSend) are the real abuse protection.
		if strings.TrimSpace(req.Captcha) == "" {
			writeSmsError(w, http.StatusBadRequest, "missing_captcha", 0)
			return
		}

		ip := clientIP(r)
		code, retryAfter, errKey := ctx.SmsStore.CheckAndSend(req.Phone, ip, smsMaxPhone, smsMaxIP, smsMaxGlobal)
		if errKey != "" {
			switch errKey {
			case "global_limit":
				writeSmsError(w, http.StatusServiceUnavailable, errKey, retryAfter)
			case "internal":
				logx.WithContext(r.Context()).Error("sms store crypto/rand failure")
				writeSmsError(w, http.StatusInternalServerError, "server_error", 0)
			default:
				writeSmsError(w, http.StatusTooManyRequests, errKey, retryAfter)
			}
			return
		}

		if err := ctx.SmsSender.Send(r.Context(), req.Phone, code); err != nil {
			logx.WithContext(r.Context()).Errorw("sms send failed", logx.Field("err", err))
			writeSmsError(w, http.StatusInternalServerError, "send_failed", 0)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, map[string]any{"ok": true})
	}
}

// SmsVerifyHandler handles POST /api/auth/sms/verify.
func SmsVerifyHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req smsVerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeSmsError(w, http.StatusBadRequest, "invalid_body", 0)
			return
		}
		req.Phone = strings.TrimSpace(req.Phone)
		req.Code = strings.TrimSpace(req.Code)
		if !phoneRe.MatchString(req.Phone) {
			writeSmsError(w, http.StatusBadRequest, "invalid_phone", 0)
			return
		}
		if req.Code == "" {
			writeSmsError(w, http.StatusBadRequest, "missing_code", 0)
			return
		}
		if len(req.Code) != 6 {
			writeSmsError(w, http.StatusBadRequest, "invalid_code_format", 0)
			return
		}

		if errKey := ctx.SmsStore.Verify(req.Phone, req.Code); errKey != "" {
			writeSmsError(w, http.StatusBadRequest, errKey, 0)
			return
		}

		user, err := ctx.UserRepo.UpsertByPhone(r.Context(), req.Phone)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("upsert user by phone", logx.Field("err", err))
			writeSmsError(w, http.StatusInternalServerError, "server_error", 0)
			return
		}

		token, err := ctx.JWTSigner.Sign(user.ID)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("sign jwt", logx.Field("err", err))
			writeSmsError(w, http.StatusInternalServerError, "server_error", 0)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, map[string]any{"token": token})
	}
}

// clientIP returns the TCP peer address for rate-limiting purposes.
// Use only the TCP peer address to prevent X-Forwarded-For spoofing.
// If this server runs behind a trusted reverse proxy, replace with
// proxy-aware logic that validates the peer before trusting XFF headers.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeSmsError(w http.ResponseWriter, status int, errKey string, retryAfter int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}
	w.WriteHeader(status)
	resp := map[string]any{"error": errKey}
	if retryAfter > 0 {
		resp["retry_after"] = retryAfter
	}
	_ = json.NewEncoder(w).Encode(resp)
}

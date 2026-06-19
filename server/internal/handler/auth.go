package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"hls.dev/server/internal/svc"
)

type QrStartResponse struct {
	State     string `json:"state"`
	IframeURL string `json:"iframe_url"`
}

type QrStatusResponse struct {
	Status string `json:"status"` // "pending" | "done" | "expired"
	Token  string `json:"token,omitempty"`
}

type MeResponse struct {
	ID        int64  `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// QrStartHandler handles GET /api/auth/wechat/qr/start.
func QrStartHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := uuid.New().String()
		if err := ctx.AuthStateRepo.Create(r.Context(), state); err != nil {
			writeError(w, r, http.StatusInternalServerError, "create state", err)
			return
		}
		callbackURL := fmt.Sprintf("https://%s/api/auth/wechat/qr/callback",
			ctx.Config.WechatOpen.CallbackDomain)
		iframeURL := fmt.Sprintf(
			"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s",
			ctx.Config.WechatOpen.AppID,
			url.QueryEscape(callbackURL),
			state,
		)
		httpx.OkJsonCtx(r.Context(), w, QrStartResponse{State: state, IframeURL: iframeURL})
	}
}

// QrCallbackHandler handles GET /api/auth/wechat/qr/callback (WeChat redirects here after scan).
func QrCallbackHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		frontendURL := ctx.Config.WechatOpen.FrontendURL

		if code == "" || state == "" {
			http.Redirect(w, r, frontendURL+"/login?error=missing_params", http.StatusFound)
			return
		}

		tokResp, err := ctx.WechatClient.ExchangeCode(code)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("wechat exchange code", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=wechat_code_invalid", http.StatusFound)
			return
		}

		userInfo, err := ctx.WechatClient.FetchUserInfo(tokResp.AccessToken, tokResp.OpenID)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("wechat fetch userinfo", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=wechat_timeout", http.StatusFound)
			return
		}

		user, err := ctx.UserRepo.UpsertByWechat(r.Context(),
			tokResp.OpenID, tokResp.UnionID, userInfo.Nickname, userInfo.HeadImgURL)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("upsert user", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=server_error", http.StatusFound)
			return
		}

		jwtStr, err := ctx.JWTSigner.Sign(user.ID)
		if err != nil {
			logx.WithContext(r.Context()).Errorw("sign jwt", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=server_error", http.StatusFound)
			return
		}

		if err := ctx.AuthStateRepo.SetJWT(r.Context(), state, jwtStr); err != nil {
			logx.WithContext(r.Context()).Errorw("set jwt in state", logx.Field("err", err))
			http.Redirect(w, r, frontendURL+"/login?error=server_error", http.StatusFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding-top:60px"><p>✅ 登录成功，页面正在跳转…</p></body></html>`)
	}
}

// QrStatusHandler handles GET /api/auth/wechat/qr/status?state=xxx.
func QrStatusHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		if state == "" {
			writeError(w, r, http.StatusBadRequest, "missing state", nil)
			return
		}

		as, err := ctx.AuthStateRepo.Get(r.Context(), state)
		if errors.Is(err, sqlx.ErrNotFound) {
			httpx.OkJsonCtx(r.Context(), w, QrStatusResponse{Status: "expired"})
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "get state", err)
			return
		}
		if !as.JWT.Valid || as.JWT.String == "" {
			httpx.OkJsonCtx(r.Context(), w, QrStatusResponse{Status: "pending"})
			return
		}

		tok := as.JWT.String
		if err := ctx.AuthStateRepo.Delete(r.Context(), state); err != nil {
			logx.WithContext(r.Context()).Errorw("delete auth state", logx.Field("err", err))
		}
		httpx.OkJsonCtx(r.Context(), w, QrStatusResponse{Status: "done", Token: tok})
	}
}

// MeHandler handles GET /api/auth/me — requires Authorization: Bearer <jwt>.
func MeHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, r, http.StatusUnauthorized, "missing token", nil)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := ctx.JWTSigner.Verify(tokenStr)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "invalid token", err)
			return
		}

		user, err := ctx.UserRepo.FindByID(r.Context(), userID)
		if errors.Is(err, sqlx.ErrNotFound) {
			writeError(w, r, http.StatusUnauthorized, "user not found", nil)
			return
		}
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "find user", err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, MeResponse{
			ID:        user.ID,
			Nickname:  user.Nickname,
			AvatarURL: user.AvatarURL,
		})
	}
}

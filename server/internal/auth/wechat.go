package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const wechatBaseURL = "https://api.weixin.qq.com"

// WechatClient calls WeChat Open Platform APIs to exchange codes and fetch user info.
type WechatClient struct {
	AppID      string
	AppSecret  string
	httpClient *http.Client
	baseURL    string
}

// NewWechatClient creates a WechatClient for the given Open Platform credentials.
func NewWechatClient(appID, appSecret string) *WechatClient {
	return &WechatClient{
		AppID:      appID,
		AppSecret:  appSecret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    wechatBaseURL,
	}
}

// SetBaseURL overrides the base URL — used in tests to point at a mock server.
func (c *WechatClient) SetBaseURL(url string) {
	c.baseURL = url
}

// WechatTokenResp is the response from the access_token endpoint.
type WechatTokenResp struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// WechatUserInfo is the response from the userinfo endpoint.
type WechatUserInfo struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// ExchangeCode exchanges an OAuth code for an access token + openid.
func (c *WechatClient) ExchangeCode(code string) (*WechatTokenResp, error) {
	url := fmt.Sprintf(
		"%s/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		c.baseURL, c.AppID, c.AppSecret, code,
	)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("wechat exchange code: %w", err)
	}
	defer resp.Body.Close()
	var result WechatTokenResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("wechat decode token response: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

// FetchUserInfo fetches the user's nickname and avatar using the access token.
func (c *WechatClient) FetchUserInfo(accessToken, openID string) (*WechatUserInfo, error) {
	url := fmt.Sprintf(
		"%s/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		c.baseURL, accessToken, openID,
	)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("wechat fetch userinfo: %w", err)
	}
	defer resp.Body.Close()
	var result WechatUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("wechat decode userinfo: %w", err)
	}
	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat userinfo error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

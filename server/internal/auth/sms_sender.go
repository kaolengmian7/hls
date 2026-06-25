package auth

import (
	"context"
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"

	"hls.dev/server/internal/config"
)

// SmsSender sends a single SMS verification code to a phone number.
type SmsSender interface {
	Send(ctx context.Context, phone, code string) error
}

// TencentSmsSender sends SMS via Tencent Cloud SMS service.
type TencentSmsSender struct {
	client     *sms.Client
	sdkAppID   string
	signName   string
	templateID string
}

// NewTencentSmsSender creates a TencentSmsSender from config.
func NewTencentSmsSender(cfg config.TencentSMSConfig) *TencentSmsSender {
	cred := common.NewCredential(cfg.SecretID, cfg.SecretKey)
	cpf := profile.NewClientProfile()
	client, _ := sms.NewClient(cred, "ap-guangzhou", cpf)
	return &TencentSmsSender{
		client:     client,
		sdkAppID:   cfg.SmsSdkAppID,
		signName:   cfg.SignName,
		templateID: cfg.TemplateID,
	}
}

// Send sends a 6-digit code to the given phone number (without country prefix).
func (s *TencentSmsSender) Send(_ context.Context, phone, code string) error {
	req := sms.NewSendSmsRequest()
	req.SmsSdkAppId = common.StringPtr(s.sdkAppID)
	req.SignName = common.StringPtr(s.signName)
	req.TemplateId = common.StringPtr(s.templateID)
	req.TemplateParamSet = common.StringPtrs([]string{code, "10"})
	req.PhoneNumberSet = common.StringPtrs([]string{"+86" + phone})

	resp, err := s.client.SendSms(req)
	if err != nil {
		return fmt.Errorf("tencent sms send: %w", err)
	}
	for _, st := range resp.Response.SendStatusSet {
		if st.Code == nil || *st.Code != "Ok" {
			msg := ""
			if st.Message != nil {
				msg = *st.Message
			}
			errCode := "unknown"
			if st.Code != nil {
				errCode = *st.Code
			}
			return fmt.Errorf("tencent sms status %s: %s", errCode, msg)
		}
	}
	return nil
}

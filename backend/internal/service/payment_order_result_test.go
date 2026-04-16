package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

type paymentOrderResultSettingRepoStub struct{}

func (paymentOrderResultSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (paymentOrderResultSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (paymentOrderResultSettingRepoStub) Set(context.Context, string, string) error {
	return nil
}

func (paymentOrderResultSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (paymentOrderResultSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (paymentOrderResultSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (paymentOrderResultSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

func TestBuildCreateOrderResponseDefaultsToOrderCreated(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	resp := buildCreateOrderResponse(
		&dbent.PaymentOrder{
			ID:         42,
			Amount:     12.34,
			FeeRate:    0.03,
			ExpiresAt:  expiresAt,
			OutTradeNo: "sub2_42",
		},
		CreateOrderRequest{PaymentType: payment.TypeWxpay},
		12.71,
		&payment.InstanceSelection{PaymentMode: "qrcode"},
		&payment.CreatePaymentResponse{
			TradeNo: "sub2_42",
			QRCode:  "weixin://wxpay/bizpayurl?pr=test",
		},
		payment.CreatePaymentResultOrderCreated,
	)

	if resp.ResultType != payment.CreatePaymentResultOrderCreated {
		t.Fatalf("result type = %q, want %q", resp.ResultType, payment.CreatePaymentResultOrderCreated)
	}
	if resp.OutTradeNo != "sub2_42" {
		t.Fatalf("out_trade_no = %q, want %q", resp.OutTradeNo, "sub2_42")
	}
	if resp.PaymentType != payment.TypeWxpay {
		t.Fatalf("payment_type = %q, want %q", resp.PaymentType, payment.TypeWxpay)
	}
	if resp.QRCode != "weixin://wxpay/bizpayurl?pr=test" {
		t.Fatalf("qr_code = %q, want %q", resp.QRCode, "weixin://wxpay/bizpayurl?pr=test")
	}
	if resp.JSAPI != nil || resp.JSAPIPayload != nil {
		t.Fatal("legacy order_created response should not include jsapi payload")
	}
	if !resp.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expires_at = %v, want %v", resp.ExpiresAt, expiresAt)
	}
}

func TestBuildCreateOrderResponseCopiesJSAPIPayload(t *testing.T) {
	t.Parallel()

	jsapiPayload := &payment.WechatJSAPIPayload{
		AppID:     "wx123",
		TimeStamp: "1712345678",
		NonceStr:  "nonce-123",
		Package:   "prepay_id=wx123",
		SignType:  "RSA",
		PaySign:   "signed-payload",
	}
	resp := buildCreateOrderResponse(
		&dbent.PaymentOrder{
			ID:         88,
			Amount:     66.88,
			FeeRate:    0.01,
			ExpiresAt:  time.Date(2026, 4, 16, 13, 0, 0, 0, time.UTC),
			OutTradeNo: "sub2_88",
		},
		CreateOrderRequest{PaymentType: payment.TypeWxpay},
		67.55,
		&payment.InstanceSelection{PaymentMode: "popup"},
		&payment.CreatePaymentResponse{
			TradeNo:    "sub2_88",
			ResultType: payment.CreatePaymentResultJSAPIReady,
			JSAPI:      jsapiPayload,
		},
		payment.CreatePaymentResultJSAPIReady,
	)

	if resp.ResultType != payment.CreatePaymentResultJSAPIReady {
		t.Fatalf("result type = %q, want %q", resp.ResultType, payment.CreatePaymentResultJSAPIReady)
	}
	if resp.JSAPI == nil || resp.JSAPIPayload == nil {
		t.Fatal("expected jsapi payload aliases to be populated")
	}
	if resp.JSAPI != jsapiPayload {
		t.Fatal("jsapi pointer should be preserved")
	}
	if resp.JSAPIPayload != jsapiPayload {
		t.Fatal("jsapi_payload pointer should mirror jsapi")
	}
	if resp.PayURL != "" || resp.QRCode != "" || resp.ClientSecret != "" {
		t.Fatal("jsapi response should not populate legacy redirect/qr/client_secret fields")
	}
}

func TestMaybeBuildWeChatOAuthRequiredResponse(t *testing.T) {
	t.Parallel()

	settingSvc := NewSettingService(paymentOrderResultSettingRepoStub{}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			AppID:               "wx123456",
			AppSecret:           "wechat-secret",
			Scopes:              "snsapi_login snsapi_base",
			RedirectURL:         "https://merchant.example/api/v1/auth/oauth/wechat/callback",
			FrontendRedirectURL: "/auth/wechat/callback",
		},
	})
	svc := &PaymentService{settingService: settingSvc}

	resp, err := svc.maybeBuildWeChatOAuthRequiredResponse(context.Background(), CreateOrderRequest{
		Amount:          12.5,
		PaymentType:     payment.TypeWxpay,
		IsWeChatBrowser: true,
		RequestScheme:   "https",
		SrcHost:         "merchant.example",
		SrcURL:          "https://merchant.example/payment?from=wechat",
		OrderType:       payment.OrderTypeBalance,
	}, 12.5, 12.88, 0.03)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected oauth_required response, got nil")
	}
	if resp.ResultType != payment.CreatePaymentResultOAuthRequired {
		t.Fatalf("result type = %q, want %q", resp.ResultType, payment.CreatePaymentResultOAuthRequired)
	}
	if resp.Amount != 12.5 || resp.PayAmount != 12.88 || resp.FeeRate != 0.03 {
		t.Fatalf("unexpected amounts: amount=%.2f pay=%.2f fee=%.2f", resp.Amount, resp.PayAmount, resp.FeeRate)
	}
	if resp.OAuth == nil {
		t.Fatal("expected oauth payload, got nil")
	}
	if resp.OAuth.AppID != "wx123456" {
		t.Fatalf("appid = %q, want %q", resp.OAuth.AppID, "wx123456")
	}
	if resp.OAuth.Scope != "snsapi_base" {
		t.Fatalf("scope = %q, want %q", resp.OAuth.Scope, "snsapi_base")
	}
	if resp.OAuth.RedirectURL != "/auth/wechat/callback" {
		t.Fatalf("redirect_url = %q, want %q", resp.OAuth.RedirectURL, "/auth/wechat/callback")
	}
	if resp.OAuth.AuthorizeURL != "/api/v1/auth/oauth/wechat/payment/start?amount=12.5&order_type=balance&payment_type=wxpay&redirect=%2Fpurchase%3Ffrom%3Dwechat&scope=snsapi_base" {
		t.Fatalf("authorize_url = %q", resp.OAuth.AuthorizeURL)
	}
}

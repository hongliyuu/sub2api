package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestNewWeChatPayServiceDisabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.WeChatPay.Enabled = false

	svc := NewWeChatPayService(cfg)
	if svc.IsEnabled() {
		t.Fatal("IsEnabled() should be false when disabled")
	}
}

func TestWeChatPayServiceGetClientNotInitialized(t *testing.T) {
	cfg := &config.Config{}
	cfg.WeChatPay.Enabled = false

	svc := NewWeChatPayService(cfg)
	_, err := svc.GetClient()
	if err == nil {
		t.Fatal("GetClient() should return error when not initialized")
	}
}

func TestWeChatPayServiceGetPrivateKeyNotLoaded(t *testing.T) {
	cfg := &config.Config{}
	cfg.WeChatPay.Enabled = false

	svc := NewWeChatPayService(cfg)
	_, err := svc.GetPrivateKey()
	if err == nil {
		t.Fatal("GetPrivateKey() should return error when not loaded")
	}
}

func TestWeChatPayServiceGetConfigSanitized(t *testing.T) {
	cfg := &config.Config{}
	cfg.WeChatPay.Enabled = true
	cfg.WeChatPay.AppID = "wx123"
	cfg.WeChatPay.MchID = "mch456"
	cfg.WeChatPay.APIv3Key = "secret-key-should-not-be-returned"
	cfg.WeChatPay.CertSerialNo = "serial-should-not-be-returned"
	cfg.WeChatPay.PrivateKeyPath = "/path/should-not-be-returned"
	cfg.WeChatPay.NotifyURL = "https://example.com/callback"

	// Don't call initClient() since it requires real key file
	svc := &WeChatPayService{cfg: cfg}

	result := svc.GetConfig()
	if result.AppID != "wx123" {
		t.Fatalf("GetConfig().AppID = %q, want wx123", result.AppID)
	}
	if result.MchID != "mch456" {
		t.Fatalf("GetConfig().MchID = %q, want mch456", result.MchID)
	}
	if result.NotifyURL != "https://example.com/callback" {
		t.Fatalf("GetConfig().NotifyURL = %q, want https://example.com/callback", result.NotifyURL)
	}
	// Sensitive fields must be empty (sanitized)
	if result.APIv3Key != "" {
		t.Fatal("GetConfig().APIv3Key should be empty (sanitized)")
	}
	if result.CertSerialNo != "" {
		t.Fatal("GetConfig().CertSerialNo should be empty (sanitized)")
	}
	if result.PrivateKeyPath != "" {
		t.Fatal("GetConfig().PrivateKeyPath should be empty (sanitized)")
	}
}

func TestWeChatPayServiceGetRechargeConfig(t *testing.T) {
	cfg := &config.Config{}
	svc := &WeChatPayService{cfg: cfg}

	rechargeCfg := svc.GetRechargeConfig()

	if rechargeCfg.MinAmount != 1.0 {
		t.Fatalf("GetRechargeConfig().MinAmount = %f, want 1.0", rechargeCfg.MinAmount)
	}
	if rechargeCfg.MaxAmount != 1000.0 {
		t.Fatalf("GetRechargeConfig().MaxAmount = %f, want 1000.0", rechargeCfg.MaxAmount)
	}
	if len(rechargeCfg.DefaultAmounts) != 5 {
		t.Fatalf("GetRechargeConfig().DefaultAmounts length = %d, want 5", len(rechargeCfg.DefaultAmounts))
	}
}

func TestWeChatPayServiceValidateRechargeAmount(t *testing.T) {
	cfg := &config.Config{}
	svc := &WeChatPayService{cfg: cfg}

	tests := []struct {
		name      string
		amount    float64
		expectErr bool
	}{
		{"valid amount 10", 10.0, false},
		{"valid amount 100", 100.0, false},
		{"valid amount min boundary", 1.0, false},
		{"valid amount max boundary", 1000.0, false},
		{"invalid amount too small", 0.5, true},
		{"invalid amount zero", 0, true},
		{"invalid amount too large", 1001.0, true},
		{"invalid amount negative", -10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateRechargeAmount(tt.amount)
			if tt.expectErr && err == nil {
				t.Errorf("ValidateRechargeAmount(%f) expected error, got nil", tt.amount)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateRechargeAmount(%f) unexpected error: %v", tt.amount, err)
			}
		})
	}
}

func TestAmountToFen(t *testing.T) {
	tests := []struct {
		name     string
		yuan     float64
		expected int64
	}{
		{"整数金额", 10.0, 1000},
		{"小数金额", 10.50, 1050},
		{"两位小数", 99.99, 9999},
		{"最小金额", 0.01, 1},
		{"大金额", 1000.0, 100000},
		{"带精度问题的金额", 19.9, 1990},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AmountToFen(tt.yuan)
			if result != tt.expected {
				t.Errorf("AmountToFen(%f) = %d, want %d", tt.yuan, result, tt.expected)
			}
		})
	}
}

func TestCreateWeChatPayOrderRequest(t *testing.T) {
	cfg := &config.Config{}
	cfg.WeChatPay.AppID = "wx123456"
	cfg.WeChatPay.MchID = "1234567890"
	cfg.WeChatPay.NotifyURL = "https://example.com/api/v1/wechat/pay/notify"

	svc := &WeChatPayService{cfg: cfg}

	tests := []struct {
		name           string
		orderNo        string
		amount         float64
		description    string
		channel        string
		openID         string
		expectChannel  PaymentChannel
	}{
		{
			name:          "Native支付",
			orderNo:       "RECH20260201000001",
			amount:        10.0,
			description:   "充值 10.00 元",
			channel:       "native",
			openID:        "",
			expectChannel: WeChatPayChannelNative,
		},
		{
			name:          "JSAPI支付",
			orderNo:       "RECH20260201000002",
			amount:        50.0,
			description:   "充值 50.00 元",
			channel:       "jsapi",
			openID:        "oUpF8uMuAJO",
			expectChannel: WeChatPayChannelJSAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := svc.buildPaymentRequest(tt.orderNo, tt.amount, tt.description, tt.channel, tt.openID)
			if req == nil {
				t.Fatal("buildPaymentRequest returned nil")
			}
			if req.AppID != cfg.WeChatPay.AppID {
				t.Errorf("AppID = %s, want %s", req.AppID, cfg.WeChatPay.AppID)
			}
			if req.MchID != cfg.WeChatPay.MchID {
				t.Errorf("MchID = %s, want %s", req.MchID, cfg.WeChatPay.MchID)
			}
			if req.OutTradeNo != tt.orderNo {
				t.Errorf("OutTradeNo = %s, want %s", req.OutTradeNo, tt.orderNo)
			}
			if req.AmountInFen != AmountToFen(tt.amount) {
				t.Errorf("AmountInFen = %d, want %d", req.AmountInFen, AmountToFen(tt.amount))
			}
			if req.Description != tt.description {
				t.Errorf("Description = %s, want %s", req.Description, tt.description)
			}
			if req.NotifyURL != cfg.WeChatPay.NotifyURL {
				t.Errorf("NotifyURL = %s, want %s", req.NotifyURL, cfg.WeChatPay.NotifyURL)
			}
			if req.Channel != tt.expectChannel {
				t.Errorf("Channel = %v, want %v", req.Channel, tt.expectChannel)
			}
			if tt.channel == "jsapi" && req.OpenID != tt.openID {
				t.Errorf("OpenID = %s, want %s", req.OpenID, tt.openID)
			}
		})
	}
}

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestPcParseFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		defaultVal float64
		expected   float64
	}{
		{"empty string returns default", "", 1.0, 1.0},
		{"valid float", "3.14", 0, 3.14},
		{"valid integer as float", "42", 0, 42.0},
		{"invalid string returns default", "notanumber", 9.99, 9.99},
		{"zero value", "0", 5.0, 0},
		{"negative value", "-10.5", 0, -10.5},
		{"very large value", "99999999.99", 0, 99999999.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pcParseFloat(tt.input, tt.defaultVal)
			if got != tt.expected {
				t.Fatalf("pcParseFloat(%q, %v) = %v, want %v", tt.input, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestPcParseInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		defaultVal int
		expected   int
	}{
		{"empty string returns default", "", 30, 30},
		{"valid int", "10", 0, 10},
		{"invalid string returns default", "abc", 5, 5},
		{"float string returns default", "3.14", 0, 0},
		{"zero value", "0", 99, 0},
		{"negative value", "-1", 0, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pcParseInt(tt.input, tt.defaultVal)
			if got != tt.expected {
				t.Fatalf("pcParseInt(%q, %v) = %v, want %v", tt.input, tt.defaultVal, got, tt.expected)
			}
		})
	}
}

func TestParsePaymentConfig(t *testing.T) {
	t.Parallel()

	svc := &PaymentConfigService{}

	t.Run("empty vals uses defaults", func(t *testing.T) {
		t.Parallel()
		cfg := svc.parsePaymentConfig(map[string]string{})
		if cfg.Enabled {
			t.Fatal("expected Enabled=false by default")
		}
		if cfg.MinAmount != 1 {
			t.Fatalf("expected MinAmount=1, got %v", cfg.MinAmount)
		}
		if cfg.MaxAmount != 0 {
			t.Fatalf("expected MaxAmount=0 (no limit), got %v", cfg.MaxAmount)
		}
		if cfg.OrderTimeoutMin != 30 {
			t.Fatalf("expected OrderTimeoutMin=30, got %v", cfg.OrderTimeoutMin)
		}
		if cfg.MaxPendingOrders != 3 {
			t.Fatalf("expected MaxPendingOrders=3, got %v", cfg.MaxPendingOrders)
		}
		if cfg.LoadBalanceStrategy != payment.DefaultLoadBalanceStrategy {
			t.Fatalf("expected LoadBalanceStrategy=%s, got %q", payment.DefaultLoadBalanceStrategy, cfg.LoadBalanceStrategy)
		}
		if len(cfg.EnabledTypes) != 0 {
			t.Fatalf("expected empty EnabledTypes, got %v", cfg.EnabledTypes)
		}
	})

	t.Run("all values populated", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingPaymentEnabled:      "true",
			SettingMinRechargeAmount:   "5.00",
			SettingMaxRechargeAmount:   "1000.00",
			SettingDailyRechargeLimit:  "5000.00",
			SettingOrderTimeoutMinutes: "15",
			SettingMaxPendingOrders:    "5",
			SettingEnabledPaymentTypes: "alipay,wxpay,stripe",
			SettingBalancePayDisabled:  "true",
			SettingLoadBalanceStrategy: "least_amount",
			SettingProductNamePrefix:   "PRE",
			SettingProductNameSuffix:   "SUF",
		}
		cfg := svc.parsePaymentConfig(vals)

		if !cfg.Enabled {
			t.Fatal("expected Enabled=true")
		}
		if cfg.MinAmount != 5 {
			t.Fatalf("MinAmount = %v, want 5", cfg.MinAmount)
		}
		if cfg.MaxAmount != 1000 {
			t.Fatalf("MaxAmount = %v, want 1000", cfg.MaxAmount)
		}
		if cfg.DailyLimit != 5000 {
			t.Fatalf("DailyLimit = %v, want 5000", cfg.DailyLimit)
		}
		if cfg.OrderTimeoutMin != 15 {
			t.Fatalf("OrderTimeoutMin = %v, want 15", cfg.OrderTimeoutMin)
		}
		if cfg.MaxPendingOrders != 5 {
			t.Fatalf("MaxPendingOrders = %v, want 5", cfg.MaxPendingOrders)
		}
		if len(cfg.EnabledTypes) != 3 {
			t.Fatalf("EnabledTypes len = %d, want 3", len(cfg.EnabledTypes))
		}
		if cfg.EnabledTypes[0] != "alipay" || cfg.EnabledTypes[1] != "wxpay" || cfg.EnabledTypes[2] != "stripe" {
			t.Fatalf("EnabledTypes = %v, want [alipay wxpay stripe]", cfg.EnabledTypes)
		}
		if !cfg.BalanceDisabled {
			t.Fatal("expected BalanceDisabled=true")
		}
		if cfg.LoadBalanceStrategy != "least_amount" {
			t.Fatalf("LoadBalanceStrategy = %q, want %q", cfg.LoadBalanceStrategy, "least_amount")
		}
		if cfg.ProductNamePrefix != "PRE" {
			t.Fatalf("ProductNamePrefix = %q, want %q", cfg.ProductNamePrefix, "PRE")
		}
		if cfg.ProductNameSuffix != "SUF" {
			t.Fatalf("ProductNameSuffix = %q, want %q", cfg.ProductNameSuffix, "SUF")
		}
	})

	t.Run("enabled types with spaces are trimmed", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingEnabledPaymentTypes: " alipay , wxpay ",
		}
		cfg := svc.parsePaymentConfig(vals)
		if len(cfg.EnabledTypes) != 2 {
			t.Fatalf("EnabledTypes len = %d, want 2", len(cfg.EnabledTypes))
		}
		if cfg.EnabledTypes[0] != "alipay" || cfg.EnabledTypes[1] != "wxpay" {
			t.Fatalf("EnabledTypes = %v, want [alipay wxpay]", cfg.EnabledTypes)
		}
	})

	t.Run("empty enabled types string", func(t *testing.T) {
		t.Parallel()
		vals := map[string]string{
			SettingEnabledPaymentTypes: "",
		}
		cfg := svc.parsePaymentConfig(vals)
		if len(cfg.EnabledTypes) != 0 {
			t.Fatalf("expected empty EnabledTypes for empty string, got %v", cfg.EnabledTypes)
		}
	})
}

func TestGetBasePaymentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{payment.TypeEasyPay, payment.TypeEasyPay},
		{payment.TypeStripe, payment.TypeStripe},
		{payment.TypeCard, payment.TypeStripe},
		{payment.TypeLink, payment.TypeStripe},
		{payment.TypeAlipay, payment.TypeAlipay},
		{payment.TypeAlipayDirect, payment.TypeAlipay},
		{payment.TypeWxpay, payment.TypeWxpay},
		{payment.TypeWxpayDirect, payment.TypeWxpay},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := payment.GetBasePaymentType(tt.input)
			if got != tt.expected {
				t.Fatalf("GetBasePaymentType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMergePaymentConfigPatchPreservesOmittedFields(t *testing.T) {
	t.Parallel()

	current := &PaymentConfig{
		Enabled:                true,
		MinAmount:              10,
		MaxAmount:              200,
		DailyLimit:             500,
		OrderTimeoutMin:        15,
		MaxPendingOrders:       4,
		EnabledTypes:           []string{"alipay", "wxpay"},
		BalanceDisabled:        true,
		LoadBalanceStrategy:    string(payment.StrategyLeastAmount),
		ProductNamePrefix:      "PRE",
		ProductNameSuffix:      "SUF",
		HelpImageURL:           "https://example.com/help.png",
		HelpText:               "help",
		CancelRateLimitEnabled: true,
		CancelRateLimitMax:     5,
		CancelRateLimitWindow:  60,
		CancelRateLimitUnit:    "minute",
		CancelRateLimitMode:    "rolling",
	}

	newMin := 25.0
	req := UpdatePaymentConfigRequest{
		MinAmount: &newMin,
	}

	merged := mergePaymentConfigPatch(current, req)

	if merged.MinAmount != 25 {
		t.Fatalf("MinAmount = %v, want 25", merged.MinAmount)
	}
	if merged.MaxAmount != current.MaxAmount {
		t.Fatalf("MaxAmount = %v, want %v", merged.MaxAmount, current.MaxAmount)
	}
	if len(merged.EnabledTypes) != 2 || merged.EnabledTypes[0] != "alipay" || merged.EnabledTypes[1] != "wxpay" {
		t.Fatalf("EnabledTypes = %v, want [alipay wxpay]", merged.EnabledTypes)
	}
	if !merged.BalanceDisabled {
		t.Fatal("expected BalanceDisabled to be preserved")
	}
	if merged.HelpText != current.HelpText {
		t.Fatalf("HelpText = %q, want %q", merged.HelpText, current.HelpText)
	}
	if !merged.CancelRateLimitEnabled || merged.CancelRateLimitMax != 5 || merged.CancelRateLimitWindow != 60 {
		t.Fatalf("cancel rate limit settings were not preserved: %+v", merged)
	}
}

func TestMergePaymentConfigPatchAllowsClearingEnabledTypes(t *testing.T) {
	t.Parallel()

	current := &PaymentConfig{
		Enabled:      true,
		EnabledTypes: []string{"alipay", "wxpay"},
	}
	req := UpdatePaymentConfigRequest{
		EnabledTypes: []string{},
	}

	merged := mergePaymentConfigPatch(current, req)
	if len(merged.EnabledTypes) != 0 {
		t.Fatalf("EnabledTypes = %v, want empty slice", merged.EnabledTypes)
	}

	serialized := serializePaymentConfig(merged)
	if got := serialized[SettingEnabledPaymentTypes]; got != "" {
		t.Fatalf("serialized enabled types = %q, want empty string", got)
	}
}

func TestIsPaymentTypeEnabled(t *testing.T) {
	t.Parallel()

	cfg := &PaymentConfig{
		EnabledTypes: []string{"alipay", "stripe"},
	}

	if !isPaymentTypeEnabled(cfg, "alipay") {
		t.Fatal("expected alipay to be enabled")
	}
	if !isPaymentTypeEnabled(cfg, " Stripe ") {
		t.Fatal("expected stripe to be enabled with case/space normalization")
	}
	if isPaymentTypeEnabled(cfg, "wxpay") {
		t.Fatal("expected wxpay to be disabled")
	}
	if isPaymentTypeEnabled(cfg, "") {
		t.Fatal("expected empty payment type to be disabled")
	}
}

package service

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

type refundProviderStub struct {
	key string
}

func (s refundProviderStub) Name() string { return s.key }

func (s refundProviderStub) ProviderKey() string { return s.key }

func (s refundProviderStub) SupportedTypes() []payment.PaymentType { return nil }

func (s refundProviderStub) CreatePayment(context.Context, payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	return nil, nil
}

func (s refundProviderStub) QueryOrder(context.Context, string) (*payment.QueryOrderResponse, error) {
	return nil, nil
}

func (s refundProviderStub) VerifyNotification(context.Context, string, map[string]string) (*payment.PaymentNotification, error) {
	return nil, nil
}

func (s refundProviderStub) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, nil
}

func TestResolveRefundTradeNo(t *testing.T) {
	t.Parallel()

	svc := &PaymentService{}
	order := &dbent.PaymentOrder{
		ID:         42,
		OutTradeNo: "sub2_test_order",
	}

	t.Run("returns stored payment trade number when present", func(t *testing.T) {
		t.Parallel()

		o := *order
		o.PaymentTradeNo = "pi_123"
		tradeNo, err := svc.resolveRefundTradeNo(context.Background(), refundProviderStub{key: payment.TypeStripe}, &o)
		if err != nil {
			t.Fatalf("resolveRefundTradeNo returned error: %v", err)
		}
		if tradeNo != "pi_123" {
			t.Fatalf("tradeNo = %q, want pi_123", tradeNo)
		}
	})

	t.Run("easypay allows refund by out_trade_no when trade number is missing", func(t *testing.T) {
		t.Parallel()

		tradeNo, err := svc.resolveRefundTradeNo(context.Background(), refundProviderStub{key: payment.TypeEasyPay}, order)
		if err != nil {
			t.Fatalf("resolveRefundTradeNo returned error: %v", err)
		}
		if tradeNo != "" {
			t.Fatalf("tradeNo = %q, want empty string", tradeNo)
		}
	})

	t.Run("stripe requires payment intent id when trade number is missing", func(t *testing.T) {
		t.Parallel()

		if _, err := svc.resolveRefundTradeNo(context.Background(), refundProviderStub{key: payment.TypeStripe}, order); err == nil {
			t.Fatal("expected missing stripe payment trade number to fail")
		}
	})
}

func TestPrepDeductAllowsPartialBalanceDeductionWithoutForce(t *testing.T) {
	t.Parallel()

	svc := &PaymentService{
		userRepo: &userRepoStub{
			user: &User{ID: 1, Balance: 12},
		},
	}
	order := &dbent.PaymentOrder{
		UserID:    1,
		OrderType: payment.OrderTypeBalance,
	}
	plan := &RefundPlan{
		RefundAmount: 20,
	}

	result := svc.prepDeduct(context.Background(), order, plan, false)
	if result != nil {
		t.Fatalf("prepDeduct returned unexpected early result: %+v", result)
	}
	if plan.DeductionType != payment.DeductionTypeBalance {
		t.Fatalf("DeductionType = %q, want %q", plan.DeductionType, payment.DeductionTypeBalance)
	}
	if plan.BalanceToDeduct != 12 {
		t.Fatalf("BalanceToDeduct = %v, want 12", plan.BalanceToDeduct)
	}
}

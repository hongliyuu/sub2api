package service

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// ---------------------------------------------------------------------------
// TestCryptoRandFloat64
// ---------------------------------------------------------------------------

func TestCryptoRandFloat64(t *testing.T) {
	t.Run("values in [0, 1) range", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			v := cryptoRandFloat64()
			if v < 0 || v >= 1 {
				t.Fatalf("cryptoRandFloat64() = %v; want [0, 1)", v)
			}
		}
	})

	t.Run("not all same value", func(t *testing.T) {
		first := cryptoRandFloat64()
		allSame := true
		for i := 0; i < 999; i++ {
			if cryptoRandFloat64() != first {
				allSame = false
				break
			}
		}
		if allSame {
			t.Fatal("cryptoRandFloat64() returned the same value 1000 times; expected variance")
		}
	})
}

// ---------------------------------------------------------------------------
// TestWeightConfigMap
// ---------------------------------------------------------------------------

func TestWeightConfigMap(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		wantKeys map[string]float64 // subset checks
	}{
		{
			name:   "default config when empty",
			config: "",
			wantKeys: map[string]float64{
				LotteryUserCategoryNewUser:    3.0,
				LotteryUserCategoryRegular:    1.0,
				LotteryUserCategoryPaid:       0.3,
				LotteryUserCategorySubscriber: 0.1,
			},
		},
		{
			name:   "custom JSON overrides defaults",
			config: `{"new_user":5.0,"regular":2.0,"paid":0.5,"subscriber":0.2}`,
			wantKeys: map[string]float64{
				LotteryUserCategoryNewUser:    5.0,
				LotteryUserCategoryRegular:    2.0,
				LotteryUserCategoryPaid:       0.5,
				LotteryUserCategorySubscriber: 0.2,
			},
		},
		{
			name:   "partial JSON merges with defaults",
			config: `{"new_user":10.0}`,
			wantKeys: map[string]float64{
				LotteryUserCategoryNewUser:    10.0,
				LotteryUserCategoryRegular:    1.0,
				LotteryUserCategoryPaid:       0.3,
				LotteryUserCategorySubscriber: 0.1,
			},
		},
		{
			name:   "invalid JSON falls back to defaults",
			config: `{invalid-json`,
			wantKeys: map[string]float64{
				LotteryUserCategoryNewUser:    3.0,
				LotteryUserCategoryRegular:    1.0,
				LotteryUserCategoryPaid:       0.3,
				LotteryUserCategorySubscriber: 0.1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &LotteryActivity{WeightConfig: tc.config}
			got := a.WeightConfigMap()
			for k, want := range tc.wantKeys {
				if got[k] != want {
					t.Errorf("WeightConfigMap()[%q] = %v; want %v", k, got[k], want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLotteryCouponIsActive
// ---------------------------------------------------------------------------

func TestLotteryCouponIsActive(t *testing.T) {
	tests := []struct {
		name   string
		coupon LotteryCoupon
		want   bool
	}{
		{
			name: "active and not expired",
			coupon: LotteryCoupon{
				Status:    LotteryCouponStatusActive,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			want: true,
		},
		{
			name: "active but expired",
			coupon: LotteryCoupon{
				Status:    LotteryCouponStatusActive,
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			want: false,
		},
		{
			name: "used status",
			coupon: LotteryCoupon{
				Status:    LotteryCouponStatusUsed,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			want: false,
		},
		{
			name: "expired status",
			coupon: LotteryCoupon{
				Status:    LotteryCouponStatusExpired,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			want: false,
		},
		{
			name: "active with exact now boundary (expires at past instant)",
			coupon: LotteryCoupon{
				Status:    LotteryCouponStatusActive,
				ExpiresAt: time.Now().Add(-1 * time.Millisecond),
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.coupon.IsActive(); got != tc.want {
				t.Errorf("LotteryCoupon.IsActive() = %v; want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestLotteryActivityIsActive
// ---------------------------------------------------------------------------

func TestLotteryActivityIsActive(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active status", LotteryStatusActive, true},
		{"pending status", LotteryStatusPending, false},
		{"drawing status", LotteryStatusDrawing, false},
		{"completed status", LotteryStatusCompleted, false},
		{"cancelled status", LotteryStatusCancelled, false},
		{"expired status", LotteryStatusExpired, false},
		{"empty status", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &LotteryActivity{Status: tc.status}
			if got := a.IsActive(); got != tc.want {
				t.Errorf("LotteryActivity.IsActive() with status %q = %v; want %v", tc.status, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGenerateShareCode
// ---------------------------------------------------------------------------

func TestGenerateShareCode(t *testing.T) {
	t.Run("length is 16 hex characters", func(t *testing.T) {
		code := generateShareCode()
		if len(code) != 16 {
			t.Errorf("generateShareCode() length = %d; want 16", len(code))
		}
	})

	t.Run("valid hex characters", func(t *testing.T) {
		code := generateShareCode()
		if _, err := hex.DecodeString(code); err != nil {
			t.Errorf("generateShareCode() = %q is not valid hex: %v", code, err)
		}
	})

	t.Run("uniqueness over 100 iterations", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code := generateShareCode()
			if seen[code] {
				t.Fatalf("duplicate share code generated: %s", code)
			}
			seen[code] = true
		}
	})
}

// ---------------------------------------------------------------------------
// TestClassifyUser - uses minimal mock repos
// ---------------------------------------------------------------------------

// mockUserSubRepoForClassify is a minimal mock for UserSubscriptionRepository.
// It embeds the interface so only the methods we need are implemented;
// calling any other method will panic (acceptable in unit tests).
type mockUserSubRepoForClassify struct {
	UserSubscriptionRepository // embed to satisfy interface
	subs                       []UserSubscription
	err                        error
}

func (m *mockUserSubRepoForClassify) ListActiveByUserID(_ context.Context, _ int64) ([]UserSubscription, error) {
	return m.subs, m.err
}

// mockRechargeOrderRepoForClassify is a minimal mock for RechargeOrderRepository.
type mockRechargeOrderRepoForClassify struct {
	RechargeOrderRepository // embed to satisfy interface
	result                  *ListRechargeOrdersResult
	err                     error
}

func (m *mockRechargeOrderRepoForClassify) ListByUserID(_ context.Context, _ int64, _ *ListRechargeOrdersRequest) (*ListRechargeOrdersResult, error) {
	return m.result, m.err
}

func TestClassifyUser(t *testing.T) {
	activityStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		user            *User
		subs            []UserSubscription
		subErr          error
		rechargeResult  *ListRechargeOrdersResult
		rechargeErr     error
		wantCategory    string
	}{
		{
			name: "new user registered after activity start",
			user: &User{
				ID:        1,
				CreatedAt: activityStart.Add(24 * time.Hour), // registered after activity start
			},
			wantCategory: LotteryUserCategoryNewUser,
		},
		{
			name: "subscriber with active subscriptions",
			user: &User{
				ID:        2,
				CreatedAt: activityStart.Add(-30 * 24 * time.Hour), // registered before
			},
			subs:         []UserSubscription{{ID: 1, UserID: 2}},
			wantCategory: LotteryUserCategorySubscriber,
		},
		{
			name: "paid user with recharge orders",
			user: &User{
				ID:        3,
				CreatedAt: activityStart.Add(-30 * 24 * time.Hour),
			},
			subs: nil, // no active subscriptions
			rechargeResult: &ListRechargeOrdersResult{
				Orders: []*RechargeOrder{{ID: 1, UserID: 3}},
			},
			wantCategory: LotteryUserCategoryPaid,
		},
		{
			name: "regular user with no subscriptions and no orders",
			user: &User{
				ID:        4,
				CreatedAt: activityStart.Add(-30 * 24 * time.Hour),
			},
			subs:           nil,
			rechargeResult: &ListRechargeOrdersResult{Orders: nil},
			wantCategory:   LotteryUserCategoryRegular,
		},
		{
			name: "regular user when repos return errors",
			user: &User{
				ID:        5,
				CreatedAt: activityStart.Add(-30 * 24 * time.Hour),
			},
			subs:           nil,
			subErr:         context.DeadlineExceeded,
			rechargeResult: nil,
			rechargeErr:    context.DeadlineExceeded,
			wantCategory:   LotteryUserCategoryRegular,
		},
		{
			name: "subscriber takes priority over paid",
			user: &User{
				ID:        6,
				CreatedAt: activityStart.Add(-30 * 24 * time.Hour),
			},
			subs: []UserSubscription{{ID: 1, UserID: 6}},
			rechargeResult: &ListRechargeOrdersResult{
				Orders: []*RechargeOrder{{ID: 1, UserID: 6}},
			},
			wantCategory: LotteryUserCategorySubscriber,
		},
		{
			name: "new user takes priority over everything",
			user: &User{
				ID:        7,
				CreatedAt: activityStart.Add(1 * time.Hour),
			},
			subs: []UserSubscription{{ID: 1, UserID: 7}},
			rechargeResult: &ListRechargeOrdersResult{
				Orders: []*RechargeOrder{{ID: 1, UserID: 7}},
			},
			wantCategory: LotteryUserCategoryNewUser,
		},
		{
			name: "regular when recharge result is nil",
			user: &User{
				ID:        8,
				CreatedAt: activityStart.Add(-30 * 24 * time.Hour),
			},
			subs:           nil,
			rechargeResult: nil,
			wantCategory:   LotteryUserCategoryRegular,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &LotteryDrawService{
				userSubRepo: &mockUserSubRepoForClassify{
					subs: tc.subs,
					err:  tc.subErr,
				},
				rechargeOrderRepo: &mockRechargeOrderRepoForClassify{
					result: tc.rechargeResult,
					err:    tc.rechargeErr,
				},
			}

			got := svc.classifyUser(context.Background(), tc.user, activityStart)
			if got != tc.wantCategory {
				t.Errorf("classifyUser() = %q; want %q", got, tc.wantCategory)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestClassifyUser_EdgeCases
// ---------------------------------------------------------------------------

func TestClassifyUser_EdgeCases(t *testing.T) {
	t.Run("user created exactly at activity start is not new", func(t *testing.T) {
		activityStart := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
		user := &User{
			ID:        100,
			CreatedAt: activityStart, // exact same time
		}

		svc := &LotteryDrawService{
			userSubRepo:       &mockUserSubRepoForClassify{},
			rechargeOrderRepo: &mockRechargeOrderRepoForClassify{},
		}

		// time.After returns false when equal, so user should NOT be classified as new
		got := svc.classifyUser(context.Background(), user, activityStart)
		if got == LotteryUserCategoryNewUser {
			t.Errorf("classifyUser() = %q; user created at exact activity start should not be new_user", got)
		}
	})

	t.Run("recharge result with empty orders slice is regular", func(t *testing.T) {
		activityStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		user := &User{
			ID:        200,
			CreatedAt: activityStart.Add(-24 * time.Hour),
		}

		svc := &LotteryDrawService{
			userSubRepo: &mockUserSubRepoForClassify{},
			rechargeOrderRepo: &mockRechargeOrderRepoForClassify{
				result: &ListRechargeOrdersResult{
					Orders:     []*RechargeOrder{},
					Pagination: &pagination.PaginationResult{},
				},
			},
		}

		got := svc.classifyUser(context.Background(), user, activityStart)
		if got != LotteryUserCategoryRegular {
			t.Errorf("classifyUser() = %q; want %q", got, LotteryUserCategoryRegular)
		}
	})
}

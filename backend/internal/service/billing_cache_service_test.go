package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type billingCacheWorkerStub struct {
	balanceUpdates      int64
	subscriptionUpdates int64
}

func (b *billingCacheWorkerStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateUserBalance(ctx context.Context, userID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *billingCacheWorkerStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	return nil
}

func (b *billingCacheWorkerStub) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	return nil
}

type billingCacheFallbackStub struct {
	billingCacheWorkerStub
	rateLimitUpdates atomic.Int64
}

func (b *billingCacheFallbackStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	b.rateLimitUpdates.Add(1)
	return nil
}

type userRepoBase struct{}

func (userRepoBase) Create(context.Context, *User) error               { return nil }
func (userRepoBase) GetByID(context.Context, int64) (*User, error)     { return nil, nil }
func (userRepoBase) GetByEmail(context.Context, string) (*User, error) { return nil, nil }
func (userRepoBase) GetFirstAdmin(context.Context) (*User, error)      { return nil, nil }
func (userRepoBase) Update(context.Context, *User) error               { return nil }
func (userRepoBase) Delete(context.Context, int64) error               { return nil }
func (userRepoBase) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (userRepoBase) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (userRepoBase) UpdateBalance(context.Context, int64, float64) error { return nil }
func (userRepoBase) DeductBalance(context.Context, int64, float64) error { return nil }
func (userRepoBase) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (userRepoBase) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }
func (userRepoBase) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (userRepoBase) AddGroupToAllowedGroups(context.Context, int64, int64) error          { return nil }
func (userRepoBase) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error { return nil }
func (userRepoBase) UpdateTotpSecret(context.Context, int64, *string) error               { return nil }
func (userRepoBase) EnableTotp(context.Context, int64) error                              { return nil }
func (userRepoBase) DisableTotp(context.Context, int64) error                             { return nil }

type fallbackUserRepo struct {
	userRepoBase
	balance float64
}

func (f *fallbackUserRepo) GetByID(context.Context, int64) (*User, error) {
	return &User{Balance: f.balance}, nil
}

type userSubscriptionRepoBase struct{}

func (userSubscriptionRepoBase) Create(context.Context, *UserSubscription) error {
	return nil
}
func (userSubscriptionRepoBase) GetByID(context.Context, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubscriptionRepoBase) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubscriptionRepoBase) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, nil
}
func (userSubscriptionRepoBase) Update(context.Context, *UserSubscription) error { return nil }
func (userSubscriptionRepoBase) Delete(context.Context, int64) error             { return nil }
func (userSubscriptionRepoBase) ListByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}
func (userSubscriptionRepoBase) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}
func (userSubscriptionRepoBase) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (userSubscriptionRepoBase) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string, string) ([]UserSubscription, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (userSubscriptionRepoBase) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (userSubscriptionRepoBase) ExtendExpiry(context.Context, int64, time.Time) error     { return nil }
func (userSubscriptionRepoBase) UpdateStatus(context.Context, int64, string) error        { return nil }
func (userSubscriptionRepoBase) UpdateNotes(context.Context, int64, string) error         { return nil }
func (userSubscriptionRepoBase) ActivateWindows(context.Context, int64, time.Time) error  { return nil }
func (userSubscriptionRepoBase) ResetDailyUsage(context.Context, int64, time.Time) error  { return nil }
func (userSubscriptionRepoBase) ResetWeeklyUsage(context.Context, int64, time.Time) error { return nil }
func (userSubscriptionRepoBase) ResetMonthlyUsage(context.Context, int64, time.Time) error {
	return nil
}
func (userSubscriptionRepoBase) IncrementUsage(context.Context, int64, float64) error { return nil }
func (userSubscriptionRepoBase) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	return 0, nil
}

type fallbackSubscriptionRepo struct {
	userSubscriptionRepoBase
	subscription UserSubscription
}

func (f *fallbackSubscriptionRepo) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	cp := f.subscription
	return &cp, nil
}

func TestBillingCacheServiceQueueHighLoad(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	start := time.Now()
	for i := 0; i < cacheWriteBufferSize*2; i++ {
		svc.QueueDeductBalance(1, 1)
	}
	require.Less(t, time.Since(start), 2*time.Second)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceEnqueueAfterStopReturnsFalse(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, &config.Config{})
	svc.Stop()

	enqueued := svc.enqueueCacheWrite(cacheWriteTask{
		kind:   cacheWriteDeductBalance,
		userID: 1,
		amount: 1,
	})
	require.False(t, enqueued)
}

func TestBillingCacheServiceGetUserBalanceFallbackWhenQueueFull(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := &BillingCacheService{
		cache:          cache,
		userRepo:       &fallbackUserRepo{balance: 38.5},
		cacheWriteChan: make(chan cacheWriteTask, 1),
		cfg:            &config.Config{},
	}
	// Fill the queue so the next enqueue hits the fallback path.
	svc.cacheWriteChan <- cacheWriteTask{kind: cacheWriteSetBalance}

	balance, err := svc.GetUserBalance(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, 38.5, balance)
	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) >= 1
	}, time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceGetSubscriptionStatusFallbackWhenQueueFull(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := &BillingCacheService{
		cache:          cache,
		subRepo:        &fallbackSubscriptionRepo{subscription: UserSubscription{Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour)}},
		cacheWriteChan: make(chan cacheWriteTask, 1),
		cfg:            &config.Config{},
	}
	svc.cacheWriteChan <- cacheWriteTask{kind: cacheWriteSetSubscription}

	data, err := svc.GetSubscriptionStatus(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, data.Status)
	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) >= 1
	}, time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceQueueUpdateAPIKeyRateLimitUsageFallbackWhenQueueClosed(t *testing.T) {
	cache := &billingCacheFallbackStub{}
	svc := &BillingCacheService{
		cache:          cache,
		cacheWriteChan: make(chan cacheWriteTask, 1),
		cfg:            &config.Config{},
	}
	svc.Stop()

	svc.QueueUpdateAPIKeyRateLimitUsage(99, 3.5)
	require.Eventually(t, func() bool {
		return cache.rateLimitUpdates.Load() >= 1
	}, time.Second, 10*time.Millisecond)
}

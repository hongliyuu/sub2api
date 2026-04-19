package service

import "context"

type balanceUpdateContextKey string

const skipTotalRechargedTrackingKey balanceUpdateContextKey = "skip_total_recharged_tracking"

// WithSkipTotalRechargedTracking marks a balance update as a non-recharge credit.
func WithSkipTotalRechargedTracking(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipTotalRechargedTrackingKey, true)
}

// ShouldTrackTotalRecharged reports whether positive balance updates should
// contribute to the cumulative recharge counter.
func ShouldTrackTotalRecharged(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	skip, _ := ctx.Value(skipTotalRechargedTrackingKey).(bool)
	return !skip
}

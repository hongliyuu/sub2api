package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/lotterycoupon"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type lotteryCouponRepository struct {
	client *dbent.Client
}

func NewLotteryCouponRepository(client *dbent.Client) service.LotteryCouponRepository {
	return &lotteryCouponRepository{client: client}
}

func (r *lotteryCouponRepository) Create(ctx context.Context, coupon *service.LotteryCoupon) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryCoupon.Create().
		SetActivityID(coupon.ActivityID).
		SetUserID(coupon.UserID).
		SetCouponType(coupon.CouponType).
		SetApplicableScope(coupon.ApplicableScope).
		SetStatus(coupon.Status).
		SetExpiresAt(coupon.ExpiresAt).
		SetReminded(coupon.Reminded)

	if coupon.DiscountPercent != nil {
		builder.SetDiscountPercent(*coupon.DiscountPercent)
	}
	if coupon.ReductionAmount != nil {
		builder.SetReductionAmount(*coupon.ReductionAmount)
	}
	if coupon.UsedAt != nil {
		builder.SetUsedAt(*coupon.UsedAt)
	}
	if coupon.UsedOrderID != nil {
		builder.SetUsedOrderID(*coupon.UsedOrderID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}

	coupon.ID = created.ID
	coupon.CreatedAt = created.CreatedAt
	return nil
}

func (r *lotteryCouponRepository) GetByID(ctx context.Context, id int64) (*service.LotteryCoupon, error) {
	m, err := r.client.LotteryCoupon.Query().
		Where(lotterycoupon.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrLotteryCouponNotFound
		}
		return nil, err
	}
	return lotteryCouponEntityToService(m), nil
}

func (r *lotteryCouponRepository) ListByUser(ctx context.Context, userID int64, status string, params pagination.PaginationParams) ([]service.LotteryCoupon, *pagination.PaginationResult, error) {
	q := r.client.LotteryCoupon.Query().
		Where(lotterycoupon.UserIDEQ(userID))

	if status != "" {
		q = q.Where(lotterycoupon.StatusEQ(status))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	coupons, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(lotterycoupon.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return lotteryCouponEntitiesToService(coupons), paginationResultFromTotal(int64(total), params), nil
}

func (r *lotteryCouponRepository) ListByActivity(ctx context.Context, activityID int64) ([]service.LotteryCoupon, error) {
	coupons, err := r.client.LotteryCoupon.Query().
		Where(lotterycoupon.ActivityIDEQ(activityID)).
		Order(dbent.Desc(lotterycoupon.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return lotteryCouponEntitiesToService(coupons), nil
}

func (r *lotteryCouponRepository) MarkUsed(ctx context.Context, id int64, orderID string) error {
	client := clientFromContext(ctx, r.client)
	now := time.Now()
	// 使用 WHERE status='active' 条件防止并发双重使用
	affected, err := client.LotteryCoupon.Update().
		Where(
			lotterycoupon.IDEQ(id),
			lotterycoupon.StatusEQ("active"),
		).
		SetStatus("used").
		SetUsedAt(now).
		SetUsedOrderID(orderID).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrLotteryCouponNotFound
	}
	return nil
}

func (r *lotteryCouponRepository) ReleaseByOrderID(ctx context.Context, orderID string) error {
	client := clientFromContext(ctx, r.client)
	// 将该订单占用的优惠券释放回 active 状态（仅当当前状态为 used 且未过期时）
	_, err := client.LotteryCoupon.Update().
		Where(
			lotterycoupon.UsedOrderIDEQ(orderID),
			lotterycoupon.StatusEQ("used"),
		).
		SetStatus("active").
		ClearUsedAt().
		ClearUsedOrderID().
		Save(ctx)
	return err
}

func (r *lotteryCouponRepository) ExpireByActivity(ctx context.Context, activityID int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.LotteryCoupon.Update().
		Where(
			lotterycoupon.ActivityIDEQ(activityID),
			lotterycoupon.StatusEQ("active"),
		).
		SetStatus("expired").
		Save(ctx)
	return err
}

func (r *lotteryCouponRepository) ListUnremindedExpiringSoon(ctx context.Context, withinHours int) ([]service.LotteryCoupon, error) {
	now := time.Now()
	expiryThreshold := now.Add(time.Duration(withinHours) * time.Hour)

	coupons, err := r.client.LotteryCoupon.Query().
		Where(
			lotterycoupon.StatusEQ("active"),
			lotterycoupon.RemindedEQ(false),
			lotterycoupon.ExpiresAtLTE(expiryThreshold),
			lotterycoupon.ExpiresAtGT(now),
		).
		Order(dbent.Asc(lotterycoupon.FieldExpiresAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return lotteryCouponEntitiesToService(coupons), nil
}

func (r *lotteryCouponRepository) MarkReminded(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.LotteryCoupon.UpdateOneID(id).
		SetReminded(true).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrLotteryCouponNotFound
		}
		return err
	}
	return nil
}

func (r *lotteryCouponRepository) FindActiveForUser(ctx context.Context, userID int64, scope string) (*service.LotteryCoupon, error) {
	now := time.Now()
	m, err := r.client.LotteryCoupon.Query().
		Where(
			lotterycoupon.UserIDEQ(userID),
			lotterycoupon.StatusEQ("active"),
			lotterycoupon.ApplicableScopeEQ(scope),
			lotterycoupon.ExpiresAtGT(now),
		).
		Order(dbent.Desc(lotterycoupon.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return lotteryCouponEntityToService(m), nil
}

// Entity to Service conversions

func lotteryCouponEntityToService(m *dbent.LotteryCoupon) *service.LotteryCoupon {
	if m == nil {
		return nil
	}
	return &service.LotteryCoupon{
		ID:              m.ID,
		ActivityID:      m.ActivityID,
		UserID:          m.UserID,
		CouponType:      m.CouponType,
		DiscountPercent: m.DiscountPercent,
		ReductionAmount: m.ReductionAmount,
		ApplicableScope: m.ApplicableScope,
		Status:          m.Status,
		UsedAt:          m.UsedAt,
		UsedOrderID:     m.UsedOrderID,
		ExpiresAt:       m.ExpiresAt,
		Reminded:        m.Reminded,
		CreatedAt:       m.CreatedAt,
	}
}

func lotteryCouponEntitiesToService(models []*dbent.LotteryCoupon) []service.LotteryCoupon {
	out := make([]service.LotteryCoupon, 0, len(models))
	for i := range models {
		if s := lotteryCouponEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

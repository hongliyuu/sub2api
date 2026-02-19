package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/lotteryparticipant"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type lotteryParticipantRepository struct {
	client *dbent.Client
}

func NewLotteryParticipantRepository(client *dbent.Client) service.LotteryParticipantRepository {
	return &lotteryParticipantRepository{client: client}
}

func (r *lotteryParticipantRepository) Create(ctx context.Context, p *service.LotteryParticipant) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryParticipant.Create().
		SetActivityID(p.ActivityID).
		SetUserID(p.UserID).
		SetUserCategory(p.UserCategory).
		SetWeightMultiplier(p.WeightMultiplier)

	if p.IsWinner != nil {
		builder.SetIsWinner(*p.IsWinner)
	}
	if p.CouponID != nil {
		builder.SetCouponID(*p.CouponID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return service.ErrLotteryAlreadyParticipated
		}
		return err
	}

	p.ID = created.ID
	p.ParticipatedAt = created.ParticipatedAt
	return nil
}

func (r *lotteryParticipantRepository) GetByActivityAndUser(ctx context.Context, activityID, userID int64) (*service.LotteryParticipant, error) {
	m, err := r.client.LotteryParticipant.Query().
		Where(
			lotteryparticipant.ActivityIDEQ(activityID),
			lotteryparticipant.UserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return lotteryParticipantEntityToService(m), nil
}

func (r *lotteryParticipantRepository) ListByActivity(ctx context.Context, activityID int64, params pagination.PaginationParams) ([]service.LotteryParticipant, *pagination.PaginationResult, error) {
	q := r.client.LotteryParticipant.Query().
		Where(lotteryparticipant.ActivityIDEQ(activityID))

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	participants, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(lotteryparticipant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return lotteryParticipantEntitiesToService(participants), paginationResultFromTotal(int64(total), params), nil
}

func (r *lotteryParticipantRepository) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.LotteryParticipant, *pagination.PaginationResult, error) {
	q := r.client.LotteryParticipant.Query().
		Where(lotteryparticipant.UserIDEQ(userID))

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	participants, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(lotteryparticipant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return lotteryParticipantEntitiesToService(participants), paginationResultFromTotal(int64(total), params), nil
}

func (r *lotteryParticipantRepository) ListAllByActivity(ctx context.Context, activityID int64) ([]service.LotteryParticipant, error) {
	participants, err := r.client.LotteryParticipant.Query().
		Where(lotteryparticipant.ActivityIDEQ(activityID)).
		Order(dbent.Asc(lotteryparticipant.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return lotteryParticipantEntitiesToService(participants), nil
}

func (r *lotteryParticipantRepository) UpdateWinnerStatus(ctx context.Context, id int64, isWinner bool, couponID *int64) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryParticipant.UpdateOneID(id).
		SetIsWinner(isWinner)

	if couponID != nil {
		builder.SetCouponID(*couponID)
	} else {
		builder.ClearCouponID()
	}

	_, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrLotteryParticipantNotFound
		}
		return err
	}
	return nil
}

// Entity to Service conversions

func lotteryParticipantEntityToService(m *dbent.LotteryParticipant) *service.LotteryParticipant {
	if m == nil {
		return nil
	}
	return &service.LotteryParticipant{
		ID:               m.ID,
		ActivityID:       m.ActivityID,
		UserID:           m.UserID,
		UserCategory:     m.UserCategory,
		WeightMultiplier: m.WeightMultiplier,
		IsWinner:         m.IsWinner,
		CouponID:         m.CouponID,
		ParticipatedAt:   m.ParticipatedAt,
	}
}

func lotteryParticipantEntitiesToService(models []*dbent.LotteryParticipant) []service.LotteryParticipant {
	out := make([]service.LotteryParticipant, 0, len(models))
	for i := range models {
		if s := lotteryParticipantEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

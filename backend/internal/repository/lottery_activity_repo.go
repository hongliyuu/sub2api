package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/lotteryactivity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type lotteryActivityRepository struct {
	client *dbent.Client
}

func NewLotteryActivityRepository(client *dbent.Client) service.LotteryActivityRepository {
	return &lotteryActivityRepository{client: client}
}

func (r *lotteryActivityRepository) Create(ctx context.Context, activity *service.LotteryActivity) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryActivity.Create().
		SetTitle(activity.Title).
		SetShareCode(activity.ShareCode).
		SetStatus(activity.Status).
		SetDrawAt(activity.DrawAt).
		SetActivityStartAt(activity.ActivityStartAt).
		SetActivityEndAt(activity.ActivityEndAt).
		SetMinParticipants(activity.MinParticipants).
		SetBaseWinRate(activity.BaseWinRate).
		SetWeightConfig(activity.WeightConfig).
		SetWinnerDiscountPercent(activity.WinnerDiscountPercent).
		SetLoserCouponAmount(activity.LoserCouponAmount).
		SetParticipantCount(activity.ParticipantCount).
		SetWinnerCount(activity.WinnerCount).
		SetCreatedBy(activity.CreatedBy)

	if activity.Description != "" {
		builder.SetDescription(activity.Description)
	}
	if activity.TempGroupID != nil {
		builder.SetTempGroupID(*activity.TempGroupID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}

	activity.ID = created.ID
	activity.CreatedAt = created.CreatedAt
	activity.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *lotteryActivityRepository) GetByID(ctx context.Context, id int64) (*service.LotteryActivity, error) {
	m, err := r.client.LotteryActivity.Query().
		Where(lotteryactivity.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrLotteryActivityNotFound
		}
		return nil, err
	}
	return lotteryActivityEntityToService(m), nil
}

func (r *lotteryActivityRepository) GetByShareCode(ctx context.Context, code string) (*service.LotteryActivity, error) {
	m, err := r.client.LotteryActivity.Query().
		Where(lotteryactivity.ShareCodeEQ(code)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrLotteryActivityNotFound
		}
		return nil, err
	}
	return lotteryActivityEntityToService(m), nil
}

func (r *lotteryActivityRepository) Update(ctx context.Context, activity *service.LotteryActivity) error {
	client := clientFromContext(ctx, r.client)
	builder := client.LotteryActivity.UpdateOneID(activity.ID).
		SetTitle(activity.Title).
		SetShareCode(activity.ShareCode).
		SetStatus(activity.Status).
		SetDrawAt(activity.DrawAt).
		SetActivityStartAt(activity.ActivityStartAt).
		SetActivityEndAt(activity.ActivityEndAt).
		SetMinParticipants(activity.MinParticipants).
		SetBaseWinRate(activity.BaseWinRate).
		SetWeightConfig(activity.WeightConfig).
		SetWinnerDiscountPercent(activity.WinnerDiscountPercent).
		SetLoserCouponAmount(activity.LoserCouponAmount).
		SetParticipantCount(activity.ParticipantCount).
		SetWinnerCount(activity.WinnerCount).
		SetCreatedBy(activity.CreatedBy)

	if activity.Description != "" {
		builder.SetDescription(activity.Description)
	} else {
		builder.ClearDescription()
	}
	if activity.TempGroupID != nil {
		builder.SetTempGroupID(*activity.TempGroupID)
	} else {
		builder.ClearTempGroupID()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrLotteryActivityNotFound
		}
		return err
	}

	activity.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *lotteryActivityRepository) List(ctx context.Context, params pagination.PaginationParams, status string) ([]service.LotteryActivity, *pagination.PaginationResult, error) {
	q := r.client.LotteryActivity.Query()

	if status != "" {
		q = q.Where(lotteryactivity.StatusEQ(status))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	activities, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(lotteryactivity.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return lotteryActivityEntitiesToService(activities), paginationResultFromTotal(int64(total), params), nil
}

func (r *lotteryActivityRepository) ListByStatus(ctx context.Context, status string) ([]service.LotteryActivity, error) {
	activities, err := r.client.LotteryActivity.Query().
		Where(lotteryactivity.StatusEQ(status)).
		Order(dbent.Desc(lotteryactivity.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return lotteryActivityEntitiesToService(activities), nil
}

func (r *lotteryActivityRepository) ListActiveForDraw(ctx context.Context) ([]service.LotteryActivity, error) {
	now := time.Now()
	activities, err := r.client.LotteryActivity.Query().
		Where(
			lotteryactivity.StatusEQ("active"),
			lotteryactivity.DrawAtLTE(now),
		).
		Order(dbent.Asc(lotteryactivity.FieldDrawAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return lotteryActivityEntitiesToService(activities), nil
}

func (r *lotteryActivityRepository) ListExpired(ctx context.Context) ([]service.LotteryActivity, error) {
	now := time.Now()
	activities, err := r.client.LotteryActivity.Query().
		Where(
			lotteryactivity.StatusIn("completed", "cancelled"),
			lotteryactivity.ActivityEndAtLTE(now),
		).
		Order(dbent.Asc(lotteryactivity.FieldActivityEndAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return lotteryActivityEntitiesToService(activities), nil
}

func (r *lotteryActivityRepository) IncrementParticipantCount(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.LotteryActivity.UpdateOneID(id).
		AddParticipantCount(1).
		Save(ctx)
	return err
}

func (r *lotteryActivityRepository) UpdateDrawResult(ctx context.Context, id int64, winnerCount int, activityEndAt time.Time) error {
	client := clientFromContext(ctx, r.client)

	_, err := client.LotteryActivity.UpdateOneID(id).
		SetStatus("completed").
		SetWinnerCount(winnerCount).
		SetActivityEndAt(activityEndAt).
		Save(ctx)
	return err
}

// Entity to Service conversions

func lotteryActivityEntityToService(m *dbent.LotteryActivity) *service.LotteryActivity {
	if m == nil {
		return nil
	}
	return &service.LotteryActivity{
		ID:                    m.ID,
		Title:                 m.Title,
		Description:           derefString(m.Description),
		ShareCode:             m.ShareCode,
		Status:                m.Status,
		DrawAt:                m.DrawAt,
		ActivityStartAt:       m.ActivityStartAt,
		ActivityEndAt:         m.ActivityEndAt,
		MinParticipants:       m.MinParticipants,
		BaseWinRate:           m.BaseWinRate,
		WeightConfig:          m.WeightConfig,
		WinnerDiscountPercent: m.WinnerDiscountPercent,
		LoserCouponAmount:     m.LoserCouponAmount,
		TempGroupID:           m.TempGroupID,
		ParticipantCount:      m.ParticipantCount,
		WinnerCount:           m.WinnerCount,
		CreatedBy:             m.CreatedBy,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}

func lotteryActivityEntitiesToService(models []*dbent.LotteryActivity) []service.LotteryActivity {
	out := make([]service.LotteryActivity, 0, len(models))
	for i := range models {
		if s := lotteryActivityEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

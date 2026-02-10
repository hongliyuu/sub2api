package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/balancelot"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type balanceLotRepository struct {
	client *dbent.Client
}

// NewBalanceLotRepository 创建余额批次仓储
func NewBalanceLotRepository(client *dbent.Client) service.BalanceLotRepository {
	return &balanceLotRepository{client: client}
}

func (r *balanceLotRepository) Create(ctx context.Context, lot *service.BalanceLot) error {
	client := clientFromContext(ctx, r.client)
	builder := client.BalanceLot.Create().
		SetUserID(lot.UserID).
		SetSourceType(lot.SourceType).
		SetOriginalAmount(lot.OriginalAmount).
		SetRemainingAmount(lot.RemainingAmount).
		SetStatus(lot.Status).
		SetExpiresAt(lot.ExpiresAt).
		SetDescription(lot.Description)

	if lot.SourceRef != nil {
		builder.SetSourceRef(*lot.SourceRef)
	}
	if lot.ExpiredAt != nil {
		builder.SetExpiredAt(*lot.ExpiredAt)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	lot.ID = created.ID
	lot.CreatedAt = created.CreatedAt
	lot.UpdatedAt = created.UpdatedAt
	return nil
}

// ListActiveByUserForUpdate 获取用户活跃批次并加行锁（FOR UPDATE），按过期时间升序
// 使用 Ent client 的 QueryContext 执行原生 SQL，在事务中自动使用事务连接
func (r *balanceLotRepository) ListActiveByUserForUpdate(ctx context.Context, userID int64) ([]*service.BalanceLot, error) {
	query := `
		SELECT id, user_id, source_type, source_ref, original_amount, remaining_amount,
		       status, expires_at, expired_at, description, created_at, updated_at
		FROM balance_lots
		WHERE user_id = $1 AND status = 'active' AND remaining_amount > 0
		ORDER BY expires_at ASC
		FOR UPDATE
	`

	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list active lots for update: %w", err)
	}
	defer rows.Close()

	return r.scanBalanceLots(rows)
}

func (r *balanceLotRepository) UpdateRemainingAndStatus(ctx context.Context, lotID int64, remaining float64, status string) error {
	client := clientFromContext(ctx, r.client)
	n, err := client.BalanceLot.Update().
		Where(balancelot.IDEQ(lotID)).
		SetRemainingAmount(remaining).
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("balance lot %d not found", lotID)
	}
	return nil
}

// BatchExpire 批量过期处理
// 查询 status=active 且 expires_at < now 的批次，将其标记为 expired
func (r *balanceLotRepository) BatchExpire(ctx context.Context, batchSize int) ([]*service.BalanceLot, error) {
	now := time.Now()

	// 先查询待过期的批次
	lots, err := r.client.BalanceLot.Query().
		Where(
			balancelot.StatusEQ(service.BalanceLotStatusActive),
			balancelot.ExpiresAtLT(now),
			balancelot.RemainingAmountGT(0),
		).
		Order(dbent.Asc(balancelot.FieldExpiresAt)).
		Limit(batchSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query expired lots: %w", err)
	}

	if len(lots) == 0 {
		return nil, nil
	}

	result := make([]*service.BalanceLot, 0, len(lots))
	for _, lot := range lots {
		result = append(result, r.toServiceModel(lot))
	}

	return result, nil
}

func (r *balanceLotRepository) SumActiveRemaining(ctx context.Context, userID int64) (float64, error) {
	var sum float64
	query := `SELECT COALESCE(SUM(remaining_amount), 0) FROM balance_lots WHERE user_id = $1 AND status = 'active'`

	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("sum active remaining: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&sum); err != nil {
			return 0, fmt.Errorf("scan sum: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return sum, nil
}

func (r *balanceLotRepository) ListByUser(ctx context.Context, userID int64, status string, page, pageSize int) ([]*service.BalanceLot, int64, error) {
	predicates := []predicate.BalanceLot{balancelot.UserIDEQ(userID)}
	if status != "" {
		predicates = append(predicates, balancelot.StatusEQ(status))
	}

	total, err := r.client.BalanceLot.Query().
		Where(predicates...).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count user lots: %w", err)
	}

	offset := (page - 1) * pageSize
	lots, err := r.client.BalanceLot.Query().
		Where(predicates...).
		Order(dbent.Desc(balancelot.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list user lots: %w", err)
	}

	result := make([]*service.BalanceLot, 0, len(lots))
	for _, lot := range lots {
		result = append(result, r.toServiceModel(lot))
	}

	return result, int64(total), nil
}

func (r *balanceLotRepository) ListExpiringSoon(ctx context.Context, advanceDays int) ([]*service.BalanceLot, error) {
	now := time.Now()
	deadline := now.Add(time.Duration(advanceDays) * 24 * time.Hour)

	lots, err := r.client.BalanceLot.Query().
		Where(
			balancelot.StatusEQ(service.BalanceLotStatusActive),
			balancelot.RemainingAmountGT(0),
			balancelot.ExpiresAtGT(now),
			balancelot.ExpiresAtLTE(deadline),
		).
		Order(dbent.Asc(balancelot.FieldExpiresAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expiring soon lots: %w", err)
	}

	result := make([]*service.BalanceLot, 0, len(lots))
	for _, lot := range lots {
		result = append(result, r.toServiceModel(lot))
	}

	return result, nil
}

func (r *balanceLotRepository) ListExpiringSoonByUser(ctx context.Context, userID int64, advanceDays int) ([]*service.BalanceLot, error) {
	now := time.Now()
	deadline := now.Add(time.Duration(advanceDays) * 24 * time.Hour)

	lots, err := r.client.BalanceLot.Query().
		Where(
			balancelot.UserIDEQ(userID),
			balancelot.StatusEQ(service.BalanceLotStatusActive),
			balancelot.RemainingAmountGT(0),
			balancelot.ExpiresAtGT(now),
			balancelot.ExpiresAtLTE(deadline),
		).
		Order(dbent.Asc(balancelot.FieldExpiresAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list expiring soon lots by user: %w", err)
	}

	result := make([]*service.BalanceLot, 0, len(lots))
	for _, lot := range lots {
		result = append(result, r.toServiceModel(lot))
	}

	return result, nil
}

func (r *balanceLotRepository) CountByUser(ctx context.Context, userID int64) (int64, error) {
	count, err := r.client.BalanceLot.Query().
		Where(balancelot.UserIDEQ(userID)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count by user: %w", err)
	}
	return int64(count), nil
}

// scanBalanceLots 扫描余额批次行
func (r *balanceLotRepository) scanBalanceLots(rows *sql.Rows) ([]*service.BalanceLot, error) {
	var result []*service.BalanceLot
	for rows.Next() {
		lot := &service.BalanceLot{}
		var sourceRef sql.NullString
		var expiredAt sql.NullTime

		err := rows.Scan(
			&lot.ID,
			&lot.UserID,
			&lot.SourceType,
			&sourceRef,
			&lot.OriginalAmount,
			&lot.RemainingAmount,
			&lot.Status,
			&lot.ExpiresAt,
			&expiredAt,
			&lot.Description,
			&lot.CreatedAt,
			&lot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan balance lot: %w", err)
		}

		if sourceRef.Valid {
			lot.SourceRef = &sourceRef.String
		}
		if expiredAt.Valid {
			lot.ExpiredAt = &expiredAt.Time
		}

		result = append(result, lot)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// toServiceModel 将 Ent 模型转换为 Service 模型
func (r *balanceLotRepository) toServiceModel(lot *dbent.BalanceLot) *service.BalanceLot {
	result := &service.BalanceLot{
		ID:              lot.ID,
		UserID:          lot.UserID,
		SourceType:      lot.SourceType,
		OriginalAmount:  lot.OriginalAmount,
		RemainingAmount: lot.RemainingAmount,
		Status:          lot.Status,
		ExpiresAt:       lot.ExpiresAt,
		Description:     lot.Description,
		CreatedAt:       lot.CreatedAt,
		UpdatedAt:       lot.UpdatedAt,
	}
	if lot.SourceRef != nil {
		result.SourceRef = lot.SourceRef
	}
	if lot.ExpiredAt != nil {
		result.ExpiredAt = lot.ExpiredAt
	}
	return result
}

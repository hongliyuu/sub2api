// backend/internal/repository/request_log_repo.go
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const maxRequestBodyBytes = 1024 * 1024 // 1 MB

type requestLogRepository struct {
	db *sql.DB
}

// NewRequestLogRepository creates a repository backed by a raw *sql.DB.
func NewRequestLogRepository(db *sql.DB) service.RequestLogRepository {
	return &requestLogRepository{db: db}
}

// truncateBody truncates body to maxRequestBodyBytes and replaces it with a JSON truncation marker.
func truncateBody(b []byte) []byte {
	if len(b) <= maxRequestBodyBytes {
		return b
	}
	marker := []byte(fmt.Sprintf(`{"_truncated":true,"_original_size":%d}`, len(b)))
	return marker
}

func (r *requestLogRepository) Save(ctx context.Context, input *service.RequestLogInput) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil request log repository")
	}

	// For JSONB columns, lib/pq requires either a string with ::jsonb cast or json.RawMessage.
	// We use sql.NullString with explicit ::jsonb cast in SQL to handle NULL cleanly.
	toJSONBArg := func(b []byte) sql.NullString {
		if b == nil {
			return sql.NullString{}
		}
		truncated := truncateBody(b)
		if !json.Valid(truncated) {
			return sql.NullString{}
		}
		return sql.NullString{String: string(truncated), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, `
INSERT INTO request_logs
  (request_id, usage_log_id, user_id, api_key_id, account_id, group_id,
   anomaly_types, request_body, upstream_request_body, upstream_response_body, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7,
        CASE WHEN $8::text IS NOT NULL THEN $8::jsonb END,
        CASE WHEN $9::text IS NOT NULL THEN $9::jsonb END,
        CASE WHEN $10::text IS NOT NULL THEN $10::jsonb END,
        $11)`,
		input.RequestID,
		toNullInt64(input.UsageLogID),
		toNullInt64(input.UserID),
		toNullInt64(input.APIKeyID),
		toNullInt64(input.AccountID),
		toNullInt64(input.GroupID),
		pq.Array(input.AnomalyTypes),
		toJSONBArg(input.RequestBody),
		toJSONBArg(input.UpstreamRequestBody),
		toJSONBArg(input.UpstreamResponseBody),
		time.Now().UTC(),
	)
	return err
}

func (r *requestLogRepository) GetByRequestID(ctx context.Context, requestID string) (*service.RequestLogData, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil request log repository")
	}
	row := r.db.QueryRowContext(ctx, `
SELECT request_id, anomaly_types, request_body, upstream_request_body, upstream_response_body
FROM request_logs
WHERE request_id = $1
ORDER BY created_at DESC
LIMIT 1`, requestID)

	var out service.RequestLogData
	var anomalyTypes pq.StringArray
	var reqBody, upReqBody, upRespBody []byte

	if err := row.Scan(
		&out.RequestID,
		&anomalyTypes,
		&reqBody,
		&upReqBody,
		&upRespBody,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	out.AnomalyTypes = []string(anomalyTypes)
	if len(reqBody) > 0 {
		out.RequestBody = make([]byte, len(reqBody))
		copy(out.RequestBody, reqBody)
	}
	if len(upReqBody) > 0 {
		out.UpstreamRequestBody = make([]byte, len(upReqBody))
		copy(out.UpstreamRequestBody, upReqBody)
	}
	if len(upRespBody) > 0 {
		out.UpstreamResponseBody = make([]byte, len(upRespBody))
		copy(out.UpstreamResponseBody, upRespBody)
	}
	return &out, nil
}

// toNullInt64 converts *int64 to sql.NullInt64.
func toNullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

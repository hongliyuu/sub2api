CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ops_error_logs_created_at_id
    ON ops_error_logs (created_at ASC, id ASC);

DROP INDEX CONCURRENTLY IF EXISTS idx_ops_error_logs_created_at;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_created_at_id
    ON usage_logs (created_at ASC, id ASC);

DROP INDEX CONCURRENTLY IF EXISTS idx_usage_logs_created_at;

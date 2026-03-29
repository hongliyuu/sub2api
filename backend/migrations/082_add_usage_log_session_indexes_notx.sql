-- 082_add_usage_log_session_indexes_notx.sql
-- Indexes for session-based queries (conversation tracing) and request correlation.
-- Uses CONCURRENTLY to avoid blocking reads/writes on the usage_logs table.

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_session_hash
    ON usage_logs (session_hash)
    WHERE session_hash IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_client_request_id
    ON usage_logs (client_request_id)
    WHERE client_request_id IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_session_hash_created_at
    ON usage_logs (session_hash, created_at DESC)
    WHERE session_hash IS NOT NULL;

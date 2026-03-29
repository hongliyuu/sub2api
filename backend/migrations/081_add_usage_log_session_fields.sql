-- 081_add_usage_log_session_fields.sql
-- Add session correlation and platform fields to usage_logs for conversation-level tracing.

ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS session_hash TEXT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS client_request_id TEXT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS platform VARCHAR(20);

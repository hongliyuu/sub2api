-- 082_add_spans_to_logs.sql
-- Adds spans JSONB column to both error_logs and usage_logs for
-- request diagnosis and tracing. Spans store raw per-phase timing events;
-- fault_owner and diagnosis are computed at query time from this data.

ALTER TABLE ops_error_logs
  ADD COLUMN IF NOT EXISTS spans JSONB;

ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS spans JSONB;

COMMENT ON COLUMN ops_error_logs.spans IS
  'Per-phase span events as JSON array. Each span: {name, start_unix_ms, duration_ms, status, attrs}.';

COMMENT ON COLUMN usage_logs.spans IS
  'Per-phase span events as JSON array. Each span: {name, start_unix_ms, duration_ms, status, attrs}.';

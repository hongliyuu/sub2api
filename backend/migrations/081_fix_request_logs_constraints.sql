-- Fix migration: update request_logs constraints and indexes added by migration 080
-- Adds data integrity constraints and improves index design

-- Drop old single-column identity indexes
DROP INDEX IF EXISTS idx_request_logs_user_id;
DROP INDEX IF EXISTS idx_request_logs_api_key_id;

-- Add composite indexes with created_at for time-scoped queries
CREATE INDEX IF NOT EXISTS idx_request_logs_user_id    ON request_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_api_key_id ON request_logs(api_key_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_account_id ON request_logs(account_id, created_at DESC);

-- Add data integrity constraints
ALTER TABLE request_logs
    ALTER COLUMN anomaly_types DROP DEFAULT,
    ADD CONSTRAINT request_logs_anomaly_types_nonempty
        CHECK (cardinality(anomaly_types) > 0),
    ADD CONSTRAINT request_logs_anomaly_types_valid
        CHECK (anomaly_types <@ ARRAY['zero_token','slow_request','timeout','error']::text[]);

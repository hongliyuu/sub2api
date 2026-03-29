-- Fix migration: update request_logs to improve constraints and indexes
-- Applies fixes to environments that ran the original migration 080.

-- Recreate identity indexes as composites with created_at for time-scoped queries
DROP INDEX IF EXISTS idx_request_logs_user_id;
DROP INDEX IF EXISTS idx_request_logs_api_key_id;

CREATE INDEX IF NOT EXISTS idx_request_logs_user_id    ON request_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_api_key_id ON request_logs(api_key_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_account_id ON request_logs(account_id, created_at DESC);

-- Drop the permissive default (anomaly_types must always be provided explicitly)
ALTER TABLE request_logs ALTER COLUMN anomaly_types DROP DEFAULT;

-- Add data integrity constraints (conditional to avoid error if already present from updated 080)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'request_logs_anomaly_types_nonempty'
          AND conrelid = 'request_logs'::regclass
    ) THEN
        ALTER TABLE request_logs
            ADD CONSTRAINT request_logs_anomaly_types_nonempty
                CHECK (cardinality(anomaly_types) > 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'request_logs_anomaly_types_valid'
          AND conrelid = 'request_logs'::regclass
    ) THEN
        ALTER TABLE request_logs
            ADD CONSTRAINT request_logs_anomaly_types_valid
                CHECK (anomaly_types <@ ARRAY['zero_token','slow_request','timeout','error']::text[]);
    END IF;
END
$$;

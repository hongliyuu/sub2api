-- Group-level daily actual_cost pre-aggregation for admin group usage summary.
-- This keeps cumulative per-group cost queries off the hot usage_logs path.

CREATE TABLE IF NOT EXISTS usage_group_daily_costs (
    bucket_date DATE NOT NULL,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    actual_cost DECIMAL(20, 10) NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bucket_date, group_id)
);

CREATE INDEX IF NOT EXISTS idx_usage_group_daily_costs_group_id_bucket_date
    ON usage_group_daily_costs (group_id, bucket_date DESC);

COMMENT ON TABLE usage_group_daily_costs IS 'Pre-aggregated daily actual_cost by group for fast admin group usage summary queries.';
COMMENT ON COLUMN usage_group_daily_costs.bucket_date IS 'UTC date bucket derived from usage_logs.created_at.';
COMMENT ON COLUMN usage_group_daily_costs.computed_at IS 'When the group/day row was last computed or refreshed.';

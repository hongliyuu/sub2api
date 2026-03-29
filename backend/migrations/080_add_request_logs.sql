-- 添加 request_logs 表，用于存储触发异常检测的请求原始数据
-- anomaly_types 字段记录触发的异常类型，可选值: 'zero_token', 'slow_request', 'timeout', 'error'
-- Requires PostgreSQL >= 12 for FK to partitioned usage_logs table.
CREATE TABLE IF NOT EXISTS request_logs (
    id                     BIGSERIAL    PRIMARY KEY,
    request_id             VARCHAR(64)  NOT NULL,
    usage_log_id           BIGINT       REFERENCES usage_logs(id) ON DELETE SET NULL,

    user_id                BIGINT,
    api_key_id             BIGINT,
    account_id             BIGINT,
    group_id               BIGINT,

    anomaly_types          TEXT[]       NOT NULL,

    request_body           JSONB,
    upstream_request_body  JSONB,
    upstream_response_body JSONB,

    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT request_logs_anomaly_types_nonempty
        CHECK (cardinality(anomaly_types) > 0),
    CONSTRAINT request_logs_anomaly_types_valid
        CHECK (anomaly_types <@ ARRAY['zero_token','slow_request','timeout','error']::text[])
);

CREATE INDEX IF NOT EXISTS idx_request_logs_request_id ON request_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_user_id    ON request_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_api_key_id ON request_logs(api_key_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_request_logs_account_id ON request_logs(account_id, created_at DESC);

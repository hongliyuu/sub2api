CREATE TABLE IF NOT EXISTS user_provider_default_grants (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_type VARCHAR(20) NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_provider_default_grants_provider_type_check
        CHECK (provider_type IN ('linuxdo', 'wechat', 'oidc'))
);

CREATE UNIQUE INDEX IF NOT EXISTS user_provider_default_grants_user_provider_key
    ON user_provider_default_grants (user_id, provider_type);

CREATE INDEX IF NOT EXISTS user_provider_default_grants_user_id_idx
    ON user_provider_default_grants (user_id);

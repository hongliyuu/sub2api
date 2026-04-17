-- Add additive profile tables for third-party user bindings and avatar metadata.
-- Existing users, logins, and payments remain unchanged because all new data is
-- stored in separate tables keyed by user_id.

CREATE TABLE IF NOT EXISTS user_external_identities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_union_id VARCHAR(255),
    provider_username VARCHAR(255) NOT NULL DEFAULT '',
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    profile_url TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_external_identities_provider_check
        CHECK (provider IN ('linuxdo', 'wechat'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_external_identities_user_provider
    ON user_external_identities (user_id, provider);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_external_identities_provider_user
    ON user_external_identities (provider, provider_user_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_external_identities_provider_union
    ON user_external_identities (provider, provider_union_id)
    WHERE provider_union_id IS NOT NULL AND provider_union_id <> '';

CREATE INDEX IF NOT EXISTS idx_user_external_identities_user_id
    ON user_external_identities (user_id);

COMMENT ON TABLE user_external_identities IS 'Per-user third-party identity bindings for login/connect providers such as LinuxDo and WeChat.';
COMMENT ON COLUMN user_external_identities.provider IS 'Supported provider key: linuxdo or wechat.';
COMMENT ON COLUMN user_external_identities.provider_user_id IS 'Stable provider subject/openid scoped to the provider.';
COMMENT ON COLUMN user_external_identities.provider_union_id IS 'Optional provider-wide union identifier when available (for example WeChat unionid).';
COMMENT ON COLUMN user_external_identities.metadata IS 'Provider-specific JSON metadata retained for future auth/profile integrations.';

CREATE TABLE IF NOT EXISTS user_avatars (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    storage_provider VARCHAR(64) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    content_type VARCHAR(128) NOT NULL,
    byte_size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL DEFAULT '',
    width INTEGER,
    height INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_avatars_byte_size_check
        CHECK (byte_size >= 0 AND byte_size <= 102400)
);

COMMENT ON TABLE user_avatars IS 'Metadata for the currently active user avatar object stored in an external or local storage backend.';
COMMENT ON COLUMN user_avatars.byte_size IS 'Avatar object size in bytes. Enforced at the DB layer with a 100 KB upper bound.';
COMMENT ON COLUMN user_avatars.storage_key IS 'Opaque storage object key/path used by the backing avatar storage provider.';

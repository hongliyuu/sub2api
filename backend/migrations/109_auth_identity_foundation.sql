ALTER TABLE users
ADD COLUMN IF NOT EXISTS signup_source VARCHAR(20) NOT NULL DEFAULT 'email';

UPDATE users
SET signup_source = 'email'
WHERE signup_source IS NULL OR signup_source = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_signup_source_check'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_signup_source_check
            CHECK (signup_source IN ('email', 'linuxdo', 'wechat', 'oidc'));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS auth_identities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_type VARCHAR(20) NOT NULL,
    provider_key TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    verified_at TIMESTAMPTZ NULL,
    issuer TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT auth_identities_provider_type_check
        CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc'))
);

CREATE UNIQUE INDEX IF NOT EXISTS auth_identities_provider_subject_key
    ON auth_identities (provider_type, provider_key, provider_subject);

CREATE INDEX IF NOT EXISTS auth_identities_user_id_idx
    ON auth_identities (user_id);

CREATE INDEX IF NOT EXISTS auth_identities_user_provider_idx
    ON auth_identities (user_id, provider_type);

CREATE TABLE IF NOT EXISTS auth_identity_channels (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL REFERENCES auth_identities(id) ON DELETE CASCADE,
    provider_type VARCHAR(20) NOT NULL,
    provider_key TEXT NOT NULL,
    channel VARCHAR(20) NOT NULL,
    channel_app_id TEXT NOT NULL,
    channel_subject TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT auth_identity_channels_provider_type_check
        CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc'))
);

CREATE UNIQUE INDEX IF NOT EXISTS auth_identity_channels_channel_key
    ON auth_identity_channels (provider_type, provider_key, channel, channel_app_id, channel_subject);

CREATE INDEX IF NOT EXISTS auth_identity_channels_identity_id_idx
    ON auth_identity_channels (identity_id);

CREATE TABLE IF NOT EXISTS pending_auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    intent VARCHAR(40) NOT NULL,
    provider_type VARCHAR(20) NOT NULL,
    provider_key TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    upstream_identity_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    target_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    state_hash VARCHAR(255) NOT NULL,
    nonce VARCHAR(255) NOT NULL,
    redirect_to TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pending_auth_sessions_intent_check
        CHECK (intent IN ('login', 'bind_current_user', 'adopt_existing_user_by_email')),
    CONSTRAINT pending_auth_sessions_provider_type_check
        CHECK (provider_type IN ('email', 'linuxdo', 'wechat', 'oidc'))
);

CREATE UNIQUE INDEX IF NOT EXISTS pending_auth_sessions_state_hash_key
    ON pending_auth_sessions (state_hash);

CREATE UNIQUE INDEX IF NOT EXISTS pending_auth_sessions_nonce_key
    ON pending_auth_sessions (nonce);

CREATE INDEX IF NOT EXISTS pending_auth_sessions_target_user_id_idx
    ON pending_auth_sessions (target_user_id);

CREATE INDEX IF NOT EXISTS pending_auth_sessions_expires_at_idx
    ON pending_auth_sessions (expires_at);

CREATE INDEX IF NOT EXISTS pending_auth_sessions_provider_idx
    ON pending_auth_sessions (provider_type, provider_key, provider_subject);

CREATE TABLE IF NOT EXISTS identity_adoption_decisions (
    id BIGSERIAL PRIMARY KEY,
    identity_id BIGINT NOT NULL REFERENCES auth_identities(id) ON DELETE CASCADE,
    adopt_display_name BOOLEAN NOT NULL DEFAULT FALSE,
    adopt_avatar BOOLEAN NOT NULL DEFAULT FALSE,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS identity_adoption_decisions_identity_id_key
    ON identity_adoption_decisions (identity_id);

-- Compatibility backfill:
-- - Existing environments still authenticate through users.email/password_hash.
-- - Seed canonical email identities now so later dual-read / dual-write code can switch over safely.
-- - Use ON CONFLICT DO NOTHING so anomalous historical rows do not block migration rollout.
INSERT INTO auth_identities (
    user_id,
    provider_type,
    provider_key,
    provider_subject,
    verified_at,
    metadata
)
SELECT
    u.id,
    'email',
    'local',
    LOWER(BTRIM(u.email)),
    COALESCE(u.updated_at, u.created_at, NOW()),
    jsonb_build_object(
        'backfill_source', 'users.email',
        'migration', '109_auth_identity_foundation'
    )
FROM users AS u
WHERE u.deleted_at IS NULL
  AND BTRIM(COALESCE(u.email, '')) <> ''
ON CONFLICT (provider_type, provider_key, provider_subject) DO NOTHING;

CREATE TABLE IF NOT EXISTS user_avatars (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    storage_provider VARCHAR(64) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    url TEXT NOT NULL DEFAULT '',
    content_type VARCHAR(128) NOT NULL,
    byte_size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_avatars_byte_size_check
        CHECK (byte_size >= 0 AND byte_size <= 102400)
);

CREATE UNIQUE INDEX IF NOT EXISTS user_avatars_user_id_idx
    ON user_avatars (user_id);

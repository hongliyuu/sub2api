-- 添加 LDAP 配置表
CREATE TABLE IF NOT EXISTS ldap_configs (
    id BIGSERIAL PRIMARY KEY,
    server_url VARCHAR(255) NOT NULL,
    bind_dn VARCHAR(255) NOT NULL,
    bind_password_encrypted TEXT NOT NULL,
    base_dn VARCHAR(255) NOT NULL,
    user_filter VARCHAR(255) NOT NULL DEFAULT '(uid=%s)',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    tls_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    tls_skip_verify BOOLEAN NOT NULL DEFAULT FALSE,
    config_source VARCHAR(50) NOT NULL DEFAULT 'database',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 添加 LDAP 用户表
CREATE TABLE IF NOT EXISTS ldap_users (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    ldap_username VARCHAR(255) NOT NULL UNIQUE,
    ldap_dn VARCHAR(500) NOT NULL,
    last_sync_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ldap_configs_enabled ON ldap_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_ldap_users_user_id ON ldap_users(user_id);
CREATE INDEX IF NOT EXISTS idx_ldap_users_ldap_username ON ldap_users(ldap_username);

-- 添加注释
COMMENT ON TABLE ldap_configs IS 'LDAP 服务器配置表';
COMMENT ON COLUMN ldap_configs.server_url IS 'LDAP 服务器 URL (例如: ldap://ldap.example.com:389)';
COMMENT ON COLUMN ldap_configs.bind_dn IS 'LDAP 绑定 DN';
COMMENT ON COLUMN ldap_configs.bind_password_encrypted IS '加密的绑定密码 (AES-256-GCM)';
COMMENT ON COLUMN ldap_configs.base_dn IS '用户搜索基础 DN';
COMMENT ON COLUMN ldap_configs.user_filter IS 'LDAP 用户过滤器模板';
COMMENT ON COLUMN ldap_configs.enabled IS '是否启用此 LDAP 配置';
COMMENT ON COLUMN ldap_configs.tls_enabled IS '是否使用 LDAP over TLS (ldaps://)';
COMMENT ON COLUMN ldap_configs.tls_skip_verify IS '是否跳过 TLS 证书验证';
COMMENT ON COLUMN ldap_configs.config_source IS '配置来源: env 或 database';

COMMENT ON TABLE ldap_users IS 'LDAP 用户映射表';
COMMENT ON COLUMN ldap_users.user_id IS '关联的用户 ID (外键到 users 表)';
COMMENT ON COLUMN ldap_users.ldap_username IS 'LDAP 用户名 (uid)';
COMMENT ON COLUMN ldap_users.ldap_dn IS 'LDAP 可分辨名称';
COMMENT ON COLUMN ldap_users.last_sync_at IS '最后同步时间';

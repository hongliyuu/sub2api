-- 推荐码唯一索引（CONCURRENTLY，不锁表）
-- 仅对非空值生效，支持软删除后推荐码复用
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS users_aff_code_unique
    ON users(aff_code) WHERE aff_code != '' AND deleted_at IS NULL;

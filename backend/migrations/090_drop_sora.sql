-- Migration: 090 (drop legacy media objects)
-- Remove legacy media-related database objects.
-- Drops legacy tables and related columns in groups/users/usage_logs

-- ============================================================
-- 1. Drop legacy media tables
-- ============================================================
DROP TABLE IF EXISTS sora_tasks;
DROP TABLE IF EXISTS sora_generations;
DROP TABLE IF EXISTS sora_accounts;

-- ============================================================
-- 2. Drop legacy columns from groups table
-- ============================================================
ALTER TABLE groups
    DROP COLUMN IF EXISTS sora_image_price_360,
    DROP COLUMN IF EXISTS sora_image_price_540,
    DROP COLUMN IF EXISTS sora_video_price_per_request,
    DROP COLUMN IF EXISTS sora_video_price_per_request_hd,
    DROP COLUMN IF EXISTS sora_storage_quota_bytes;

-- ============================================================
-- 3. Drop legacy columns from users table
-- ============================================================
ALTER TABLE users
    DROP COLUMN IF EXISTS sora_storage_quota_bytes,
    DROP COLUMN IF EXISTS sora_storage_used_bytes;

-- ============================================================
-- 4. Drop legacy column from usage_logs table
-- ============================================================
ALTER TABLE usage_logs
    DROP COLUMN IF EXISTS media_type;

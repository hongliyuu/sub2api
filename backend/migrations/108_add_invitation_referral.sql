-- 邀请返利功能：用户表新增邀请相关字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_code VARCHAR(32) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS invited_by BIGINT DEFAULT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_count INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS invite_earnings DECIMAL(20,8) DEFAULT 0;

-- 邀请码索引（用于注册时快速查找邀请人）
CREATE INDEX IF NOT EXISTS idx_users_invite_code ON users (invite_code) WHERE invite_code != '';
CREATE INDEX IF NOT EXISTS idx_users_invited_by ON users (invited_by) WHERE invited_by IS NOT NULL;

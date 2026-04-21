-- 邀请返利功能字段
-- aff_code: 用户唯一推荐码（6位字母数字，首次请求时懒加载生成）
-- aff_count: 成功邀请人数
-- aff_history_balance: 累计获得的返利余额（历史记录，不随余额消费递减）
-- inviter_id: 邀请人的用户ID（记录谁邀请了该用户）

ALTER TABLE users ADD COLUMN IF NOT EXISTS aff_code VARCHAR(8) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS aff_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS aff_history_balance DECIMAL(20,8) NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS inviter_id BIGINT DEFAULT NULL;

CREATE INDEX IF NOT EXISTS users_inviter_id_idx ON users(inviter_id);

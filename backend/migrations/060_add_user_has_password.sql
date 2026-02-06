-- 添加 has_password 字段，标记用户是否设置了密码
-- OAuth/微信登录用户的密码是系统随机生成的，用户不知道
ALTER TABLE users ADD COLUMN IF NOT EXISTS has_password BOOLEAN DEFAULT true;

-- 仅合成邮箱用户（从未绑定真实邮箱的微信/LinuxDo 登录用户）标记为未设置密码
-- 注意：已绑定真实邮箱的 OAuth 用户可能已通过忘记密码重置了密码，不能假设他们不知道密码
UPDATE users SET has_password = false WHERE (email LIKE '%@wechat-auth.invalid' OR email LIKE '%@linuxdo.invalid') AND deleted_at IS NULL;

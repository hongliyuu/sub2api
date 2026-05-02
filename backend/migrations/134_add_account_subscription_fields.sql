ALTER TABLE accounts ADD COLUMN IF NOT EXISTS plan_type varchar(100);
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS subscription_status varchar(100);
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS subscription_expires_at timestamptz;

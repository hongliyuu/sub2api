-- 订阅订单升级功能：新增升级相关字段
-- 支持从低价套餐升级到高价套餐，按剩余天数折算旧套餐价值

-- 订单类型：purchase（新购）/ upgrade（升级）
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS order_type VARCHAR(20) NOT NULL DEFAULT 'purchase';

-- 升级源订阅ID（仅升级订单有效）
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS source_subscription_id BIGINT;

-- 原价（用于展示删除线价格）
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS original_amount DECIMAL(20,2);

-- 折扣金额（旧套餐剩余价值）
ALTER TABLE subscription_orders ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(20,2);

-- 索引
CREATE INDEX IF NOT EXISTS idx_subscription_orders_source_sub_id ON subscription_orders(source_subscription_id);

-- 字段注释
COMMENT ON COLUMN subscription_orders.order_type IS '订单类型：purchase（新购）/ upgrade（升级）';
COMMENT ON COLUMN subscription_orders.source_subscription_id IS '升级源订阅ID';
COMMENT ON COLUMN subscription_orders.original_amount IS '原价（升级时展示删除线价格）';
COMMENT ON COLUMN subscription_orders.discount_amount IS '折扣金额（旧套餐剩余价值）';

-- 创建充值订单表
CREATE TABLE IF NOT EXISTS recharge_orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(50) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(20) NOT NULL,
    payment_channel VARCHAR(20) NOT NULL DEFAULT 'native',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    wechat_transaction_id VARCHAR(64),
    qrcode_url TEXT,
    prepay_id VARCHAR(64),
    expire_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    notes TEXT DEFAULT '',
    refund_no VARCHAR(50),
    refund_status VARCHAR(20),
    refunded_at TIMESTAMPTZ,
    refund_reason TEXT,
    refund_admin_id BIGINT,
    wechat_refund_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_recharge_orders_user_id ON recharge_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_status ON recharge_orders(status);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_expire_at ON recharge_orders(expire_at);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_user_id_status ON recharge_orders(user_id, status);

-- 添加注释
COMMENT ON TABLE recharge_orders IS '充值订单表';
COMMENT ON COLUMN recharge_orders.order_no IS '订单号';
COMMENT ON COLUMN recharge_orders.user_id IS '用户ID';
COMMENT ON COLUMN recharge_orders.amount IS '充值金额';
COMMENT ON COLUMN recharge_orders.payment_method IS '支付方式: wechat_pay, alipay';
COMMENT ON COLUMN recharge_orders.payment_channel IS '支付渠道: native, jsapi, h5';
COMMENT ON COLUMN recharge_orders.status IS '订单状态: pending, paid, failed, expired, cancelled';
COMMENT ON COLUMN recharge_orders.wechat_transaction_id IS '微信支付订单号';
COMMENT ON COLUMN recharge_orders.qrcode_url IS '支付二维码URL';
COMMENT ON COLUMN recharge_orders.prepay_id IS '预支付交易会话标识';
COMMENT ON COLUMN recharge_orders.expire_at IS '订单过期时间';
COMMENT ON COLUMN recharge_orders.paid_at IS '支付完成时间';
COMMENT ON COLUMN recharge_orders.notes IS '备注';
COMMENT ON COLUMN recharge_orders.refund_no IS '退款单号';
COMMENT ON COLUMN recharge_orders.refund_status IS '退款状态: pending, processing, success, failed';
COMMENT ON COLUMN recharge_orders.refunded_at IS '退款时间';
COMMENT ON COLUMN recharge_orders.refund_reason IS '退款原因';
COMMENT ON COLUMN recharge_orders.refund_admin_id IS '退款操作人ID';
COMMENT ON COLUMN recharge_orders.wechat_refund_id IS '微信退款单号';

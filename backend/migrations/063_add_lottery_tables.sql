-- 063: 抽奖拉新功能 - 创建 lottery_activities, lottery_participants, lottery_coupons 三表

-- ==========================================
-- 1. lottery_activities (抽奖活动主表)
-- ==========================================
CREATE TABLE IF NOT EXISTS lottery_activities (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    share_code VARCHAR(32) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    draw_at TIMESTAMPTZ NOT NULL,
    activity_start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activity_end_at TIMESTAMPTZ NOT NULL,
    min_participants INT NOT NULL DEFAULT 10,
    base_win_rate DECIMAL(5,4) NOT NULL DEFAULT 0.1000,
    weight_config JSONB NOT NULL DEFAULT '{"new_user":3.0,"regular":1.0,"paid":0.3,"subscriber":0.1}',
    winner_discount_percent INT NOT NULL DEFAULT 95,
    loser_coupon_amount DECIMAL(20,2) NOT NULL DEFAULT 30.00,
    temp_group_id BIGINT,
    participant_count INT NOT NULL DEFAULT 0,
    winner_count INT NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_lottery_activities_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_lottery_activities_temp_group FOREIGN KEY (temp_group_id) REFERENCES groups(id) ON DELETE SET NULL
);

-- 唯一索引：分享码
CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_activities_share_code ON lottery_activities(share_code);

-- 状态索引
CREATE INDEX IF NOT EXISTS idx_lottery_activities_status ON lottery_activities(status);

-- 开奖时间索引（用于定时任务扫描）
CREATE INDEX IF NOT EXISTS idx_lottery_activities_draw_at ON lottery_activities(draw_at);

-- 活动结束时间索引（用于过期清理）
CREATE INDEX IF NOT EXISTS idx_lottery_activities_activity_end_at ON lottery_activities(activity_end_at);

COMMENT ON TABLE lottery_activities IS '抽奖活动主表';
COMMENT ON COLUMN lottery_activities.share_code IS '分享短码（用于 QR 链接）';
COMMENT ON COLUMN lottery_activities.status IS '活动状态：pending/active/drawing/completed/cancelled';
COMMENT ON COLUMN lottery_activities.draw_at IS '开奖时间';
COMMENT ON COLUMN lottery_activities.activity_start_at IS '活动开始时间';
COMMENT ON COLUMN lottery_activities.activity_end_at IS '活动结束时间（开奖后+3天）';
COMMENT ON COLUMN lottery_activities.min_participants IS '最低参与人数';
COMMENT ON COLUMN lottery_activities.base_win_rate IS '基础中奖率';
COMMENT ON COLUMN lottery_activities.weight_config IS '权重配置 JSON';
COMMENT ON COLUMN lottery_activities.winner_discount_percent IS '中奖折扣百分比（95=付95%即九五折）';
COMMENT ON COLUMN lottery_activities.loser_coupon_amount IS '未中奖立减金额';
COMMENT ON COLUMN lottery_activities.temp_group_id IS '临时分组 ID';

-- ==========================================
-- 2. lottery_participants (参与记录)
-- ==========================================
CREATE TABLE IF NOT EXISTS lottery_participants (
    id BIGSERIAL PRIMARY KEY,
    activity_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    user_category VARCHAR(20) NOT NULL,
    weight_multiplier DECIMAL(5,2) NOT NULL DEFAULT 1.00,
    is_winner BOOLEAN,
    coupon_id BIGINT,
    participated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_lottery_participants_activity FOREIGN KEY (activity_id) REFERENCES lottery_activities(id) ON DELETE CASCADE,
    CONSTRAINT fk_lottery_participants_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 每活动每用户限一次
CREATE UNIQUE INDEX IF NOT EXISTS idx_lottery_participants_activity_user ON lottery_participants(activity_id, user_id);

-- 活动 ID 索引
CREATE INDEX IF NOT EXISTS idx_lottery_participants_activity_id ON lottery_participants(activity_id);

-- 用户 ID 索引
CREATE INDEX IF NOT EXISTS idx_lottery_participants_user_id ON lottery_participants(user_id);

COMMENT ON TABLE lottery_participants IS '抽奖参与记录';
COMMENT ON COLUMN lottery_participants.user_category IS '用户类别：new_user/regular/paid/subscriber';
COMMENT ON COLUMN lottery_participants.weight_multiplier IS '该用户的权重倍数';
COMMENT ON COLUMN lottery_participants.is_winner IS 'NULL=未开奖, true=中奖, false=未中奖';

-- ==========================================
-- 3. lottery_coupons (优惠券)
-- ==========================================
CREATE TABLE IF NOT EXISTS lottery_coupons (
    id BIGSERIAL PRIMARY KEY,
    activity_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    coupon_type VARCHAR(20) NOT NULL,
    discount_percent INT,
    reduction_amount DECIMAL(20,2),
    applicable_scope VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    used_at TIMESTAMPTZ,
    used_order_id VARCHAR(100),
    expires_at TIMESTAMPTZ NOT NULL,
    reminded BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_lottery_coupons_activity FOREIGN KEY (activity_id) REFERENCES lottery_activities(id) ON DELETE CASCADE,
    CONSTRAINT fk_lottery_coupons_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 用户 + 状态索引（查询可用优惠券）
CREATE INDEX IF NOT EXISTS idx_lottery_coupons_user_status ON lottery_coupons(user_id, status);

-- 活动索引
CREATE INDEX IF NOT EXISTS idx_lottery_coupons_activity_id ON lottery_coupons(activity_id);

-- 过期时间索引
CREATE INDEX IF NOT EXISTS idx_lottery_coupons_expires_at ON lottery_coupons(expires_at);

-- 提醒标记索引（用于到期提醒扫描）
CREATE INDEX IF NOT EXISTS idx_lottery_coupons_reminded ON lottery_coupons(reminded) WHERE status = 'active' AND reminded = false;

COMMENT ON TABLE lottery_coupons IS '抽奖优惠券';
COMMENT ON COLUMN lottery_coupons.coupon_type IS '优惠券类型：winner_discount/loser_reduction';
COMMENT ON COLUMN lottery_coupons.discount_percent IS '折扣百分比（95=付95%，中奖者）';
COMMENT ON COLUMN lottery_coupons.reduction_amount IS '立减金额（未中奖者）';
COMMENT ON COLUMN lottery_coupons.applicable_scope IS '适用范围：recharge/monthly_subscription';
COMMENT ON COLUMN lottery_coupons.status IS '状态：active/used/expired';
COMMENT ON COLUMN lottery_coupons.reminded IS '过期前是否已提醒';

-- 回写 lottery_participants.coupon_id 的外键（可选，不加也行，coupon_id 可在应用层维护）

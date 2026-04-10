-- 084_model_pricings.sql
-- 模型计费价格管理表
-- 管理员可通过页面查看/编辑各模型的 fallback 计费价格

CREATE TABLE IF NOT EXISTS model_pricings (
    id                                BIGSERIAL PRIMARY KEY,
    created_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                        TIMESTAMPTZ,
    model_key                         TEXT        NOT NULL,
    display_name                      TEXT,
    input_price_per_million           DOUBLE PRECISION NOT NULL DEFAULT 0,
    output_price_per_million          DOUBLE PRECISION NOT NULL DEFAULT 0,
    input_price_per_million_priority  DOUBLE PRECISION NOT NULL DEFAULT 0,
    output_price_per_million_priority DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_read_price_per_million      DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_read_price_per_million_priority DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_creation_price_per_million  DOUBLE PRECISION NOT NULL DEFAULT 0,
    enabled                           BOOLEAN NOT NULL DEFAULT TRUE,
    note                              TEXT
);

-- active 记录（deleted_at IS NULL）强制 model_key 唯一。
-- PostgreSQL 中 NULL != NULL，普通复合 UNIQUE 无法阻止多条 deleted_at IS NULL 的记录。
CREATE UNIQUE INDEX IF NOT EXISTS model_pricings_model_key_active_unique_idx
    ON model_pricings (model_key) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS model_pricings_enabled_idx ON model_pricings (enabled);
CREATE INDEX IF NOT EXISTS model_pricings_deleted_at_idx ON model_pricings (deleted_at);

-- 种子数据：严格对齐 billing_service.go 中的 hardcoded fallback 价格（每百万 token 为单位）
-- 价格换算：per-token * 1,000,000 = per-million
-- 注意：priority 价格字段对应 InputPricePerTokenPriority / OutputPricePerTokenPriority 等字段
INSERT INTO model_pricings (
    model_key, display_name,
    input_price_per_million, output_price_per_million,
    input_price_per_million_priority, output_price_per_million_priority,
    cache_read_price_per_million, cache_read_price_per_million_priority,
    cache_creation_price_per_million,
    enabled, note
)
VALUES
    -- Anthropic Claude 系列
    -- claude-opus-4.5: input=5e-6, output=25e-6, cache_read=0.5e-6, cache_creation=6.25e-6
    ('claude-opus-4.5',       'Claude Opus 4.5',        5.0,   25.0,  0,    0,    0.5,   0,    6.25, TRUE, 'Anthropic 官方价格'),
    -- claude-sonnet-4: input=3e-6, output=15e-6, cache_read=0.3e-6, cache_creation=3.75e-6
    ('claude-sonnet-4',       'Claude Sonnet 4',        3.0,   15.0,  0,    0,    0.3,   0,    3.75, TRUE, 'Anthropic 官方价格'),
    -- claude-3-5-sonnet: input=3e-6, output=15e-6, cache_read=0.3e-6, cache_creation=3.75e-6
    ('claude-3-5-sonnet',     'Claude 3.5 Sonnet',      3.0,   15.0,  0,    0,    0.3,   0,    3.75, TRUE, 'Anthropic 官方价格'),
    -- claude-3-5-haiku: input=1e-6, output=5e-6, cache_read=0.1e-6, cache_creation=1.25e-6
    ('claude-3-5-haiku',      'Claude 3.5 Haiku',       1.0,   5.0,   0,    0,    0.1,   0,    1.25, TRUE, 'Anthropic 官方价格'),
    -- claude-3-opus: input=15e-6, output=75e-6, cache_read=1.5e-6, cache_creation=18.75e-6
    ('claude-3-opus',         'Claude 3 Opus',          15.0,  75.0,  0,    0,    1.5,   0,    18.75,TRUE, 'Anthropic 官方价格'),
    -- claude-3-haiku: input=0.25e-6, output=1.25e-6, cache_read=0.03e-6, cache_creation=0.3e-6
    ('claude-3-haiku',        'Claude 3 Haiku',         0.25,  1.25,  0,    0,    0.03,  0,    0.3,  TRUE, 'Anthropic 官方价格'),
    -- claude-opus-4.6: 与 claude-opus-4.5 同价
    ('claude-opus-4.6',       'Claude Opus 4.6',        5.0,   25.0,  0,    0,    0.5,   0,    6.25, TRUE, '与 claude-opus-4.5 同价'),

    -- Google Gemini 系列
    -- gemini-3.1-pro: input=2e-6, output=12e-6, cache_read=0.2e-6, cache_creation=2e-6
    ('gemini-3.1-pro',        'Gemini 3.1 Pro',         2.0,   12.0,  0,    0,    0.2,   0,    2.0,  TRUE, 'Google 官方价格'),

    -- OpenAI GPT 系列
    -- gpt-5.1: input=1.25e-6, input_priority=2.5e-6, output=10e-6, output_priority=20e-6
    --          cache_read=0.125e-6, cache_read_priority=0.25e-6, cache_creation=1.25e-6
    ('gpt-5.1',               'GPT-5.1',                1.25,  10.0,  2.5,  20.0, 0.125, 0.25, 1.25, TRUE, 'OpenAI 官方价格'),
    -- gpt-5.4: input=2.5e-6, input_priority=5e-6, output=15e-6, output_priority=30e-6
    --          cache_read=0.25e-6, cache_read_priority=0.5e-6, cache_creation=2.5e-6
    --          注意：>272k token 时有长上下文价格策略（2x/1.5x），由 applyModelSpecificPricingPolicy 处理，无需在此体现
    ('gpt-5.4',               'GPT-5.4',                2.5,   15.0,  5.0,  30.0, 0.25,  0.5,  2.5,  TRUE, 'OpenAI 官方价格，>272k token 时 2x/1.5x'),
    -- gpt-5.4-mini: input=7.5e-7, output=4.5e-6, cache_read=7.5e-8（无 priority，无 cache_creation）
    ('gpt-5.4-mini',          'GPT-5.4 Mini',           0.75,  4.5,   0,    0,    0.075, 0,    0.0,  TRUE, 'OpenAI 官方价格'),
    -- gpt-5.4-nano: input=2e-7, output=1.25e-6, cache_read=2e-8
    ('gpt-5.4-nano',          'GPT-5.4 Nano',           0.2,   1.25,  0,    0,    0.02,  0,    0.0,  TRUE, 'OpenAI 官方价格'),
    -- gpt-5.2: input=1.75e-6, input_priority=3.5e-6, output=14e-6, output_priority=28e-6
    --          cache_read=0.175e-6, cache_read_priority=0.35e-6, cache_creation=1.75e-6
    ('gpt-5.2',               'GPT-5.2',                1.75,  14.0,  3.5,  28.0, 0.175, 0.35, 1.75, TRUE, 'OpenAI 官方价格'),
    -- gpt-5.1-codex: input=1.5e-6, input_priority=3e-6, output=12e-6, output_priority=24e-6
    --                cache_read=0.15e-6, cache_read_priority=0.3e-6, cache_creation=1.5e-6
    ('gpt-5.1-codex',         'GPT-5.1 Codex',          1.5,   12.0,  3.0,  24.0, 0.15,  0.3,  1.5,  TRUE, 'OpenAI Codex 官方价格'),
    -- gpt-5.2-codex: input=1.75e-6, input_priority=3.5e-6, output=14e-6, output_priority=28e-6
    --                cache_read=0.175e-6, cache_read_priority=0.35e-6, cache_creation=1.75e-6
    ('gpt-5.2-codex',         'GPT-5.2 Codex',          1.75,  14.0,  3.5,  28.0, 0.175, 0.35, 1.75, TRUE, 'OpenAI Codex 官方价格'),
    -- gpt-4-o-preview: input=5e-6, output=15e-6，无 cache
    ('gpt-4-o-preview',       'GPT-4o Preview',         5.0,   15.0,  0,    0,    0.0,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    -- gpt-41-copilot: input=2e-6, output=8e-6, cache_read=0.5e-6
    ('gpt-41-copilot',        'GPT-4.1 Copilot',        2.0,   8.0,   0,    0,    0.5,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    -- text-embedding-ada-002: input=0.1e-6，无 output
    ('text-embedding-ada-002','Text Embedding Ada 002',  0.1,   0.0,   0,    0,    0.0,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    -- text-embedding-3-small: 无 fallback 定义，按 20e-9 input 估算，此处保留原始值
    ('text-embedding-3-small','Text Embedding 3 Small',  0.02,  0.0,   0,    0,    0.0,   0,    0.0,  TRUE, 'OpenAI 官方价格'),

    -- xAI Grok 系列
    -- grok-code-fast-1: 无 fallback 定义，保留原有价格
    ('grok-code-fast-1',      'Grok Code Fast 1',        0.2,   1.5,   0,    0,    0.02,  0,    0.0,  TRUE, 'xAI 官方价格')
ON CONFLICT DO NOTHING;

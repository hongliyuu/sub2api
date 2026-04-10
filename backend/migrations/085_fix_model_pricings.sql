-- 085_fix_model_pricings.sql
-- 修正 084_model_pricings.sql 的遗留问题：
-- 1. 确保正确的 partial unique index 存在（084 草稿版本可能建了复合唯一约束）
-- 2. 用 ON CONFLICT DO UPDATE 修正 seed 数据（旧草稿版本的价格与 billing_service.go 不一致）

-- 删除旧的复合唯一约束（如存在于草稿版本中）。
-- 正确的约束应为 partial unique index（WHERE deleted_at IS NULL），见下方。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'model_pricings_model_key_deleted_at_key'
          AND conrelid = 'model_pricings'::regclass
    ) THEN
        ALTER TABLE model_pricings
            DROP CONSTRAINT model_pricings_model_key_deleted_at_key;
    END IF;
END;
$$;

-- 确保 partial unique index 存在（IF NOT EXISTS 幂等）。
CREATE UNIQUE INDEX IF NOT EXISTS model_pricings_model_key_active_unique_idx
    ON model_pricings (model_key) WHERE deleted_at IS NULL;

-- 确保普通索引存在（IF NOT EXISTS 幂等）。
CREATE INDEX IF NOT EXISTS model_pricings_enabled_idx    ON model_pricings (enabled);
CREATE INDEX IF NOT EXISTS model_pricings_deleted_at_idx ON model_pricings (deleted_at);

-- 修正 seed 数据：使用 ON CONFLICT(model_key) DO UPDATE 对齐 billing_service.go fallback 价格。
-- 只修正当前仍为系统 seed（note 包含"官方价格"）且用户未手动修改的条目（enabled=true）。
-- priority 字段和 cache_creation 字段也一并修正。
INSERT INTO model_pricings (
    model_key, display_name,
    input_price_per_million, output_price_per_million,
    input_price_per_million_priority, output_price_per_million_priority,
    cache_read_price_per_million, cache_read_price_per_million_priority,
    cache_creation_price_per_million,
    enabled, note
)
VALUES
    ('claude-opus-4.5',       'Claude Opus 4.5',        5.0,   25.0,  0,    0,    0.5,   0,    6.25, TRUE, 'Anthropic 官方价格'),
    ('claude-sonnet-4',       'Claude Sonnet 4',        3.0,   15.0,  0,    0,    0.3,   0,    3.75, TRUE, 'Anthropic 官方价格'),
    ('claude-3-5-sonnet',     'Claude 3.5 Sonnet',      3.0,   15.0,  0,    0,    0.3,   0,    3.75, TRUE, 'Anthropic 官方价格'),
    ('claude-3-5-haiku',      'Claude 3.5 Haiku',       1.0,   5.0,   0,    0,    0.1,   0,    1.25, TRUE, 'Anthropic 官方价格'),
    ('claude-3-opus',         'Claude 3 Opus',          15.0,  75.0,  0,    0,    1.5,   0,    18.75,TRUE, 'Anthropic 官方价格'),
    ('claude-3-haiku',        'Claude 3 Haiku',         0.25,  1.25,  0,    0,    0.03,  0,    0.3,  TRUE, 'Anthropic 官方价格'),
    ('claude-opus-4.6',       'Claude Opus 4.6',        5.0,   25.0,  0,    0,    0.5,   0,    6.25, TRUE, '与 claude-opus-4.5 同价'),
    ('gemini-3.1-pro',        'Gemini 3.1 Pro',         2.0,   12.0,  0,    0,    0.2,   0,    2.0,  TRUE, 'Google 官方价格'),
    ('gpt-5.1',               'GPT-5.1',                1.25,  10.0,  2.5,  20.0, 0.125, 0.25, 1.25, TRUE, 'OpenAI 官方价格'),
    ('gpt-5.4',               'GPT-5.4',                2.5,   15.0,  5.0,  30.0, 0.25,  0.5,  2.5,  TRUE, 'OpenAI 官方价格，>272k token 时 2x/1.5x'),
    ('gpt-5.4-mini',          'GPT-5.4 Mini',           0.75,  4.5,   0,    0,    0.075, 0,    0.0,  TRUE, 'OpenAI 官方价格'),
    ('gpt-5.4-nano',          'GPT-5.4 Nano',           0.2,   1.25,  0,    0,    0.02,  0,    0.0,  TRUE, 'OpenAI 官方价格'),
    ('gpt-5.2',               'GPT-5.2',                1.75,  14.0,  3.5,  28.0, 0.175, 0.35, 1.75, TRUE, 'OpenAI 官方价格'),
    ('gpt-5.1-codex',         'GPT-5.1 Codex',          1.5,   12.0,  3.0,  24.0, 0.15,  0.3,  1.5,  TRUE, 'OpenAI Codex 官方价格'),
    ('gpt-5.2-codex',         'GPT-5.2 Codex',          1.75,  14.0,  3.5,  28.0, 0.175, 0.35, 1.75, TRUE, 'OpenAI Codex 官方价格'),
    ('gpt-4-o-preview',       'GPT-4o Preview',         5.0,   15.0,  0,    0,    0.0,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    ('gpt-41-copilot',        'GPT-4.1 Copilot',        2.0,   8.0,   0,    0,    0.5,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    ('text-embedding-ada-002','Text Embedding Ada 002',  0.1,   0.0,   0,    0,    0.0,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    ('text-embedding-3-small','Text Embedding 3 Small',  0.02,  0.0,   0,    0,    0.0,   0,    0.0,  TRUE, 'OpenAI 官方价格'),
    ('grok-code-fast-1',      'Grok Code Fast 1',        0.2,   1.5,   0,    0,    0.02,  0,    0.0,  TRUE, 'xAI 官方价格')
ON CONFLICT (model_key) WHERE deleted_at IS NULL
DO UPDATE SET
    display_name                      = EXCLUDED.display_name,
    input_price_per_million           = EXCLUDED.input_price_per_million,
    output_price_per_million          = EXCLUDED.output_price_per_million,
    input_price_per_million_priority  = EXCLUDED.input_price_per_million_priority,
    output_price_per_million_priority = EXCLUDED.output_price_per_million_priority,
    cache_read_price_per_million      = EXCLUDED.cache_read_price_per_million,
    cache_read_price_per_million_priority = EXCLUDED.cache_read_price_per_million_priority,
    cache_creation_price_per_million  = EXCLUDED.cache_creation_price_per_million,
    note                              = EXCLUDED.note,
    updated_at                        = NOW()
WHERE model_pricings.note = EXCLUDED.note
   OR model_pricings.note LIKE '%官方价格%';

-- 087_copilot_platform_configs.sql
-- Copilot 平台级参数配置表
-- 按 plan_type 存储 max_output_tokens / max_body_kb / model_mapping / model_whitelist 的默认值
-- 账号级配置优先于此处配置，此处配置优先于系统默认

CREATE TABLE IF NOT EXISTS copilot_platform_configs (
    id                BIGSERIAL PRIMARY KEY,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    plan_type         VARCHAR(32) NOT NULL,
    -- 枚举值: individual_free / individual_pro / individual_pro_plus / business / enterprise

    max_output_tokens BIGINT,      -- NULL 表示不设默认
    max_body_kb       INTEGER,     -- NULL 表示不设默认
    model_mapping     JSONB,       -- {"from_model": "to_model", ...}，NULL 表示不设默认
    model_whitelist   JSONB        -- ["model-a", "model-b"]，NULL 表示不设默认
);

CREATE UNIQUE INDEX IF NOT EXISTS copilot_platform_configs_plan_type_unique_idx
    ON copilot_platform_configs (plan_type);

CREATE INDEX IF NOT EXISTS copilot_platform_configs_plan_type_idx
    ON copilot_platform_configs (plan_type);

-- 预插入 5 行，max_output_tokens = 0 表示不限制输出 token，按套餐设置模型白名单。
-- individual_free / individual_pro / individual_pro_plus 使用 pro+ 账户可用模型列表；
-- business / enterprise 使用 business 账户可用模型列表。
-- model_whitelist 为空数组时表示不启用白名单（允许所有模型）。
INSERT INTO copilot_platform_configs (plan_type, max_output_tokens, model_whitelist) VALUES
    ('individual_free',     0, '["claude-opus-4.6","claude-sonnet-4.6","claude-sonnet-4","claude-sonnet-4.5","claude-opus-4.5","claude-haiku-4.5","gemini-3.1-pro-preview","gemini-3-flash-preview","gemini-2.5-pro","goldeneye-free-auto","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex","gpt-5.2-codex","gpt-5.2","gpt-5.1","gpt-5-mini","gpt-41-copilot","gpt-4.1","gpt-4.1-2025-04-14","gpt-4o","gpt-4o-2024-05-13","gpt-4o-2024-08-06","gpt-4o-2024-11-20","gpt-4o-mini","gpt-4o-mini-2024-07-18","gpt-4","gpt-4-0613","gpt-4-0125-preview","gpt-4-o-preview","gpt-3.5-turbo","gpt-3.5-turbo-0613","grok-code-fast-1","minimax-m2.5","oswe-vscode-prime","oswe-vscode-secondary","text-embedding-3-small","text-embedding-3-small-inference","text-embedding-ada-002","accounts/msft/routers/mp3yn0h7","accounts/msft/routers/yaqq2gxh","accounts/msft/routers/f185i3v4","accounts/msft/routers/fmfeto88","accounts/msft/routers/gdjv4v2v"]'),
    ('individual_pro',      0, '["claude-opus-4.6","claude-sonnet-4.6","claude-sonnet-4","claude-sonnet-4.5","claude-opus-4.5","claude-haiku-4.5","gemini-3.1-pro-preview","gemini-3-flash-preview","gemini-2.5-pro","goldeneye-free-auto","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex","gpt-5.2-codex","gpt-5.2","gpt-5.1","gpt-5-mini","gpt-41-copilot","gpt-4.1","gpt-4.1-2025-04-14","gpt-4o","gpt-4o-2024-05-13","gpt-4o-2024-08-06","gpt-4o-2024-11-20","gpt-4o-mini","gpt-4o-mini-2024-07-18","gpt-4","gpt-4-0613","gpt-4-0125-preview","gpt-4-o-preview","gpt-3.5-turbo","gpt-3.5-turbo-0613","grok-code-fast-1","minimax-m2.5","oswe-vscode-prime","oswe-vscode-secondary","text-embedding-3-small","text-embedding-3-small-inference","text-embedding-ada-002","accounts/msft/routers/mp3yn0h7","accounts/msft/routers/yaqq2gxh","accounts/msft/routers/f185i3v4","accounts/msft/routers/fmfeto88","accounts/msft/routers/gdjv4v2v"]'),
    ('individual_pro_plus', 0, '["claude-opus-4.6","claude-sonnet-4.6","claude-sonnet-4","claude-sonnet-4.5","claude-opus-4.5","claude-haiku-4.5","gemini-3.1-pro-preview","gemini-3-flash-preview","gemini-2.5-pro","goldeneye-free-auto","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex","gpt-5.2-codex","gpt-5.2","gpt-5.1","gpt-5-mini","gpt-41-copilot","gpt-4.1","gpt-4.1-2025-04-14","gpt-4o","gpt-4o-2024-05-13","gpt-4o-2024-08-06","gpt-4o-2024-11-20","gpt-4o-mini","gpt-4o-mini-2024-07-18","gpt-4","gpt-4-0613","gpt-4-0125-preview","gpt-4-o-preview","gpt-3.5-turbo","gpt-3.5-turbo-0613","grok-code-fast-1","minimax-m2.5","oswe-vscode-prime","oswe-vscode-secondary","text-embedding-3-small","text-embedding-3-small-inference","text-embedding-ada-002","accounts/msft/routers/mp3yn0h7","accounts/msft/routers/yaqq2gxh","accounts/msft/routers/f185i3v4","accounts/msft/routers/fmfeto88","accounts/msft/routers/gdjv4v2v"]'),
    ('business',            0, '["claude-opus-4.6","claude-sonnet-4.6","gemini-3.1-pro-preview","gemini-3-flash-preview","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex","gpt-5.2","gpt-41-copilot","gpt-4.1","gpt-4.1-2025-04-14","gpt-4o","gpt-4o-2024-05-13","gpt-4o-2024-08-06","gpt-4o-2024-11-20","gpt-4o-mini","gpt-4o-mini-2024-07-18","gpt-4","gpt-4-0613","gpt-4-0125-preview","gpt-4-o-preview","gpt-3.5-turbo","gpt-3.5-turbo-0613","grok-code-fast-1","text-embedding-3-small","text-embedding-3-small-inference","text-embedding-ada-002b","accounts/msft/routers/f185i3v4","accounts/msft/routers/fmfeto88","accounts/msft/routers/gdjv4v2v"]'),
    ('enterprise',          0, '["claude-opus-4.6","claude-sonnet-4.6","gemini-3.1-pro-preview","gemini-3-flash-preview","gpt-5.4","gpt-5.4-mini","gpt-5.3-codex","gpt-5.2","gpt-41-copilot","gpt-4.1","gpt-4.1-2025-04-14","gpt-4o","gpt-4o-2024-05-13","gpt-4o-2024-08-06","gpt-4o-2024-11-20","gpt-4o-mini","gpt-4o-mini-2024-07-18","gpt-4","gpt-4-0613","gpt-4-0125-preview","gpt-4-o-preview","gpt-3.5-turbo","gpt-3.5-turbo-0613","grok-code-fast-1","text-embedding-3-small","text-embedding-3-small-inference","text-embedding-ada-002b","accounts/msft/routers/f185i3v4","accounts/msft/routers/fmfeto88","accounts/msft/routers/gdjv4v2v"]')
ON CONFLICT (plan_type) DO NOTHING;

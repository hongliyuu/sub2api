-- Add user_agent column to accounts table
-- Allows per-account custom User-Agent for upstream API requests (Claude/OpenAI/Gemini)

ALTER TABLE accounts ADD COLUMN IF NOT EXISTS user_agent VARCHAR(200);

COMMENT ON COLUMN accounts.user_agent IS 'Custom User-Agent for upstream API requests. Null means use platform default.';

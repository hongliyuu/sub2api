-- P2: migrate OpenAI account extra key from openai_passthrough/openai_oauth_passthrough
-- to forward_passthrough_only.

UPDATE accounts
SET extra = CASE
	WHEN extra ? 'forward_passthrough_only' THEN
		extra - 'openai_passthrough' - 'openai_oauth_passthrough'
	WHEN extra ? 'openai_passthrough' THEN
		jsonb_set(
			extra - 'openai_passthrough' - 'openai_oauth_passthrough',
			'{forward_passthrough_only}',
			extra->'openai_passthrough',
			true
		)
	WHEN extra ? 'openai_oauth_passthrough' THEN
		jsonb_set(
			extra - 'openai_oauth_passthrough',
			'{forward_passthrough_only}',
			extra->'openai_oauth_passthrough',
			true
		)
	ELSE
		extra
END
WHERE platform = 'openai'
  AND extra IS NOT NULL
  AND (extra ? 'openai_passthrough' OR extra ? 'openai_oauth_passthrough');

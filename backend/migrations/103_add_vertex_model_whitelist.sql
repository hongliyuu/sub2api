-- Split legacy Vertex allowlists out of model_mapping.
--
-- Vertex now supports a dedicated credentials.model_whitelist field. For older
-- accounts that encoded allowlists as identity model_mapping entries, migrate
-- those keys into model_whitelist while preserving the original model_mapping.

UPDATE accounts
SET credentials = jsonb_set(
    credentials,
    '{model_whitelist}',
    (
        SELECT jsonb_agg(entry.key ORDER BY entry.key)
        FROM jsonb_each_text(credentials->'model_mapping') AS entry(key, value)
        WHERE entry.key = entry.value
    ),
    true
)
WHERE type = 'vertex'
  AND deleted_at IS NULL
  AND credentials->'model_whitelist' IS NULL
  AND credentials->'model_mapping' IS NOT NULL
  AND EXISTS (
      SELECT 1
      FROM jsonb_each_text(credentials->'model_mapping') AS entry(key, value)
      WHERE entry.key = entry.value
  );

-- P2: disable error_passthrough_rules.passthrough_body capability.
-- Runtime no longer allows forwarding upstream raw fault text to clients.

ALTER TABLE error_passthrough_rules
ALTER COLUMN passthrough_body SET DEFAULT false;

UPDATE error_passthrough_rules
SET passthrough_body = false
WHERE passthrough_body = true;

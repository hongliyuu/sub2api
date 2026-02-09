-- Seed account expiry reminder advance days setting (9 days = 3 weeks into a 30-day cycle)
INSERT INTO settings (key, value, updated_at)
VALUES ('account_expiry_reminder_advance_days', '9', NOW())
ON CONFLICT (key) DO NOTHING;

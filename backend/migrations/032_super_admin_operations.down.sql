DROP INDEX IF EXISTS idx_companies_deletion_scheduled;
DROP INDEX IF EXISTS idx_login_events_user;
DROP INDEX IF EXISTS idx_login_events_created;
DROP TABLE IF EXISTS login_events;

ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS last_used_at;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS user_agent;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS ip_address;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_enabled;
ALTER TABLE companies DROP COLUMN IF EXISTS deletion_scheduled_at;
ALTER TABLE companies DROP COLUMN IF EXISTS suspended_at;

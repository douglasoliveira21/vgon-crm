DROP TABLE IF EXISTS widget_sessions;
ALTER TABLE users
    DROP COLUMN IF EXISTS two_factor_confirmed_at,
    DROP COLUMN IF EXISTS two_factor_pending_secret,
    DROP COLUMN IF EXISTS two_factor_secret;
DROP INDEX IF EXISTS idx_refresh_tokens_active_session;
DROP INDEX IF EXISTS idx_refresh_tokens_session_id;
ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS device_name,
    DROP COLUMN IF EXISTS revoked_at,
    DROP COLUMN IF EXISTS session_id;

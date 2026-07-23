ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS session_id UUID,
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS device_name VARCHAR(160);

UPDATE refresh_tokens SET session_id = id WHERE session_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_session_id ON refresh_tokens(session_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_active_session
    ON refresh_tokens(user_id, session_id, expires_at)
    WHERE revoked_at IS NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS two_factor_secret TEXT,
    ADD COLUMN IF NOT EXISTS two_factor_pending_secret TEXT,
    ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS two_factor_confirmed_at TIMESTAMP;

CREATE TABLE IF NOT EXISTS widget_sessions (
    id UUID PRIMARY KEY,
    widget_id UUID NOT NULL REFERENCES widgets(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    visitor_id VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_widget_sessions_conversation
    ON widget_sessions(widget_id, conversation_id, visitor_id)
    WHERE revoked_at IS NULL;

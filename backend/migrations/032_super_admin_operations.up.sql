ALTER TABLE companies ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS deletion_scheduled_at TIMESTAMP;

ALTER TABLE users ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN DEFAULT false;

ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS ip_address VARCHAR(80);
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS user_agent TEXT;
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS last_used_at TIMESTAMP DEFAULT NOW();

CREATE TABLE IF NOT EXISTS login_events (
    id UUID PRIMARY KEY,
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    email VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL,
    failure_reason TEXT,
    ip_address VARCHAR(80),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_login_events_created ON login_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_login_events_user ON login_events(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_companies_deletion_scheduled ON companies(deletion_scheduled_at)
WHERE deletion_scheduled_at IS NOT NULL;

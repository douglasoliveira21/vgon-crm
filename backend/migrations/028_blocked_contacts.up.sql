ALTER TABLE contacts ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS blocked_at TIMESTAMP;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS blocked_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_company_blocked
ON contacts(company_id, is_blocked, created_at DESC);

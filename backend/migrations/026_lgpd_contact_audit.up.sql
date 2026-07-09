ALTER TABLE contacts ADD COLUMN IF NOT EXISTS consent_status VARCHAR(30) DEFAULT 'unknown';
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS consent_source VARCHAR(100);
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS consent_text TEXT;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS consent_given_at TIMESTAMP;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS consent_revoked_at TIMESTAMP;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS consent_updated_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS opt_out_reason TEXT;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS opt_out_source VARCHAR(100);

UPDATE contacts
SET consent_status = CASE WHEN COALESCE(is_opted_out, false) THEN 'revoked' ELSE COALESCE(consent_status, 'unknown') END,
    consent_revoked_at = CASE WHEN COALESCE(is_opted_out, false) THEN COALESCE(consent_revoked_at, opted_out_at) ELSE consent_revoked_at END
WHERE consent_status IS NULL OR COALESCE(is_opted_out, false);

CREATE TABLE IF NOT EXISTS contact_consents (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(30) NOT NULL,
    source VARCHAR(100),
    reason TEXT,
    consent_text TEXT,
    ip_address VARCHAR(80),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contact_consents_contact ON contact_consents(contact_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contact_consents_company ON contact_consents(company_id, created_at DESC);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID,
    ip_address VARCHAR(80),
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entity_type VARCHAR(80);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entity_id UUID;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS resource_type VARCHAR(100);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS resource_id UUID;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS details JSONB DEFAULT '{}'::jsonb;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS ip_address VARCHAR(80);
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_agent TEXT;

UPDATE audit_logs
SET entity_type = COALESCE(entity_type, resource_type),
    entity_id = COALESCE(entity_id, resource_id),
    metadata = COALESCE(metadata, details, '{}'::jsonb)
WHERE entity_type IS NULL OR metadata IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_company_created ON audit_logs(company_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id, created_at DESC);

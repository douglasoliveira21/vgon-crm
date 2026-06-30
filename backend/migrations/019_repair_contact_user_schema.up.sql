-- Repair contact/user columns expected by the current CRM screens.
-- Some production databases were created before these fields existed and were
-- then marked as baselined, so incremental migrations never added them.

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_online BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP;
ALTER TABLE users ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS availability_status VARCHAR(20) DEFAULT 'offline';

UPDATE users
SET is_active = COALESCE(is_active, true),
    is_online = COALESCE(is_online, false),
    settings = COALESCE(settings, '{}'::jsonb),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW()),
    availability_status = COALESCE(
        availability_status,
        CASE WHEN COALESCE(is_online, false) THEN 'online' ELSE 'offline' END
    );

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS company_name VARCHAR(255);
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS position VARCHAR(255);
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS city VARCHAR(255);
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS state VARCHAR(100);
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS origin VARCHAR(100);
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS custom_fields JSONB DEFAULT '{}';
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS is_opted_out BOOLEAN DEFAULT false;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS opted_out_at TIMESTAMP;
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;

UPDATE contacts
SET custom_fields = COALESCE(custom_fields, '{}'::jsonb),
    is_opted_out = COALESCE(is_opted_out, false),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

CREATE INDEX IF NOT EXISTS idx_contacts_customer_company ON contacts(customer_company_id);
CREATE INDEX IF NOT EXISTS idx_contacts_assigned ON contacts(assigned_to);

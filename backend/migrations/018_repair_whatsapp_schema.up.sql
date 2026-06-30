-- Repair WhatsApp/channel tables for deployments that were baselined before
-- all Evolution API fields existed.

CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    team_id UUID REFERENCES teams(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'whatsapp',
    status VARCHAR(50) DEFAULT 'disconnected',
    settings JSONB DEFAULT '{}',
    business_hours JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE channels ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES teams(id);
ALTER TABLE channels ADD COLUMN IF NOT EXISTS type VARCHAR(50) NOT NULL DEFAULT 'whatsapp';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'disconnected';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS business_hours JSONB DEFAULT '{}';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE channels ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE channels ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

CREATE TABLE IF NOT EXISTS whatsapp_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    instance_name VARCHAR(255) NOT NULL,
    instance_id VARCHAR(255),
    token VARCHAR(500),
    status VARCHAR(50) DEFAULT 'disconnected',
    qrcode TEXT,
    phone_number VARCHAR(20),
    webhook_url TEXT,
    settings JSONB DEFAULT '{}',
    connected_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES channels(id) ON DELETE SET NULL;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS instance_id VARCHAR(255);
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS token VARCHAR(500);
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'disconnected';
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS qrcode TEXT;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20);
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS webhook_url TEXT;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS connected_at TIMESTAMP;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE channels SET status = COALESCE(status, 'disconnected'), type = COALESCE(type, 'whatsapp'), updated_at = COALESCE(updated_at, NOW());
UPDATE whatsapp_instances SET status = COALESCE(status, 'disconnected'), created_at = COALESCE(created_at, NOW()), updated_at = COALESCE(updated_at, NOW());

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_instances_instance_name_unique ON whatsapp_instances(instance_name);
CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_company ON whatsapp_instances(company_id);
CREATE INDEX IF NOT EXISTS idx_whatsapp_instances_channel ON whatsapp_instances(channel_id);
CREATE INDEX IF NOT EXISTS idx_channels_company_type ON channels(company_id, type, status);

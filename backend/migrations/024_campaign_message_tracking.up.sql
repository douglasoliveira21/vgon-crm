CREATE TABLE IF NOT EXISTS campaign_contact_messages (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    campaign_contact_id UUID NOT NULL REFERENCES campaign_contacts(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL UNIQUE,
    message_type VARCHAR(50) DEFAULT 'text',
    item_index INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'sent',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_contact_messages_campaign_id ON campaign_contact_messages(campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_contact_messages_contact_id ON campaign_contact_messages(campaign_contact_id);
CREATE INDEX IF NOT EXISTS idx_campaign_contact_messages_external_id ON campaign_contact_messages(external_id);

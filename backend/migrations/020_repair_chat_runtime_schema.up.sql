-- Final runtime repair for chat/conversation pages on older baselined databases.
-- These columns are read by /api/conversations, /api/conversations/:id/messages,
-- /api/funnels and /api/deals.

CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id),
    assigned_to UUID REFERENCES users(id),
    team_id UUID REFERENCES teams(id),
    status VARCHAR(50) DEFAULT 'open',
    priority VARCHAR(20) DEFAULT 'normal',
    subject VARCHAR(500),
    last_message_at TIMESTAMP,
    last_message_preview TEXT,
    unread_count INT DEFAULT 0,
    sla_deadline TIMESTAMP,
    first_response_at TIMESTAMP,
    resolved_at TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL,
    first_response_due_at TIMESTAMP,
    resolution_due_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES channels(id);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES teams(id);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'open';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'normal';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS subject VARCHAR(500);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_preview TEXT;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS unread_count INT DEFAULT 0;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS sla_deadline TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolution_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE conversations
SET status = COALESCE(status, 'open'),
    priority = COALESCE(priority, 'normal'),
    unread_count = COALESCE(unread_count, 0),
    metadata = COALESCE(metadata, '{}'::jsonb),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW()),
    last_message_at = COALESCE(last_message_at, updated_at, created_at, NOW());

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL,
    sender_id UUID,
    content TEXT,
    message_type VARCHAR(50) DEFAULT 'text',
    media_url TEXT,
    media_mime_type VARCHAR(100),
    media_filename VARCHAR(255),
    external_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'sent',
    is_private BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    reply_to_content TEXT,
    reply_to_sender VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_id UUID;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_type VARCHAR(50) DEFAULT 'text';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_url TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_mime_type VARCHAR(100);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_filename VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS external_id VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'sent';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();

UPDATE messages
SET message_type = COALESCE(message_type, 'text'),
    status = COALESCE(status, 'sent'),
    is_private = COALESCE(is_private, false),
    metadata = COALESCE(metadata, '{}'::jsonb),
    created_at = COALESCE(created_at, NOW());

ALTER TABLE channels ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'whatsapp';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'disconnected';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE channels ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE channels ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE channels
SET type = COALESCE(type, 'whatsapp'),
    status = COALESCE(status, 'disconnected'),
    settings = COALESCE(settings, '{}'::jsonb),
    is_active = COALESCE(is_active, true),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

ALTER TABLE funnels ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT false;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE funnels
SET is_default = COALESCE(is_default, false),
    is_active = COALESCE(is_active, true),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS color VARCHAR(7) DEFAULT '#3B82F6';
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS position INT DEFAULT 0;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS is_won BOOLEAN DEFAULT false;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS is_lost BOOLEAN DEFAULT false;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();

UPDATE funnel_stages
SET color = COALESCE(color, '#3B82F6'),
    position = COALESCE(position, 0),
    is_won = COALESCE(is_won, false),
    is_lost = COALESCE(is_lost, false),
    created_at = COALESCE(created_at, NOW());

ALTER TABLE deals ADD COLUMN IF NOT EXISTS funnel_id UUID REFERENCES funnels(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS stage_id UUID REFERENCES funnel_stages(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS contact_id UUID REFERENCES contacts(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS conversation_id UUID REFERENCES conversations(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS title VARCHAR(500) DEFAULT 'Negocio';
ALTER TABLE deals ADD COLUMN IF NOT EXISTS value DECIMAL(15,2) DEFAULT 0;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS expected_close_date DATE;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS origin VARCHAR(100);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS loss_reason TEXT;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'open';
ALTER TABLE deals ADD COLUMN IF NOT EXISTS won_at TIMESTAMP;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS lost_at TIMESTAMP;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE deals ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE deals
SET title = COALESCE(title, 'Negocio'),
    value = COALESCE(value, 0),
    status = COALESCE(status, 'open'),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

CREATE INDEX IF NOT EXISTS idx_conversations_company_status_last_message
    ON conversations(company_id, status, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_company_assigned_status_last_message
    ON conversations(company_id, assigned_to, status, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_company_unassigned_status_last_message
    ON conversations(company_id, status, last_message_at DESC)
    WHERE assigned_to IS NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_company_contact_status
    ON conversations(company_id, contact_id, status);
CREATE INDEX IF NOT EXISTS idx_conversations_company_channel_status
    ON conversations(company_id, channel_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_company_created
    ON messages(conversation_id, company_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_messages_company_external_id
    ON messages(company_id, external_id);
CREATE INDEX IF NOT EXISTS idx_deals_company_contact_status
    ON deals(company_id, contact_id, status);
CREATE INDEX IF NOT EXISTS idx_deals_company_stage_status
    ON deals(company_id, stage_id, status);
CREATE INDEX IF NOT EXISTS idx_funnel_stages_funnel_position
    ON funnel_stages(funnel_id, position);

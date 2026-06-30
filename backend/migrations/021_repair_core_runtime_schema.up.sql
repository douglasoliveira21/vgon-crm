-- Repair core CRM tables that may exist partially in older Supabase databases.
-- This migration is intentionally idempotent.

CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255),
    description TEXT,
    distribution_rule VARCHAR(50) DEFAULT 'round-robin',
    business_hours JSONB DEFAULT '{}'::jsonb,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE teams ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE teams ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS distribution_rule VARCHAR(50) DEFAULT 'round-robin';
ALTER TABLE teams ADD COLUMN IF NOT EXISTS business_hours JSONB DEFAULT '{}'::jsonb;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE teams ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE teams ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE teams
SET name = COALESCE(NULLIF(name, ''), 'Time'),
    distribution_rule = COALESCE(NULLIF(distribution_rule, ''), 'round-robin'),
    business_hours = COALESCE(business_hours, '{}'::jsonb),
    is_active = COALESCE(is_active, true),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

CREATE TABLE IF NOT EXISTS team_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE team_users ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES teams(id) ON DELETE CASCADE;
ALTER TABLE team_users ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE team_users ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();

CREATE TABLE IF NOT EXISTS customer_companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255),
    cnpj VARCHAR(20),
    trade_name VARCHAR(255),
    legal_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    city VARCHAR(120),
    state VARCHAR(2),
    address TEXT,
    initial_response_sla_minutes INTEGER DEFAULT 30,
    resolution_sla_minutes INTEGER DEFAULT 240,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS cnpj VARCHAR(20);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS trade_name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS legal_name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS phone VARCHAR(50);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS city VARCHAR(120);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS state VARCHAR(2);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS initial_response_sla_minutes INTEGER DEFAULT 30;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS resolution_sla_minutes INTEGER DEFAULT 240;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE customer_companies
SET name = COALESCE(NULLIF(name, ''), trade_name, legal_name, 'Empresa'),
    initial_response_sla_minutes = COALESCE(NULLIF(initial_response_sla_minutes, 0), 30),
    resolution_sla_minutes = COALESCE(NULLIF(resolution_sla_minutes, 0), 240),
    is_active = COALESCE(is_active, true),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS channel_id UUID REFERENCES channels(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS team_id UUID REFERENCES teams(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'open';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'normal';
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS subject VARCHAR(255);
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_preview TEXT;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS unread_count INTEGER DEFAULT 0;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolution_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE conversations
SET status = COALESCE(NULLIF(status, ''), 'open'),
    priority = COALESCE(NULLIF(priority, ''), 'normal'),
    unread_count = COALESCE(unread_count, 0),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

ALTER TABLE messages ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_type VARCHAR(20) DEFAULT 'system';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_type VARCHAR(20) DEFAULT 'text';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_url TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_mime_type VARCHAR(120);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_filename VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS external_id VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'sent';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_internal BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();

UPDATE messages
SET sender_type = COALESCE(NULLIF(sender_type, ''), 'system'),
    message_type = COALESCE(NULLIF(message_type, ''), 'text'),
    status = COALESCE(NULLIF(status, ''), 'sent'),
    metadata = COALESCE(metadata, '{}'::jsonb),
    is_internal = COALESCE(is_internal, false),
    is_private = COALESCE(is_private, false),
    created_at = COALESCE(created_at, NOW());

ALTER TABLE funnels ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT false;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE funnels
SET name = COALESCE(NULLIF(name, ''), 'Funil'),
    is_default = COALESCE(is_default, false),
    is_active = COALESCE(is_active, true),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS funnel_id UUID REFERENCES funnels(id) ON DELETE CASCADE;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS color VARCHAR(20) DEFAULT '#3B82F6';
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS position INTEGER DEFAULT 0;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS is_won BOOLEAN DEFAULT false;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS is_lost BOOLEAN DEFAULT false;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE funnel_stages
SET name = COALESCE(NULLIF(name, ''), 'Etapa'),
    color = COALESCE(NULLIF(color, ''), '#3B82F6'),
    position = COALESCE(position, 0),
    is_won = COALESCE(is_won, false),
    is_lost = COALESCE(is_lost, false),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

ALTER TABLE deals ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS funnel_id UUID REFERENCES funnels(id) ON DELETE SET NULL;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS stage_id UUID REFERENCES funnel_stages(id) ON DELETE SET NULL;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS title VARCHAR(255);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS value NUMERIC(12,2) DEFAULT 0;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'open';
ALTER TABLE deals ADD COLUMN IF NOT EXISTS loss_reason TEXT;
ALTER TABLE deals ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE deals ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

UPDATE deals
SET title = COALESCE(NULLIF(title, ''), 'Negocio'),
    value = COALESCE(value, 0),
    status = COALESCE(NULLIF(status, ''), 'open'),
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

CREATE INDEX IF NOT EXISTS idx_teams_company ON teams(company_id);
CREATE INDEX IF NOT EXISTS idx_team_users_team ON team_users(team_id);
CREATE INDEX IF NOT EXISTS idx_team_users_user ON team_users(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_team_users_team_user_unique ON team_users(team_id, user_id);
CREATE INDEX IF NOT EXISTS idx_customer_companies_company ON customer_companies(company_id);
CREATE INDEX IF NOT EXISTS idx_conversations_company_status ON conversations(company_id, status);
CREATE INDEX IF NOT EXISTS idx_conversations_contact ON conversations(contact_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_funnels_company ON funnels(company_id);
CREATE INDEX IF NOT EXISTS idx_funnel_stages_funnel ON funnel_stages(funnel_id);
CREATE INDEX IF NOT EXISTS idx_deals_company ON deals(company_id);

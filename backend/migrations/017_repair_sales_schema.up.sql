-- Repair sales/funnel tables for older databases that were marked as baseline
-- before these structures existed.

CREATE TABLE IF NOT EXISTS funnels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE funnels ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS is_default BOOLEAN DEFAULT false;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE funnels ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

CREATE TABLE IF NOT EXISTS funnel_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(7) DEFAULT '#3B82F6',
    position INT NOT NULL DEFAULT 0,
    is_won BOOLEAN DEFAULT false,
    is_lost BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS color VARCHAR(7) DEFAULT '#3B82F6';
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS position INT NOT NULL DEFAULT 0;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS is_won BOOLEAN DEFAULT false;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS is_lost BOOLEAN DEFAULT false;
ALTER TABLE funnel_stages ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();

CREATE TABLE IF NOT EXISTS deals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    funnel_id UUID REFERENCES funnels(id),
    stage_id UUID REFERENCES funnel_stages(id),
    contact_id UUID REFERENCES contacts(id),
    conversation_id UUID REFERENCES conversations(id),
    assigned_to UUID REFERENCES users(id),
    title VARCHAR(500) NOT NULL DEFAULT 'Negocio',
    value DECIMAL(15,2) DEFAULT 0,
    expected_close_date DATE,
    origin VARCHAR(100),
    loss_reason TEXT,
    notes TEXT,
    status VARCHAR(50) DEFAULT 'open',
    won_at TIMESTAMP,
    lost_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE deals ADD COLUMN IF NOT EXISTS funnel_id UUID REFERENCES funnels(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS stage_id UUID REFERENCES funnel_stages(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS contact_id UUID REFERENCES contacts(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS conversation_id UUID REFERENCES conversations(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS assigned_to UUID REFERENCES users(id);
ALTER TABLE deals ADD COLUMN IF NOT EXISTS title VARCHAR(500) NOT NULL DEFAULT 'Negocio';
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

CREATE INDEX IF NOT EXISTS idx_deals_company_funnel ON deals(company_id, funnel_id, status);
CREATE INDEX IF NOT EXISTS idx_deals_contact ON deals(company_id, contact_id, status);

-- Reinforce conversation/message columns that the conversations page needs.
CREATE TABLE IF NOT EXISTS customer_companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    cnpj VARCHAR(32),
    trade_name VARCHAR(255),
    legal_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    city VARCHAR(120),
    state VARCHAR(50),
    address TEXT,
    initial_response_sla_minutes INT NOT NULL DEFAULT 30,
    resolution_sla_minutes INT NOT NULL DEFAULT 240,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS cnpj VARCHAR(32);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS trade_name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS legal_name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS phone VARCHAR(50);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS city VARCHAR(120);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS state VARCHAR(50);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS initial_response_sla_minutes INT NOT NULL DEFAULT 30;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS resolution_sla_minutes INT NOT NULL DEFAULT 240;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolution_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_customer_companies_company ON customer_companies(company_id);
CREATE INDEX IF NOT EXISTS idx_contacts_customer_company ON contacts(customer_company_id);
CREATE INDEX IF NOT EXISTS idx_conversations_customer_company ON conversations(customer_company_id);

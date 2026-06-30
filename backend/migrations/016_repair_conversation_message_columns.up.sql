CREATE TABLE IF NOT EXISTS customer_companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    cnpj VARCHAR(18),
    trade_name VARCHAR(255),
    legal_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(30),
    city VARCHAR(120),
    state VARCHAR(2),
    address TEXT,
    initial_response_sla_minutes INT NOT NULL DEFAULT 30,
    resolution_sla_minutes INT NOT NULL DEFAULT 240,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS cnpj VARCHAR(18);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS trade_name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS legal_name VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS phone VARCHAR(30);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS city VARCHAR(120);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS state VARCHAR(2);
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS address TEXT;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS initial_response_sla_minutes INT NOT NULL DEFAULT 30;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS resolution_sla_minutes INT NOT NULL DEFAULT 240;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE customer_companies ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_companies_company_cnpj
    ON customer_companies(company_id, cnpj)
    WHERE cnpj IS NOT NULL AND cnpj <> '';

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolution_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;

ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_sender VARCHAR(255);

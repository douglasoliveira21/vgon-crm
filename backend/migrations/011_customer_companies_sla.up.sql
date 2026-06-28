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

CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_companies_company_cnpj
    ON customer_companies(company_id, cnpj)
    WHERE cnpj IS NOT NULL AND cnpj <> '';

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS customer_company_id UUID REFERENCES customer_companies(id) ON DELETE SET NULL;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS first_response_due_at TIMESTAMP;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS resolution_due_at TIMESTAMP;

UPDATE contacts c
SET customer_company_id = cc.id
FROM customer_companies cc
WHERE c.customer_company_id IS NULL
  AND c.company_id = cc.company_id
  AND c.company_name IS NOT NULL
  AND LOWER(TRIM(c.company_name)) = LOWER(TRIM(cc.name));

UPDATE conversations conv
SET customer_company_id = c.customer_company_id
FROM contacts c
WHERE conv.contact_id = c.id
  AND conv.customer_company_id IS NULL
  AND c.customer_company_id IS NOT NULL;

UPDATE conversations conv
SET first_response_due_at = COALESCE(conv.first_response_due_at, conv.created_at + (cc.initial_response_sla_minutes || ' minutes')::interval),
    resolution_due_at = COALESCE(conv.resolution_due_at, conv.created_at + (cc.resolution_sla_minutes || ' minutes')::interval)
FROM customer_companies cc
WHERE conv.customer_company_id = cc.id;

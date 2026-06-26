CREATE TABLE IF NOT EXISTS telephony_ivr (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    welcome_message TEXT NOT NULL DEFAULT 'Bem-vindo. Escolha uma opção.',
    business_hours_start VARCHAR(5) DEFAULT '08:00',
    business_hours_end VARCHAR(5) DEFAULT '18:00',
    off_hours_message TEXT DEFAULT 'Estamos fora do horário de atendimento.',
    timeout_message TEXT DEFAULT 'Não entendi sua opção. Tente novamente.',
    invalid_message TEXT DEFAULT 'Opção inválida. Tente novamente.',
    max_retries INT DEFAULT 3,
    options JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_telephony_ivr_company ON telephony_ivr(company_id);

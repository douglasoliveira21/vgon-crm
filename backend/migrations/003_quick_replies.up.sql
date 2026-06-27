-- Quick Replies (Respostas Prontas)
CREATE TABLE IF NOT EXISTS quick_replies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    shortcut VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    category VARCHAR(100),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE quick_replies ADD COLUMN IF NOT EXISTS category VARCHAR(100);
ALTER TABLE quick_replies ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_quick_replies_company ON quick_replies(company_id);
CREATE INDEX IF NOT EXISTS idx_quick_replies_shortcut ON quick_replies(company_id, shortcut);

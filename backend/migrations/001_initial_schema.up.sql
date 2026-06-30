-- EvoCRM Pro - Initial Schema Migration
-- Multiempresa/Multitenant SaaS CRM

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- COMPANIES (Empresas/Tenants)
-- ============================================
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    document VARCHAR(20),
    email VARCHAR(255),
    phone VARCHAR(20),
    logo_url TEXT,
    plan VARCHAR(50) DEFAULT 'free',
    max_users INT DEFAULT 5,
    max_channels INT DEFAULT 2,
    is_active BOOLEAN DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- ROLES (Cargos)
-- ============================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    description TEXT,
    permissions JSONB DEFAULT '[]',
    is_system BOOLEAN DEFAULT false,
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- USERS (Usuários)
-- ============================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT true,
    is_online BOOLEAN DEFAULT false,
    last_seen_at TIMESTAMP,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(email, company_id)
);

-- ============================================
-- TEAMS (Times)
-- ============================================
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    distribution_rule VARCHAR(50) DEFAULT 'round-robin',
    business_hours JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE team_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_supervisor BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(team_id, user_id)
);

-- ============================================
-- CHANNELS (Canais)
-- ============================================
CREATE TABLE channels (
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

-- ============================================
-- WHATSAPP INSTANCES (Instâncias Evolution API)
-- ============================================
CREATE TABLE whatsapp_instances (
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
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(instance_name)
);

-- ============================================
-- CONTACTS (Contatos)
-- ============================================
CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255),
    phone VARCHAR(20),
    email VARCHAR(255),
    company_name VARCHAR(255),
    position VARCHAR(255),
    city VARCHAR(255),
    state VARCHAR(100),
    origin VARCHAR(100),
    avatar_url TEXT,
    notes TEXT,
    assigned_to UUID REFERENCES users(id),
    custom_fields JSONB DEFAULT '{}',
    is_opted_out BOOLEAN DEFAULT false,
    opted_out_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_contacts_phone_company ON contacts(phone, company_id) WHERE phone IS NOT NULL;

-- ============================================
-- TAGS (Etiquetas)
-- ============================================
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) DEFAULT '#3B82F6',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE contact_tags (
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (contact_id, tag_id)
);

-- ============================================
-- CONVERSATIONS (Conversas)
-- ============================================
CREATE TABLE conversations (
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
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_conversations_company_status ON conversations(company_id, status);
CREATE INDEX idx_conversations_assigned ON conversations(assigned_to, status);
CREATE INDEX idx_conversations_team ON conversations(team_id, status);

-- ============================================
-- MESSAGES (Mensagens)
-- ============================================
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL, -- 'contact', 'user', 'system', 'bot'
    sender_id UUID,
    content TEXT,
    message_type VARCHAR(50) DEFAULT 'text', -- text, image, audio, video, document, sticker, location
    media_url TEXT,
    media_mime_type VARCHAR(100),
    media_filename VARCHAR(255),
    external_id VARCHAR(255),
    status VARCHAR(20) DEFAULT 'sent', -- sent, delivered, read, failed
    is_private BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    reply_to_content TEXT,
    reply_to_sender VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_external_id ON messages(external_id);

-- ============================================
-- MESSAGE ATTACHMENTS
-- ============================================
CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_url TEXT NOT NULL,
    file_name VARCHAR(255),
    file_size BIGINT,
    mime_type VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- CONVERSATION NOTES (Observações internas)
-- ============================================
CREATE TABLE conversation_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- QUICK REPLIES (Respostas rápidas)
-- ============================================
CREATE TABLE quick_replies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    shortcut VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    attachment_url TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- FUNNELS (Funis de vendas)
-- ============================================
CREATE TABLE funnels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE funnel_stages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(7) DEFAULT '#3B82F6',
    position INT NOT NULL DEFAULT 0,
    is_won BOOLEAN DEFAULT false,
    is_lost BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- DEALS (Oportunidades/Negócios)
-- ============================================
CREATE TABLE deals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    funnel_id UUID NOT NULL REFERENCES funnels(id),
    stage_id UUID NOT NULL REFERENCES funnel_stages(id),
    contact_id UUID REFERENCES contacts(id),
    conversation_id UUID REFERENCES conversations(id),
    assigned_to UUID REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    value DECIMAL(15,2) DEFAULT 0,
    expected_close_date DATE,
    origin VARCHAR(100),
    loss_reason TEXT,
    notes TEXT,
    status VARCHAR(50) DEFAULT 'open', -- open, won, lost
    won_at TIMESTAMP,
    lost_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_deals_company_funnel ON deals(company_id, funnel_id, status);

-- ============================================
-- BOT FLOWS (Fluxos de bots)
-- ============================================
CREATE TABLE bot_flows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_type VARCHAR(50) NOT NULL, -- new_conversation, keyword, off_hours, no_response, transfer, funnel_stage, tag_added
    trigger_value TEXT,
    channel_id UUID REFERENCES channels(id),
    is_active BOOLEAN DEFAULT false,
    priority INTEGER NOT NULL DEFAULT 10,
    stop_on_match BOOLEAN NOT NULL DEFAULT true,
    nodes JSONB DEFAULT '[]',
    edges JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE bot_nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flow_id UUID NOT NULL REFERENCES bot_flows(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- send_message, send_media, ask_question, transfer_team, transfer_agent, add_tag, change_stage, call_webhook, condition, delay
    position_x FLOAT DEFAULT 0,
    position_y FLOAT DEFAULT 0,
    data JSONB DEFAULT '{}',
    next_node_id UUID,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE bot_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flow_id UUID NOT NULL REFERENCES bot_flows(id),
    conversation_id UUID REFERENCES conversations(id),
    contact_id UUID REFERENCES contacts(id),
    current_node_id UUID,
    status VARCHAR(50) DEFAULT 'running', -- running, completed, paused, error
    context JSONB DEFAULT '{}',
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- ============================================
-- CAMPAIGNS (Campanhas de marketing)
-- ============================================
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    message_content TEXT,
    message_type VARCHAR(50) DEFAULT 'text',
    media_url TEXT,
    variables JSONB DEFAULT '[]',
    filter_tags UUID[],
    filter_funnel_id UUID,
    filter_stage_id UUID,
    filter_city VARCHAR(255),
    filter_origin VARCHAR(100),
    status VARCHAR(50) DEFAULT 'draft', -- draft, scheduled, sending, completed, paused, cancelled
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    send_speed INT DEFAULT 30, -- messages per minute
    total_contacts INT DEFAULT 0,
    sent_count INT DEFAULT 0,
    delivered_count INT DEFAULT 0,
    read_count INT DEFAULT 0,
    replied_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE campaign_contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id),
    status VARCHAR(50) DEFAULT 'pending', -- pending, sent, delivered, read, replied, failed
    sent_at TIMESTAMP,
    delivered_at TIMESTAMP,
    read_at TIMESTAMP,
    replied_at TIMESTAMP,
    error_message TEXT
);

-- ============================================
-- CALLS (Chamadas de áudio)
-- ============================================
CREATE TABLE calls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES contacts(id),
    user_id UUID REFERENCES users(id),
    conversation_id UUID REFERENCES conversations(id),
    channel_id UUID REFERENCES channels(id),
    direction VARCHAR(20) DEFAULT 'outbound', -- inbound, outbound
    status VARCHAR(50) DEFAULT 'initiated', -- initiated, ringing, answered, completed, missed, declined, no_answer
    duration INT DEFAULT 0, -- seconds
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    recording_url TEXT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- WIDGETS (Widget para site)
-- ============================================
CREATE TABLE widgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    channel_id UUID REFERENCES channels(id),
    team_id UUID REFERENCES teams(id),
    primary_color VARCHAR(7) DEFAULT '#3B82F6',
    greeting_message TEXT DEFAULT 'Olá! Como podemos ajudar?',
    channels_displayed JSONB DEFAULT '["whatsapp"]',
    position VARCHAR(20) DEFAULT 'bottom-right',
    show_pre_chat_form BOOLEAN DEFAULT true,
    pre_chat_fields JSONB DEFAULT '["name", "phone"]',
    business_hours JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- INTERNAL ANNOUNCEMENTS (Recados internos)
-- ============================================
CREATE TABLE internal_announcements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high, urgent
    target_type VARCHAR(50) DEFAULT 'all', -- all, team
    target_team_id UUID REFERENCES teams(id),
    is_pinned BOOLEAN DEFAULT false,
    published_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE announcement_reads (
    announcement_id UUID NOT NULL REFERENCES internal_announcements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (announcement_id, user_id)
);

-- ============================================
-- AUDIT LOGS (Logs de auditoria)
-- ============================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_company ON audit_logs(company_id, created_at DESC);

-- ============================================
-- REFRESH TOKENS
-- ============================================
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================
-- SEED: Default Roles
-- ============================================
INSERT INTO roles (id, name, slug, description, permissions, is_system) VALUES
    (uuid_generate_v4(), 'Administrador Geral', 'super-admin', 'Acesso total ao sistema', '["*"]', true),
    (uuid_generate_v4(), 'Administrador', 'admin', 'Administrador da empresa', '["company.*", "users.*", "teams.*", "channels.*", "conversations.*", "contacts.*", "funnels.*", "bots.*", "campaigns.*", "metrics.*", "widget.*", "announcements.*"]', true),
    (uuid_generate_v4(), 'Supervisor', 'supervisor', 'Supervisor de time', '["conversations.*", "contacts.*", "teams.view", "metrics.view", "funnels.*"]', true),
    (uuid_generate_v4(), 'Atendente', 'agent', 'Atendente de conversas', '["conversations.own", "contacts.view", "contacts.edit", "funnels.view"]', true),
    (uuid_generate_v4(), 'Vendedor', 'sales', 'Vendedor com foco em funil', '["conversations.own", "contacts.*", "funnels.*", "deals.*"]', true),
    (uuid_generate_v4(), 'Marketing', 'marketing', 'Responsável por campanhas', '["campaigns.*", "contacts.view", "metrics.campaigns"]', true),
    (uuid_generate_v4(), 'Financeiro', 'finance', 'Acesso a relatórios financeiros', '["metrics.view", "deals.view", "funnels.view"]', true);

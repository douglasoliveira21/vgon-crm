CREATE TABLE IF NOT EXISTS durable_jobs (
    id UUID PRIMARY KEY,
    company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
    job_type VARCHAR(80) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    available_at TIMESTAMP NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMP,
    locked_by VARCHAR(160),
    last_error TEXT,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(job_type, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_durable_jobs_claim
    ON durable_jobs(status, available_at, created_at);

CREATE TABLE IF NOT EXISTS dead_letter_jobs (
    id UUID PRIMARY KEY,
    original_job_id UUID,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    job_type VARCHAR(80) NOT NULL,
    payload JSONB NOT NULL,
    attempts INT NOT NULL,
    last_error TEXT,
    failed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE bot_flows
    ADD COLUMN IF NOT EXISTS lifecycle_status VARCHAR(20) NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS published_version INT,
    ADD COLUMN IF NOT EXISTS current_version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS published_by UUID REFERENCES users(id) ON DELETE SET NULL;

UPDATE bot_flows
SET lifecycle_status = CASE WHEN is_active THEN 'published' ELSE 'draft' END,
    published_version = CASE WHEN is_active THEN 1 ELSE NULL END,
    published_at = CASE WHEN is_active THEN COALESCE(updated_at, created_at) ELSE NULL END
WHERE lifecycle_status = 'draft' AND current_version = 1;

CREATE TABLE IF NOT EXISTS bot_flow_versions (
    id UUID PRIMARY KEY,
    flow_id UUID NOT NULL REFERENCES bot_flows(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    version INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    bot_name VARCHAR(255),
    description TEXT,
    trigger_type VARCHAR(100),
    trigger_value TEXT,
    priority INT,
    stop_on_match BOOLEAN,
    nodes JSONB NOT NULL DEFAULT '[]',
    edges JSONB NOT NULL DEFAULT '[]',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(flow_id, version)
);
ALTER TABLE bot_flow_versions
    ADD COLUMN IF NOT EXISTS bot_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS priority INT,
    ADD COLUMN IF NOT EXISTS stop_on_match BOOLEAN;

INSERT INTO bot_flow_versions
    (id, flow_id, company_id, version, name, bot_name, description, trigger_type,
     trigger_value, priority, stop_on_match, nodes, edges, created_at)
SELECT
    bf.id, bf.id, bf.company_id, 1, bf.name, bf.bot_name, bf.description, bf.trigger_type,
    bf.trigger_value, bf.priority, bf.stop_on_match, COALESCE(bf.nodes, '[]'::jsonb),
    COALESCE(bf.edges, '[]'::jsonb), COALESCE(bf.updated_at, bf.created_at, NOW())
FROM bot_flows bf
WHERE bf.is_active = true
ON CONFLICT (flow_id, version) DO NOTHING;

CREATE TABLE IF NOT EXISTS bot_execution_events (
    id UUID PRIMARY KEY,
    execution_id UUID NOT NULL REFERENCES bot_executions(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    node_id VARCHAR(255),
    event_type VARCHAR(60) NOT NULL,
    status VARCHAR(30) NOT NULL,
    input JSONB NOT NULL DEFAULT '{}',
    output JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_bot_execution_events_execution
    ON bot_execution_events(execution_id, created_at);

ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS timezone VARCHAR(80) NOT NULL DEFAULT 'America/Sao_Paulo',
    ADD COLUMN IF NOT EXISTS approval_status VARCHAR(20) NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS frequency_cap_days INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS campaign_suppressions (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    channel VARCHAR(30) NOT NULL,
    destination VARCHAR(320) NOT NULL,
    reason VARCHAR(120),
    source VARCHAR(80),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, channel, destination)
);

CREATE TABLE IF NOT EXISTS campaign_delivery_events (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    channel VARCHAR(30) NOT NULL,
    destination VARCHAR(320),
    event_type VARCHAR(40) NOT NULL,
    provider_event_id VARCHAR(255),
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, provider_event_id, event_type)
);

CREATE TABLE IF NOT EXISTS contact_channel_consents (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel VARCHAR(30) NOT NULL,
    purpose VARCHAR(60) NOT NULL DEFAULT 'marketing',
    status VARCHAR(30) NOT NULL,
    source VARCHAR(100),
    consent_text TEXT,
    proof JSONB NOT NULL DEFAULT '{}',
    given_at TIMESTAMP,
    revoked_at TIMESTAMP,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(contact_id, channel, purpose)
);

CREATE TABLE IF NOT EXISTS data_subject_requests (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    request_type VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'open',
    requester_name VARCHAR(255),
    requester_email VARCHAR(320),
    notes TEXT,
    due_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS data_retention_policies (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    resource_type VARCHAR(40) NOT NULL,
    retention_days INT NOT NULL CHECK (retention_days >= 30),
    action VARCHAR(20) NOT NULL DEFAULT 'anonymize',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, resource_type)
);

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS previous_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS entry_hash VARCHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_entry_hash
    ON audit_logs(entry_hash) WHERE entry_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS contact_merge_history (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    primary_contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE RESTRICT,
    merged_contact_id UUID,
    merged_snapshot JSONB NOT NULL,
    merged_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

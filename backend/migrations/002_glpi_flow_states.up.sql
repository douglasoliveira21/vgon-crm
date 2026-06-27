-- GLPI Flow States - tracks multi-step GLPI ticket creation conversations
CREATE TABLE IF NOT EXISTS glpi_flow_states (
    conversation_id UUID PRIMARY KEY,
    state JSONB NOT NULL DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_glpi_flow_states_updated ON glpi_flow_states(updated_at);

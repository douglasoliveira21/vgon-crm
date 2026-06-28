-- Separate SIP operators/trunks from WebRTC extensions.

CREATE TABLE IF NOT EXISTS sip_trunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    sip_server VARCHAR(255) NOT NULL,
    sip_port INT NOT NULL DEFAULT 5060,
    transport VARCHAR(20) NOT NULL DEFAULT 'UDP',
    sip_domain VARCHAR(255) DEFAULT '',
    username VARCHAR(255) DEFAULT '',
    password TEXT DEFAULT '',
    caller_id VARCHAR(100) DEFAULT '',
    realm VARCHAR(255) DEFAULT '',
    outbound_proxy TEXT DEFAULT '',
    codecs JSONB DEFAULT '["ulaw","alaw"]',
    nat BOOLEAN DEFAULT true,
    keep_alive INT DEFAULT 60,
    dtmf VARCHAR(50) DEFAULT 'rfc4733',
    register_expires INT DEFAULT 300,
    pjsip_config TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sip_trunks_company ON sip_trunks(company_id, is_active);

ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS outbound_trunk_id UUID REFERENCES sip_trunks(id) ON DELETE SET NULL;
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS queue_id UUID REFERENCES call_queues(id) ON DELETE SET NULL;
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS group_name VARCHAR(100) DEFAULT '';
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS webrtc_domain VARCHAR(255) DEFAULT 'voip.vgon.com.br';
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS webrtc_ws_url TEXT DEFAULT 'wss://voip.vgon.com.br:8089/ws';
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS stun_server TEXT DEFAULT 'stun:stun.l.google.com:19302';
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS sip_username VARCHAR(255) DEFAULT '';

UPDATE phone_extensions
SET sip_username = extension_number
WHERE COALESCE(sip_username, '') = '';

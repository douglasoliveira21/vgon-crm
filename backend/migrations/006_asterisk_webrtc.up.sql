-- Asterisk WebRTC integration for the existing telephony module.

CREATE TABLE IF NOT EXISTS telephony_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT 'Asterisk VGoN',
    provider_type VARCHAR(50) NOT NULL DEFAULT 'asterisk',
    sip_host VARCHAR(255) NOT NULL DEFAULT 'voip.vgon.com.br',
    sip_port INT NOT NULL DEFAULT 5060,
    sip_user VARCHAR(255) DEFAULT '',
    sip_password TEXT DEFAULT '',
    sip_domain VARCHAR(255) NOT NULL DEFAULT 'voip.vgon.com.br',
    webrtc_domain VARCHAR(255) NOT NULL DEFAULT 'voip.vgon.com.br',
    webrtc_ws_url TEXT NOT NULL DEFAULT 'wss://voip.vgon.com.br:8089/ws',
    transport VARCHAR(20) NOT NULL DEFAULT 'WSS',
    caller_id VARCHAR(100) DEFAULT '',
    stun_server TEXT DEFAULT 'stun:stun.l.google.com:19302',
    ari_url TEXT NOT NULL DEFAULT 'http://voip.vgon.com.br:8088/ari',
    ari_user VARCHAR(255) DEFAULT '',
    ari_password TEXT DEFAULT '',
    ami_host VARCHAR(255) NOT NULL DEFAULT '85.239.248.224',
    ami_port INT NOT NULL DEFAULT 5038,
    ami_user VARCHAR(255) DEFAULT '',
    ami_password TEXT DEFAULT '',
    recording_path TEXT NOT NULL DEFAULT '/var/spool/asterisk/monitor',
    recording_enabled BOOLEAN DEFAULT true,
    status VARCHAR(50) DEFAULT 'offline',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS webrtc_domain VARCHAR(255) NOT NULL DEFAULT 'voip.vgon.com.br';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS webrtc_ws_url TEXT NOT NULL DEFAULT 'wss://voip.vgon.com.br:8089/ws';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ari_url TEXT NOT NULL DEFAULT 'http://voip.vgon.com.br:8088/ari';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ari_user VARCHAR(255) DEFAULT '';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ari_password TEXT DEFAULT '';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ami_host VARCHAR(255) NOT NULL DEFAULT '85.239.248.224';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ami_port INT NOT NULL DEFAULT 5038;
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ami_user VARCHAR(255) DEFAULT '';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS ami_password TEXT DEFAULT '';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS recording_path TEXT NOT NULL DEFAULT '/var/spool/asterisk/monitor';
ALTER TABLE telephony_providers ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

CREATE TABLE IF NOT EXISTS phone_extensions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    display_name VARCHAR(255) NOT NULL,
    extension_number VARCHAR(50) NOT NULL,
    extension_password TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'offline',
    can_call_external BOOLEAN DEFAULT true,
    can_receive_calls BOOLEAN DEFAULT true,
    can_transfer BOOLEAN DEFAULT true,
    can_access_recordings BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(company_id, extension_number)
);

ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS can_receive_calls BOOLEAN DEFAULT true;
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS can_transfer BOOLEAN DEFAULT true;
ALTER TABLE phone_extensions ADD COLUMN IF NOT EXISTS can_access_recordings BOOLEAN DEFAULT false;

CREATE TABLE IF NOT EXISTS call_queues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    strategy VARCHAR(50) DEFAULT 'ringall',
    max_wait_time INT DEFAULT 120,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS call_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    extension_id UUID REFERENCES phone_extensions(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    channel_id TEXT,
    call_direction VARCHAR(20) NOT NULL,
    call_status VARCHAR(50) NOT NULL DEFAULT 'initiated',
    from_number VARCHAR(80) DEFAULT '',
    to_number VARCHAR(80) DEFAULT '',
    started_at TIMESTAMP DEFAULT NOW(),
    answered_at TIMESTAMP,
    ended_at TIMESTAMP,
    duration_seconds INT DEFAULT 0,
    recording_url TEXT,
    notes TEXT,
    call_result VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE call_records ADD COLUMN IF NOT EXISTS extension_id UUID REFERENCES phone_extensions(id) ON DELETE SET NULL;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS channel_id TEXT;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS call_direction VARCHAR(20) NOT NULL DEFAULT 'outbound';
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS call_status VARCHAR(50) NOT NULL DEFAULT 'initiated';
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS from_number VARCHAR(80) DEFAULT '';
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS to_number VARCHAR(80) DEFAULT '';
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS started_at TIMESTAMP DEFAULT NOW();
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS answered_at TIMESTAMP;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS ended_at TIMESTAMP;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS duration_seconds INT DEFAULT 0;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS recording_url TEXT;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS call_result VARCHAR(100);
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();
ALTER TABLE call_records ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_call_records_company_created ON call_records(company_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_call_records_contact ON call_records(contact_id);
CREATE INDEX IF NOT EXISTS idx_phone_extensions_user ON phone_extensions(user_id);

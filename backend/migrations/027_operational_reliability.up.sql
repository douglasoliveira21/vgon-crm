ALTER TABLE campaign_contacts ADD COLUMN IF NOT EXISTS retry_count INT DEFAULT 0;
ALTER TABLE campaign_contacts ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMP DEFAULT NOW();
ALTER TABLE campaign_contacts ADD COLUMN IF NOT EXISTS locked_at TIMESTAMP;
ALTER TABLE campaign_contacts ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_campaign_contacts_queue
ON campaign_contacts(campaign_id, status, next_attempt_at, id);

ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS last_status_check_at TIMESTAMP;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS last_status_error TEXT;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS disconnected_alerted_at TIMESTAMP;
ALTER TABLE whatsapp_instances ADD COLUMN IF NOT EXISTS last_known_connected_at TIMESTAMP;

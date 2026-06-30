ALTER TABLE bot_flows
ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 10,
ADD COLUMN IF NOT EXISTS stop_on_match BOOLEAN NOT NULL DEFAULT true;

UPDATE bot_flows
SET priority = CASE
    WHEN trigger_type IN ('off_hours', 'trigger_off_hours') THEN 100
    WHEN trigger_type IN ('no_response', 'trigger_no_response') THEN 80
    WHEN trigger_type IN ('new_conversation', 'trigger_new_conversation', 'trigger_inbox_message') THEN 10
    ELSE COALESCE(priority, 10)
END
WHERE priority = 10 OR priority IS NULL;

CREATE INDEX IF NOT EXISTS idx_bot_flows_company_active_priority
ON bot_flows(company_id, is_active, priority DESC, created_at ASC);

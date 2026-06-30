ALTER TABLE bot_flows ADD COLUMN IF NOT EXISTS bot_name VARCHAR(120);

UPDATE bot_flows
SET bot_name = COALESCE(NULLIF(bot_name, ''), 'Assistente')
WHERE bot_name IS NULL OR bot_name = '';

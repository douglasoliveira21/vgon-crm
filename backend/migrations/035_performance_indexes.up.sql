CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_conversations_company_status_assigned ON conversations(company_id, status, assigned_to);
CREATE INDEX IF NOT EXISTS idx_conversations_company_unread ON conversations(company_id) WHERE unread_count > 0;

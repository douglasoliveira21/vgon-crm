-- Index for fast lookup of widget contacts by visitor_id stored in custom_fields
CREATE INDEX IF NOT EXISTS idx_contacts_visitor_id
ON contacts ((custom_fields->>'visitor_id'))
WHERE custom_fields->>'visitor_id' IS NOT NULL;

-- Indexes for queries that were doing sequential scans on tables that grow
-- without bound.

-- The campaign frequency cap runs a NOT EXISTS against campaign_contacts once
-- per recipient, so sending to a large list scanned the whole table for every
-- contact in it.
CREATE INDEX IF NOT EXISTS idx_campaign_contacts_contact
    ON campaign_contacts(contact_id);

-- Resuming a campaign selects its next pending recipient on every iteration.
CREATE INDEX IF NOT EXISTS idx_campaign_contacts_campaign_status
    ON campaign_contacts(campaign_id, status);

-- Dashboard agent productivity aggregates messages by who sent them.
CREATE INDEX IF NOT EXISTS idx_messages_company_sender
    ON messages(company_id, sender_id);

-- Inbound webhook deduplication looks a message up by its WhatsApp id before
-- inserting, on every single incoming message.
CREATE INDEX IF NOT EXISTS idx_messages_company_external
    ON messages(company_id, external_id)
    WHERE external_id IS NOT NULL;

-- Resuming an interrupted multi-message campaign checks which items a
-- recipient already received.
CREATE INDEX IF NOT EXISTS idx_campaign_contact_messages_recipient_item
    ON campaign_contact_messages(campaign_contact_id, item_index);

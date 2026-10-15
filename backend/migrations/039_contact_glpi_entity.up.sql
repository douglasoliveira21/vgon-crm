-- Remember a GLPI CNPJ that was validated against the GLPI entity database for
-- a contact, so the bot flow does not need to ask the CNPJ again on future
-- conversations with the same phone number.
--
-- Only CNPJs confirmed to exist in GLPI are written here (see the matches-found
-- branch of handleCNPJLookup). Invalid or not-found CNPJs are never persisted.
ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS glpi_cnpj VARCHAR(14),
    ADD COLUMN IF NOT EXISTS glpi_entity_id INTEGER,
    ADD COLUMN IF NOT EXISTS glpi_entity_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS glpi_validated_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_contacts_glpi_cnpj
    ON contacts(company_id, glpi_cnpj)
    WHERE glpi_cnpj IS NOT NULL;

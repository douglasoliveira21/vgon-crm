-- Add tsvector column to contacts for faster search
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create index on the tsvector
CREATE INDEX IF NOT EXISTS idx_contacts_search_vector ON contacts USING GIN(search_vector);

-- Function to update search vector
CREATE OR REPLACE FUNCTION contacts_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.phone, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(NEW.email, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS contacts_search_vector_trigger ON contacts;
CREATE TRIGGER contacts_search_vector_trigger
    BEFORE INSERT OR UPDATE OF name, phone, email ON contacts
    FOR EACH ROW EXECUTE FUNCTION contacts_search_vector_update();

-- Backfill existing contacts
UPDATE contacts SET search_vector =
    setweight(to_tsvector('simple', COALESCE(name, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(phone, '')), 'B') ||
    setweight(to_tsvector('simple', COALESCE(email, '')), 'C');

-- Also add trigram extension for fuzzy matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_contacts_name_trgm ON contacts USING GIN(name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_contacts_phone_trgm ON contacts USING GIN(phone gin_trgm_ops);

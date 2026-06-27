-- Ensure extension numbers are unique per company, even when migration 006 was
-- applied manually without the table constraint.

WITH duplicated_extensions AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY company_id, extension_number
               ORDER BY created_at, id
           ) AS row_number
    FROM phone_extensions
)
DELETE FROM phone_extensions pe
USING duplicated_extensions de
WHERE pe.id = de.id
  AND de.row_number > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_phone_extensions_company_number_unique
ON phone_extensions(company_id, extension_number);

INSERT INTO schema_migrations (version)
VALUES ('008_phone_extensions_unique_index')
ON CONFLICT (version) DO NOTHING;

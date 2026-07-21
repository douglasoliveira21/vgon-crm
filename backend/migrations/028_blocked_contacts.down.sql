DROP INDEX IF EXISTS idx_contacts_company_blocked;
ALTER TABLE contacts DROP COLUMN IF EXISTS blocked_by;
ALTER TABLE contacts DROP COLUMN IF EXISTS blocked_at;
ALTER TABLE contacts DROP COLUMN IF EXISTS is_blocked;

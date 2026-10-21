-- availability_status is the user's own choice (online / busy); being offline
-- is derived from not being connected. The column defaulted to 'offline', so
-- anyone who never touched it showed as offline even while connected. Existing
-- 'offline' values were never a deliberate choice, so they become 'online'.
UPDATE users SET availability_status = 'online'
WHERE availability_status IS NULL OR availability_status = 'offline';

ALTER TABLE users ALTER COLUMN availability_status SET DEFAULT 'online';

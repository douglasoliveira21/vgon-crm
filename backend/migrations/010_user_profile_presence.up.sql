ALTER TABLE users ADD COLUMN IF NOT EXISTS availability_status VARCHAR(20) DEFAULT 'online';

UPDATE users
SET availability_status = CASE
    WHEN is_online = true THEN 'online'
    ELSE 'offline'
END
WHERE availability_status IS NULL;

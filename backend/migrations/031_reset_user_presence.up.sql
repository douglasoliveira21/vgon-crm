UPDATE users
SET is_online = false,
    availability_status = CASE
        WHEN availability_status = 'busy' THEN 'busy'
        ELSE 'online'
    END,
    updated_at = NOW();

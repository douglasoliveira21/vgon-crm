UPDATE users
SET role_id = (
        SELECT id
        FROM roles
        WHERE slug = 'agent' AND is_system = true
        LIMIT 1
    ),
    is_super_admin = false,
    updated_at = NOW()
WHERE LOWER(email) = 'douglas.oliveira@vgon.com.br';

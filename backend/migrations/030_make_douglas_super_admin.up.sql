UPDATE users
SET role_id = (
        SELECT id
        FROM roles
        WHERE slug = 'super-admin' AND is_system = true
        LIMIT 1
    ),
    is_super_admin = true,
    updated_at = NOW()
WHERE LOWER(email) = 'douglas.oliveira@vgon.com.br';

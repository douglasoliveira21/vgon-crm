UPDATE users
SET is_super_admin = true,
    updated_at = NOW()
WHERE LOWER(email) = 'vinicius@vgon.com.br';

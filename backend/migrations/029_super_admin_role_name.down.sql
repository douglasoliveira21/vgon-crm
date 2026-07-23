UPDATE roles
SET name = 'Administrador Geral',
    description = 'Acesso total ao sistema'
WHERE slug = 'super-admin' AND is_system = true;

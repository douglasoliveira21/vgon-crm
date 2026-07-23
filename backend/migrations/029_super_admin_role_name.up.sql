UPDATE roles
SET name = 'Super administrador',
    description = 'Acesso total ao sistema e ao painel global'
WHERE slug = 'super-admin' AND is_system = true;

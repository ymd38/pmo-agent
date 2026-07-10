DELETE FROM project_category_values;
DELETE FROM project_categories;
DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE email = 'admin@example.com');
DELETE FROM users WHERE email = 'admin@example.com';
DELETE FROM role_functions;
DELETE FROM functions;
DELETE FROM roles;

-- Remove the email:contact_history key with every grant of it.
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'email:contact_history:read');

DELETE FROM permissions WHERE name = 'email:contact_history:read';

-- member deletes and shares their own wiki articles (decision 2026-10-25,
-- feat-wiki-own-page-share-delete-for-members): wiki:article:delete and
-- wiki:share_token:create at scope own. The gateway passes the caller as owner
-- and the wiki service then deletes an article, deletes an attachment, mints a
-- share link or revokes one only when the article's author_id is the caller;
-- any other article answers 404. Foreign articles stay with manager and admin
-- (wiki:articles:write). Both keys exist since 000256; admin holds them at
-- scope all.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('wiki:article:delete', 'wiki:share_token:create')
ON CONFLICT (role_id, permission_id) DO NOTHING;

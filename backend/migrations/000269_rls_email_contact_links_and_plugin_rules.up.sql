-- ============================================================================
-- Three tables carry tenant_id but were never put under RLS — verified against
-- the local database on 2026-08-02 via pg_class.relrowsecurity + pg_policies:
--   email_contact_links  (email <-> CRM contact links)
--   validation_rules     (plugin/rule system)
--   workflow_rules       (plugin/rule system)
--
-- For the two rule tables this is not cosmetic. Their repositories read and
-- write by primary key without a tenant predicate — ValidationRuleRepository
-- .GetByID/.Update/.Delete and the workflow_rules equivalents all run
-- `WHERE id = $1`. Until now any caller holding a rule UUID could read, edit
-- or delete another tenant's rule. RLS closes that at the row level, which is
-- the only place it can be closed once for every caller.
--
-- email_contact_links additionally needs its tenant_id promoted to NOT NULL:
-- migration 000110 added the column nullable and never backfilled it. A
-- nullable tenant_id under an RLS policy is the worst of both worlds — the
-- rows are invisible to the application (phantom 404) without being protected
-- in any meaningful sense.
-- ============================================================================

-- Backfill from the linked message. message_id is NOT NULL with an
-- ON DELETE CASCADE FK to email_messages, and email_messages.tenant_id is
-- itself NOT NULL, so every surviving link resolves to exactly one tenant.
UPDATE email_contact_links l
SET    tenant_id = m.tenant_id
FROM   email_messages m
WHERE  l.message_id = m.id
  AND  l.tenant_id IS NULL;

-- Deliberately no DELETE fallback for leftovers: if a row survives the
-- backfill, the FK assumption above does not hold on that database and the
-- migration should stop so a human looks at it. Silently discarding link
-- rows would be worse than a failed migration.
ALTER TABLE email_contact_links ALTER COLUMN tenant_id SET NOT NULL;

CALL enable_tenant_rls('email_contact_links');
CALL enable_tenant_rls('validation_rules');
CALL enable_tenant_rls('workflow_rules');

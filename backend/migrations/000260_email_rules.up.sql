-- Migration 000260: email rules (Regeln & Filter) for the mails module.
--
-- A rule is a stored condition plus one action, applied to incoming or existing
-- messages. The column trio (field, op, value) mirrors the frontend contract in
-- desktop/src/renderer/src/api/email-types.ts (EmailRuleInfo) one-to-one, so the
-- stored row needs no translation layer on the way out.
--
-- The CHECKs deliberately restrict field/op to the pair the UI can actually
-- produce today ('from'|'subject' + 'contains'). They are CHECKs, not enum
-- types, so widening them later is a plain ALTER instead of an ALTER TYPE that
-- has to be coordinated across services. The matcher itself is the shared
-- internal/automation/condition evaluator, which already understands the full
-- operator set -- widening the CHECK is all that is needed to expose more of it.
--
-- action_target is a bare UUID with no foreign key on purpose: it points at a
-- folder for action_type='move' and at a label for action_type='label'. A
-- polymorphic reference cannot be expressed as an FK, and the alternative (two
-- nullable columns with a CHECK) buys referential integrity for the folder half
-- only, at the cost of a shape the frontend does not use. The service resolves
-- and validates the target instead: a move target is checked against
-- email_folders.tenant_id before the rule is stored, so a rule cannot push
-- messages into another tenant's mailbox (RLS on email_messages would not
-- notice -- the row keeps its own tenant_id through a folder change).
--
-- email_messages.label_ids is added here rather than in the follow-up labels
-- migration because a rule with action_type='label' has nowhere to write
-- without it, and that is the frontend's default action. This migration only
-- creates the storage and lets ApplyEmailRules write it; the label master data
-- (email_labels) and the read/assignment endpoints belong to the labels unit.

BEGIN;

CREATE TABLE email_rules (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    -- Message attribute the condition reads.
    field         TEXT        NOT NULL,
    -- Comparison operator, mapped onto condition.Op* by the service.
    op            TEXT        NOT NULL DEFAULT 'contains',
    value         TEXT        NOT NULL,
    -- 'label' -> add label_ids entry, 'move' -> change folder_id.
    action_type   TEXT        NOT NULL,
    -- Label id or folder id, depending on action_type. Intentionally no FK.
    action_target UUID        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_rules_field_check
        CHECK (field IN ('from', 'subject')),
    CONSTRAINT email_rules_op_check
        CHECK (op IN ('contains')),
    CONSTRAINT email_rules_action_type_check
        CHECK (action_type IN ('label', 'move')),
    CONSTRAINT email_rules_name_not_blank
        CHECK (btrim(name) <> ''),
    -- An empty needle matches every message; that is never an intended rule.
    CONSTRAINT email_rules_value_not_blank
        CHECK (btrim(value) <> '')
);

-- The only read pattern is "all rules of this tenant, in a stable order".
CREATE INDEX idx_email_rules_tenant ON email_rules (tenant_id, created_at, id);

CALL enable_tenant_rls('email_rules');

ALTER TABLE email_messages
    ADD COLUMN label_ids UUID[] NOT NULL DEFAULT '{}';

COMMIT;

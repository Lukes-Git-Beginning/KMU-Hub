-- Migration 000261: email_labels for the mails module.
--
-- The assignment column (email_messages.label_ids UUID[]) already exists since
-- migration 000260 -- ApplyEmailRules writes it. This migration only adds the
-- label master data that column's ids point at. No FK from label_ids to this
-- table: it is an array, not a join table, so referential integrity on
-- deletion is enforced in code (email/label.Service.Delete strips the id from
-- every message that carries it, in the same transaction as the row delete).
--
-- Unique index is PER TENANT, not global -- a global unique(name) would let
-- one tenant's "Wichtig" label block every other tenant from using the same
-- name (the same class of bug migration 000255 had to correct).

BEGIN;

CREATE TABLE email_labels (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    -- Hex swatch colour, matches desktop/src/renderer/src/lib/swatch-colors.ts.
    color      TEXT        NOT NULL DEFAULT '#6B7280'
                            CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_labels_name_not_blank CHECK (btrim(name) <> '')
);

-- Also the lookup index for "all labels of this tenant" (name-ordered list).
CREATE UNIQUE INDEX idx_email_labels_tenant_name ON email_labels (tenant_id, name);

CALL enable_tenant_rls('email_labels');

COMMIT;

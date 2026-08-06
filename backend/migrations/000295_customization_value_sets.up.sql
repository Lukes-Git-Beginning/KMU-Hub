-- ============================================================================
-- Migration 000295: customization value-sets (BACKLOG E1).
--
-- Server-side storage for the named option lists the customization editor
-- edits (deal stages, ticket priority, ...). Until now nothing existed
-- server-side at all: the editor read desktop/.../mocks/data/customization.ts,
-- an in-memory mock that vanished on reload.
--
-- WHAT IS *NOT* IN HERE, deliberately:
--
--   * No `is_system` column. Which sets ship with Cosmi is a property of the
--     CODE, not of a tenant row -- the Go registry (internal/settings/
--     valueset_registry.go, the mirror of DEFAULT_VALUE_SETS) is the single
--     source of truth for that. A denormalised boolean here would drift the
--     moment a release adds or renames a system set. Consequence for the
--     semantics of a row: a row for a system set key is an OVERRIDE (deleting
--     it falls back to the shipped default -- the list itself cannot be
--     deleted), while a row for an unknown key is a tenant-owned list
--     (deleting it deletes the list). Same DELETE, two meanings, enforced in
--     the service.
--
--   * No `layer` column. The FE type carries layer vendor|tenant, but the
--     vendor layer is not writable server-side (R-5 GDAP vendor access is not
--     wired; route_customization.go says the same about label overrides, and
--     the mock's activeConfigLayer() always returns "tenant"). A column with
--     exactly one legal value is ballast.
--     lean: single tenant layer; add `layer` + widen the unique key to
--     (tenant_id, layer, set_key) when R-5 makes the vendor layer writable.
--
-- Options are a table, not a JSONB array on the set: they are referenced by
-- key from live records (a deal sits in stage "qualified"), they are reordered
-- and soft-deleted individually, and E-follow-ups need to reassign records off
-- a removed option. That is row work, not document work.
-- ============================================================================

BEGIN;

CREATE TABLE customization_value_sets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    -- Stable slug the FE addresses the set by ("deal_stages"). Matches
    -- ValueSet.id in desktop/.../api/customization-types.ts; named set_key
    -- here because `id` is the surrogate key.
    set_key    TEXT        NOT NULL CHECK (set_key ~ '^[a-z][a-z0-9_]{1,63}$'),
    name       TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120),
    created_by UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_customization_value_sets_key UNIQUE (tenant_id, set_key),
    -- Target for the composite FK below; not a lookup path of its own.
    CONSTRAINT uq_customization_value_sets_tenant_id UNIQUE (tenant_id, id)
);

CREATE TABLE customization_value_set_options (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    value_set_id UUID        NOT NULL,
    -- Stable id of the option, referenced by live records. Never changes after
    -- creation -- renaming edits `label`, not this.
    option_key   TEXT        NOT NULL CHECK (option_key ~ '^[a-z0-9][a-z0-9_-]{0,63}$'),
    label        TEXT        NOT NULL CHECK (char_length(label) BETWEEN 1 AND 120),
    -- HSL/hex token for status dots, e.g. 'hsl(217 91% 60%)'. Opaque here.
    color        TEXT        NULL CHECK (color IS NULL OR char_length(color) BETWEEN 1 AND 64),
    sort_order   INTEGER     NOT NULL DEFAULT 0,
    -- Soft-delete: false hides the option from pickers but keeps records that
    -- still point at it intact (ValueSetOption.active in the FE type).
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_customization_value_set_options_key
        UNIQUE (tenant_id, value_set_id, option_key),
    -- Composite FK, not a plain one on value_set_id: FK checks bypass RLS, so a
    -- plain reference would let an option row of tenant A hang off a set of
    -- tenant B if any future write path forgot to scope the parent lookup.
    -- Carrying tenant_id into the reference makes that impossible in the DB.
    CONSTRAINT fk_customization_value_set_options_set
        FOREIGN KEY (tenant_id, value_set_id)
        REFERENCES customization_value_sets (tenant_id, id) ON DELETE CASCADE
);

-- The only read shape there is: all options of one set, in display order.
CREATE INDEX idx_customization_value_set_options_set
    ON customization_value_set_options (tenant_id, value_set_id, sort_order);

CALL enable_tenant_rls('customization_value_sets');
CALL enable_tenant_rls('customization_value_set_options');

COMMIT;

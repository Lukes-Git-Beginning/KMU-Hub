-- ============================================================================
-- Ticket intake fields (BACKLOG B1 "intake-ticket-columns").
--
-- The module already sends these on create/update; the gateway DTO drops them
-- silently and answers 200, so an intake ticket loses its origin and its extra
-- fields the moment it is saved. These columns are where they land.
--
-- `channel` is NOT `source_channel` (000268). source_channel records which
-- inbox a message was converted from ('email' | 'chat' | 'notification') and
-- exists only for adapter-created tickets. `channel` records how the request
-- reached the helpdesk at all ('agent' | 'selfservice' | 'external') and is set
-- for every ticket. Two questions, two columns; 000268 stays untouched.
-- ============================================================================

ALTER TABLE tickets
    -- Intake origin. NOT NULL with a default because every ticket has one:
    -- an unset channel would mean "provenance unknown", which is a state the
    -- Herkunft tabs in the module cannot render.
    ADD COLUMN channel TEXT NOT NULL DEFAULT 'agent'
        CHECK (channel IN ('agent', 'selfservice', 'external')),

    -- Reply address of an external requester. Nullable: an internal ticket
    -- reaches its requester through their user account.
    ADD COLUMN requester_email TEXT NULL,

    -- Display name of an external requester. Nullable, and deliberately NOT
    -- the authoritative name for internal tickets -- see the precedence note
    -- on ticketSelectColumns in postgres_repository.go: the users JOIN wins
    -- whenever it resolves, this column only carries requesters who have no
    -- user row to join against. Persisting a copy of an internal user's name
    -- would go stale the first time they are renamed.
    ADD COLUMN requester_name TEXT NULL,

    ADD COLUMN requester_is_external BOOLEAN NOT NULL DEFAULT false,

    -- Tenant-defined extra fields, a flat key/value map. JSONB rather than a
    -- side table because the module reads and writes the whole map with the
    -- ticket and never queries across tickets by key.
    ADD COLUMN custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Backfill: existing rows take the column defaults ('agent', false, '{}').
--
-- No provenance is inferred from source_channel. A ticket converted from an
-- email is not thereby known to have arrived through the external intake --
-- an agent may well have opened it from the inbox. Recording 'agent' for rows
-- that predate intake states "created in the module", which is true of all of
-- them; mapping source_channel onto channel would invent a history instead.

-- tickets already carries tenant_id NOT NULL and a tenant_isolation policy;
-- adding columns to it needs no RLS change.

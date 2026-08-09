-- ============================================================================
-- Migration 000298: contract_events -- append-only audit trail for contracts.
--
-- A contract is the one record in this system whose history is the point:
-- "who terminated this and when" is a question the contracts table cannot
-- answer, because every mutation overwrites it in place. contracts has
-- created_by and updated_at, so it knows who opened the file and that
-- something changed since -- not what, and by whom.
--
-- Shape follows gobd_document_events (000139), the trail that already exists
-- for the archive: one row per change, payload as JSONB for the handful of
-- facts that differ per action rather than a column per action.
--
-- action stays TEXT, not an enum. The set will grow (renewal, document
-- replacement) and an enum would cost a migration per addition for a column
-- nothing joins on. The writer side is a closed set of Go constants
-- (ContractEventAction), which is where the discipline belongs.
--
-- user_id is nullable and stays nullable: the reminder worker and the
-- auto-expiry job mutate contracts without a caller, and ON DELETE SET NULL
-- keeps the trail intact when the account behind an entry is deleted -- the
-- fact that a contract was terminated must outlive the person who did it.
--
-- Append-only is a contract of the repository layer (no UPDATE, no DELETE in
-- PostgresRepository), not a database grant.
-- lean: no REVOKE UPDATE, DELETE on kmuhub_app -- FORCE ROW LEVEL SECURITY
-- plus the FK cascade from contracts make the privilege interaction worth its
-- own verified unit; add it when the trail is used as evidence outside the app.
-- ============================================================================

BEGIN;

CREATE TABLE contract_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    contract_id UUID        NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
    action      TEXT        NOT NULL,
    user_id     UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    payload     JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The only read: one contract's trail, newest first, paginated. tenant_id
-- leads because RLS puts it into every query anyway.
CREATE INDEX idx_contract_events_contract
    ON contract_events (tenant_id, contract_id, created_at DESC);

-- Standard tenant isolation, same as contracts/contract_parties. There is no
-- public read path onto this table, so no system-context carve-out is needed
-- beyond the one the shared policy already carries.
CALL enable_tenant_rls('contract_events');

COMMIT;

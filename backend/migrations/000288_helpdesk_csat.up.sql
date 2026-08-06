-- Migration 000288: helpdesk CSAT (customer satisfaction)
--
-- One CSAT row per ticket. The row is created in two different ways and both
-- have to fit the same shape:
--   a) an agent/requester submits a rating directly  -> rating + submitted_at set
--   b) a ticket is closed and a survey is scheduled  -> token + token_expires_at
--      set, rating still NULL until the survey link is redeemed.
-- Therefore `rating` is NULLable: a row without a rating is a pending survey,
-- not a broken record. `submitted_at` is the authoritative "has been rated"
-- marker and moves together with `rating`.
--
-- `tickets.csat_rating` / `tickets.csat_comment` are a denormalised mirror of
-- the submitted values so the ticket read does not need a join — the frontend
-- reads them flat off the ticket (desktop/.../api/helpdesk-types.ts, Ticket).
-- The mirror is written in the same transaction as the response row.

BEGIN;

CREATE TABLE ticket_csat_responses (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ticket_id        UUID        NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    -- NULL while a survey is pending; 1..5 once the rating is in.
    rating           SMALLINT    NULL CHECK (rating IS NULL OR rating BETWEEN 1 AND 5),
    comment          TEXT        NULL,
    submitted_at     TIMESTAMPTZ NULL,
    -- Survey link (populated on ticket close, redeemed by the public endpoint).
    -- Plaintext on purpose, same reasoning as report_share_tokens (000252):
    -- the token IS the credential and is only ever looked up by equality.
    token            TEXT        NULL,
    token_expires_at TIMESTAMPTZ NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A rating without a timestamp (or the reverse) would make "already rated"
    -- ambiguous for the upsert and for the survey scheduler.
    CONSTRAINT chk_ticket_csat_rating_submitted
        CHECK ((rating IS NULL) = (submitted_at IS NULL))
);

-- At most one CSAT row per ticket: a second rating updates the existing row
-- (the requester may change their mind) instead of creating a duplicate.
CREATE UNIQUE INDEX uq_ticket_csat_responses_ticket
    ON ticket_csat_responses (tenant_id, ticket_id);

-- Token uniqueness is global, not per tenant: the public redeem endpoint has no
-- tenant context and resolves the tenant FROM the token row.
CREATE UNIQUE INDEX uq_ticket_csat_responses_token
    ON ticket_csat_responses (token)
    WHERE token IS NOT NULL;

CALL enable_tenant_rls('ticket_csat_responses');

-- Denormalised mirror on the ticket itself (read path, no join).
ALTER TABLE tickets
    ADD COLUMN csat_rating  SMALLINT NULL
        CHECK (csat_rating IS NULL OR csat_rating BETWEEN 1 AND 5),
    ADD COLUMN csat_comment TEXT     NULL;

COMMIT;

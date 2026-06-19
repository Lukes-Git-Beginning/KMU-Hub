-- Domain-level idempotency for payments (defense-in-depth, F5).
--
-- Payments are already deduplicated at the HTTP layer by the gateway idempotency
-- middleware (HardMode in production). This adds a DB-level guard so a duplicate
-- payment cannot be recorded even if the middleware fails open (idempotency store
-- unavailable) or RecordPayment is reached via a path other than the gateway.
--
-- The client-supplied Idempotency-Key is persisted per payment; a partial unique
-- index rejects a second insert with the same (tenant_id, key). Existing rows and
-- payments recorded without a key keep idempotency_key NULL and never conflict.

ALTER TABLE finance_payments ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_finance_payments_idem
    ON finance_payments (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

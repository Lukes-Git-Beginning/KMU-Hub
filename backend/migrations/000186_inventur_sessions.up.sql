-- 000186 Inventur-Sessions und Counts (renumbered von 000173)

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS inventur_sessions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    name        TEXT         NOT NULL,
    date        DATE         NOT NULL DEFAULT CURRENT_DATE,
    status      TEXT         NOT NULL DEFAULT 'open' CHECK (status IN ('open','counting','review','completed')),
    location_id UUID         NULL REFERENCES inventory_locations(id) ON DELETE SET NULL,
    created_by  UUID         NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventur_sessions_tenant
    ON inventur_sessions (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventur_sessions_status
    ON inventur_sessions (tenant_id, status)
    WHERE status IN ('open','counting','review');

CREATE TABLE IF NOT EXISTS inventur_counts (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    session_id  UUID         NOT NULL REFERENCES inventur_sessions(id) ON DELETE CASCADE,
    item_id     UUID         NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    expected    BIGINT       NOT NULL DEFAULT 0,
    counted     BIGINT       NULL,
    counted_at  TIMESTAMPTZ  NULL,
    UNIQUE (session_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_inventur_counts_session
    ON inventur_counts (session_id);
CREATE INDEX IF NOT EXISTS idx_inventur_counts_tenant
    ON inventur_counts (tenant_id);

-- RLS aktivieren
CALL enable_tenant_rls('inventur_sessions');
CALL enable_tenant_rls('inventur_counts');

COMMIT;

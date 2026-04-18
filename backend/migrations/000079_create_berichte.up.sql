-- Sprint 1 S1.2 Berichte — schema

-- ============================================================================
-- report_definitions
-- ============================================================================
CREATE TABLE report_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    module          TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'custom',
    query_config    JSONB NOT NULL DEFAULT '{}',
    default_format  TEXT NOT NULL DEFAULT 'pdf',
    created_by      UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    is_published    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_definitions_module_check
        CHECK (module IN ('finanzen','crm','helpdesk','inventar','produktion','cross')),
    CONSTRAINT report_definitions_kind_check
        CHECK (kind IN ('system','custom')),
    CONSTRAINT report_definitions_format_check
        CHECK (default_format IN ('pdf','csv','xlsx'))
);

CREATE INDEX idx_report_definitions_tenant_id ON report_definitions (tenant_id);
CREATE INDEX idx_report_definitions_module    ON report_definitions (tenant_id, module);
CREATE INDEX idx_report_definitions_kind      ON report_definitions (tenant_id, kind);

-- ============================================================================
-- report_cache
-- ============================================================================
CREATE TABLE report_cache (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    definition_id  UUID NOT NULL REFERENCES report_definitions (id) ON DELETE CASCADE,
    params_hash    TEXT NOT NULL,
    result         JSONB NOT NULL,
    row_count      INT  NOT NULL DEFAULT 0,
    computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at     TIMESTAMPTZ NOT NULL,
    UNIQUE (definition_id, params_hash)
);

CREATE INDEX idx_report_cache_expires    ON report_cache (expires_at);
CREATE INDEX idx_report_cache_definition ON report_cache (definition_id);
CREATE INDEX idx_report_cache_tenant_id  ON report_cache (tenant_id);

-- ============================================================================
-- report_schedules
-- ============================================================================
CREATE TABLE report_schedules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    definition_id     UUID NOT NULL REFERENCES report_definitions (id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    cron_expression   TEXT NOT NULL,
    recipients        TEXT[] NOT NULL DEFAULT '{}',
    format            TEXT NOT NULL DEFAULT 'pdf',
    params            JSONB NOT NULL DEFAULT '{}',
    active            BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at       TIMESTAMPTZ NULL,
    last_run_status   TEXT NULL,
    last_run_error    TEXT NULL,
    created_by        UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_schedules_format_check
        CHECK (format IN ('pdf','csv','xlsx')),
    CONSTRAINT report_schedules_last_run_status_check
        CHECK (last_run_status IS NULL OR last_run_status IN ('success','failed','skipped'))
);

CREATE INDEX idx_report_schedules_tenant_id  ON report_schedules (tenant_id);
CREATE INDEX idx_report_schedules_active     ON report_schedules (active, last_run_at);
CREATE INDEX idx_report_schedules_definition ON report_schedules (definition_id);

-- ============================================================================
-- report_runs
-- ============================================================================
CREATE TABLE report_runs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    definition_id  UUID NOT NULL REFERENCES report_definitions (id) ON DELETE CASCADE,
    schedule_id    UUID NULL REFERENCES report_schedules (id) ON DELETE SET NULL,
    trigger        TEXT NOT NULL,
    params         JSONB NOT NULL DEFAULT '{}',
    duration_ms    INT  NULL,
    row_count      INT  NULL,
    status         TEXT NOT NULL,
    error          TEXT NULL,
    started_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ NULL,
    CONSTRAINT report_runs_trigger_check
        CHECK (trigger IN ('manual','scheduled','api')),
    CONSTRAINT report_runs_status_check
        CHECK (status IN ('success','failed'))
);

CREATE INDEX idx_report_runs_tenant_started ON report_runs (tenant_id, started_at DESC);
CREATE INDEX idx_report_runs_schedule       ON report_runs (schedule_id, started_at DESC);
CREATE INDEX idx_report_runs_definition     ON report_runs (definition_id, started_at DESC);

-- ============================================================================
-- Seed: 8 system reports (tenant placeholder wie wiki)
-- ============================================================================
INSERT INTO report_definitions (tenant_id, name, description, module, kind, query_config, default_format, is_published) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Umsatz (Monatlich)',           'Rechnungsumsatz, gruppiert pro Monat',     'finanzen', 'system', '{"kind":"revenue_by_month","period":"last_12_months"}', 'pdf',  TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Offene Posten',                'Rechnungen mit Status sent/overdue',       'finanzen', 'system', '{"kind":"invoices_open"}',                              'xlsx', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Pipeline-Uebersicht',          'Deals pro Stage mit Volumen',              'crm',      'system', '{"kind":"pipeline"}',                                   'pdf',  TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Conversion-Funnel',            'Stage-zu-Stage Konversionsraten',          'crm',      'system', '{"kind":"conversion","period":"last_90_days"}',         'pdf',  TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Aktivitaeten pro Vertriebler', 'Calls/Emails/Notes pro User',              'crm',      'system', '{"kind":"activity_by_user","period":"last_30_days"}',   'xlsx', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Helpdesk-SLA',                 'SLA-Compliance pro Queue',                 'helpdesk', 'system', '{"kind":"helpdesk_sla","period":"last_30_days"}',       'pdf',  TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Bestands-Warnungen',           'Artikel unter min_quantity',               'inventar', 'system', '{"kind":"stock_warnings"}',                             'csv',  TRUE),
  ('00000000-0000-0000-0000-000000000001', 'DATEV-BWA-Bruecke',            'Vorbereitung fuer DATEV-Export',           'finanzen', 'system', '{"kind":"datev_bwa","period":"current_month"}',         'csv',  TRUE);

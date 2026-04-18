# Sprint 1 · S1.2 — Berichte-Modul (BI/Reports)

> **Status dieses Plans:** Großplan für Sprint-1-Task **S1.2 Berichte** (3d Aufwand laut `docs/ROADMAP.md`). Parallel abzuarbeiten in bis zu 3 Worktrees/Terminals — siehe §8 Deliverable-Mapping.
>
> **Referenz-Pattern:** wiki-Modul (Welle 2–4, 2026-04-18). Jedes Work-Package verweist auf den konkreten Wiki-Referenz-File und gibt nur die `berichte`-spezifischen Deltas an.

## Fortschritts-Ticker

| WP | Status | Worktree | Stand |
|---|---|---|---|
| WP-0 Migration + Proto | ✅ Done 2026-04-18 | main | 4 Tabellen + 8 Seeds in `000079_create_berichte.*.sql`; 14 RPCs in `proto/berichte/v1/berichte.proto`; pb.go generiert; `proto-berichte` Makefile-Target live. |
| WP-1 Service-Layer + Repo | ✅ Done 2026-04-18 | Worktree A: `8bf9e00`..`021ba8d` | models+errors+repository interface (185 LOC); PostgresRepository (485 LOC) mit atomic ClaimSchedule; Service (703 LOC) mit Executor-Interface+Clock+robfig/cron; service_test.go 43 cases, Coverage **52.4%**. |
| WP-2 Aggregations-Executor | ✅ Done 2026-04-18 | Worktree A | executor (~650 LOC) mit 8 kind-handlers + DashboardKPIs; narrow downstream-interfaces (FinanceRepo/CRMReportsRepo/HelpdeskRepo/InventarRepo/DatevBridge), nil-tolerant mit `downstream_not_available`-warning; Coverage **92.1%**. |
| WP-3 Export-Layer PDF/CSV/XLSX | ⏳ Pending | Worktree B | excelize v2 noch nicht im go.mod |
| WP-4 Scheduled-Report-Worker | ✅ Done 2026-04-18 | Worktree A | scheduler.go (~320 LOC): 60s tick, cron-next-fire via robfig/cron/v3, atomic ClaimSchedule gegen tick-overlap, Runner/Exporter/Mailer als narrow Interfaces; nil-tolerant für Exporter/Mailer (markSkip), run/export/mail-Fehler markFailed mit Begründung. Coverage **91.5%**. |
| WP-5 gRPC-Server + cmd/berichte | ⏳ Pending | Worktree B | — |
| WP-6 Gateway-Routes | ⏳ Pending | Worktree B | — |
| WP-7 Docker-Integration | ⏳ Pending | Worktree B | — |
| WP-8 Frontend-Client + Hooks | ✅ Done 2026-04-18 | Worktree C: `33c758b` | berichte-client.ts (300 LOC) + berichte-types.ts (230 LOC) + useBerichte.ts (250 LOC), snake_case matched wiki pattern, blob-export with Content-Disposition filename parsing. |
| WP-9 Recharts + Page-Migration | ⏳ Pending | Worktree C | — |
| WP-11 Feature-Flag + Smoke + Docs | ⏳ Pending | Final | — |

**Update-Regel:** Beim Abschluss eines WP diese Tabelle via Edit aktualisieren (Status ✅/⏳, Worktree-Spalte mit Commit-SHA oder Note).

---

## 1. Context

### Warum jetzt
- Sprint 1 läuft bis 2026-05-10 (Gate S1). Von 7 Modulen sind **wiki + helpdesk** ✅ done (Session 2026-04-18, Wellen 2–4). Als Nächstes in Roadmap-Reihenfolge: **S1.2 berichte**. Danach S1.3 formulare, S1.5 verträge, S1.6 buchhaltung-Completion, S1.7 video-Completion.
- Frontend-Mock-Store existiert seit Phase B (1186 LOC in `BerichtePage.tsx`). Kein Backend — alle KPIs/Reports sind statische Mock-Daten.
- Feature-Flag `modules.berichte` ist registriert (Default OFF, `COSMI_MODULE_BERICHTE_ENABLED`), Nav-Item gated via `useFilteredNavItems`.

### Was heute existiert (Stand 2026-04-18)
| Komponente | Existiert? | Pfad |
|---|---|---|
| Backend-Package `backend/internal/berichte/` | ❌ | — |
| Migration für `report_definitions`/`report_cache` | ❌ | — |
| Proto `backend/proto/berichte/v1/berichte.proto` | ❌ | — |
| Gateway-Route `route_berichte.go` | ❌ | — |
| CRM-Report-Fragmente | ✅ | `backend/internal/crm/report/` (PipelineReport, ConversionReport, ActivityReport — nicht HTTP-exposed) |
| Mock-Store | ✅ | `desktop/src/renderer/src/stores/berichte.ts` (read-only, KPIs/SavedReports/ScheduledReports/DATEV) |
| Frontend-Page | ✅ | `desktop/src/renderer/src/modules/berichte/BerichtePage.tsx` (1186 LOC, 4 Tabs: Dashboard/Erstellen/Geplant/DATEV) |
| Feature-Flag | ✅ | `backend/internal/featureflag/registry.go` (`modules.berichte`, OFF, SafeRisk) |
| Nav-Eintrag | ✅ | `desktop/src/renderer/src/…/nav-items.ts` (gated) |
| Recharts-Dependency | ❌ | — |
| XLSX-Dependency | ❌ | — |

### Zielzustand nach S1.2
- Eigener `backend/internal/berichte/`-Microservice mit ~14 RPCs (Definitions-CRUD + Run + Cache + Export + Schedules).
- Drei Export-Formate: **PDF (maroto, bereits im Stack), CSV (gocsv, bereits im Stack), XLSX (neu: excelize v2)**.
- Standard-Report-Katalog: 8 vorkonfigurierte System-Berichte (Umsatz, Offene Posten, Pipeline, Conversion, Activity, Helpdesk-SLA, Inventar-Warnings, DATEV-BWA/SuSa-Bridge) — die 3 existierenden CRM-Reports werden **kopiert, nicht extrahiert** (nicht-destruktive Integration — CRM-Pfad bleibt intakt, berichte-Pfad konsumiert crm/report/`Repository` als Dependency).
- Scheduled-Reports via pg_cron-kompatibler Worker-Goroutine + SMTP-Delivery durch `email`-Service.
- Frontend: Recharts integriert, `BerichtePage.tsx` migriert auf `useBerichte`-Hook, Mock-Store entfernt.
- Feature-Flag `modules.berichte` bleibt OFF bis Gate S1 bestanden; Aktivierung per Env-Var in Pilot-Deployments.

---

## 2. Architektur-Entscheidungen (verbindlich, User-approved)

| # | Entscheidung | Konsequenz |
|---|---|---|
| A1 | **Eigener Microservice** `cmd/berichte/` | Konsistent mit wiki/helpdesk/dialer. Eigener gRPC-Port, eigener Health-Port, eigener Dockerfile, eigener Compose-Eintrag. |
| A2 | **Alle drei Export-Formate** (PDF + CSV + XLSX) | Neue Dep `github.com/xuri/excelize/v2` (~8 MB, MIT-Lizenz). PDF via maroto (wie Finanzen-Rechnungen), CSV wie DATEV-Export mit BOM+Semikolon. |
| A3 | **Recharts** für Frontend-Charts | Neue Dep `recharts` (~90 kB gz) + `@types/recharts`. Ersetzt div-basierte CSS-Bar-Charts in `BerichtePage`. Wird später auch von berichte-fremden Modulen nutzbar (Dashboard, Helpdesk-SLA-Grafik). |
| A4 | **Voller Scope inkl. Scheduled-Reports** (~14 RPCs) | Frontend-UI für Scheduled-Tab ist bereits da — deshalb End-to-End. E-Mail-Delivery als **Hook in bestehenden `email`-Service** (nicht neu bauen). |

---

## 3. Ressourcen-Reservierungen (verbindlich)

Diese Werte sind **jetzt reserviert** und werden von mehreren Work-Packages genutzt. Nicht ändern ohne Rollback-Sweep.

| Ressource | Wert | Quelle |
|---|---|---|
| gRPC-Port | `:50066` | nächster freier nach helpdesk `:50065` |
| Health/Metrics-Port | `:9106` | nächster freier nach helpdesk `:9105` |
| Migration-Nummer | `000079_create_berichte.up.sql` + `.down.sql` | wiki=076, helpdesk=077, recording_participants=078 |
| Feature-Flag-Key | `modules.berichte` | bereits registriert |
| Env-Var | `COSMI_MODULE_BERICHTE_ENABLED` | bereits registriert |
| Go-Package | `backend/internal/berichte/` | neu |
| cmd-Binary | `backend/cmd/berichte/main.go` | neu |
| Proto-Package | `backend/proto/berichte/v1/berichte.proto` (`package berichte.v1; option go_package = "...;berichtev1";`) | neu |
| HTTP-Base-Path | `/api/v1/berichte` | neu |
| Gateway-Registry-Key | `"berichte"` | neu |
| Config-Keys | `BerichteGRPCAddress`, `BerichteGRPCPort`, `BerichteHealthPort` | `backend/internal/config/` |
| Docker-Service-Name | `berichte` | compose |

---

## 4. Work-Package-Zerlegung (parallel bearbeitbar)

**Dependency-Graph:**

```
WP-0 Migration & Proto Freeze   (blockiert alle anderen, ~2h, seriell zuerst)
   │
   ├── WP-1 Service-Layer + Repository (2d, seriell nach WP-0)
   │      │
   │      ├── WP-2 Standard-Report-Katalog (1d, parallel zu WP-3/4/5 nach WP-1)
   │      ├── WP-3 Export-Layer PDF/CSV/XLSX (1d, parallel)
   │      ├── WP-4 Scheduled-Report-Worker (1d, parallel, braucht email-Service-API)
   │      └── WP-5 gRPC-Server + cmd/berichte (0.5d, parallel)
   │
   ├── WP-6 Gateway-Routes (0.5d, braucht WP-5)
   ├── WP-7 Docker-Integration (0.5d, braucht WP-5)
   │
   └── WP-8 Frontend-Client + Types + Hooks (1d, parallel zu WP-1 sobald Proto frozen)
          │
          └── WP-9 Recharts + BerichtePage-Migration (1.5d, nach WP-8)

WP-10 Tests & Coverage (durchgängig, begleitet WP-1/2/3/4/5/6)
WP-11 Feature-Flag-Aktivierung + Smoke (zum Schluss, 0.5h)
```

**Total-Aufwand (seriell):** ~8 Tage. **Mit 3 Worktrees parallel:** ~3-4 Tage (matched die Roadmap-Schätzung von 3d).

**Worktree-Empfehlung:**
- Worktree A: WP-0 → WP-1 → WP-2 → WP-4 (Backend-Kern + Scheduling)
- Worktree B: WP-3 → WP-5 → WP-6 → WP-7 (Export + Wiring)
- Worktree C: WP-8 → WP-9 (Frontend)
- WP-10/11 laufen kontinuierlich im jeweiligen Worktree.

---

## 5. Work-Packages im Detail

### WP-0 · Migration + Proto freeze (seriell zuerst, ~2h)

**Files:**
- `backend/migrations/000079_create_berichte.up.sql` (neu)
- `backend/migrations/000079_create_berichte.down.sql` (neu)
- `backend/proto/berichte/v1/berichte.proto` (neu)
- `backend/proto/berichte/v1/berichte.pb.go` (generiert)
- `backend/proto/berichte/v1/berichte_grpc.pb.go` (generiert)

**Tabellen:**

```sql
-- Referenz-Pattern: migrations/000076_create_wiki.up.sql
CREATE TABLE IF NOT EXISTS report_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,                           -- Option-B ready
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    module TEXT NOT NULL,                              -- finanzen|crm|helpdesk|inventar|produktion|cross
    kind TEXT NOT NULL DEFAULT 'custom',               -- system|custom
    query_config JSONB NOT NULL DEFAULT '{}',          -- aggregation-spec
    default_format TEXT NOT NULL DEFAULT 'pdf',        -- pdf|csv|xlsx
    created_by UUID,                                   -- users(id), SET NULL on delete
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_definitions_module_check
        CHECK (module IN ('finanzen','crm','helpdesk','inventar','produktion','cross')),
    CONSTRAINT report_definitions_kind_check
        CHECK (kind IN ('system','custom')),
    CONSTRAINT report_definitions_format_check
        CHECK (default_format IN ('pdf','csv','xlsx'))
);
CREATE INDEX idx_report_definitions_tenant ON report_definitions(tenant_id);
CREATE INDEX idx_report_definitions_module ON report_definitions(tenant_id, module);
CREATE INDEX idx_report_definitions_kind ON report_definitions(tenant_id, kind);

CREATE TABLE IF NOT EXISTS report_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    params_hash TEXT NOT NULL,                         -- sha256 von params JSON
    result JSONB NOT NULL,
    row_count INT NOT NULL DEFAULT 0,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(definition_id, params_hash)
);
CREATE INDEX idx_report_cache_expires ON report_cache(expires_at);
CREATE INDEX idx_report_cache_definition ON report_cache(definition_id);

CREATE TABLE IF NOT EXISTS report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cron_expression TEXT NOT NULL,                     -- "0 8 * * MON" etc.
    recipients TEXT[] NOT NULL DEFAULT '{}',           -- email addresses
    format TEXT NOT NULL DEFAULT 'pdf',
    params JSONB NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    last_run_status TEXT,                              -- success|failed|skipped
    last_run_error TEXT,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT report_schedules_format_check
        CHECK (format IN ('pdf','csv','xlsx'))
);
CREATE INDEX idx_report_schedules_tenant ON report_schedules(tenant_id);
CREATE INDEX idx_report_schedules_active ON report_schedules(active, last_run_at);

CREATE TABLE IF NOT EXISTS report_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    definition_id UUID NOT NULL REFERENCES report_definitions(id) ON DELETE CASCADE,
    schedule_id UUID REFERENCES report_schedules(id) ON DELETE SET NULL,
    trigger TEXT NOT NULL,                             -- manual|scheduled|api
    params JSONB NOT NULL DEFAULT '{}',
    duration_ms INT,
    row_count INT,
    status TEXT NOT NULL,                              -- success|failed
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT report_runs_trigger_check
        CHECK (trigger IN ('manual','scheduled','api')),
    CONSTRAINT report_runs_status_check
        CHECK (status IN ('success','failed'))
);
CREATE INDEX idx_report_runs_tenant_started ON report_runs(tenant_id, started_at DESC);
CREATE INDEX idx_report_runs_schedule ON report_runs(schedule_id, started_at DESC);

-- Seed: 8 System-Berichte (siehe WP-2)
-- Platzhalter-Tenant-ID wie in wiki: 00000000-0000-0000-0000-000000000001
INSERT INTO report_definitions (tenant_id, name, description, module, kind, query_config, default_format, is_published)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'Umsatz (Monatlich)', 'Rechnungsumsatz, gruppiert pro Monat', 'finanzen', 'system', '{"kind":"revenue_by_month","period":"last_12_months"}', 'pdf', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Offene Posten', 'Rechnungen mit status=sent/overdue', 'finanzen', 'system', '{"kind":"invoices_open"}', 'xlsx', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Pipeline-Uebersicht', 'Deals pro Stage mit Volumen', 'crm', 'system', '{"kind":"pipeline"}', 'pdf', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Conversion-Funnel', 'Stage-zu-Stage Konversionsraten', 'crm', 'system', '{"kind":"conversion","period":"last_90_days"}', 'pdf', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Aktivitaeten pro Vertriebler', 'Calls/Emails/Notes pro User', 'crm', 'system', '{"kind":"activity_by_user","period":"last_30_days"}', 'xlsx', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Helpdesk-SLA', 'SLA-Compliance pro Queue', 'helpdesk', 'system', '{"kind":"helpdesk_sla","period":"last_30_days"}', 'pdf', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'Bestands-Warnungen', 'Artikel unter min_quantity', 'inventar', 'system', '{"kind":"stock_warnings"}', 'csv', TRUE),
  ('00000000-0000-0000-0000-000000000001', 'DATEV-BWA-Bruecke', 'Vorbereitung fuer DATEV-Export', 'finanzen', 'system', '{"kind":"datev_bwa","period":"current_month"}', 'csv', TRUE);
```

**Proto-Skelett:**
```proto
syntax = "proto3";
package berichte.v1;
option go_package = "github.com/Lukes-Git-Beginning/KMU-Hub/backend/proto/berichte/v1;berichtev1";
import "google/protobuf/timestamp.proto";

service BerichteService {
  // Definitions
  rpc CreateDefinition (CreateDefinitionRequest) returns (DefinitionResponse);
  rpc UpdateDefinition (UpdateDefinitionRequest) returns (DefinitionResponse);
  rpc DeleteDefinition (DeleteDefinitionRequest) returns (Empty);
  rpc GetDefinition (GetDefinitionRequest) returns (DefinitionResponse);
  rpc ListDefinitions (ListDefinitionsRequest) returns (ListDefinitionsResponse);

  // Run & Cache
  rpc RunReport (RunReportRequest) returns (RunReportResponse);
  rpc GetCachedResult (GetCachedResultRequest) returns (RunReportResponse);
  rpc InvalidateCache (InvalidateCacheRequest) returns (Empty);

  // Export
  rpc ExportReport (ExportReportRequest) returns (ExportReportResponse);  // format=pdf|csv|xlsx im request

  // Schedules
  rpc CreateSchedule (CreateScheduleRequest) returns (ScheduleResponse);
  rpc UpdateSchedule (UpdateScheduleRequest) returns (ScheduleResponse);
  rpc DeleteSchedule (DeleteScheduleRequest) returns (Empty);
  rpc ListSchedules (ListSchedulesRequest) returns (ListSchedulesResponse);
  rpc ToggleSchedule (ToggleScheduleRequest) returns (ScheduleResponse);

  // KPI-Shortcut (von BerichtePage Dashboard konsumiert)
  rpc GetDashboardKPIs (DashboardKPIsRequest) returns (DashboardKPIsResponse);
}
```

Message-Konventionen wie wiki: UUIDs als `string`, Timestamps `google.protobuf.Timestamp`, List-Responses `repeated <T> items + int32 total`, Delete liefert leere `Empty{}`. `query_config` und `params` als `bytes` (raw JSONB).

**Verifikation WP-0:**
- `make migrate-up` grün, `make migrate-down` grün.
- `buf generate` bzw. `protoc ... berichte.proto` erzeugt `.pb.go` ohne Fehler.
- `go build ./backend/proto/berichte/...` grün.

---

### WP-1 · Service-Layer + Repository (2d, nach WP-0)

**Files (1:1-Parallele zu `backend/internal/wiki/`):**
| File | LOC-Schätzung | Zweck |
|---|---|---|
| `backend/internal/berichte/models.go` | ~80 | `Definition`, `CacheEntry`, `Schedule`, `Run`, `KPI` |
| `backend/internal/berichte/errors.go` | ~15 | `ErrDefinitionNotFound`, `ErrScheduleNotFound`, `ErrInvalidQueryConfig`, `ErrInvalidCron`, `ErrCacheMiss` |
| `backend/internal/berichte/repository.go` | ~70 | Interface + `ListDefinitionsFilter`, `ListSchedulesFilter` |
| `backend/internal/berichte/postgres_repository.go` | ~550 | `PostgresRepository` mit pgxpool, Scan-Helper, tenant_id in jedem Query |
| `backend/internal/berichte/service.go` | ~500 | Business-Layer: Validation, Cache-Handling, SHA-Hash-Keying, Cron-Parsing (`robfig/cron/v3`) |
| `backend/internal/berichte/service_test.go` | ~800 | In-memory mockRepository, Unit-Tests für alle Service-Methoden, Coverage-Ziel ≥35% |

**Tenant-Filter-Pattern:** Application-Level wie wiki (kein RLS in dieser Migration — RLS kommt Sprint-2 S2.MT.1 über Option-B-Phase-1). Jede Query: `WHERE tenant_id = $N`.

**Cron-Parsing:** `github.com/robfig/cron/v3` (bereits indirect dep via bestehendes Job-System? — prüfen in go.mod; falls nicht, `go get`). `Service.ValidateCron(expr string) error` nutzt den Parser ohne Aktivierung.

**Cache-Logik:**
```go
func (s *Service) RunReport(ctx context.Context, in RunReportInput) (*ReportResult, error) {
    hash := sha256Hex(canonicalJSON(in.Params))
    if cached, err := s.repo.GetCacheEntry(ctx, in.TenantID, in.DefinitionID, hash); err == nil {
        if cached.ExpiresAt.After(time.Now()) {
            return cached.Result, nil
        }
    }
    // Cache miss oder expired -> echte Aggregation
    result, err := s.executor.Run(ctx, def, in.Params)
    // ...
    s.repo.UpsertCacheEntry(ctx, tenantID, defID, hash, result, defaultTTL)
    s.repo.InsertRun(ctx, runRecord)
}
```

**Verifikation WP-1:**
- `go test ./backend/internal/berichte/... -cover` ≥35%.
- `go vet` + `golangci-lint` sauber.
- Keine DB-Zugriffe im Service-Layer (Repository-Interface-Disziplin).

---

### WP-2 · Standard-Report-Katalog / Aggregations-Executor (1d, nach WP-1)

**Zweck:** Map `query_config.kind` → konkrete Aggregations-Implementation. Executor konsumiert bestehende Repositories **per Dependency-Injection**.

**File:** `backend/internal/berichte/executor/executor.go` (~450 LOC)

```go
type Executor struct {
    crmReports    crmReport.Repository   // aus backend/internal/crm/report/
    financeRepo   biz.FinanceRepository  // aus backend/internal/biz/
    helpdeskRepo  helpdesk.Repository
    inventarRepo  inventar.Repository    // kommt erst S2 - bis dahin stub/nil-Guard
    datevExporter *datev.Exporter
}

func (e *Executor) Run(ctx context.Context, def *berichte.Definition, params json.RawMessage) (*berichte.ReportResult, error) {
    cfg, err := parseQueryConfig(def.QueryConfig)
    switch cfg.Kind {
    case "revenue_by_month":  return e.revenueByMonth(ctx, def.TenantID, cfg, params)
    case "invoices_open":     return e.invoicesOpen(ctx, def.TenantID, cfg, params)
    case "pipeline":          return e.pipeline(ctx, def.TenantID, cfg, params)
    case "conversion":        return e.conversion(ctx, def.TenantID, cfg, params)
    case "activity_by_user":  return e.activityByUser(ctx, def.TenantID, cfg, params)
    case "helpdesk_sla":      return e.helpdeskSLA(ctx, def.TenantID, cfg, params)
    case "stock_warnings":    return e.stockWarnings(ctx, def.TenantID, cfg, params)
    case "datev_bwa":         return e.datevBWA(ctx, def.TenantID, cfg, params)
    default:                  return nil, berichte.ErrInvalidQueryConfig
    }
}
```

**Reuse (nicht kopieren):**
- `crm/report.Repository.GetPipelineReport` → wird über Dependency genutzt, kein Code-Duplikat.
- `crm/report.Repository.GetConversionReport` → analog.
- `crm/report.Repository.GetActivityReport` → analog.
- Finanzen-Aggregations: falls `biz.FinanceRepository` die Queries nicht hat, neue Methoden dort hinzufügen (`GetRevenueByMonth(ctx, tenant, from, to)`, `GetOpenInvoices(ctx, tenant)`) — nicht im berichte-Repo! Thick-Service-Prinzip.

**Ergebnis-Format:** `ReportResult` ist einheitlich JSON-serialisierbar:
```go
type ReportResult struct {
    Columns []Column         `json:"columns"`     // für Tabellen
    Rows    []map[string]any `json:"rows"`
    Series  []Series         `json:"series,omitempty"`   // für Charts (label, data-points)
    Totals  map[string]any   `json:"totals,omitempty"`
    Meta    ReportMeta       `json:"meta"`         // generated_at, row_count, definition_id
}
```

**Verifikation WP-2:**
- Unit-Test pro Kind mit gemocktem Repo → Result-Shape stimmt.
- Integration-Test mit echter DB (optional, gate-guarded): `make test-integration`.

---

### WP-3 · Export-Layer PDF/CSV/XLSX (1d, nach WP-1)

**Files:**
- `backend/internal/berichte/export/pdf.go` (~200 LOC, maroto)
- `backend/internal/berichte/export/csv.go` (~80 LOC, gocsv, BOM+Semikolon)
- `backend/internal/berichte/export/xlsx.go` (~150 LOC, excelize v2)
- `backend/internal/berichte/export/exporter.go` (~50 LOC, Dispatcher-Interface)

**Dep-Add:** `go get github.com/xuri/excelize/v2@latest` (prüfen, stabile Version pinnen).

**Dispatcher-Interface:**
```go
type Exporter interface {
    Export(result *ReportResult, w io.Writer) error
    ContentType() string
    FileExtension() string
}

func NewExporter(format string) (Exporter, error) {
    switch format {
    case "pdf":  return &PDFExporter{}, nil
    case "csv":  return &CSVExporter{}, nil
    case "xlsx": return &XLSXExporter{}, nil
    }
    return nil, ErrUnsupportedFormat
}
```

**Streaming:** PDF + XLSX sind Buffer-basiert (wie Rechnungs-PDF in `biz/pdf/generator.go`) — OK für Launch, Streaming-Upgrade in Phase D. CSV streamt direkt durch `gocsv.Marshal` + `io.Writer`.

**Verifikation WP-3:**
- Golden-File-Tests pro Format (3 Tests) mit synthetischem `ReportResult`.
- MIME-Type-Check: `Content-Type: application/pdf` / `text/csv; charset=utf-8` / `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`.
- UTF-8-BOM bei CSV vorhanden (wie DATEV-Pattern).

---

### WP-4 · Scheduled-Report-Worker (1d, nach WP-1)

**Files:**
- `backend/internal/berichte/scheduler/scheduler.go` (~250 LOC)
- `backend/internal/berichte/scheduler/scheduler_test.go` (~150 LOC)

**Ansatz:** In-Process-Cron-Loop (`robfig/cron/v3`), kein pg_cron (pg_cron-Partitionierung kommt erst S4 R2-P1.10). Worker startet beim `cmd/berichte/main.go`-Start, läuft im eigenen Goroutine, nutzt `context.Context` für Shutdown.

```go
type Scheduler struct {
    repo    ScheduleRepository
    svc     *berichte.Service
    exec    *executor.Executor
    mailer  email.Sender          // aus bestehendem email-Service
    clock   Clock                 // für Tests
    log     *slog.Logger
}

func (s *Scheduler) Run(ctx context.Context) error {
    ticker := time.NewTicker(60 * time.Second)  // minute-granular
    for {
        select {
        case <-ctx.Done(): return nil
        case now := <-ticker.C:
            schedules, _ := s.repo.ListDueSchedules(ctx, now)
            for _, sch := range schedules {
                go s.runScheduleSafe(ctx, sch, now)
            }
        }
    }
}
```

**E-Mail-Delivery:** Konsumiert `email.Sender`-Interface aus `backend/internal/email/` — wenn dort kein passendes Interface existiert, minimalen Port-Adapter `email.Sender` dort einfügen (nicht mit einem parallelen SMTP-Client arbeiten!).

**Consent-Wrapper:** Nicht nötig — Scheduled Reports gehen an **User-definierte Recipients** (interne Team-Adressen, keine Customer-Contacts). Falls Recipients in `contacts` gespiegelt werden sollen, muss `AssertConsent` davor — NICHT in Sprint 1.

**Idempotenz:** `report_schedules.last_run_at` wird atomar UPDATE bei Run-Start gesetzt; Loop-Query filtert `last_run_at < next_due_at(cron, now) - INTERVAL '1 min'` — keine Doppel-Runs bei Tick-Überlappung.

**Verifikation WP-4:**
- Scheduler-Test mit `Clock`-Mock: Cron-Expression `* * * * *`, 2 Tick-Advances → 2 Runs + 2 `report_runs`-Einträge.
- E-Mail-Sender-Mock capturn → Attachment-Filename stimmt mit Format+Definition-Name überein.

---

### WP-5 · gRPC-Server + cmd/berichte (0.5d, nach WP-1)

**Files:**
- `backend/internal/server/berichte_grpc.go` (~400 LOC)
- `backend/cmd/berichte/main.go` (~130 LOC, 1:1-Template von `cmd/wiki/main.go`)
- `backend/internal/config/config.go` → `BerichteGRPCAddress/Port/HealthPort` ergänzen

**gRPC-Server-Pattern (1:1 wiki):**
- Struct `BerichteGRPCServer{ svc *berichte.Service; exec *executor.Executor; export *export.Dispatcher }`.
- Jedes UUID-Feld mit `uuid.Parse` validieren → `codes.InvalidArgument`.
- `mapBerichteError` mit `errors.Is` (NotFound/AlreadyExists/InvalidArgument/Internal).
- Conversion-Helper `definitionToProto`, `scheduleToProto`, `runToProto`, `resultToProto` mit nil-Guards + `timestamppb.New`.
- ExportReport-RPC streamt per `stream server-side`? **Nein** — Initial Buffer-Response (`bytes payload + string filename + string content_type`), bei Files >10 MB Upgrade zu Server-Stream in Phase C.

**cmd/berichte/main.go:** Exakte Kopie des wiki-main.go, ersetze `wiki` → `berichte`, Ports `50062`→`50066`, `9102`→`9106`, Scheduler-Goroutine starten:

```go
scheduler := scheduler.New(repo, svc, exec, mailer, slog.Default())
go func() {
    if err := scheduler.Run(ctx); err != nil {
        slog.Error("scheduler stopped", "err", err)
    }
}()
```

**Verifikation WP-5:**
- `go run ./backend/cmd/berichte` startet, `/health` antwortet 200.
- `grpcurl -plaintext localhost:50066 berichte.v1.BerichteService/ListDefinitions` funktioniert.
- Graceful Shutdown: SIGTERM stoppt Scheduler + gRPC.

---

### WP-6 · Gateway-Routes + Feature-Flag-Guard (0.5d, nach WP-5)

**Files:**
- `backend/internal/gateway/route_berichte.go` (~400 LOC, Template `route_wiki.go`)
- `backend/cmd/gateway/main.go` → `registry.Register("berichte", cfg.BerichteGRPCAddress)` + `gateway.NewBerichteRoutes(registry, flagRegistry)` in `registrars`-Slice
- `backend/internal/config/config.go` → `BerichteGRPCAddress` env-bind

**Route-Tree unter `/api/v1/berichte` (RESTful):**
```
GET    /definitions                     → ListDefinitions
POST   /definitions                     → CreateDefinition
GET    /definitions/{id}                → GetDefinition
PATCH  /definitions/{id}                → UpdateDefinition
DELETE /definitions/{id}                → DeleteDefinition
POST   /definitions/{id}/run            → RunReport   (body: params JSON)
POST   /definitions/{id}/export         → ExportReport (body: params JSON, query: format)
DELETE /definitions/{id}/cache          → InvalidateCache

GET    /schedules                       → ListSchedules
POST   /schedules                       → CreateSchedule
GET    /schedules/{id}                  → (implicit via List)
PATCH  /schedules/{id}                  → UpdateSchedule
DELETE /schedules/{id}                  → DeleteSchedule
POST   /schedules/{id}/toggle           → ToggleSchedule

GET    /kpis                            → GetDashboardKPIs  (query: modules=finanzen,crm,...)
```

**Feature-Flag:** `if !br.flags.IsEnabled("modules.berichte") { return }` als allererste Zeile in `RegisterRoutes` (Pattern aus wiki).

**Permission:** `middleware.RequirePermission("berichte:reports", "read"|"write")` — RBAC-Key im ACL-Seed ergänzen, falls noch nicht vorhanden.

**Export-Response:** `Content-Disposition: attachment; filename="<def-name>-<timestamp>.<ext>"`, Body = raw bytes.

**Verifikation WP-6:**
- `curl http://localhost:8080/api/v1/berichte/definitions` mit Flag OFF → 404 (route nicht registriert).
- Flag ON via `COSMI_MODULE_BERICHTE_ENABLED=true` + Gateway-Restart → `[]` Liste (leer, System-Seeds kommen mit Migration).
- Export-Endpoint liefert gültiges PDF/CSV/XLSX (MIME + Datei-Signatur-Check).

---

### WP-7 · Docker-Integration (0.5d, nach WP-5)

**Files:**
- `backend/Dockerfile` (oder `deploy/docker/Dockerfile.berichte` — **wiki hat kein dediziertes Dockerfile** laut Sub-Agent-Report, Build läuft via zentralem Dockerfile im backend-Root. Gleiches Pattern übernehmen: cmd-Binary als Build-Arg.)
- `deploy/docker/docker-compose.yml` → neuen `berichte`-Service (Template aus `wiki`-Service)
- `deploy/docker/docker-compose.prod.yml` → Resource-Limits `memory: 256M`, `cpus: 0.25`
- `deploy/docker/docker-compose.yml` Gateway → `depends_on: berichte` + env `BERICHTE_GRPC_ADDRESS: berichte:50066`

**Compose-Entry-Template:**
```yaml
berichte:
  build:
    context: ../../backend
    dockerfile: Dockerfile
    args:
      SERVICE: berichte
  environment:
    DATABASE_URL: ${DATABASE_URL}
    JWT_SECRET: ${JWT_SECRET}
    BERICHTE_GRPC_PORT: ":50066"
    BERICHTE_HEALTH_PORT: ":9106"
    SMTP_HOST: ${SMTP_HOST}           # für Scheduled-Reports
    SMTP_PORT: ${SMTP_PORT}
    SMTP_USER: ${SMTP_USER}
    SMTP_PASS: ${SMTP_PASS}
  ports:
    - "50066:50066"
    - "9106:9106"
  depends_on:
    postgres: {condition: service_healthy}
    migrate:  {condition: service_completed_successfully}
  healthcheck:
    test: ["CMD", "wget", "--spider", "http://localhost:9106/health"]
    interval: 30s
    timeout: 5s
    retries: 3
  restart: unless-stopped
```

**Verifikation WP-7:**
- `docker compose up -d berichte` → Container healthy.
- `docker compose logs berichte` → keine Panics, Scheduler-Start-Log sichtbar.
- Gateway kann `berichte:50066` auflösen: `docker compose exec gateway wget --spider http://berichte:9106/health`.

---

### WP-8 · Frontend-Client + Types + Hooks (1d, parallel zu WP-1 ab Proto-Freeze)

**Files (1:1-Template wiki):**
- `desktop/src/renderer/src/api/berichte-client.ts` (~300 LOC)
- `desktop/src/renderer/src/api/berichte-types.ts` (~150 LOC)
- `desktop/src/renderer/src/api/hooks/useBerichte.ts` (~400 LOC)

**Client-Wrapper:** Fetch-basiert wie `wiki-client.ts`. Endpoints:
- `listDefinitions(filter)`, `getDefinition(id)`, `createDefinition(input)`, `updateDefinition(id, input)`, `deleteDefinition(id)`
- `runReport(defId, params)`, `exportReport(defId, format, params)` — returnt `Blob` für Download-Triggering via `URL.createObjectURL`
- `invalidateCache(defId)`
- `listSchedules(filter)`, `createSchedule(input)`, `updateSchedule(id, input)`, `deleteSchedule(id)`, `toggleSchedule(id, active)`
- `getDashboardKPIs(modules?)` — leichtgewichtig, kein Cache

**Types:**
```ts
export type ReportFormat = 'pdf' | 'csv' | 'xlsx';
export type ReportModule = 'finanzen'|'crm'|'helpdesk'|'inventar'|'produktion'|'cross';
export type ReportKind = 'system'|'custom';

export interface ReportDefinition { id: string; tenantId: string; name: string; description: string;
  module: ReportModule; kind: ReportKind; queryConfig: Record<string, unknown>;
  defaultFormat: ReportFormat; createdBy: string | null; isPublished: boolean;
  createdAt: string; updatedAt: string; }

export interface ReportResult { columns: ReportColumn[]; rows: Record<string, unknown>[];
  series?: ReportSeries[]; totals?: Record<string, unknown>; meta: ReportMeta; }

export interface ReportSchedule { id: string; definitionId: string; name: string;
  cronExpression: string; recipients: string[]; format: ReportFormat; params: Record<string, unknown>;
  active: boolean; lastRunAt: string | null; lastRunStatus: 'success'|'failed'|'skipped' | null;
  lastRunError: string | null; createdAt: string; updatedAt: string; }

export interface DashboardKPI { id: string; label: string; value: number | string; unit?: string;
  changePercent?: number; moduleId: ReportModule; }
```

**React-Query-Keys:**
```ts
export const berichteKeys = {
  root: ['berichte'] as const,
  definitions: (filter?: any) => ['berichte','definitions', filter] as const,
  definition: (id: string) => ['berichte','definitions', id] as const,
  result: (id: string, paramsHash: string) => ['berichte','definitions', id,'result', paramsHash] as const,
  schedules: (filter?: any) => ['berichte','schedules', filter] as const,
  kpis: (modules?: string[]) => ['berichte','kpis', modules] as const,
};
```

**Export-Download-Helper:**
```ts
async function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url; a.download = filename; a.click();
  URL.revokeObjectURL(url);
}
```

**Verifikation WP-8:**
- `npm run typecheck` grün.
- `npm run lint` grün.
- Smoke mit Mock-Fetch: `useBerichte().useDefinitions()` → liefert System-Berichte-Array.

---

### WP-9 · Recharts-Integration + BerichtePage-Migration (1.5d, nach WP-8)

**Dep-Add:**
```bash
cd desktop
npm install recharts
npm install --save-dev @types/recharts  # falls nicht in recharts bundled
```

**Files:**
- `desktop/src/renderer/src/modules/berichte/BerichtePage.tsx` — **refactoren**, nicht neu (1186 LOC → ~1000 LOC netto durch Mock-Entfernung)
- Neue Components-Extraktion (~250 LOC, reduziert BerichtePage):
  - `components/KPICard.tsx` (mit `changePercent`-Indikator)
  - `components/DashboardGrid.tsx` (Recharts-Bar + LineChart + PieChart je nach `ReportResult.series`)
  - `components/ReportBuilder.tsx` (Tab "Erstellen" extrahiert, echter Export-Button)
  - `components/ScheduleList.tsx` (Tab "Geplant" extrahiert, API-driven)
  - `components/DatevView.tsx` (Tab "DATEV" — konsumiert `datev_bwa`-System-Bericht)

**Mock-Store-Entfernung:**
- `stores/berichte.ts` → **Datei behalten für Rollback-Sicherheit** aber exports entfernen, Kommentar `// SUPERSEDED by api/hooks/useBerichte.ts — remove in Sprint 2 cleanup`.

**Chart-Patterns:**
- KPI-Cards: kein Chart, nur Value + Trend-Arrow (Recharts nicht nötig)
- Drilldown-Timeline: `<LineChart>` mit `series.dataPoints`
- Vergleichs-Balken: `<BarChart>` mit `series[].label + data`
- Pipeline-Funnel: `<FunnelChart>` (Recharts shape)

**Design-Compliance (CLAUDE.md UI-Directive):**
- Keine generischen Recharts-Defaults — eigene `ChartTheme`-Utility (Primär + Akzent + Neutral aus `var(--primary)`/`var(--accent-1)`).
- Responsive via `<ResponsiveContainer>`, Anti-Pattern "fixed height grid of identical cards" vermeiden: 1 Hero-Chart + Cluster kleinerer KPIs.
- `prefers-reduced-motion` respektieren: Recharts-Animationen via `isAnimationActive={!prefersReducedMotion}`.

**Verifikation WP-9:**
- `npm run dev` → BerichtePage zeigt echte Daten aus Backend (mit Flag=ON + Scheduler=OFF für schnelleren Start).
- Alle 4 Tabs funktional: Dashboard zeigt KPIs aus `getDashboardKPIs`, Erstellen erzeugt echten Export-Download, Geplant listet echte Schedules, DATEV rendert `datev_bwa`-System-Bericht.
- Dev-Server Screenshot gegen bestehenden Mock verglichen — keine Regressionen.

---

### WP-10 · Tests & Coverage (durchgängig)

**Backend:**
- `backend/internal/berichte/service_test.go` — ≥35% Coverage (wiki=38.2%, helpdesk=39.3% als Benchmark)
- `backend/internal/berichte/executor/executor_test.go` — pro Kind ein Test mit gemockten Repos
- `backend/internal/berichte/export/*_test.go` — Golden-File-Tests
- `backend/internal/berichte/scheduler/scheduler_test.go` — Clock-Mock + Email-Mock
- `backend/internal/server/berichte_grpc_test.go` — Request-Validation-Edge-Cases

**Gateway:**
- `backend/internal/gateway/route_berichte_test.go` — Feature-Flag-OFF → 404, Flag-ON → Route erreichbar

**Frontend:**
- `desktop/src/renderer/src/api/hooks/useBerichte.test.ts` — Query-Key-Stabilität, Invalidation-Ketten

**Integration:**
- `backend/integration/berichte_test.go` (gate-guarded via build-tag `integration`) — End-to-End: Create Definition → RunReport → GetCachedResult → Export → InvalidateCache.

---

### WP-11 · Feature-Flag-Aktivierung + Smoke (0.5h, zum Schluss)

**Steps:**
1. `COSMI_MODULE_BERICHTE_ENABLED=true` in `deploy/docker/.env.dev` setzen (nicht `.env.example` — der ist für prod-Template!).
2. `deploy/scripts/smoke.sh` um berichte-Checks erweitern:
   - `/api/v1/berichte/definitions` → 200, Liste mit 8 System-Berichten.
   - `/api/v1/berichte/definitions/<umsatz-id>/run` → 200, `ReportResult` mit Rows.
   - `/api/v1/berichte/definitions/<umsatz-id>/export?format=pdf` → 200, MIME `application/pdf`.
3. Commit: `feat(berichte): activate module flag for dev + extend smoke script`.

---

## 6. Test-Strategie (Überblick)

| Layer | Coverage-Ziel | Tooling |
|---|---|---|
| Service (`internal/berichte/`) | ≥35% | go test + mockRepository |
| Executor (`executor/`) | ≥60% | go test + mocked downstream repos |
| Export (`export/`) | ≥80% | golden-file |
| Scheduler (`scheduler/`) | ≥50% | go test + Clock-Mock + Email-Mock |
| gRPC-Server | ≥40% | go test |
| Gateway-Routes | ≥30% | httptest + fake gRPC-Client |
| Frontend-Hooks | Smoke | jest + MSW oder React-Query-Test-Utils |
| Integration E2E | 1 happy-path + 1 error-path | Docker-Compose-Test |

---

## 7. Verifikations-Gate für S1.2 "Done"

Checkliste — alle Punkte grün bevor S1.2 in `docs/ROADMAP.md` als ✅ Done markiert wird:

- [ ] Migration `000079_create_berichte.up.sql` auf main, `make migrate-up` + `make migrate-down` rund
- [ ] 8 System-Berichte-Seeds in Migration enthalten
- [ ] `backend/internal/berichte/` vollständig, Coverage ≥35% (wie wiki/helpdesk)
- [ ] 14 RPCs in Proto + gRPC-Server implementiert
- [ ] Executor deckt alle 8 `kind`-Werte ab (Test pro Kind)
- [ ] PDF/CSV/XLSX Export je mit Golden-Test
- [ ] Scheduler-Loop startet/stoppt sauber, 1 Test-Schedule läuft in Integration-Test
- [ ] `cmd/berichte/main.go` baut, `go run` erreicht `/health` 200
- [ ] Gateway-Route hinter `modules.berichte`-Flag, OFF → 404, ON → funktional
- [ ] `route_berichte_test.go` prüft Flag-Gate
- [ ] Docker: `docker compose up berichte` healthy, Gateway depends_on aktualisiert
- [ ] `deploy/docker/docker-compose.prod.yml` Resource-Limits gesetzt
- [ ] Frontend: `useBerichte.ts` + `berichte-client.ts` + `berichte-types.ts` vorhanden, `npm run typecheck` + `npm run lint` grün
- [ ] Recharts integriert, BerichtePage auf API migriert, Mock-Store-Exports entfernt
- [ ] `deploy/scripts/smoke.sh` um 3 berichte-Checks erweitert
- [ ] `.knowledge/api.md` + `.knowledge/datenbank.md` + `.knowledge/_index.md` aktualisiert (`berichte` Endpoint-Domain + Migration 079)
- [ ] `docs/ROADMAP.md` S1.2 → ✅ Done + Commit-SHA
- [ ] `memory/project_sprint1_progress.md` um Welle 5 erweitert

---

## 8. Deliverable-Mapping pro Worktree

Für parallele Abarbeitung (User-Wunsch "Context-Schoner"):

### Worktree A — "berichte-core" (Backend-Kern)
- WP-0 (ganz)
- WP-1 Service-Layer
- WP-2 Executor + Standard-Katalog
- WP-4 Scheduler
- WP-10 Backend-Tests

**Branch:** `sprint1/berichte-core` oder direct-to-main commits `feat(berichte): scaffold migration+proto+models`, `feat(berichte): repository+service`, `feat(berichte): executor for 8 report kinds`, `feat(berichte): scheduled report worker`.

### Worktree B — "berichte-wiring" (Export + Serve)
- WP-3 Export-Layer
- WP-5 gRPC-Server + cmd-Binary
- WP-6 Gateway-Routes
- WP-7 Docker-Integration

**Branch:** direct-to-main commits `feat(berichte): pdf+csv+xlsx exporters`, `feat(berichte): grpc server + cmd binary`, `feat(berichte): gateway routes behind feature flag`, `feat(berichte): compose integration`.

### Worktree C — "berichte-frontend"
- WP-8 Client + Hooks
- WP-9 Recharts + Page-Refactor

**Branch:** direct-to-main commits `feat(desktop): berichte api client + react-query hooks`, `feat(desktop): recharts integration`, `feat(desktop): migrate BerichtePage to live api`.

### Final (nach allen 3 Worktrees grün)
- WP-11 Smoke-Integration
- Doc-Updates (`.knowledge/`, ROADMAP, Sprint-1-Progress)

---

## 9. Risiken & Mitigationen

| Risiko | Wahrscheinlichkeit | Mitigation |
|---|---|---|
| excelize v2 zieht große Abhängigkeitskette | mittel | go.sum-Audit, ggf. nur CSV+PDF wenn Dep-Bloat >20 MB |
| Recharts-Bundle-Size-Impact auf Desktop | niedrig | lazy-import in `BerichtePage.tsx`, `React.lazy` + `Suspense` |
| Scheduler-Race bei Gateway-Restart | niedrig | `last_run_at` atomar via `UPDATE ... WHERE last_run_at IS NULL OR last_run_at < $now_minus_grace`. Multi-Instance-Safety kommt mit R2-P1.7 (Redis-WS-State) — bis dahin Single-Instance OK (berichte-Service skaliert nicht horizontal in S1) |
| Bestehender `crm/report`-Code deckt Report-Shape nicht vollständig ab | mittel | Executor darf eigene Queries durch `crmReports`-Adapter durchreichen; bei Shape-Mismatch: `ReportResult`-Mapper-Layer in Executor |
| Frontend-Tab "Erstellen" ist komplex (Form-Builder für query_config) | hoch | Sprint-1-Scope: nur System-Berichte (`kind=system`). Custom-Definitions-Editor als S1.2-Followup markieren, in Sprint 2 oder Phase C erledigen |
| E-Mail-Delivery benötigt SMTP-Config die erst in Prod vorliegt | hoch | Scheduler no-op wenn `email.Sender` nil zurückgibt — Env-Var-Check `SMTP_HOST`, grace skip bis Pilot-Go-Live |

---

## 10. Follow-ups (außerhalb S1.2, nicht blockierend)

- **Custom-Definitions-Editor** (visuelles Query-Builder-UI) → Phase C
- **Report-Sharing via Short-Links** (wie wiki_share_tokens) → Phase C
- **Streaming-Export für >10 MB Files** → Phase D
- **pg_cron-Integration statt in-Process-Scheduler** → zusammen mit R2-P1.10 (Partitionierung, Sprint 4)
- **Report-RLS-Policy** (Option-B Phase 1) → Sprint 2 S2.MT.1 zieht `report_definitions/report_cache/report_schedules/report_runs` mit rein
- **Redis-Backed Cache für report_cache** → Phase D zusammen mit Performance-Redis-Layer

---

## 11. Referenz-Commits (bereits auf main, als Vorlage)

Alle Referenz-Commits sind aus der Wiki/Helpdesk-Arbeit vom 2026-04-18:
- Proto + Migration: siehe `backend/migrations/000076_create_wiki.up.sql` + `backend/proto/wiki/v1/wiki.proto`
- Service-Layer: `backend/internal/wiki/` (16k LOC insg.)
- gRPC-Server + cmd: `backend/internal/server/wiki_grpc.go` + `backend/cmd/wiki/main.go`
- Gateway-Route: `backend/internal/gateway/route_wiki.go` (Feature-Flag-Pattern in Zeile 1)
- Docker: `docker-compose.yml`/`docker-compose.prod.yml`-Diffs aus Welle 4
- Frontend: `desktop/src/renderer/src/api/wiki-client.ts` + `wiki-types.ts` + `api/hooks/useWiki.ts`

**Wichtig:** Wiki hat **kein dediziertes `Dockerfile.wiki`** — der zentrale `backend/Dockerfile` akzeptiert `SERVICE`-Build-Arg. Gleiches Muster für berichte übernehmen (keine Dockerfile-Proliferation).

---

## 12. Nach Approval

1. Diesen Plan-File 1:1 nach `docs/SPRINT1_BERICHTE.md` kopieren (Commit: `docs(sprint-1): add S1.2 berichte work-package plan`).
2. `memory/project_sprint1_progress.md` um Zeile ergänzen: "Welle 5 geplant: S1.2 berichte — Plan in `docs/SPRINT1_BERICHTE.md`".
3. Worktree A starten mit WP-0 (seriell), sobald Proto + Migration gemerged sind Worktree B+C freischalten.
4. Nach erfolgreichem Gate S1.2 → `docs/SPRINT1_BERICHTE.md` bekommt Header-Badge "✅ Shipped YYYY-MM-DD" (wie MODULES_SCOPE_MATRIX es konsistent macht).

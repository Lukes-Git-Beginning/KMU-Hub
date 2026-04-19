---
tags: [architektur, backend, frontend, ci-cd]
updated: 2026-04-19
---
# Architektur

## Backend (Go Microservices)

Gateway (HTTP/chi) auf Port 8080 → gRPC-Services intern:

| Service | Port | Domain |
|---------|------|--------|
| auth | :50051 | Users, Sessions, TOTP, RBAC |
| crm | :50052 | Contacts, Companies, Deals, Pipeline, Activities, Tags |
| chat | :50053 | Channels, Messages, DMs, Threads, Reactions, Presence |
| notification | :50054 | Notifications, Delivery, Preferences |
| work | :50055 | Projects, Tasks, Time Entries, Calendar, Events, Video |
| email | :50056 | IMAP/SMTP Accounts, Messages, Signatures |
| document | :50057 | Files, Folders, Versions, WOPI Locks |
| biz | :50058 | Finance, HR, Bexio, Lexware, DATEV, PDF, Dunning |
| automation | :50059 | Workflows, Conditions, Actions, Templates |
| plugin | :50060 | WASM Sandbox, SDK, Host API, Lifecycle |
| dialer | :50061 | Campaigns, Call Sessions, Agent Status, Outcomes |

Gateway `/health`: status, checks, registered_services, version, commit, build_time (via ldflags)

### Gateway Route-Dateien (38)

Alle Handler sind duenne gRPC-Proxies. Keine direkte DB-Abfrage im Gateway (ausser Dashboard, gekapselt).
Shared Helpers: `validateUUIDParam`, `RequireAuthenticated` Middleware, `respondGRPCError`, `parsePagination`.
Entry Point: `cmd/gateway/main.go` (~324 LoC) + `setup.go` + `adapters.go`.

| Datei | Domain |
|-------|--------|
| `route_auth.go` | Auth, 2FA, Passwoerter |
| `route_bexio.go` | Bexio OAuth + Sync |
| `route_biz.go` | Finance Struct + RegisterRoutes + Enum-Helpers |
| `route_biz_quotes.go` | Angebote (CRUD, PDF, Send, Accept, Reject) |
| `route_biz_invoices.go` | Rechnungen (CRUD, PDF, ZUGFeRD, Send) |
| `route_biz_billing.go` | Zahlungen, Mahnwesen, Gutschriften, DATEV |
| `route_biz_ext.go` | Time-to-Invoice (duenner gRPC-Proxy) |
| `route_caldav.go` | CalDAV/CardDAV Proxy + App-Passwoerter |
| `route_calendar.go` | Kalender, Events, Ressourcen |
| `route_chat.go` | Chat, Channels, Nachrichten |
| `route_crm.go` | CRM Struct + RegisterRoutes |
| `route_crm_contacts.go` | Kontakte (CRUD, Import/Export, Tags) |
| `route_crm_companies.go` | Firmen (CRUD) |
| `route_crm_pipeline.go` | Pipeline Stages + Deals |
| `route_crm_activities.go` | Aktivitaeten, Suche, Filter, Reports, Custom Fields, Tags |
| `route_crm_ext.go` | CRM Extended (Consent, Duplikate, Timeline — gRPC) |
| `route_dashboard.go` | Dashboard-Widgets (gateway-local) |
| `route_datev_upload.go` | DATEV CSV-Upload |
| `route_dialer.go` | Dialer (Campaigns, Calls, Agent Status, Outcomes) |
| `route_feature_flags.go` | `GET /api/v1/feature-flags` — Resolver-Output fuer Frontend-Gate |
| `route_document.go` | Dokumente (CRUD) |
| `route_email.go` | E-Mail-Konten, IMAP/SMTP |
| `route_guest.go` | Guest Chat (public, kein Auth) |
| `route_health.go` | `/health` (public) |
| `route_hr.go` | HR (Mitarbeiter, Abwesenheit, Urlaub, Schichten) |
| `route_inbox.go` | Unified Inbox (Messages, Routing, Teams) |
| `route_integration.go` | Integrations-Config, Teams/Slack Webhooks |
| `route_lexware.go` | Lexware API |
| `route_notification.go` | Benachrichtigungen |
| `route_plugin.go` | Plugin Install/Run/Manifests/Templates |
| `route_registrar.go` | RouteRegistrar Interface |
| `route_search_global.go` | Cross-Service Global Search (500ms Timeout) |
| `route_security.go` | Audit Logs, GDPR Export, Passwort-Policy |
| `route_video.go` | Video-Calls (LiveKit) |
| `route_wopi.go` | WOPI Document Protocol (OnlyOffice) |
| `route_work.go` | Work Struct + RegisterRoutes |
| `route_work_projects.go` | Projekte (CRUD, Members, Templates, Statuses) |
| `route_work_tasks.go` | Tasks (CRUD, Comments, Dependencies, Files) |
| `route_work_time.go` | Timer + Zeiterfassung |

## Frontend (Electron + React 19 + TypeScript)

### Module (36 in `modules/`)
admin, auth, automatisierung, berichte, buchhaltung, calendar, chat, crm, dashboard, **dialer**, dokumente, einkauf, finanzen, formulare, fuhrpark, helpdesk, inventar, kalender, kommunikation, kontakte, mails, meetings, notifications, produktion, profil, rapporte, schichten, security, settings, team, vermietung, vertraege, video, wiki, work, zeiterfassung

### Dialer-Modul (Phase 1 MVP ✅ 2026-04-09)
- 6 Pages: DialerLayout (4-Tab Sub-Nav), CampaignList, CampaignDetail, DialerWorkspace, AgentDashboard, DialerSettings
- 15 Komponenten in `components/` (workspace/, dashboard/, settings/)
- Workspace: 4-Phasen Call-Flow (idle → dialing → on_call → wrap_up) mit `animate-scale-in` Transitions
- Data: `api/dialer-client.ts` (typed fetch, nicht openapi-fetch) + `api/hooks/useDialer.ts` (24 Hooks)
- Zusatz-Hooks: `api/hooks/useTimeline.ts` (CRM Contact Timeline), `api/hooks/useSavedFilters.ts` (typed apiClient)
- Nav: `PhoneCall` Icon, Farbe h:142 s:72 (Gruen), Live-Dot Badge bei aktivem Call
- AddContactsDialog: Tab-Toggle (Einzelne Kontakte / Gespeicherter CRM-Filter)
- ContactTimeline: Live-Daten via `GET /api/v1/crm/contacts/{id}/timeline` (kein Mock mehr)
- EventEmitter: `PGEventEmitter` in `backend/internal/dialer/event_emitter.go` — emittiert `dialer.call.outcome_logged`, `dialer.contact.callback_scheduled`, `dialer.campaign.completed`
- NotificationType `dialer_callback` + PhoneCall-Icon im NotificationCenter

### Zustand Stores (37 in `stores/`)
ai, auth, automatisierung, berichte, calendar, contacts, dashboard, **dialer**, einkauf, finance, formulare, fuhrpark, helpdesk, integrations, inventar, kommunikation, locale, mails, meetings, navigation, notifications, presence, produktion, profile, rapporte, schichten, search, settings, team, timetracking, tour, ui, vermietung, vertraege, video, wiki, work

### Standalone
- Guest Chat: Separate Vite SPA unter `/guest/`

## Performance-Patterns

### N+1 Query Elimination (2026-04-08)
- Contact-Liste: `enrichWithRelationsBatch()` — 4 Queries statt 61 (Batch-Loading per `ANY($1)`)
- Deal-Liste: `enrichWithRelationsBatch()` — 7 Queries statt 121
- Batch-Inserts: Tags per `unnest($2::uuid[])`, Custom Fields per `pgx.Batch`
- Einzelabfragen (`getWithRelations`) bleiben fuer GetByID/Create/Update (nur 1 Entity)

### Gateway Performance & Security
- **Audit Logger:** Buffered Channel (1000) + Worker Pool (10 Workers) statt unbegrenzte Goroutines
- **gRPC Keep-Alive:** 60s/10s, PermitWithoutStream (in `registry.go`)
- **gRPC mTLS:** Optional via Env-Vars, `ServiceRegistry` akzeptiert `*tls.Config` (nil = insecure)
- **IP Filter:** TTL-basierter Fail-Close (5min Max-Staleness), Details in [[security]]
- **pprof:** `/debug/pprof` hinter `ENABLE_PPROF=true` Env-Var (nur Staging/Dev)
- **Connection Pool:** MaxConns=10 pro Service (10×10=100 = PG-Limit), MaxConnLifetime=1h, HealthCheckPeriod=1m

### Gateway Bloat Refactoring (2026-04-10)
- 14 neue gRPC RPCs (11 CRM: Duplikate, Merge, Timeline, GDPR Consent; 3 Biz: TimeToInvoice, QuoteFromDeal, ZUGFeRD)
- `*pgxpool.Pool` Nutzung im Gateway von 5 auf 1 (Dashboard, gekapselt via `NewDashboardStack`)
- Domain-Imports eliminiert: `internal/biz/pdf`, `internal/crm/*`, `shopspring/decimal`, `internal/models`
- CalDAV raw SQL → `caldav.UserPreferenceRepository`
- Boilerplate: `validateUUIDParam` + `RequireAuthenticated` Middleware (~570 Zeilen reduziert)
- 3 grosse Dateien gesplittet: route_crm (2335→164), route_work (1800→140), route_biz (1437→308)
- main.go (525→324) via Extraktion nach setup.go + adapters.go

### React Compiler (2026-04-08)
- `babel-plugin-react-compiler` im `annotation` Mode
- `"use memo"` Directive auf DashboardPage, DealPipelineView, ContactsListPage
- Automatisches Memoization ohne manuelle useMemo/useCallback

## Feature-Flag-Subsystem (2026-04-18, Sprint 0 S0.6)

- **Backend:** `backend/internal/featureflag/registry.go`
  - Typed `Flag{Key, DefaultEnabled, EnvVar, Description, Risk, LLMToggleSafe}`
  - `Risk`-Enum: `safe | breaking | security` — plus `LLMToggleSafe`-Flag, damit spaetere LLM-gesteuerte Toggles wissen, was sie anfassen duerfen
  - Env-Loader, `IsEnabled(key)`, `All()`, sortierte `Keys()`
  - 16 Flags registriert: 14 Modul-Flags (`modules.<name>`, Default off) + `plugins.wasm` (off) + `plugins.config` (on)
  - EnvVar-Konvention Modul-Flags: `COSMI_MODULE_<UPPER>_ENABLED`
- **API:** `GET /api/v1/feature-flags` (auth-required) → `{flags: {[key]: bool}, version: "v1"}`
- **Frontend:**
  - `api/hooks/useFeatureFlags.ts` — React-Query, Refetch on focus, `isEnabled(key)`-Helper
  - `components/shared/FeatureGate.tsx` — `<FeatureGate flag="modules.wiki" fallback={null}>`
  - `hooks/useFilteredNavItems.ts` — Nav-Items nach Modul-Flag filtern (Mapping video-Nav-ID `meetings` → `modules.video` explizit)
- **Wiring:** Registry in `cmd/gateway/main.go` gebaut (explizit, kein Package-Global, keine `init()`).
- **Zweck:** Safety-Net fuer Modul-Slippage — jedes Modul, das bis Launch nicht produktionsreif wird, laesst sich per Env-Flag ausblenden.

## Consent Enforcement Wrapper (2026-04-18, Sprint 0 S0.2)

- **Package:** `backend/internal/crm/consent/`
  - `Asserter.Assert(ctx, contactID, channel)` mit `ChannelEmail | ChannelPhone`
  - `ErrNoConsent`, `ErrContactMissing`
  - Transactional Skip bei `ChannelEmail` wenn Contact keine E-Mail hat
  - Blocked Attempts: `slog.Warn("consent_block", ...)`
- **Integration:**
  - `backend/internal/email/send/service.go` — Check vor SMTP-Dispatch
  - `backend/internal/dialer/service.go` — Check vor Twilio/Dialer-Call
  - Additive `NewServiceWithConsent()`-Constructors (Gateway-Wiring separater Schritt)
- **Repo-Query:** `consent_records WHERE contact_id=$1 AND consent_type=$2 AND granted=true AND revoked_at IS NULL`
- Details: [[security]]

## WASM-Plugin-System — Feature-Flag OFF (Sprint 0 S0.6 / R2-P1.2)

- Runtime-Files (`runtime.go`, `sandbox.go`, `hostapi.go`, `memory.go`, `lifecycle.go`) mit Build-Tag `//go:build !no_wasm`
- Stub: `backend/internal/plugin/wasm/runtime_disabled.go` mit `//go:build no_wasm` — exportiert die gleiche API, alle Operationen no-op
- Runtime-Toggle zusaetzlich ueber `plugins.wasm`-Flag (Default off)
- `make build-prod` setzt `-tags no_wasm` → Production-Builds enthalten keinen WASM-Code-Pfad mehr
- Details: [[integrationen]]

## Demo Mode

- Aktiviert via `RENDERER_VITE_DEMO_MODE=true` (Build-Time Env)
- **Fetch-Interceptor** (nicht MSW Service Worker)
- Grund: Electron Production nutzt `file://` Protokoll — Service Workers + dynamic import() nicht unterstuetzt
- Handler: `mocks/handlers/` (19 Dateien inkl. Dialer mit 617 LoC, statisch importiert, Vite eliminiert in non-demo Builds)
- Mock-Daten: `mocks/data/` (8 typisierte Dateien)
- Dev: `npm run dev` → `--mode demo`; `npm run dev:test` → + CDP Port 9222

## i18n-Architektur

- i18next v26 + react-i18next v17 + i18next-icu v2
- 41 Additions-JSONs mergen in `de.json` via `mergedDE` in `i18n.ts`
- Vollstaendig dokumentiert in [[i18n]]

## Datenbank

- PostgreSQL 16 + Redis 7 (Redis = Cache only, kein Dual-Write!)
- Aenderungen NUR via golang-migrate
- Details: [[datenbank]]

## Auth

- JWT 15min + opaque refresh 7d, SHA-256, rotation + theft detection
- RBAC: admin/manager/member, resource:action permissions
- 2FA (TOTP), vault, audit logging, GDPR erasure
- Details: [[security]]

## CI/CD

### Workflows (5 Dateien in `.github/workflows/`)

| Workflow | Trigger | Zweck |
|----------|---------|-------|
| `ci.yml` | Push/PR auf main/develop (backend/**) | Lint → Test → Build → E2E → Smoke → OpenAPI Validate |
| `ci-desktop.yml` | Push/PR auf main/develop (desktop/**) | Lint → Typecheck → Test → Build |
| `cd.yml` | Manual (workflow_dispatch) | SSH → deploy.sh → Health Check |
| `claude-pr.yml` | PR open/sync, @claude Mention | Automatisches Claude Code PR-Review |
| `security-review.yml` | PR | Security-fokussiertes Code-Review |

### CI Details
- Go 1.25.6, golangci-lint v2.8 (action v7)
- Postgres + Redis Service Containers fuer Tests
- E2E: `backend/test/e2e/` (Build Tag `e2e`)
- Smoke: Dual — Bash (`smoke.sh`, 19 Tests) + Go (`test/smoke/`, 11 Tests)

## Config-Dateien

| Datei | Zweck |
|-------|-------|
| `backend/.golangci.yml` | Linter-Config (version: "2") |
| `backend/api/openapi.yaml` | API-Spec (14.000+ Zeilen, OpenAPI 3.0.3) |
| `deploy/docker/docker-compose.yml` | Lokale + Self-Hosted Infrastruktur |
| `deploy/docker/docker-compose.prod.yml` | Production Overlay (Caddy, Prometheus, Grafana) |
| `deploy/k8s/` | Kubernetes-Manifeste (sekundaer, Kustomize) |
| `desktop/electron.vite.config.ts` | Vite + Electron Build-Config |

## Entwicklungs-Kommandos

### Backend (Go)

```bash
cd backend
make run-gateway                    # API Gateway starten (Port 8080)
make run-crm                        # CRM Service starten
make run-chat                       # Chat Service starten
make run-auth                       # Auth Service starten

make build                          # Alle Services bauen
make build-prod                     # Production Build (-tags no_wasm)
make test                           # Alle Tests
make test-coverage                  # Mit Coverage-Report
make lint                           # golangci-lint

make migrate-up                     # Migrations ausfuehren
make migrate-down                   # Letzte Migration zurueckrollen
make migrate-create name=xxx        # Neue Migration erstellen
```

### Desktop (Electron)

```bash
cd desktop
npm install
npm run dev                         # Electron Dev-Modus
npm run dev:test                    # Dev + CDP Port 9222 (Playwright MCP)
npm run build                       # Production Build
npm run test                        # Vitest
npm run lint                        # ESLint
```

### Docker (Lokale Entwicklung)

```bash
cd deploy/docker
docker-compose up -d                # Alle Services starten
docker-compose logs -f gateway      # Logs eines Services
docker-compose down                 # Alles stoppen (Volumes bleiben)
docker-compose down -v              # Inkl. Volumes (zerstoerend!)
```

## Architektur-Regeln (Detail)

Bullet-Liste mit Quick-Hinweis steht in `CLAUDE.md`. Hier die ausfuehrliche Version mit Code-Beispielen.

### 1. Thick Services, Thin Handlers

Business-Logik gehoert in Services, NICHT in HTTP-Handler. Handler sind nur fuer:
- Request parsen / validieren
- Service aufrufen
- Response formatieren

```go
// RICHTIG
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateContactRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, err)
        return
    }
    contact, err := h.contactService.Create(r.Context(), req)
    if err != nil {
        respondError(w, http.StatusInternalServerError, err)
        return
    }
    respondJSON(w, http.StatusCreated, contact)
}

// FALSCH - Business-Logik im Handler
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
    // ... Validierung, DB-Zugriff, E-Mail senden direkt hier
}
```

### 2. Centralized Service Registry

Alle Services ueber zentrale Stelle initialisieren und injizieren. Kein `init()`-Missbrauch, keine globalen Variablen.

```go
type ServiceRegistry struct {
    ContactService *contact.Service
    DealService    *deal.Service
    AuthService    *auth.Service
    // ...
}
```

Aktueller Stand: explizit gebaut in `cmd/gateway/main.go` + `setup.go` + `adapters.go` (Bloat-Refactoring 2026-04-10, siehe oben).

### 3. Structured Logging von Tag 1

Kein `fmt.Println()`, immer `slog`:

```go
// RICHTIG
slog.Info("contact created",
    "contact_id", contact.ID,
    "user_id", userID,
)

// FALSCH
fmt.Printf("Created contact %s\n", contact.ID)
log.Println("contact created")
```

### 4. API-First Design

OpenAPI-Spec VOR Implementation schreiben. Code wird gegen die Spec validiert (siehe `backend/api/openapi.yaml`, OpenAPI 3.0.3, ~14k Zeilen). Validation-Step in CI: `OpenAPI Validate`.

### 5. Database Migrations

IMMER via Migration-Tool (golang-migrate), NIE manuell SQL auf der DB:

```bash
make migrate-create name=add_contacts_table
# -> backend/migrations/000001_add_contacts_table.up.sql
# -> backend/migrations/000001_add_contacts_table.down.sql
```

Index-Naming-Convention: `idx_{table}_{column}` (z.B. `idx_contacts_email`).

### 6. Test Coverage

- **Gesamt:** 15%+ Minimum (CI-enforced), Ziel 40% bis Q3 2026
- **Kritische Pfade (Auth, Payments, Data):** 60%+ Ziel
- **Jeder PR:** Muss Tests enthalten fuer neuen Code
- **Test-Isolation:** Jeder Test raeumt seine Daten auf, keine Abhaengigkeiten zwischen Tests

Details: [[testing]].

### 7. Security First

- Auth + Rate Limiting + Input Validation von Anfang an
- CSRF-Schutz fuer alle mutierenden Endpoints
- SQL-Injection: Immer Prepared Statements, nie String-Concatenation
- CORS: Explizite Allowlist, kein Wildcard
- Secrets: Immer ueber Environment Variables, nie im Code

Details: [[security]].

### 8. Graceful Degradation

Services muessen unabhaengig ausfallen koennen. Beispiele:
- CRM funktioniert, auch wenn Chat offline ist
- Dialer-Workspace bleibt nutzbar, wenn Notification-Service down ist
- Frontend faellt auf Read-only zurueck, statt komplett zu crashen, wenn ein Backend-Service nicht erreichbar ist

### 9. Config ueber Environment Variables

```bash
# .env (NIE committen — wird vom Pre-Commit-Hook geblockt, siehe troubleshooting)
DATABASE_URL=postgres://user:pass@localhost:5432/kmuhub
REDIS_URL=redis://localhost:6379
JWT_SECRET=...
LIVEKIT_URL=...
LIVEKIT_API_KEY=...
LIVEKIT_API_SECRET=...
```

Nie hardcoded. Production-Secrets werden beim Start asserted (`backend/internal/config/secrets.go`), siehe [[security]].

### 10. Idempotente Operationen

Alle mutierenden API-Calls muessen sicher wiederholbar sein. Idempotency-Keys fuer POST-Requests (`Idempotency-Key`-Header). Nuetzlich besonders bei Dialer-Outcomes, Finance-Postings, Webhook-Retries.

### 11. Tenant-Modell

Cosmi ist aktuell **Single-Tenant-only**. Multi-Tenant-Support (Option-B-Full mit `tenant_id` auf ~50 Tabellen) ist fuer Sprint 2/3 geplant. Bis dahin:
- Kein SaaS mit mehreren Mandanten auf einer DB-Instanz
- Self-Hosted: Ein Deployment pro Kunde (Hetzner-Instanz pro Pilot ab M3, ~287 EUR/M bei 10 Piloten)
- Neue Tabellen MUESSEN `tenant_id` von Anfang an haben — auch wenn der Wert vor Migration noch konstant ist

Details: `project_multi_tenancy.md` im Memory-Index.

## Quick-Regeln (Top 3, ohne Detail)

- Thick services, thin handlers
- Structured logging (slog), kein fmt.Println
- Test-Coverage-Min 15% gesamt, 60% kritische Pfade

## Verwandte Notes
- [[stack]] — Strategy Decisions & Frontend-Bibliotheken
- [[i18n]] — Internationalisierung
- [[design]] — Frontend Design System
- [[datenbank]] — Schema & Migrations
- [[api]] — API-Endpoints & OpenAPI
- [[security]] — Auth, RBAC, Middleware
- [[deployment]] — Deploy Scripts, Rollback, Smoke Tests
- [[testing]] — Test-Strategie

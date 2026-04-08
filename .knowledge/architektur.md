---
tags: [architektur, backend, frontend, ci-cd]
updated: 2026-04-08
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

Gateway `/health`: status, checks, registered_services, version, commit, build_time (via ldflags)

### Gateway Route-Dateien (27)

| Datei | Domain |
|-------|--------|
| `route_auth.go` | Auth, 2FA, Passwoerter |
| `route_bexio.go` | Bexio OAuth + Sync |
| `route_biz.go` | Finance (Quotes, Invoices, Payments) |
| `route_biz_ext.go` | Finance Extended (HR-Einstellungen, Dunning) |
| `route_caldav.go` | CalDAV/CardDAV Proxy |
| `route_calendar.go` | Kalender, Events, Ressourcen |
| `route_chat.go` | Chat, Channels, Nachrichten |
| `route_crm.go` | CRM (Contacts, Companies, Deals) |
| `route_crm_ext.go` | CRM Extended (Consent, Duplicate Detection) |
| `route_dashboard.go` | Dashboard-Widgets |
| `route_datev_upload.go` | DATEV CSV-Upload |
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
| `route_registrar.go` | Service-Registrierung |
| `route_search_global.go` | Cross-Service Global Search (500ms Timeout) |
| `route_security.go` | Audit Logs, GDPR Export, Passwort-Policy |
| `route_video.go` | Video-Calls (LiveKit) |
| `route_wopi.go` | WOPI Document Protocol (OnlyOffice) |
| `route_work.go` | Projekte, Tasks, Zeiterfassung |

## Frontend (Electron + React 19 + TypeScript)

### Module (35 in `modules/`)
admin, auth, automatisierung, berichte, buchhaltung, calendar, chat, crm, dashboard, dokumente, einkauf, finanzen, formulare, fuhrpark, helpdesk, inventar, kalender, kommunikation, kontakte, mails, meetings, notifications, produktion, profil, rapporte, schichten, security, settings, team, vermietung, vertraege, video, wiki, work, zeiterfassung

### Zustand Stores (36 in `stores/`)
ai, auth, automatisierung, berichte, calendar, contacts, dashboard, einkauf, finance, formulare, fuhrpark, helpdesk, integrations, inventar, kommunikation, locale, mails, meetings, navigation, notifications, presence, produktion, profile, rapporte, schichten, search, settings, team, timetracking, tour, ui, vermietung, vertraege, video, wiki, work

### Standalone
- Guest Chat: Separate Vite SPA unter `/guest/`

## Performance-Patterns

### N+1 Query Elimination (2026-04-08)
- Contact-Liste: `enrichWithRelationsBatch()` — 4 Queries statt 61 (Batch-Loading per `ANY($1)`)
- Deal-Liste: `enrichWithRelationsBatch()` — 7 Queries statt 121
- Batch-Inserts: Tags per `unnest($2::uuid[])`, Custom Fields per `pgx.Batch`
- Einzelabfragen (`getWithRelations`) bleiben fuer GetByID/Create/Update (nur 1 Entity)

### Gateway Performance
- **Audit Logger:** Buffered Channel (1000) + Worker Pool (10 Workers) statt unbegrenzte Goroutines
- **gRPC Keep-Alive:** 60s/10s, PermitWithoutStream (in `registry.go`)
- **pprof:** `/debug/pprof` hinter `ENABLE_PPROF=true` Env-Var (nur Staging/Dev)
- **Connection Pool:** MaxConns=10 pro Service (10×10=100 = PG-Limit), MaxConnLifetime=1h, HealthCheckPeriod=1m

### React Compiler (2026-04-08)
- `babel-plugin-react-compiler` im `annotation` Mode
- `"use memo"` Directive auf DashboardPage, DealPipelineView, ContactsListPage
- Automatisches Memoization ohne manuelle useMemo/useCallback

## Demo Mode

- Aktiviert via `RENDERER_VITE_DEMO_MODE=true` (Build-Time Env)
- **Fetch-Interceptor** (nicht MSW Service Worker)
- Grund: Electron Production nutzt `file://` Protokoll — Service Workers + dynamic import() nicht unterstuetzt
- Handler: `mocks/handlers/` (18 Dateien, statisch importiert, Vite eliminiert in non-demo Builds)
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

## Regeln
- Thick services, thin handlers
- Structured logging (slog), kein fmt.Println
- 80%+ test coverage, 95%+ fuer kritische Pfade

## Verwandte Notes
- [[stack]] — Strategy Decisions & Frontend-Bibliotheken
- [[i18n]] — Internationalisierung
- [[design]] — Frontend Design System
- [[datenbank]] — Schema & Migrations
- [[api]] — API-Endpoints & OpenAPI
- [[security]] — Auth, RBAC, Middleware
- [[deployment]] — Deploy Scripts, Rollback, Smoke Tests
- [[testing]] — Test-Strategie

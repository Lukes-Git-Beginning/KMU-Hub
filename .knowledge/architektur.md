---
tags: [architektur, backend, frontend, ci-cd, rls]
updated: 2026-06-28
---
# Architektur

## Backend (Go Microservices)

Gateway (HTTP/chi) auf Port 8080 → gRPC-Services intern. Tenant-Isolation auf HTTP-Ebene seit Welle 2D (2026-04-28) durch JWT `tid`-Claim + `middleware.GetTenantID()` (fail-closed) enforced — kein Placeholder-Fallback mehr. Details: [[security]].

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
| plugin | :50060 | WASM Sandbox (flag off), SDK, Host API, Lifecycle |
| dialer | :50061 | Campaigns, Call Sessions, Agent Status, Outcomes |
| wiki | :50062 | Articles, Versions, Attachments, Categories, FTS (tsvector+GIN), Share-Links (Sprint 1 Welle 2) |
| berichte | :50063 | Report-Definitions, Schedules, Cache, Runs, KPI-Dashboard, PDF/CSV/XLSX-Export, In-Process-Cron-Scheduler (Sprint 1 Welle 5-6) |
| formulare | :50064 | Form-Schemas (JSONB), Submissions, Webhook-Worker (HMAC-SHA256, Exp-Backoff, Dead-Letter), CSV+XLSX Export (Sprint 1 S1.3) |
| helpdesk | :50065 | Tickets, Messages, Queues, Canned-Responses, SLA-Policies, Merge (Sprint 1 Welle 2) |
| inventar | :50070 | Items, Movements, Stock-Warnings, Oversell-Guard (Sprint 2 Welle 1) |
| einkauf | :50071 | Suppliers, Purchase-Orders, PO-Lines, Wareneingang-Stub (Sprint 2 Welle 1) |
| produktion | :50072 | Production-Orders, Machine-Bookings (advisory-lock), Plans (Sprint 2 Welle 1) |
| vertraege | :50073 | Contracts, Parties, Reminders (advisory-lock-claim, 5+60min Ticker) (Sprint 1 S1.5) |
| **rapporte** | :50074 | Work-Reports, Lines, Attachments, Approval-State-Machine, GPS-Tag (Sprint 2 Welle 2A) |
| **schichten** | :50075 | Shifts, Assignments, Templates, ArbZG §5 Pre-Check (11h Ruhezeit, DST-aware) (Sprint 2 Welle 2A) |
| **fuhrpark** | :50076 | Vehicles, Services, Damages, TÜV-Reminder-Cron (advisory-lock 7d/1d) (Sprint 2 Welle 2A) |
| **vermietung** | :50077 | Rental-Objects, Rentals, Inspections, GIST tstzrange-Overlap-Index (Sprint 2 Welle 2A) |

Gateway `/health`: status, checks, registered_services, version, commit, build_time (via ldflags)

### Gateway Route-Dateien (43)

Alle Handler sind duenne gRPC-Proxies. Keine direkte DB-Abfrage im Gateway (ausser Dashboard, gekapselt).
Shared Helpers: `validateUUIDParam`, `RequireAuthenticated` Middleware, `respondGRPCError`, `parsePagination`.
Alle Modul-Routes (wiki/helpdesk/berichte) sind gated via `featureflag.Registry` — Flag OFF = Route nicht gemountet (kein 404/405, sondern komplett unsichtbar).
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
| `route_booking.go` | Public Booking (unauthenticated, IP-Rate-Limit): `GET /api/v1/public/booking-pages/{slug}`, `GET /{slug}/availability`, `POST /bookings` — Muster `route_guest.go`, kein authMiddleware |
| `route_crm_advisory.go` | Beratungsprotokoll ZFA (2026-06-10): CRUD + hand-over + PDF + referral-report. Methode auf `CRMRoutes`, separate Registrierung via `crmRoutes.RegisterAdvisoryRoutes(r, …)` NACH dem Registrar-Loop in `main.go`. Service: `internal/crm/advisoryprotocol` |
| `route_settings.go` | Settings-Fundament (2026-06-10): Module-Leads CRUD + 3-Ebenen-Settings (`/settings/{module_id}` resolved, `/tenant`, `/user`). Service `internal/settings` **co-located im auth-Binary** (Muster security/auth auf :50051, kein 25. Microservice). Proto-Regen: `make proto-settings` |
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
| `route_wiki.go` | Wiki (Articles, Versions, Attachments, Categories, FTS) — hinter `modules.wiki` |
| `route_helpdesk.go` | Helpdesk (Tickets, SLA, Merge, Canned) — hinter `modules.helpdesk` |
| `route_berichte.go` | Berichte (Definitions, Schedules, KPIs, Export PDF/CSV/XLSX) — hinter `modules.berichte`, Permission `berichte:reports` |
| `route_formulare.go` | Formulare (Schemas, Submissions, Webhooks) — hinter `modules.formulare` |
| `route_inventar.go` | Inventar (Items, Movements, Warnings) — hinter `modules.inventar` |
| `route_einkauf.go` | Einkauf (Suppliers, POs, Lines) — hinter `modules.einkauf` |
| `route_produktion.go` | Produktion (Orders, Bookings, Plans) — hinter `modules.produktion` |
| `route_vertraege.go` | Vertraege (Contracts, Parties, Reminders) — hinter `modules.vertraege` |
| `route_rapporte.go` | Rapporte (Reports, Lines, Attachments, Approval) — hinter `modules.rapporte` |
| `route_schichten.go` | Schichten (Shifts, Assignments, Templates, ArbZG-Compliance) — hinter `modules.schichten` |
| `route_fuhrpark.go` | Fuhrpark (Vehicles, Services, Damages, TÜV-Reminder) — hinter `modules.fuhrpark` |
| `route_vermietung.go` | Vermietung (Objects, Rentals, Inspections, Calendar) — hinter `modules.vermietung` |

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

### Zustand Stores (59 in `stores/`, Stand 2026-06-11)
Kern-Stores pro Modul (ai, auth, dashboard, dialer, finance, vertraege, work, …) plus etabliertes Begleit-Muster: `<modul>Prefs.ts` (UI-Präferenzen) und `<modul>Settings.ts` (Modul-Einstellungen, von den Settings-Panels der FE-Lane konsumiert) — z.B. dashboardPrefs/dashboardSettings, vertraegePrefs/vertraegeSettings, workPrefs/workSettings. `dashboard.ts` ist persist-versioniert (v2, siehe FE-Lane-Abschnitt).

### API-Layer-Pattern (2026-04-29, Sprint 2 Welle 4A)
- **Zentraler Helper:** `desktop/src/renderer/src/api/utils/authenticatedFetch.ts` — kapselt Auth-Header (`Authorization: Bearer …`), Idempotency-Key-Generierung (UUIDv4 fuer Mutations), Offline-Queue-Hooks (enqueue bei `!navigator.onLine`), Error-Mapping (Backend-Error-Shape → typed Errors).
- **32 API-Clients konsumieren ihn:** automation, berichte, bexio, caldav, calendar, crm-import, datev-upload, dialer, einkauf, email, finance, formulare, fuhrpark, helpdesk, hr, inbox, integration, inventar, lexware, notification, plugin, produktion, rapporte, schichten, security, vermietung, vertraege, video, wiki, plus 3 weitere. Eliminiert Duplikat-Code in jedem Client.
- **Test-Coverage:** `api/__tests__/idempotency-coverage.test.ts` mit 29 Cases (eine pro API-Client-Familie) verifiziert Idempotency-Key-Header-Setzung — Voraussetzung fuer Idempotency-HardMode-Switch in Welle 4B.

### FE-Lane „Luke-Block" — profil/vertraege/dashboard P1–P4 (2026-06-10/11, Branch `marathon/luke-fe`, 14 Commits bis `ee743c0d`)
Komplett-Ausbau der drei Module, mock-first wo kein Backend existiert. **Noch NICHT auf main gemerged** — wartet auf Darien-Feinschliff-Review via `.planning/reviews/{vertraege,dashboard,profil,buchhaltung}.md`.
- **vertraege:** Settings-Panel, Audit-Log+Reminder im DetailPanel, Dokumente↔Dokumente-Modul (Picker/Upload/Preview/Versionen), Canvas-E-Signatur EES (Hybrid-Dispatch), CRM/Finanzen-Verknüpfung (1 Kontakt / 1 Deal / n Rechnungen als Chips mit Navigation `?contact=`/`?invoice=`, 6 History-Codes), KI-Fristencheck als FE-lokales Heuristik-Panel. Store `stores/vertraege.ts` bleibt mock-first; API-Swap → `entity_links`/`useLinkFile` (separate Phase).
- **profil:** Presence-Picker, Avatar-Upload-UI, Notification-Quick-Card, `components/user/UserProfileCard.tsx` (Radix-Popover-Overlay an 5 Call-Sites) mit Ping→Chat via `useGetOrCreateDM()`; `NavigationIntent` um `userId`/`channelId` erweitert, ChatLayout konsumiert reaktiv.
- **dashboard:** Settings-Panel, Widget-Gating per `modules.<id>`-Flags (fail-open NUR dashboard-lokal — `useFeatureFlags`/`FeatureGate` app-weit fail-closed unangetastet), `hooks/useAlerts.ts` (aggregiert vertraege-expiring/invoices-overdue/helpdesk-SLA), Widget `cross-module-overview`, Scope-Umschalter Persönlich/Team mit zustand-persist **v1→v2-Migration** (flat `layouts`/`activeWidgets` → `personal*`/`team*`, verlustfrei, Team-Layout mock-first localStorage, debounced PUT nur personal), Widgets `team-worktime` (CSS-Bars) + `open-tickets`.
- **Fremd-Touches:** FinanzenPage konsumiert `?invoice=<id>` reaktiv (wartet auf Laden, ignoriert unbekannte IDs); finance-MSW-Handler normalisiert `items`→`line_items`/`issue_date`→`invoice_date`.
- QA-Belege (Playwright-Scripts `scripts/qa-phase*.mjs` + Screenshots + qa-result.json) sind pro Phase mitcommitted.

### Backend R3 Welle F/G + Notifications/Wiki (2026-06-21, alle auf main, prod-deployt)
- **R3 Welle F (Data-Integrity + P0-1-Rest):** alle 25 Proto-Handler in `route_video.go` auf `response.Proto` (protojson, RFC3339-Timestamps statt `{seconds,nanos}` — s. [[api]]); Helpdesk `MergeTickets` jetzt atomar via `Repository.MergeTicketTx` (eine pgx-TX `pool.Begin`→ReassignMessages+UpdateTicket→Commit, statt zwei getrennter Writes); `audit_log` DB-Level Append-Only-Trigger (Migr.222, GoBD — s. [[security]]); 29 aufgeschobene `tenant_id`-FKs validiert (Migr.223 — s. [[datenbank]]).
- **R3 Welle G:** RBAC manager/member-Permission-Seed fuer 5 Module booking-pages/schichten/hr:time_*/inventar/einkauf (Migr.224, manager voll operativ / member Self-Service — s. [[security]]).
- **Meeting-Parität (2026-06-23/28):** `route_video.go` erweitert um Host-Controls (cohosts/lock/mute, Migr.226), CRM-Link (227), KI-Summary (228, `LLM_BASE_URL`-gated) und **6A Breakout-Rooms** (2026-06-28, deployt). Breakout: 7 RPCs am VideoService (`CreateBreakoutRooms`/`ListBreakoutRooms`/`AssignBreakoutParticipant`/`JoinBreakoutRoom`/`GetBreakoutAssignment`/`ReturnToMainRoom`/`CloseBreakoutRooms`), Routen `/api/v1/meetings/{id}/breakout-rooms[...]`, eigene LiveKit-Sub-Rooms `breakout-{meetingID}-{i}` (LiveKit auto-create → kein expliziter CreateRoom), Authz rein service-layer `isHostOrCoHost` (kein neuer RBAC-Guard; join nutzt geseedetes `meetings:write`). FE-Room-Switch: 8s-Poll `GetBreakoutAssignment` ist autoritativ (DataChannel ist room-scoped → erreicht Sub-Rooms nicht → nur Accelerator für main→breakout); Store-`switchBreakoutRoom` tauscht den LiveKit-Token → `<LiveKitRoom>` reconnectet. Tabellen `meeting_breakout_rooms`+`meeting_breakout_assignments` (Migr.235, tenant_id+RLS, s. [[datenbank]]). Mitgenommen: Display-Name-Fix (`GetAttendees` JOIN users) + Screenshare-Diagnose-Logging.
- **Notifications (Darien, N-1…N-5):** DND/Quiet-Hours-Gating vor Live-Toasts (`modules/notifications/notification-gating.ts`), Toast+Widget auf einheitliche MSW-Pipeline mit Live-Arrivals, Sidebar-Nav-Eintrag + gruppierte/gemutete Demo-Seeds, Pin/Dismiss-Persistenz via MSW + Deep-Link zu konkreten Items, lesbare Modul-Labels in den Event-Type-Preferences.
- **Wiki (Darien, PB-1):** Artikel-Authoring auf die **shared block-document engine** umgezogen — `WikiRichEditor.tsx` (≈490 Zeilen) entfernt, `wiki-blocks.tsx` + erweiterter `wiki-adapter.ts` neu, `WikiArticle`/`WikiEditor` schlanker.

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
  - 17 Flags registriert: 14 Modul-Flags (`modules.<name>`, Default off) + `plugins.wasm` (off) + `plugins.config` (on) + `plugins.api` (off — Plugin-HTTP-API-Routen, security-risk, bis Phase D)
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

## Sprint 2 Welle 2D — JWT-Tenant-Hardening (2026-04-28)

Welle-1-Altlast (`<modul>PlaceholderTenantID = "00000000-...-000000000001"` in 11 Gateway-Routes) komplett entfernt. Drei Commits:

1. `33450e7` (W2D-A): `auth.Claims.TenantID string \`json:"tid"\``, `CreateAccessToken(userID, tenantID, ...)`, Migration 000104 `users.tenant_id`, Middleware-Helper `GetTenantID(ctx) (uuid.UUID, error)` mit `ErrMissingTenantID` (fail-closed). 11 Routes refactored: rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion/berichte/formulare/wiki/vertraege.
2. `c421fac` (W2D-B): `auth/postgres_repository.go` SELECTed `tenant_id` jetzt — vorher leer → `tid`-Claim immer empty trotz Issuance.
3. `8f055e3` (W2D-C): 5 Cross-Layer-Holes geschlossen — `dialer_grpc.go`/`helpdesk_grpc.go` (13 Proto-Requests um `tenant_id` Field erweitert + pb.go regeneriert), `route_wiki.go` (4 Handler verwarfen tenantID), `route_biz.go::getTenantID(r)` (rief `GetUserID` statt `GetTenantID` → UserID-als-TenantID-Surrogate in 90 Call-Sites quer durch biz/billing/invoices/quotes/ext/hr/lexware/bexio/datev).

**Pflicht-Pattern fuer neue Routes:** `tenantID, err := middleware.GetTenantID(r.Context())` als erste Aktion → `respondError(w, http.StatusUnauthorized, ...)` bei Fehler. Kein Default-Tenant. Details: [[security]] "JWT Tenant-Claim & Cross-Layer-Hardening".

## Sprint 2 Welle 2A — Modul-Patterns (2026-04-28)

Vier Handwerk-Module (`rapporte`, `schichten`, `fuhrpark`, `vermietung`) sind das frischeste Beispiel des Standard-Modul-Templates aus `MODULES_SCOPE_MATRIX.md`. Inventar (Welle 1) ist der Anker, Welle 2A wiederholt das Pattern × 4 mit drei Pflicht-Guards aus dem `ad04191`-Bugfix-Sweep:

1. **Pre-Check vor State-Transitions / Mengenwrites** — Service-Layer, nicht Repository. Beispiele: `rapporte.ApproveReport` prueft `status == submitted`, `schichten.AssignEmployee` ruft `validateRestPeriod`, `vermietung.CreateRental` macht GIST-Overlap-Query, `fuhrpark.CompleteService` prueft `status == scheduled`.
2. **`tenant_id` in jedem Get-by-ID** — `WHERE id = $1 AND tenant_id = $2` in allen `postgres_repository.go` SELECTs.
3. **`RowsAffected() == 0` Sentinel** — jede `pool.Exec()` fuer UPDATE/DELETE returniert `Err<Modul>NotFound`.

Modul-spezifische Patterns:

- **TÜV-Reminder-Cron (`fuhrpark`):** `cmd/fuhrpark/worker.go` mit `pg_try_advisory_xact_lock` Leader-Election, 7d/1d-Vorlauf-Fenster, idempotent via `vehicles.tuev_reminder_sent_at` (skip bei Stamp <23h alt). Notification-Delivery noch Stub — Sprint-3-Wiring an `notification`-gRPC noetig (siehe `docs/sprint2-welle2-issues.md`).
- **ArbZG §5 Pre-Check (`schichten`):** `validateRestPeriod(ctx, employeeID, newStart, newEnd)` in `service.go`. Bei <11h Ruhezeit zwischen letzter und neuer Schicht: `ErrArbzgViolation`. DST-Spring-Forward-Test mit `time.LoadLocation("Europe/Berlin")`.
- **GIST tstzrange-Overlap (`vermietung`):** `idx_rentals_object_dates ON rentals USING GIST (object_id, tstzrange(start_date, end_date))`. `CheckAvailability`/`CreateRental` nutzen `&&`-Operator. Doppelbuchung-Schutz auf DB-Ebene + Service-Pre-Check.
- **Approval-State-Machine (`rapporte`):** `Draft → Submitted → Approved/Rejected`, nach Approved blockt jeder Edit/Rollback (`ErrAlreadyApproved`).

Subagent-Strategie: 4 parallele Sonnet-Subagents schreiben direkt auf main, Done-Reports max 200 Worte. Race-Risiken auf shared Files (`config.go`, `cmd/gateway/main.go`, `docker-compose.yml`) wurden durch Edit-Tool-Konfliktdetection aufgeloest. Self-Commit-Anomalie: ein Subagent (fuhrpark) hat eigenmaechtig commited — fuer kuenftige Briefings explizit `kein git add/commit` ergaenzen. Details: `memory/project_sprint2_welle2.md`.

## Sprint 4 Welle 1 — RLS-Foundation + Wiring-Gap-Sweep + Pilot-0-Interceptor + Error-Mapper-Audit (2026-05-10)

7 Direct-to-Main-Commits, live in Production auf `25af970` mit Migration-Head 119. Welle 1 legt die Foundation für Row-Level-Security (Aktivierung kommt in Welle 2), schliesst 13 stille NOT-NULL-Wiring-Gaps aus Welle 4B und macht den `mapChatError`-Antipattern (Welle 0.6) systemweit unmöglich.

| Commit | Inhalt |
|--------|--------|
| `e4568f0 feat(welle1a)` | Migration 118 RLS-Foundation: `current_tenant_id/user_id/role`, `is_system_context`, `enable_tenant_rls()/_via_join()` Procedures, Database-GUC-Defaults. Pool-Layer: `database.WithSystemContext` + `database.BeginRLSTx` (set_config LOCAL). `postgres.go` AfterRelease-Reset. 10 Worker-Sites (`berichte/scheduler`, `automation/trigger.poller`, `fuhrpark/worker`, `email/sync.worker+engine`, `vertraege`, `formulare`, `biz/lexware+bexio`, `inbox/StartSnoozeWorker`) wrappen Entry-Context mit `WithSystemContext`. |
| `0bc867e feat(welle1b-stream-c)` | Wiring für 7 Tabellen (Stream C, vom Sub-Agent eigenmächtig commited): `automation_executions`, `task_entity_links`, `task_custom_field_values`, `user_project_preferences`, `document_file_versions`, `document_shares`, `document_tags`. Service-Sigs erweitert: `task.SetCustomFieldValues(tenantID)`, `tag.CreateTag(tenantID)`. |
| `f501d97 feat(welle1b)` | Stream A+B + Migration 119. Wiring für 8 weitere Tabellen: `UserSession`, `RecoveryCode`, `AppSpecificPassword`, `PushSubscription`, `ChatFile`, `Mention`, `GuestSession`/`GuestChannelConfig`, `CallSession`/`CallParticipant`. Migration 119 backfilled `dialer_campaign_contacts/agent_status_log/call_events + recording_consents` (siehe [[datenbank]]) + promotet `consent_records.tenant_id` auf NOT NULL. Service-Sigs: `auth.session.CreateSession(tenantID)`, `auth.totp.generateRecoveryCodes(tenantID)`, `caldav.AppPasswordService.Create(tenantID)`. INSERT-SELECT-Trick für `message_reactions` (Tenant aus Parent-Message via Subquery, kein Interface-Bruch). |
| `9e2a9fa feat(welle1c)` | 28 grpc-Default-Branches in 23 `*_grpc.go`-Files mit `slog.Error("unhandled <service> service error", "error", err)` versehen. Pattern aus `chat_grpc.go` `mapChatError` (3d6ff8d). 17 BORDERLINE-Branches verlieren zusätzlich den `err.Error()`-Leak im gRPC-Status. 20 Files mit `log/slog`-Import erweitert. |
| `ba93d9d feat(welle1d)` | `middleware.TenantInboundUnaryInterceptor` in 4 Pilot-0-Services (`cmd/auth`, `cmd/crm`, `cmd/dialer`, `cmd/work`) wired. Gateway-Outbound-Interceptor war seit Welle 0.6 schon GLOBAL aktiv (`internal/gateway/registry.go:94`). Auth-Service-Edge-Case dokumentiert: Login/Register/RefreshToken/AcceptInvitation/Validate2FALogin sind unauthenticated, brauchen Whitelist falls Welle 5 den Interceptor hardenet. Neue `internal/middleware/grpc_tenant_test.go` mit 4 Unit-Tests. |
| `043dd53 fix(welle1b)` | Hotfix nach Production-Deploy-Crash: `dialer_call_events.dialer_call_session_id` (echter Spaltenname, nicht `session_id`). Migration 119 Backfill-JOIN korrigiert. |
| `25af970 feat(deploy)` | `--skip-smoke`-Flag in deploy.sh — Notbremse für False-Positive-Smoke-Cascades wenn DB schon forward gewandert ist und Auto-Rollback Drift erzeugen würde (siehe [[deployment]] + [[troubleshooting]]). |

**Welle-1-Pattern für künftige RLS-Wellen:** Migration mit `enable_tenant_rls('<table>')` → Repository-INSERT-Wiring (Spalte + Bind) → Service-Konstruktion mit `middleware.GetTenantID(ctx)` ODER `parent.TenantID` → Cross-Tenant-Test in `tenant_isolation_test.go` (Pattern aus Welle 4B `b868fb6`). System-Operationen (Worker, Migrations, Audit-Inserts) wrappen Entry-Context mit `database.WithSystemContext(ctx)`.

## E2E-Modernisierung — gRPC-Tenant/Actor-Trust komplettiert (2026-06-05, `a02f3632`)

Die E2E-Suite deckte auf, dass das Tenant-Trust-Pattern an mehreren Stellen unvollstaendig war. Kanonischer Stand seither:

- **Inbound-Interceptor ist Pflicht fuer JEDEN gRPC-Service.** `cmd/document` war der letzte ohne `middleware.TenantInboundUnaryInterceptor()` — jeder tenant-scoped Document-RPC failte mit "missing tenant context". Checkliste fuer neue Services: Interceptor in `grpc.ChainUnaryInterceptor` registrieren.
- **Service-zu-Service-Verbindungen brauchen den OUTBOUND-Interceptor ebenfalls.** Gateway→Service ist global abgedeckt (`registry.go`), aber direkte Bruecken wie Dialer→CRM (`cmd/dialer/main.go` crmConn) muessen `grpc.WithUnaryInterceptor(middleware.TenantOutboundUnaryInterceptor())` als DialOption setzen — sonst ist der Folge-Call Unauthenticated. Der Outbound liest Tenant+User aus dem ctx, den der Inbound befuellt hat (Re-Propagation).
- **Tenant NIE aus Request-Proto-Feldern lesen** (spoofbar + Gateway setzt sie nicht mehr): 4 Dialer-RPC-Leftovers (ListCampaigns/SetAgentStatus/ListCallOutcomes/CreateCallOutcome) auf `middleware.GetTenantID(ctx)` umgestellt. Die 7 `tenant_id`-Felder in dialer.proto sind jetzt tote Felder (Cleanup = Followup F3).
- **Actor-IDs (created_by/deleted_by) aus `middleware.GetUserID(ctx)`** (x-user-id-Metadata), nicht uuid.Nil und nicht Proto-Felder: `work_grpc.DeleteTask` (uuid.Nil failte den Membership-Check fuer jeden Caller) + `dialer_grpc.CreateCampaign` (uuid.Nil verletzte den users-FK).
- **Read-Seite-Falle (2× gefunden):** Repo-`GetByID` filtert per tenant_id, scannt die Spalte aber nicht → `model.TenantID = uuid.Nil` → tenant-gefilterte UPDATEs matchen 0 Rows → **Phantom-404**. Details + Sweep-Followup F2: [[troubleshooting]].

**Cross-Stream-Lessons:**
- Sub-Agents committen gelegentlich eigenmächtig trotz expliziter "do not commit"-Anweisung (Stream C in dieser Welle, fuhrpark in Welle 2A). Bei parallelen Wellen einkalkulieren.
- 5 parallele Streams produzierten 4 Cross-Stream-Compile-Konflikte (Service-Signaturen erweitert, Tests/Adapter nicht mitgezogen). Hauptsession patcht im Konsolidierungs-Pass.
- Production-Deploy hatte 3 Anläufe: Migration-119-Spaltenname-Bug → Hotfix → Smoke-False-Positive (SMOKE_ADMIN_TOKEN expired) → Auto-Rollback zog Code zurück aber NICHT Migrations → Drift → `--skip-smoke`-Flag als finaler Forward.

## Sprint-4-Vorzug — R2-P1-Batch + LiveKit/COSMI_ENV-Cluster (2026-06-05, 2 Sessions)

**Session 1 (R2-P1-Batch, `f5788d8d`/`98337921`/`5dd862eb`/`19d5adb7`):**
- **Neues Package `internal/circuitbreaker`** — 3-State (closed/open/half-open), injectable clock. Konsumenten: Bexio + DATEV (siehe [[integrationen]]). Bei neuen Drittanbieter-HTTP-Clients wiederverwenden.
- **Redis-backed WS-Subscriptions** (`ws:subscriptions:{channel}`, 24h TTL, graceful Fallback) — Gateway-Restart verliert keine Subscription-Zuordnung mehr (R2-P1.7).
- **Automation-Semaphor per-Tenant** (5/Tenant in global 20, R2-P1.3); **CleanupExpiredRecordings** 24h-Cron via `WithSystemContext` (R2-P1.11).
- WS-Token-Revalidierung, Organizer-only-StartMeeting, Join-with-Consent: [[security]] "Realtime-Haertung".

**Session 2 (LiveKit/COSMI_ENV, `68158907`…`ce2a5e5d`):**
- **`config.Load(ctx, ...Requirement)`** — variadic Requirements-API (`RequireVault`/`RequireMinIO`/`RequireWOPI`); Prod-Assertion prueft nur konsumierte Gruppen, Compose-Dev-Werte bleiben universal verboten. Details [[security]]. Neue Services mit Vault/MinIO/WOPI-Konsum MUESSEN das Requirement deklarieren.
- **LiveKit-URL-Split** intern (`LIVEKIT_INTERNAL_URL`, twirp) vs. public (`LIVEKIT_WS_URL`, Clients) + `cfg.LiveKitServerAPIURL()`-Fallback. `NewVideoGRPCServer(..., tokenGen video.RoomManager, publicWSURL string)` — Join/Start-Responses tragen echte `token`/`ws_url` (vorher IMMER leer). Details [[integrationen]].
- Compose-Secret-Interpolation `${VAR:-dev-default}` als Hardrule: [[deployment]].

## Sprint 2 Welle 4B — Option-B Phase 2 + Idempotency HardMode-Bereitschaft (2026-05-07)

Drei Sub-Wellen (4B.1 + 4B.2 + 4B.3), zwei Direct-to-Main-Commits. Schliesst die Tenant-Isolation-Front auf den verbleibenden Hot-Path-Tabellen und macht den Idempotency-Stack HardMode-ready (Code + Dev-Default Hard, Prod-Cutover als separate Sprint-3-Aktion).

| Commit | Inhalt |
|--------|--------|
| `b868fb6 feat(welle4b): option-b phase 2 + idempotency hardmode readiness` | 105 Files, +3687/-1358. 5 neue Migrations (000109-000113, siehe [[datenbank]]). 16+ Repository-Wirings (work/calendar+meeting+resource, email/*, notification/*, inbox/*, crm/tag+consent+search). 13 gRPC-Handler auf `middleware.GetTenantID(ctx)`. Idempotency `Complete()`-Composite-PK-Fix + HardMode-Env-Flag `IDEMPOTENCY_MODE` in `cmd/gateway/main.go` (Default WarnMode). 12 Cross-Tenant-Tests in `tenant_isolation_test.go` + 3 finance JSONB-Tests. P2-Followups integral (P2-1, -2, -3, -5, -6, -7, -8, P3-3). |
| `1b1eb37 fix(welle4b): close 5 P0+P1 findings from welle 4B.3 sweep` | 9 Files, +195/-97. **F1 P0:** `video_grpc.StartRecording` extrahiert tenantID nun vor if/else und reicht sie an `recordingService.StartRecording` durch (vorher uuid.Nil als tenant_id auf jeder neuen recording-Row). **F2 P0:** 12 Deal/Activity-Handler in `crm_grpc.go` von `req.TenantId`-Spoof auf `middleware.GetTenantID(ctx)` (Welle-3.5-P0-Carryover). **F3+F10 P1:** `meeting_action_items` INSERT/UPDATE/DELETE/Get mit tenant_id-Filter + neue `GetActionItemByID`-Methode. **F4 P1:** `ConvertActionItemsToTasks` uuid.Nil-Guard via `GetMeetingIDForActionItem`-Lookup. **F5 P1:** `.env.example IDEMPOTENCY_MODE=hard` auskommentiert (docker-compose setzt es bereits hart fuer Dev). |

**Idempotency HardMode-Rollout:** Code-bereit, Dev-Default Hard. Prod-Cutover bleibt Sprint-3-Aktion nach Pilot-1 (Risiko: API-Caller die heute noch ohne Idempotency-Key senden wuerden 400en). 4 neue Tests (`Complete_TenantFilter`, `Get_TenantIsolation`, `HardMode_MissingKey_Returns400`, `HardMode_CrossTenantKeyRejected`).

**Pflicht-Pattern fuer neue Repos:** tenant_id-Filter auf jedem SELECT/UPDATE/DELETE; tenant_id-Spalte+Value in jedem INSERT; Repository-Sigs `(ctx, tenantID uuid.UUID, ...)` als zweiten Param; gRPC-Handler liest `middleware.GetTenantID(ctx)` als erste Aktion. 4 Plan-Notes (Out-of-Scope mit Begruendung): `work/caldav` (Auth-Layer-Problem, Sprint 3), `auth.GetUserByEmail` (by-design global), `gateway/dashboard_repository` (Architektur-Klaerung), `finance_line_items` (JSONB in Parent-Tables, kein Retrofit noetig).

**Cross-Stream-Drift-Lesson:** Drei Mal trafen stale IDE-Diagnostics — Subagent-Output sagte "alles gruen", IDE-Diagnostics zeigten Sig-Drift in `cmd/*/main.go` und gRPC-Handlern, `go build ./...` direkt war clean. **Authoritative Verifikation ist `go build ./...`, nicht IDE-Diagnostics.** Details: [[troubleshooting]].

Pause-Gates zwischen 4B.1 ↔ 4B.2 ↔ 4B.3 wie in `feedback_pause_between_waves.md` gefordert. 4 P2/P3-Followups (echte DB-Backed Cross-Tenant-Tests, idempotency-time.Sleep-flaky, ListAllActive-Caller-Audit, ListBrowsable-shared-Filter-Frage) in `docs/sprint2-welle4b-followups.md` deferred.

## Sprint 3 Session 2026-05-08 — Option-B Phase 2 Abschluss + Dialer-Tx + CI-Security

9 Direct-to-Main-Commits in 4 Wellen. Option-B Phase 2 komplett (Migrations 000114+000115, ~38 neue Tabellen). Alle Sprint-2-Server-Drift-Fixes in main geschlossen.

**Dialer-Transaktion-Pattern** (`backend/internal/dialer/postgres_repository.go`):
`UpdateSessionWithEvent` ist jetzt eine **atomare Tx-Methode**: öffnet Transaktion, updated `dialer_call_sessions` und inserted `dialer_call_events` im selben Commit-Block. Verhindert Split-Brain bei Concurrent-Calls auf denselben Session-State. Neuer Concurrent-Test `TestLogCallOutcome_Concurrent_SameSession` mit 5 Goroutinen + Mutex auf `mockCallRepo`. Pattern fuer kuenftige Event-Sourcing-Repos: immer Session-Update + Event-Append in einer Transaktion kapseln.

**Repository-Layer Neue Wirings (Sprint 3):**
- `gateway/cached_dashboard_repository.go` — `tenantID` im Cache-Key (Dashboard-Isolation)
- `document/folder` postgres_repository — `tenant_id`-First-Filter auf allen SELECTs
- `biz/bexio`, `biz/lexware` — `ListEntityMappings`/`ListSyncLogs` mit JOIN-Tenant-Fence

**Welle-4B-Followups erledigt (Sprint 3 Welle 1A):**
- F6: Echte DB-Backed Cross-Tenant-Tests fuer Calendar/Email/Recordings (`test(welle4b): close f6-f9 followups`)
- F7: `assert.Eventually` statt `time.Sleep(20ms)` in `TestHardMode_CrossTenantKeyRejected` und `TestIdempotency_Replay_ReturnsCached`
- F8: `ListAllActive` Audit-Kommentar bestaetigt (kein Intent-Change)
- F9: `ListBrowsable` shared-Filter-Semantik verifiziert und bestaetigt

## Sprint 2 Welle 3.5 — Hardening-Sweep (2026-04-29)

Bugfix-Sweep nach Welle 3 (Commit `d443ab4`, 34 Findings closed: 17 P0 + 17 P1, P2/P3 in `docs/sprint2-welle3-followups.md` deferred). Drei Themen-Cluster:

1. **gRPC Tenant-Spoof-Sweep** — `chat_grpc.go`/`crm_grpc.go`/`work_grpc.go`/`video_grpc.go`/`dialer_grpc.go` ziehen `tenant_id` jetzt aus `middleware.GetTenantID(ctx)` statt `req.GetTenantId()` (Proto-Feld). Verhindert Tenant-Spoof bei Service-zu-Service-Calls oder kompromittiertem Gateway. Pflicht-Pattern fuer alle neuen gRPC-Server-Methoden. Details: [[security]] "gRPC Tenant-Spoof-Sweep".
2. **Repository-Tenant-Filter-Sweep** — `deal/activity/task/pipelinestage/chat-message/recording postgres_repository.go` enforcen `WHERE id=$1 AND tenant_id=$2` auf jedem UPDATE/DELETE/GetByID/Search-Pfad plus `RowsAffected==0`-Sentinel. Repository-Interfaces an PostgresRepository-Signaturen synchronisiert (`CachedRepository` reicht tenant_id durch). 14 Model-Structs (deal/activity/task/etc.) tragen einheitliche `TenantID`-Tags. `pipelinestage.scanStage` liest `tenant_id` aus Migration 000106 (vorher Scan-Mismatch nach migrate-up).
3. **Idempotency + Recording-Robustness** — Middleware-Stack-Position fix nach Auth+RBAC. `errors.Is` statt String-Equality. Atomares `Reserve` via `INSERT ... ON CONFLICT DO UPDATE RETURNING`. `context.WithoutCancel` fuer async Complete. Cleanup-Worker echte `pg_try_advisory_xact_lock`-Leader-Election. Migration 000108 setzt PK auf `(tenant_id, key)`. `Recording.StartRecording` reordert Pre-Consent-Check VOR `CreateRecording` (verhindert Orphan-Rows). Frontend: `CallControls` hat Doppelklick-Guard via `isPending`-Flags + try/catch-Toast auf `confirmInitiatorConsent`-Failure; `offline-queue` retried bei 409 statt silent-drop. Details: [[security]] Idempotency + Pre-Recording-Consent.

Pause-Gate: Nach Welle 3.5 → User-Review + Welle-4-Plan (Idempotency HardMode, restliche 15 Top-20-Repos, Top-30+ Tabellen).

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
- Dev: `npm run dev` → `--mode demo`

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

### Workflows (6 Dateien in `.github/workflows/`)

| Workflow | Trigger | Zweck |
|----------|---------|-------|
| `ci.yml` | Push/PR auf main/develop (backend/**) | Lint → Test → Build → E2E → Smoke → OpenAPI Validate + Security-Scans |
| `ci-desktop.yml` | Push/PR auf main/develop (desktop/**) | Lint → Typecheck → Test → Build |
| `cd.yml` | `workflow_run` (nach ci.yml success) + `workflow_dispatch` | SSH → deploy.sh → Health Check → Slack-Notify; concurrency-Group verhindert parallele Deploys |
| `claude-pr.yml` | PR open/sync, @claude Mention | Automatisches Claude Code PR-Review |
| `security-review.yml` | PR | Security-fokussiertes Code-Review |
| `security-scans.yml` | Jeder Push (S3.2, ab 2026-05-08) | gosec + trivy fs-scan + npm audit parallel (3 Jobs) — Details: [[security]] |

### CI Details
- Go 1.25.6, golangci-lint v2.8 (action v7)
- Postgres + Redis Service Containers fuer Tests
- E2E: `backend/test/e2e/` (Build Tag `e2e`)
- Smoke: Dual — Bash (`smoke.sh`, 22 Tests) + Go (`test/smoke/`, 11 Tests)
- Image-Tags in `docker-compose.yml` + `docker-compose.prod.yml` seit S3.4 gepinnt (kein `latest`)

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

Cosmi ist aktuell **Single-Tenant-only**. Option-B-Full-Retrofit ist **abgeschlossen** (Sprint 2+3, Migrations 000106–000115, ~123 Tabellen retrofitted). Neue Tabellen MUESSEN `tenant_id` von Anfang an haben.
- Self-Hosted: Ein Deployment pro Kunde (Hetzner-Instanz pro Pilot ab M3, ~287 EUR/M bei 10 Piloten)
- Sentinel-UUID fuer Single-Tenant: `00000000-0000-0000-0000-000000000001`
- Dashboard-Cache-Key enthaelt `tenantID` seit Sprint 3 (`gateway/cached_dashboard_repository.go`)

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

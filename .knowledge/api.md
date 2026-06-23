---
tags: [api, endpoints, openapi]
updated: 2026-06-21
---
# API-Referenz

## OpenAPI Spec
- Datei: `backend/api/openapi.yaml` (14.000+ Zeilen, OpenAPI 3.0.3)
- TypeScript-Generierung: `npm run api:generate` → `desktop/src/renderer/src/api/types.ts` (232KB)
- Gateway: HTTP REST auf Port 8080, leitet intern an gRPC-Services weiter
- **Fehler-Shape (vereinheitlicht 2026-06-19, F20):** Gateway-Handler antworten mit JSON `{error}` (via `response.Error`) bzw. `{error, code, details}` fuer Validierungsfehler (via `decodeAndValidate`/`internal/validation`). Kein `text/plain http.Error` mehr — ~420 Stellen ueber 27 `route_*.go` umgestellt.

## Endpoint-Gruppen

| Domain | Praefix | Notizen |
|--------|---------|---------|
| Auth | `/api/v1/auth/…` | Login, Register, 2FA, Refresh, /auth/me, **forgot-password** (rate-limited/IP+email, kein User-Enumeration-Leak), **reset-password** (single-use SHA-256-Token, Refresh-Token-Revocation) |
| CRM | `/api/v1/contacts`, `/companies`, `/deals` | Pipeline-Stages, Activities, Custom-Fields, Tags |
| CRM Extended | `/api/v1/contacts/…/consent`, `/…/duplicates` | Consent-Management, Duplicate Detection (pg_trgm) |
| **Advisory Protocols** | `/api/v1/contacts/{id}/advisory-protocols`, `/api/v1/advisory-protocols/{id}` (+`/hand-over`, `/pdf`), `/api/v1/contacts/referral-report` | Beratungsprotokoll ZFA (2026-06-10, Migr. 000137): CRUD nur im Draft (410 nach Finalisierung), `POST /hand-over` setzt immutable (idempotent), PDF-Endpoint aktuell 501 (FE nutzt `window.print()`). RBAC `advisory-protocols:{read,write,delete}` — siehe [[datenbank]] |
| **Settings/Module-Leads** | `/api/v1/tenant/module-leads…` (+`/me`), `/api/v1/settings/{module_id}` (+`/tenant`, `/user`) | 3-Ebenen-Scope (2026-06-10, Migr. 000138): `GET /settings/{module_id}` = serverseitig resolved (user > tenant), tenant-Writes nur Module-Lead/Admin (service-enforced), PUT mit Patch-Semantik `{"settings": {key: value}}`. RBAC `module-leads:{read,write}`, `settings:{read,write}` — siehe [[security]] |
| Chat | `/api/v1/chat/channels`, `/chat/messages` | Presence, Files. **Reactions echt seit 2026-06-11 (`c9c19380`):** `POST/GET /api/v1/messages/{id}/reactions` (Toggle/List) + `POST /api/v1/messages/reactions/summary` (Batch) — work/reaction-Service in ChatGRPCServer verdrahtet; FE nutzt `useToggleReaction` + `useReactionSummary`-Batch (`507487b9`) |
| Notifications | `/api/v1/notifications` | List, Read, Preferences, Mutes |
| Dashboard | `/api/v1/dashboard` | Layout-Management pro User/Rolle |
| Work | `/api/v1/projects`, `/tasks`, `/time-entries` | Task-Comments, Files, Time Tracking. **Seit 2026-06-11:** Label-CRUD `/api/v1/work/labels` + `PUT /tasks/{id}/labels` (Migr. 145/147, RBAC `work_labels:*`), `label_ids` batch-geladen in Get/ListTasks, `filter_label_ids` als SQL-Filter; Custom-Field-Definitionen (Migr. 146, RBAC `work_custom_fields:*`) |
| Calendar | `/api/v1/calendars`, `/events`, `/resources` | Resources, Holidays, Meetings |
| **Calendar Booking-Pages** (auth) | `/api/v1/calendar/booking-pages`, `/{id}` | CRUD, RequirePermission `booking-pages`, hinter `modules.calendar`-Flag. Verwaltet `booking_pages`+`public_bookings` — siehe [[datenbank]] Chain PILOT |
| **Public Booking** (unauthenticated) | `/api/v1/public/booking-pages/{slug}`, `/{slug}/availability`, `/bookings` | Eigene Route-Gruppe ohne authMiddleware (Muster `route_guest.go`), IP-Rate-Limit. `GET /{slug}`: öffentliche Page-Info. `GET /{slug}/availability`: freie Slots (aus `availability_rules` minus belegte Calendar-Events). `POST /bookings`: legt Calendar-Event an + Bestaetigungsmail |
| Video | `/api/v1/video/calls`, `/recordings` | LiveKit-basiert |
| **Meetings** (auth) | `/api/v1/meetings`, `/{id}/join`, `/notes`, `/action-items` | **2026-06-23 Meeting-Parität:** `POST/GET /{id}/chat` (persistierter In-Call-Chat, `meeting_chat`); Host-Controls (server-autoritativ, meeting-scoped Authz Organisator+Co-Host, KEIN RBAC-Guard): `POST/DELETE/GET /{id}/cohosts`, `POST /{id}/moderation/{mute,mute-all,kick}`, `POST /{id}/lock`. Alle flaches JSON via `response.Proto` (Timestamps RFC3339). gRPC: VideoService +7 RPCs (regen). ⚠ `meeting.locked` ist NUR im SetMeetingLockRequest, noch nicht im Meeting-Response-DTO → FE-Lock-Indikator-Fix offen (Wave 5) |
| Finance | `/api/v1/quotes`, `/invoices`, `/datev/export` | Payments, Credit Notes, Dunning, ZUGFeRD. **2026-06-19 (Wave 4):** CreateQuoteFromDeal akzeptiert optionales `tax_mode` (Proto-Feld 4; leer → Settings-Fallback via `resolveTaxMode`). Kunden-`ust_id_nr` wird bei Invoice/Quote/CreditNote-Create gegen DACH-Pruefziffer validiert (`ustid_eu`, `validateCustomerVAT`) — siehe [[security]] |
| Inbox | `/api/v1/inbox` | Messages, Routing Rules, Teams (Unified Inbox) |
| Automation | `/api/v1/automations` | Workflow CRUD, Execution Logs, Templates |
| HR | `/api/v1/hr/employees`, `/hr/leave`, `/hr/absences` | Teilt "biz" gRPC-Server. **Seit 2026-06-11:** `POST /hr/employees` (CreateEmployee mit Schema-Defaults `97f30324`), FE↔BE-Shape via `adaptEmployee()` (camelCase, ContractType `intern`/`temporary`/`mini_job` statt `praktikum`/`freelance`, `67fd78b9`); hr_grpc komplett auf `middleware.GetTenantID(ctx)` (`6ff7989a`) |
| Security | `/api/v1/security/…` | Audit Logs, 2FA, App-Passwords, GDPR Export/Erasure (seit 2026-06-10 echte Handler statt Stubs, siehe [[security]]) — teilt "auth" gRPC |
| Plugin | `/api/v1/plugins/…` | Manifests, Installations, Execution-Logs, Templates |
| Global Search | `/api/v1/search` | Cross-Service (CRM + Dokumente), 500ms Timeout |
| Guest Chat | `/api/v1/guest/…` | Public, kein Auth, eigene Session-Tokens |
| Health | `/health` | Public, kein Auth |
| Bexio | `/api/v1/integrations/bexio/…` | OAuth-Flow, Sync Trigger/Status/Logs |
| Lexware | `/api/v1/integrations/lexware/…` | API-Key-basiert |
| DATEV Upload | `/api/v1/datev/upload` | CSV-Upload (Buchungsstapel) |
| CalDAV/CardDAV | `/caldav/…`, `/carddav/…` | go-webdav Proxy, App-Passwords |
| WOPI | `/api/v1/wopi/…` | Document Lock/Unlock, CheckFileInfo (OnlyOffice) |
| Dialer | `/api/v1/dialer/…` | Campaigns, Calls, Agent Status, Outcomes, Dashboards (25 Endpoints) |
| Wiki | `/api/v1/wiki/articles`, `/versions`, `/attachments`, `/categories`, `/search` | 14 Endpoints, Postgres-FTS (tsvector+GIN, deutsch), hinter `modules.wiki`-Flag |
| Helpdesk | `/api/v1/helpdesk/tickets`, `/messages`, `/queues`, `/canned-responses`, `/sla-policies` | 22 Endpoints, SLA-Engine + Ticket-Merge, hinter `modules.helpdesk`-Flag |
| Berichte | `/api/v1/berichte/definitions`, `/schedules`, `/kpis` | 14 RPCs live (Sprint 1 Welle 5-6, 2026-04-19). Definitions-CRUD (5), Run/Cache/Export/Invalidate (4), Schedules-CRUD + Toggle (5), DashboardKPIs (1). Export als `Content-Disposition: attachment` mit PDF/CSV/XLSX-Bytes. Hinter `modules.berichte`-Flag + RBAC-Permission `berichte:reports:{read,write}`. |
| Formulare | `/api/v1/formulare/schemas`, `/submissions`, `/webhooks`, `/deliveries` | 18 RPCs live (Sprint 1 S1.3, 2026-04-19). Schema-CRUD + Duplicate + Stats (7), Submissions (5: List/Get/Create/UpdateStatus/Export), Webhooks (5: CRUD + List), Deliveries (1: ListDeliveries). Export als CSV/XLSX via `format`-Query-Param. Webhook-Delivery via DB-Queue + Worker (HMAC-SHA256 `X-Cosmi-Signature`, Exp-Backoff 30s→2h, Dead-Letter nach 5 Versuchen). Hinter `modules.formulare`-Flag + RBAC-Permissions `formulare:schemas:{read,write}`, `formulare:submissions:{read,write}`, `formulare:webhooks:write`. Ports: gRPC 50064, Health 9104. |
| Inventar | `/api/v1/inventar/items`, `/movements`, `/warnings`, `/transfer`, `/report`, `/export` | 14 Endpoints (Sprint 2 Welle 1). Items-CRUD, Stock-Adjust mit Oversell-Guard, Transfer, Movements, Warnings (auto bei `quantity <= min_quantity`). Hinter `modules.inventar`-Flag + RBAC `inventar:{item,movement,warning}:{read,write}`. Ports: 50070/9110. |
| Einkauf | `/api/v1/einkauf/suppliers`, `/purchase-orders`, `/po-lines`, `/receive` | ~18 Endpoints (Sprint 2 Welle 1). Supplier-CRUD, PO-Lifecycle Submit→Sent→PartiallyReceived→Received→Closed, ReceiveGoods-Stub fuer Sprint-3-Inventar-Wiring. Hinter `modules.einkauf`-Flag. Ports: 50071/9111. |
| Produktion | `/api/v1/produktion/orders`, `/machine-bookings`, `/plans` | ~16 Endpoints (Sprint 2 Welle 1). Order-Lifecycle, Machine-Booking-Konflikt-Pruefung (advisory-lock + `pg_advisory_xact_lock`), Capacity-Overview. Hinter `modules.produktion`-Flag. Ports: 50072/9112. |
| Vertraege | `/api/v1/vertraege/contracts`, `/parties`, `/reminders` | ~14 Endpoints (Sprint 1 S1.5). Laufzeit-Engine, Reminder-Worker (advisory-lock-claim, 5+60min Ticker). E-Signatur = Canvas-EES persistiert via Migr. 143 (`d643feaf`, statt Skribble-Placeholder); Dokumente-Upload via Presign-Flow (`a362b98d`). Hinter `modules.vertraege`-Flag. Ports: 50073/9113. |
| **Rapporte** | `/api/v1/rapporte/reports`, `/lines`, `/attachments`, `/pending-approvals`, `/export` | 18 RPCs (Sprint 2 Welle 2A, 2026-04-28). Report-CRUD + Approval-State-Machine (Submit/Approve/Reject), Line-CRUD, MinIO-Photo-Upload via Gateway-File-Handler (`photo_keys` als TEXT[]), GPS-Tag (`lat`/`lon`), PDF-Export-Stub. Hinter `modules.rapporte`-Flag + RBAC `rapporte:{report,line,attachment}:{read,write}`. Ports: 50074/9114. |
| **Schichten** | `/api/v1/schichten/shifts`, `/assignments`, `/templates`, `/arbzg-check` | 16 RPCs (Sprint 2 Welle 2A, 2026-04-28). Shift-CRUD + Publish-Flow, Assign/Unassign mit ArbZG-§5-Pre-Check (11h Ruhezeit, DST-aware), Template-Apply auf Wochenmuster. Hinter `modules.schichten`-Flag + RBAC `schichten:{shift,assignment,template}:{read,write}`. Ports: 50075/9115. |
| **Fuhrpark** | `/api/v1/fuhrpark/vehicles`, `/services`, `/damages`, `/upcoming-services`, `/tuv-due` | 18 RPCs (Sprint 2 Welle 2A, 2026-04-28). Vehicle-CRUD, Service-Scheduling, Damage-Reporting mit MinIO-Photos, TÜV-Reminder-Cron-Worker (advisory-lock, 7d/1d-Fenster, idempotent via `tuev_reminder_sent_at`). Hinter `modules.fuhrpark`-Flag + RBAC `fuhrpark:{vehicle,service,damage}:{read,write}`. Ports: 50076/9116. |
| **Vermietung** | `/api/v1/vermietung/objects`, `/rentals`, `/inspections`, `/availability`, `/calendar` | 20 RPCs (Sprint 2 Welle 2A, 2026-04-28). Object-CRUD, Rental-Lifecycle Reserved→Active→Completed, GIST-tstzrange-Overlap-Check (`CheckAvailability` + Pre-Check vor `CreateRental`), Handover/Return-Inspections mit Photo-Uploads, Calendar-View. Hinter `modules.vermietung`-Flag + RBAC `vermietung:{object,rental,inspection}:{read,write}`. Ports: 50077/9117. |
| **Files (Presign)** | `/api/v1/files/presign-upload`, `/presign-download` | Generische presigned MinIO-Upload/Download-URLs (2026-06-11, `13a7b90a`+`1aef2f45`): Browser lädt direkt gegen `s3.zentria.tech` (public MinIO-Endpoint), RBAC `files:*` (Seed Migr. 144). Genutzt von vertraege-Dokumenten-Upload (`a362b98d`) |
| Feature Flags | `/api/v1/feature-flags` | Resolver-Output (17 Flags: 14 Modul-Flags + `plugins.wasm`/`plugins.config`/`plugins.api`), auth-required |
| Integration Config | `/api/v1/integrations/configs` | Teams/Slack Webhooks + OAuth |
| Registrar | (intern) | Service-Registrierung im Gateway |
| Health | `/health` | Public, kein Auth, Version/Commit/BuildTime |

## Auth-Flow
1. POST `/api/v1/auth/login` (email + password)
2. Falls 2FA aktiv: `requires_two_factor: true` + pending token
3. POST `/api/v1/auth/verify-2fa` (pending token + TOTP code)
4. Response: `access_token` (JWT, 15min) + `refresh_token` (opaque, 7d)
5. Alle geschuetzten Endpoints: `Authorization: Bearer <access_token>`
6. Token-Refresh: POST `/api/v1/auth/refresh` mit refresh_token

### JWT-Claims (seit 2026-04-28, Welle 2D)
- `uid` — User-UUID
- `tid` — Tenant-UUID (Migration 000104: `users.tenant_id`). Leer/invalid → 401 Unauthorized auf jedem Modul-Endpoint, kein Placeholder-Fallback. Details: [[security]] "JWT Tenant-Claim & Cross-Layer-Hardening".
- `roles` — RBAC-Rollen
- `perms` — Permissions (`resource:action`)

## Request/Response Pattern
- Content-Type: `application/json`
- Error-Responses: 400, 401, 403, 409, 412, 422, 429
- Rate Limiting: 100 rps pro User/IP, `429 Too Many Requests` mit `Retry-After: 1`
- **Idempotency-Key Header (seit 2026-04-28, Welle 3):** Mutations (POST/PUT/PATCH) unter `/api/v1/` SOLLEN `Idempotency-Key: <UUIDv4>` mitschicken. Aktuell WarnMode (loggt fehlende Keys), HardMode in Welle 4. Replay → cached Response, Conflict → 422, In-Flight → 409+Retry-After:2. Whitelist `/auth/login|refresh|2fa`. Details: [[security]] "Idempotency-Konvention".
- **Pre-Recording-Consent 412 (seit 2026-04-28, Welle 3):** `POST /api/v1/video/recordings/start` returniert 412 Precondition Failed wenn der Initiator nicht vorher `POST /api/v1/video/recordings/{id}/initiator-consent` aufgerufen hat. Frontend zeigt Pre-Dialog vor StartRecording. Details: [[security]] "Pre-Recording-Consent".
- **Proto-Serialisierung via `response.Proto` (Welle F / R3-P0-1, 2026-06-21):** Proto-Message-zurueckgebende Handler serialisieren ueber `response.Proto` (protojson, `UseProtoNames`+`UseEnumNumbers`) statt `response.JSON` (encoding/json) → `google.protobuf.Timestamp` als **RFC3339** statt `{seconds,nanos}`, Enums als Integer (FE-kompatibel). Umgestellt: alle 25 Proto-Handler in `route_video.go` (Meetings/Recordings/Action-Items/Presence) + Dialer (7a). Hand-geschriebene Ext-Structs ohne `proto.Message` (z.B. `GetRecordingConsents`) bleiben `response.JSON`. ⚠ `response.Proto` rendert `int64`/`uint64` als JSON-**String** (proto3-Spec) — pro Modul FE-int64-Audit noetig vor Umstellung, kein globaler Blind-Sweep. Helper: `backend/internal/server/response/response.go`. Siehe [[troubleshooting]].

## Frontend-Integration
- API-Client: `desktop/src/renderer/src/api/client.ts` (openapi-fetch)
- Automatischer Bearer-Header aus Auth-Store
- **Automatischer `Idempotency-Key`-Header (seit Welle 3)** auf jeder Mutation via `api/idempotency.ts::generateIdempotencyKey()` (UUIDv4)
- 401-Interception → transparenter Token-Refresh mit Concurrent De-Duplication
- **Offline-Queue (seit Welle 3):** Bei `!navigator.onLine` werden Mutations in IndexedDB-Queue gepuffert (`api/offline-queue.ts`, idb-keyval) statt `OfflineError` zu werfen. `useOnlineStatus`-Hook drained bei Online-Event (max 5 parallel, exp-Backoff, Dead-Letter nach 5 Versuchen). UI: `<OfflineBanner />` zeigt Queue-Count + Manual-Drain.
- 40+ React Query Hooks in `desktop/src/renderer/src/api/hooks/`
- Dialer: Eigener `dialer-client.ts` (typed fetch, nicht openapi-fetch — noch nicht in openapi.yaml)

## Verwandte Notes
- [[architektur]] — Service-Architektur & Gateway Routes
- [[datenbank]] — Schema & Tabellen
- [[security]] — Auth & Middleware
- [[integrationen]] — Bexio, Lexware, DATEV Details

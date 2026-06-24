# Backend-Gaps für Luke

> Was im Backend fehlt oder nicht „ready zum Verknüpfen" ist, damit das Frontend zu Feature-Parität andocken kann.
> Claude sammelt, Darien reicht an Luke weiter. Stand: Welle 1 (2026-06-01). **Status-Update 2026-06-10 (additiv, ✅-Markierungen):** erledigte Punkte aus Chain PILOT (2026-06-09) + Marathon-Tag-2-Wellen 1+2 (2026-06-10) sind inline markiert.
> Priorität: 🔴 ZFA-Pilot-kritisch · 🟠 wichtig · ⚪ später/Post-Launch.

## ✅ Erledigt seit 2026-06-09 (nicht ursprünglich in dieser Liste)
- **GDPR-Export/-Erasure-Handler echt** (waren Stubs): alle 14 Handler auf echte SQL, tenant+user-gefiltert, Art.-17(3)(e)-Retention (`47d210d9`, 2026-06-10).
- **Beratungsprotokoll ZFA**: `advisory_protocols` Migration 000137 + 7 CRM-RPCs (`6b211222`, 2026-06-10) — Detail-Eintrag unten ebenfalls markiert.

## 🔴 ZFA-Pilot-kritisch

### ✅ kalender — Terminbuchungs-Link (Online-Terminbuchung) — ERLEDIGT 2026-06-09 (Chain PILOT, Migrationen 000135/000136)
ZFA-Akquise hängt an Online-Terminbuchung. FE-Flow existiert komplett als Mock, BE fehlt ganz.
- `GET/POST/PUT/DELETE /api/v1/calendar/booking-pages` — Buchungsseiten (Slug, Services, Verfügbarkeitsregeln)
- `GET /api/v1/public/book/:slug` — **öffentlich/unauthenticated** (Kunde bucht ohne Login)
- `GET .../availability?date=&service=` — freie Slots aus Kalender-Belegung berechnen
- `POST /api/v1/public/bookings` — öffentliche Terminanlage → erzeugt Event + Bestätigung

### ✅ dialer — DSGVO-Consent-Absicherung — ERLEDIGT 2026-06-09 (Chain PILOT, Asserter in `cmd/dialer/main.go` verdrahtet + Regressionstest)
`consentAsserter` ist im Standard-`NewService`-Konstruktor `nil` — nur `NewServiceWithConsent` verdrahtet den Einwilligungs-Check. Prüfen, ob der Standard-Konstruktor irgendwo aktiv ist → sonst Anrufe ohne Consent möglich. Für Finanzberatung heikel.

## 🟠 Wichtig (Kern-Module, ZFA-relevant)

### kontakte
- XLSX/Excel-Import-Endpoint (CSV/vCard existieren, XLSX fehlt)

#### ✅ Kontakte/360° — fehlende Hooks/Endpoints für Kunden-360°-Ansicht — BACKEND ERLEDIGT 2026-06-10 (Welle 2, `52a74373`, Migration 000141): `GET /contracts?contact_id=` (contract_parties-EXISTS) + `GET /finance/invoices?contact_id=` (contact_id-Spalte + Backfill quote→deal→contact). FE-Hooks nachziehen = Claude/FE-Lane.
Folgende Verknüpfungen konnten im ContactDetailPanel NICHT gebaut werden, weil Hooks + Endpoints fehlen:
- **Verträge am Kontakt**: kein `GET /api/v1/contracts?contact_id={id}` und kein Frontend-Hook `useContactContracts(contactId)`. Vertragsservice hat nur generisches CRUD; die Filterung nach `contact_id` fehlt im Modell und der Route.
- **Rechnungen am Kontakt**: kein `GET /api/v1/finance/invoices?contact_id={id}` und kein Frontend-Hook `useContactInvoices(contactId)`. finance_line_items hat keinen direkten Kontakt-FK; Normalisierung (Sprint 4) Voraussetzung.
- Sobald Luke diese Endpoints + contact_id-Felder ergänzt, kann das FE die beiden Sektionen in ContactDetailPanel nachziehen (Muster: analog useDeals mit contact_id-Filter).

### crm
- Lead-Scoring: Score-Feld im Contact-Modell + Berechnungsservice (Regelwerk) + Endpoint
- Umsatz-Forecasting: dedizierter `/api/v1/reports/forecast`-Endpoint (Zeitreihe auf Deal-Wahrscheinlichkeit)
- E-Mail-Marketing/Kampagnen: kompletter Service fehlt (`/api/v1/campaigns`)

### vertraege
- ✅ **`UploadDocument` — ERLEDIGT 2026-06-11 (`a362b98d`):** Stub-Endpoint entfernt; FE nutzt den generischen Presign-Flow (presign-upload → PUT → PATCH `document_url`). ⚠ Browser-PUT braucht `MINIO_PUBLIC_ENDPOINT` in Prod (siehe §dokumente). FE-Aufrufer von `useUploadDocument` auf `{contractId, file}` umgestellt.
- Audit-Log: `contract_events`-Tabelle (action/user/timestamp/payload) + `GET /contracts/{id}/events`
- Digitale Signatur-Workflow (Phase D, Skribble/DocuSign): `POST /contracts/{id}/send-for-signing`, `/sign`, Status-Endpoint + Webhook-Receiver

### kommunikation (chat + inbox, werden zusammengeführt)
- ✅ **Reaction-Endpoints — ERLEDIGT 2026-06-11 (`c9c19380`):** `POST /api/v1/messages/{id}/reactions` (Toggle) + `GET .../reactions` + `POST /api/v1/messages/reactions/summary` (Batch). Bestehender `work/reaction`-Service in ChatGRPCServer verdrahtet, 501-Stubs aus route_video entfernt, FE + MSW umgestellt. ✅ Follow-up erledigt 2026-06-11 (`507487b9`): `MessageBubble.tsx` nutzt `useToggleReaction`, `MessageList` batch-fetcht via `useReactionSummary`, Demo-Store `stores/chatReactions.ts` gelöscht.
- Chat-Datei-Upload-Route: `POST /api/v1/channels/{id}/files` (Multipart) — Service `Upload()` existiert, Route fehlt
- **Externe Kanal-Verknüpfungen verwaltbar machen** (für Modul-Merge): Settings/CRUD um nicht-interne Kanäle (Mail/WhatsApp/Widget) anzubinden — Routing-Rules-Infra im inbox-Service ist Basis

### dokumente
- 🟠 **Echt-Schaltung-Befund 2026-06-24 (Gateway-Wire-Shape-Inkonsistenz, FE-tolerant gefixt, kanonischer Fix = Backend):** Beim Anbinden des FE an das echte document-Backend kamen mehrere Drift-Punkte hoch, die MSW verdeckt hatte. FE-seitig in `document-client.ts` abgefangen (Normalizer), damit die UI live rendert — **kanonisch gehört das ins Gateway** (`backend/internal/gateway/route_document.go`), dann kann der FE-Normalizer wieder weg:
  1. **List-Responses inkonsistent:** `HandleListFolders`/`ListTags`/`ListShares`/`ListVersions`/`ListEntityLinks`/`ListActivity`/folder-path geben das **bare Array** zurück (`response.JSON(w, …, resp.Folders)`) → bei leer serialisiert protojson das zu **`null`** (nicht `[]`). `HandleListFiles` dagegen wrappt (`{files, total}`). FE-Typen + MSW erwarten überall die gewrappte Form `{folders, total}` etc. → empfohlen: alle List-Handler konsistent in `{<key>, total}` wrappen und leere Slices als `[]` (nicht `null`) emittieren.
  2. **Single-Entity-Responses bare:** `get`/`create`/`copy` (folder+file) geben das **bare Objekt** zurück (`resp.Folder`), FE+MSW erwarten `{folder}`/`{file}`. → konsistent wrappen.
  3. **`POST /documents/folders/initialize-user` verlangt einen Body:** `decodeAndValidate[initializeUserSpaceRequest]` schlägt bei leerem Body mit **400 „invalid request body"** fehl, obwohl `user_id` optional ist. FE sendete keinen Body → init schlug immer fehl. FE schickt jetzt `{}`. → entweder leeren Body tolerieren oder im OpenAPI als pflicht-`{}` dokumentieren.
  4. **protojson-Wire-Shapes (erwartbar, FE normalisiert):** Timestamps `{seconds,nanos}` statt ISO; `space_type` als Enum-**Int** (`1`=personal/`2`=team/`3`=project) statt String; `file_count` fehlt am Folder-Objekt (FE default 0); `file_size` (int64) ggf. als String.
  - **Verifiziert:** READ live gegen lokales document-Backend (Ordner Bilder/Dokumente/Meine Dateien/Vorlagen rendern, keine Crashes/Invalid-Dates, Screenshots `desktop/.qa-screenshots/dokumente-mock-exit/`), Create-Pfad per API (Folder 201 + erscheint in der Liste). **Upload live nicht testbar** lokal (braucht `MINIO_PUBLIC_ENDPOINT`+CORS, s.u.).
- ✅ **Presign-Upload öffentlicher MinIO-Endpoint — CODE ERLEDIGT 2026-06-11 (`1aef2f45`):** `MINIO_PUBLIC_ENDPOINT`/`MINIO_PUBLIC_USE_SSL` + zweiter presign-only minio-go-Client, Caddy-Block `s3.zentria.tech → minio:9000` (docker + ansible `minio_public_domain`), CORS via `mc cors set` (`MINIO_CORS_ALLOW_ORIGIN`). ⚠ **Prod-Rollout offen:** DNS-Eintrag `s3.zentria.tech` (Cloudflare, DNS-only!) + Env in `/opt/kmuhub/.env.production` + Electron-Origin für CORS verifizieren.
- Datei-Kommentare: Comment-Tabelle + Endpoints
- Externe Share-Links: Token-Store + `GET/POST /api/v1/documents/share-links` (Ablauf, Passwort-Hash) + öffentliches Resolve-Endpoint
- Tenant-Settings dokumente (2026-06-10, Strom D): `stores/dokumenteSettings.ts` ist mock-first (Dateityp-Gruppen, Standard-Freigabe, OnlyOffice-Schalter, Papierkorb-Tage). Settings-Foundation (Migration 138, `route_settings.go`) liegt inzwischen auf main → nach Merge nur noch FE-Wiring auf `tenant_settings`, kein neues Backend nötig. Enforcement der erlaubten Dateitypen beim Upload wäre Backend-seitig sinnvoll (aktuell nur Verwaltung).
- Versionsspezifischer Download (2026-06-10, Strom D): `GET /api/v1/documents/files/{id}/versions/{n}/download` fehlt — der „Herunterladen"-Button im Versionsverlauf kann nur die aktuelle Datei laden.
- Template-Storage (2026-06-10, Strom D): echte Dokument-Vorlagen (.docx/.xlsx/.pptx) + `POST /api/v1/documents/files/from-template/{templateId}` — FE lädt bis dahin generierte Platzhalter-Dateien hoch (TemplateGalleryDialog).
- Activity-Log (2026-06-10, Strom D, aus Darien-Feedback): `document_activities`-Tabelle + `GET /api/v1/documents/files/{id}/activity` + Schreiben bei Upload/Rename/Move/Copy/Download/Share/Version — FE (Viewer-Info-Panel „Aktivität") ist fertig und läuft mock-first.
- Thumbnail-Rendering (2026-06-10, Strom D, aus Darien-Feedback): Erstseiten-Vorschau für Kacheln (`thumbnail_key` existiert am Modell, Rendering-Service + Abruf-Endpoint fehlen) — FE zeigt bis dahin eine Seiten-Optik.
- ⚠ CSP-Hinweis (kein Gap, Review): `frame-src 'self' blob:` neu in `desktop/src/renderer/index.html` (Dokument-Viewer). Der OnlyOffice-iframe (externe `VITE_ONLYOFFICE_URL`) ist von `frame-src` vermutlich weiterhin blockiert — bei OnlyOffice-Scharfschaltung CSP um die Office-Domain erweitern.

### mails
- Multi-Account: Tabelle + `ListEmailAccounts` (aktuell 1 Account/User)
- Vorlagen/Quicktext: Template-CRUD (`email/template/`)
- Regeln & Filter: `email/rule/` + Endpoints

### helpdesk
- `contact_id`/`org_id` ins Ticket-Modell (CRM-Verknüpfung)
- `source_channel` ins Ticket-Modell (Multi-Channel) + Inbox→Ticket-Adapter-Endpoint
- Knowledge-Base-Endpoint (FE-Tab existiert)
- `time_spent`-Feld (Ticket-Zeiterfassung)

### berichte
- Query-Builder: BE-Executor liest `query_config` schon — Editor-Contract festzurren
- `ExecuteKindCross`-Methode im Executor (datenquellen-übergreifend)
- Breakout/Pivot-Schema in `RunReportRequest.Params`
- **Dashboard-KPIs (P-nico-02, 2026-06-09):** `GET /api/v1/berichte/kpis` hatte im Demo-Mode KEINEN MSW-Handler → Dashboard zeigte Leerzustand (keine Karten). Jetzt Demo-Handler `mocks/handlers/berichte.ts` mit statischen KPIs gebaut. Backend: echter KPI-Service (Werte + `change_percent` pro Modul) fehlt.
- **KPI-Zeitreihe für Sparkline (P-nico-02):** Die KPI-Karten zeigen eine Mini-Trendlinie, deren Verlauf aktuell FE-seitig deterministisch aus `kpi.id` + `change_percent` synthetisiert wird (`buildSparklineSeries` in `DashboardGrid.tsx`). Echte Zeitreihe pro KPI (z.B. letzte 8 Perioden) sollte das Backend liefern → dann `sparklineData` aus echten Punkten speisen.
- **🔴 R-3b Server-PDF (Bericht-Authoring, 2026-06-20, FE B5-1 fertig):** Lese-Modus hat jetzt „Drucken / Als PDF" über `window.print()` + Print-CSS (`report-print.css`, blendet App-Shell aus, paginiert die A4-Bögen — Chromium-Druckvorschau verifiziert, 2-Seiten-PDF). Das reicht für lokalen Druck/„als PDF speichern". **Backend nötig für echten Server-Download:** `berichte-pdf`-Service (Token-geschützte Render-URL `/berichte/documents/:id/print` ohne Chrome → Playwright `page.pdf({format:'A4'})` → `application/pdf`-Blob), FE-seitig `GET …/documents/:id/export/pdf` → `<a download>`. Schriften eingebettet, Charts als Vektoren.
- **🔴 R-4 Cron-Executor + Mailer (Bericht-Scheduling, 2026-06-20, FE B5-2…B5-5 fertig):** FE-Demo vollständig — Scheduling-Modal am Bericht (Rhythmus-Picker→cron, interne/externe Empfänger, Format aus Tenant-Allowlist, aktiv-Toggle), Lauf-Historie + „Jetzt senden" laufen mock-first (`POST /schedules/:id/run` setzt `last_run_at`/`last_run_status` stateful; `ReportSchedule.definition_id = doc.id` koppelt an das Dokument). **Backend nötig:** Cron-Scheduler der fällige `ReportSchedule`s ausführt (Bericht rendern → PDF/XLSX/CSV → an `recipients` mailen), echte `report_runs`-Persistenz (statt FE-`buildRunHistory`-Seed) + Domain-Allowlist-Enforcement (`tenant_settings` `berichte` `schedule.allowed_domains`) + Release-Gate (nur `status='released'` planbar).
- **🟠 R-5 Integration/Verteilung (Bericht-Authoring, 2026-06-21, FE B6-1…B6-5 fertig):** FE-Demo vollständig — „Teilen"-Menü im Lese-Modus: an Aufgabe anhängen (`POST /tasks/:id/files` mit Verweis-Metadaten, vorhandener Endpunkt), an Kontakt anhängen (neuer stateful `POST /contacts/:id/files`-Mock), als PDF in Dokumente (`POST /documents/files/upload` mit Platzhalter-Blob + „Bericht"-Tag), externer Share-Link (neues `ReportShareToken`-Modell + stateful create/list/revoke). **Backend nötig:** (a) Bericht-Verweis als echter Typ in task-/contact-files (statt `mime_type:'application/cosmi-report'`-Konvention) + Anzeige als „Bericht-Link" in work/CRM; (b) echter PDF-Blob für R-5b (= R-3b Server-PDF statt Platzhalter); (c) **R-5c externer unauth. Zugriff**: öffentliche Token-Lese-Seite (`/share/report/:token` ohne Auth, Passwort-Check, Ablauf-Enforcement, `view_count`-Tracking) — `share_token`-Persistenz serverseitig.

### team
- ✅ **FE↔BE-Shape-Mismatch HR-Employees — ERLEDIGT 2026-06-11 (`67fd78b9`):** Doppelt-toleranter Adapter `adaptEmployee()` in `hr-client.ts` (snake_case vom Gateway, camelCase vom MSW-Demo; Enum akzeptiert Integer/Proto-String/Slug). `ContractType`-Union auf Proto-Wahrheit erweitert (`full_time|part_time|mini_job|intern|temporary`), i18n ×4 + Demo-Daten migriert (`praktikum`→`intern`, `freelance`→`temporary`). Team-Modul-API-Swap ist damit entsperrt.
- ✅ **CreateEmployee-Endpoint — TEILSCHNITT ERLEDIGT 2026-06-11 (`a3ad7158` + Fixes `97f30324`/`c2cc98ad`):** `POST /api/v1/hr/employees` (hr:admin) legt Profil für EXISTIERENDEN User an (`user_id` im Body). Prod-verifiziert 201 mit Schema-Defaults. ⚠ Follow-up: Auth-User-Anlage (Invite-Flow: email + temporary_password + roles, transaktional) — FE-Wizard muss bis dahin auf User-Picker oder zweistufigen Flow (register → create profile) umgestellt werden.
- ✅ **Latenter HR-Read-Layer-Bug GEFIXT 2026-06-11 (`c2cc98ad`):** alle users-JOINs in biz/hr referenzierten die nie existente Spalte `users.display_name` → Namens-Resolution + `GET /hr/employees/me` waren in Prod dead-on-arrival (unsichtbar wegen Demo-Mode). Jetzt `CONCAT_WS(first_name, last_name)`-Fallback auf email. ✅ Test-Gap geschlossen 2026-06-11 (`6ff7989a`): 9 pgtc-Integrationstests für employee/absence/leave (CONCAT_WS-Regression + Cross-Tenant) gegen echtes Migrations-Schema; zusätzlich 16x `req.GetTenantId()`→`middleware.GetTenantID(ctx)` in hr_grpc.go gesweept.
- Onboarding-Workflow-API (Template + Checklist)
- DATEV-**HR-Lohn**-Endpoint (bestehender `route_datev_upload.go` ist nur Buchungsdaten)
- **Lohnvorbereitung / Lohnlauf (P-team, 2026-06-07)** — siehe `team-datev-lohn-spec.md`. FE mock-first gebaut (`PayrollPrepPanel` + `payrollRuns`/`payrollSettings`-Stores). Backend: `payroll_runs` (period, group, status locked/exported, exported_at, employee_count) + **DATEV-Datei-Generierung** (LODAS / Lohn&Gehalt-Format mit Lohnarten + Abwesenheitsschlüssel) bzw. **Lohnimportdatenservice** (DATEVconnect, Akkreditierung). Bewegungsdaten-Aggregation aus Zeiterfassung+Abwesenheiten pro Periode/Gruppe. tenant_settings (module_id='team', key='payroll.*') für Berater-/Mandanten-Nr + Mappings.
- **Lohnauswertungsdatenservice** (Phase 2): Abrechnungen/Auswertungen zurück nach Cosmi importieren.
- ✅ **Demo-Daten-Lücke (modulweit) — ERLEDIGT 2026-06-11 (`7a367047`):** `mocks/handlers/team.ts` auf `EmployeeProfile`-camelCase-Shape angeglichen (`userName` befüllt). Demo zeigt jetzt Namen. ⚠ Echter API-Swap braucht noch einen Shape-Adapter — siehe neuen Gap-Eintrag oben (FE↔BE-Shape-Mismatch).

## ⚪ Später / Post-Launch / Architektur
- dialer: Gesprächsaufzeichnung (recording_url, an Video-Infra gekoppelt), AMD, Predictive (Phase 3 — bewusst)
- crm: Mobile App (PWA-Architekturentscheidung)
- mails: Exchange/EWS, PGP/S-MIME
- formulare: öffentlicher Submit-Endpoint (IsPublic-Flag da), File-Upload-Feldtyp, Submission-Mail

---

# Welle 2 — System, Produktivität, Finanzen, Automatisierung, Video

## 🔴 Pre-Launch wichtig
- ✅ **security — „Passwort vergessen"-Flow** — ERLEDIGT 2026-06-09 (Chain PILOT, Migration 000134: `password_reset_tokens` + forgot/reset-Endpoints, rate-limited, kein Enumeration-Leak).
- **profil/settings — User-Preferences-Persistenz**: Sprache/Theme/Region nur client-seitig (Store/localStorage). Für Multi-Device BE-Endpoint `GET/PUT /users/preferences`. (Für Electron-Single-Device tolerierbar.)

## 🟠 Wichtig
### admin
- Tenant-Provisioning-Endpoint (`POST /api/v1/tenants`) + Onboarding-Flow
- Super-Admin/System-Level-Rolle (über Tenant hinaus)
- Billing/License-Service (`/api/v1/billing`) — aktuell nur statische Mock-Daten
- Tenant-Ressourcen-Monitoring-API (Metrics intern da, kein HTTP-Endpoint)

### security / DSGVO  (FE-Mock-First-Batch S-1…S-5, Branch `parallel/security`)
> BE existiert weitgehend echt: GDPR-Export/-Erasure-Handler (`47d210d9`, alle 14 auf echte SQL), Audit/Sessions/PW-Policy/IP-Rules/2FA in `route_security.go`/`route_auth.go`. Das FE läuft mock-first (MSW); Verdrahtung gegen das echte BE = später (Claude/FE-Lane).
- **🔴 X-3-Spec-Lücke (alle 31 security/auth-Endpoints):** KEINER ist in `backend/api/openapi.yaml` dokumentiert (Spec endet bei `auth/reset-password`). Betroffen: `/security/audit|vault|gdpr/*|password/*|ip-rules|dsar/search`, `/auth/sessions*|2fa/*`. → openapi-Spec nachziehen, sonst bricht jede Typ-Regen-Runde + jeder Echt-Anschluss erneut.
- **Wire-Shape-Befund (encoding/json über protobuf, snake_case):** Alle Handler nutzen `response.JSON` (nicht `response.Proto`). Konsequenz für den FE-Client beim Echt-Anschluss: (a) **Listen sind gewrappt** — `{secrets:[…]}`, `{rules:[…]}`, `{export_requests:[…]}`, `{sessions:[…]}`, `{policies:[…]}`, `{entries,total}` (nie nacktes Array). FE-Client (`security-client.ts`) ist in S-1 bereits darauf ausgerichtet (entpackt gewrappte Shape). (b) **Timestamps als `{seconds,nanos}`** statt RFC3339 → beim Echt-Anschluss `normalizeWireTimestamps()` (`api/wire-time.ts`) im Client anwenden (MSW liefert ISO, geht durch).
- **Pfad-Abweichungen FE-Client ↔ echtes BE (beim Echt-Anschluss prüfen):** GDPR-Export-Request `POST /gdpr/export` (Client: `/gdpr/export/request`), Download `GET /gdpr/download/{token}` (Client: `/gdpr/export/{token}/download`), Approve/Deny über `/gdpr/exports/{id}/approve|deny`. 2FA-Policy-Pfad `policy` (singular) vs Client `policies`.
- **🔌 Verdrahten (nach Echt-Schaltung):** `security-client.ts` + Hooks gegen das echte Backend testen (Demo-Mode aus), Pfade + Wire-Shapes obiger Liste abgleichen, Timestamp-Normalizer einhängen.

### profil
- Avatar-Upload-Endpoint (MinIO) — Camera-Button im FE wartet darauf

### settings
- Workspace-Branding-Persistenz (`/api/v1/tenant/branding`) — aktuell nur localStorage
- Modul-Aktivierungs-Toggle exponieren (Flag-Registry existiert)

#### ✅ Settings-Fundament (Scope-Hierarchie) — ERLEDIGT 2026-06-10 (Welle 1, `360f92e6`, Migration 000138: `tenant_settings`/`user_settings`/`tenant_module_leads`, 3-Ebenen-Resolve serverseitig, tenant-Writes nur Lead/Admin, co-located im auth-Binary). FE-Umstellung localStorage→Endpoints = nächster Schritt.
3-Ebenen-Modell: **Tenant-Default → Modul-Leiter-Override (tenant-weit) → User-Override (persönlich)**. FE komplett gebaut (`ModuleSettingsShell`, `useIsModuleLead`, `useModuleLeadsStore`), persistiert aktuell nur in localStorage.
- **`tenant_module_leads`-Tabelle** (`tenant_id`, `user_id`, `module_id`, `granted_by`, `granted_at`) — wer ist Modul-Leiter für welches Modul. Admin setzt das im Team-Modul (MemberDetailPanel → „Erweiterte Moduleinstellungen"). Endpoints: `GET /api/v1/tenant/module-leads?user_id=`, `PUT/DELETE .../module-leads/{user_id}/{module_id}`.
- **Settings-Scope-Persistenz**: zwei Ebenen statt einem Blob. `tenant_settings` (`tenant_id`, `module_id`, `key`, `value`) für tenant-weite Defaults (nur Modul-Leiter/Admin schreibbar) + `user_settings` (`user_id`, `module_id`, `key`, `value`) für persönliche Overrides. Resolve-Reihenfolge serverseitig erzwingen (RBAC: tenant-Writes nur mit module-lead/admin).
- Beispiel CalendarSettings: `defaultView/weekStartsOn/defaultReminder` = user-scope; `workStartHour/workEndHour/holidayRegion` = tenant-scope.

### notifications
- E-Mail- + SMS-Kanal im Gateway exponieren (Dispatcher existiert intern)

### work
- `start_date`-Feld im Task-Modell + Proto (für vollwertigen Gantt; aktuell nur due_date)
- Projekt-Portfolio-Entität + Aggregations-Endpoint

### zeiterfassung
- **`GET /api/v1/hr/time/balance`** (Stundenkonto-Saldo, kumuliert + Perioden-Übertrag) — **P1 FE-mock-first verdrahtet** (`useTimeBalance`, Shape `{balanceMinutes, asOf, periodStart, targetWeeklyMinutes}`); braucht echten Endpoint.
- Export-API: CSV ist **P4 client-seitig** real; **DATEV-Lohn (LODAS)** + XLSX + PDF brauchen Serverside-Generierung.
- `tenant_settings` für zeiterfassung-Regeln (Wochensoll, Auto-Pause-Schwellen, Rundung, Feiertagsregion) — **P4 FE-mock-first** (`stores/zeiterfassungSettings`).
- **`POST /api/v1/hr/time/entries`** (manueller Eintrag) + **`GET /api/v1/hr/time/projects`** (Projekt-Taxonomie) — **P2 FE-mock-first verdrahtet** (`useCreateTimeEntry`/`useTimeProjects`); brauchen echte Endpoints.
- **`GET /api/v1/hr/time/analytics?range=week|month`** (KPI-/Tagestrend-/Projekt-/Billable-Aggregation) — **P3 FE-mock-first verdrahtet** (`useTimeAnalytics`); echter Aggregations-Endpoint oder client-seitig aus Entries.
- HR-Worktime-Entry um `project_id`/`customer_id`/`service_code` (+ `billable`) erweitern — Projekt-Liste ggf. an work-Projekte/CRM-Kunden koppeln statt eigener Taxonomie.
- **Wochen-Freigabe-Workflow** (submit/approve/reject auf Wochenebene) + **Team-Wochenübersicht** (`GET /hr/time/team`) — **P5 FE-mock-first verdrahtet** (`useSubmitWeek`/`useApproveWeek`/`useRejectWeek`/`useTeamTime`); brauchen echte Endpoints + `time_week_submissions`-Tabelle.
- Weekly-Summary braucht `totalBreakMinutes` (pro Tag + Top-Level) — im echten Endpoint mitliefern (Mock war lückenhaft, P5 gefixt).
- ✅ **Architektur-Entscheidung getroffen (Darien, 2026-06-14):** HR-API = Single Source. P1 hat das Header-Widget auf API konsolidiert (`WorkClockWidget`; `TimeTrackerWidget`+`ClockInButton` gelöscht) und die Demo-Doppelquelle behoben (idle `/hr/time/*`-Handler aus `team.ts` raus → `hr.ts` serviert). Die 10 toten Store-Views werden ab P2/P3 an die HR-API portiert, danach `stores/timetracking.ts` gelöscht. Details `reviews/zeiterfassung.md`.

### wiki
- Share-Token-Routes in `route_wiki.go` registrieren (Repo-Methoden existieren) + öffentl. Read-Endpoint
- Artikel-Templates-Endpoint (FE-Dialog existiert)

### finanzen (Symbiose-Ziel — NICHT Vollersatz, siehe finanzen-buchhaltung-strategy.md)
Strategie: Cosmi macht Vorkette (Angebot→Zahlungseingang) eigenständig, übergibt an DATEV/Bexio. Steuerberater macht Kontierung/Bilanz/USt/Lohn.
- ✅ **DATEV EXTF-Export (DE, Launch-kritisch)** — EXISTIERT KOMPLETT (Befund 2026-06-10: `internal/biz/datev/exporter.go`, EXTF ASCII/CSV Windows-1252, `POST /finance/export/datev`; Liste war veraltet).
- **Bexio-API (CH, Launch-kritisch):** OAuth2, Rechnungen/Kontakte bidirektional sync. *(Status-Check 2026-06-10: Service-Gerüst `internal/biz/bexio/` existiert substanziell — Scope-Abgleich gegen Welle 7 läuft, siehe `.planning/bexio-scope-check.md`.)*
- ✅ **E-Rechnung (Launch-Blocker)** — ERLEDIGT 2026-06-10 (Welle 2, `887d5b36`, Migration 000140): Ausgang existierte (XRechnung-UBL + ZUGFeRD); Eingang neu — `POST /finance/invoices/import` (multipart, CII/UBL-Parser + PDF-Extraktion via pdfcpu) + `finance_incoming_invoices` (received→reviewed→booked/rejected).
- ✅ **GoBD-Belegarchiv (Launch-Blocker)** — ERLEDIGT 2026-06-10 (Welle 2, `45a8ed61`, Migration 000139): `gobd_documents` immutable + SHA-256 + `gobd_document_events` append-only, Retention 31.12.(Jahr+8), Routen `/finance/gobd-archive`, MinIO-Storage.
- Wiederkehrende Rechnungen (Tabelle + Scheduler + CRUD) · OP-Liste · mehrstufiges Mahnwesen
- Zahlungsabgleich: CAMT.053/MT940-Import + Matching · später finAPI/HBCI-Banking
- `currency`-Feld + Wechselkurslogik (aktuell EUR hardcoded; CHF/USD)
- BMD-Export (AT) + Lexware/lexoffice-Anbindung (Selbstbucher) — Post-Launch

### ✅ kontakte — Beratungsprotokoll (Finanzberatung, P8) — BACKEND ERLEDIGT 2026-06-10 (Welle 1, `6b211222`, Migration 000137: 57 Spalten/8 Abschnitte, Immutability nach HandOver, `referred_by_contact_id` + `client_segment` A/B/C, 7 CRM-RPCs; PDF-Endpoint = 501-Stub, FE nutzt window.print)
- `advisory_protocols`-Tabelle (contact_id, ~40 Felder über 8 Abschnitte, **immutable nach Aushändigung**, 10-Jahre-Retention, DSGVO Art.6(1)(c)). Endpoints CRUD + PDF-Export (Aushändigung).
- „Empfohlen von"-Feld am Contact (Self-Referenz) + Empfehler-Report-Aggregation.
- Mandanten-Segment A/B/C (regelbasiert nach Umsatzpotenzial) — Feld + Berechnungsregel.

#### kontakte P7/P8 — FE-Finish (2026-06-09): mock-first → Backend-Persistenz
FE ist jetzt komplett (UI + Verdrahtung). Folgende Stores sind localStorage und müssen server-seitig/tenant-weit persistiert werden:
- **Beratungsprotokoll-PDF:** FE generiert die Geeignetheitserklärung jetzt per `window.print()` (dauerhafter Datenträger, MiFID II/§64 WpHG/FinVermV). **Backend bleibt nötig:** server-seitige PDF-Generierung + **revisionssichere, unveränderliche Ablage** finalisierter Protokolle (10 J.) — `window.print` erfüllt die Aufbewahrungspflicht NICHT (`useAdvisoryProtocolsStore`).
- **Lead-Scoring-Konfiguration:** Punkte-Regeln + Schwellen jetzt konfigurierbar (`useLeadScoringStore`) → `tenant_settings (module_id='crm', key='leadScoring.*')`. (Score-Feld/Engine am Lead siehe oben.)
- **Manuelle Segment-Überschreibung:** `useSegmentOverrideStore` pro Kontakt → Feld `contacts.segment_override` (ergänzt die regelbasierte Berechnung).
- **CustomFields-Definitionen (CRM):** `useContactsStore` localStorage → Definitions-CRUD-Endpoint (analog work, s.u.).
- **Tags:** `/api/v1/tags` in OpenAPI-Spec aufnehmen (aktuell raw-fetch, untypisiert).
- Entfernt: `NewsletterPanel` (toter Mock-Stub) — ein echtes Newsletter-/Kampagnen-Feature bräuchte ein E-Mail-Kampagnen-Backend.

### automatisierung
- Branch-/Merge-Step im Workflow-Modell + Engine (aktuell sequenziell)
- `http_request`-Action + inbound `webhook.received`-Trigger
- Zeitbasierte/Cron-Trigger: Poller-Integration verifizieren/aktivieren

### video / meetings
- Breakout-Räume (LiveKit kann es technisch)
- Recording-Download/List-UI-Endpoint exponieren
- Meeting-Recurrence-Logik

## ⚪ Später / Post-Launch
- buchhaltung (Brücke): automatische Kontierung (Kontenplan SKR03/04), EÜR-Endpoint, Steuerberater-Rolle (read-only)
- security: SSO (SAML/OAuth2/OIDC), Federation (LDAP/AD), WebAuthn/Passkeys
- admin/dashboard: zentraler modul-übergreifender Activity-Feed-Aggregator

---

# Welle 3 — Branchen-Module (Post-Launch / Solar-Pilot)

## ⭐ Cross-cutting (einmal bauen → viele Module profitieren)
- **S3/MinIO-Foto-Upload-Service**: generischer Upload-Endpoint für Foto-Anhänge — gebraucht von fuhrpark (Schaden), inventar (Bewegung), rapporte (Doku), vermietung (Protokoll), chat, profil (Avatar). Aktuell überall Mock.
- **Signatur-Persistenz**: `signature`-Feld/-Endpoint — gebraucht von rapporte, vermietung, vertraege. SignatureCanvas existiert im FE, BE nimmt es nirgends an.
- **Einkauf↔Inventar-Sync**: Wareneingang (`einkauf.ReceiveGoods`) muss `inventar.RecordMovement` triggern (Code-Kommentar „Sprint-3-Item").

## 🟠 Branchen-BE-Lücken (für Solar-Pilot ab Nov relevant)
### fuhrpark
- Führerscheinkontrolle-Modell (`LicenseCheck`: Fahrer, Ablauf, letzter Check) + Route
- Fahrtenbuch-Modell (`LogbookEntry`, finanzamtkonform) + Route `/fuhrpark/logbook` + PDF-Export
- Fahrzeugbuchung/Pool (`VehicleBooking` + Conflict-Check)
- `FuelRecord`-Modell (Tankprotokoll) + Tankkartenverwaltung
- GPS/Telematik: Provider-Integration + Webhook-Endpoint

### inventar
- `batch_number`/`serial_numbers` im Item-Modell (Chargen/Seriennummern)
- Inventur-Modell (`InventurSession`: open/counting/review/completed, Soll/Ist) + Routes
- Kommissionierung/Picklisten-Modell

### vermietung
- Strukturiertes Checklist-Format für Zustandsprotokolle (BE nur notes+photo_urls)
- `signature_url` im Inspection-Modell
- Online-Buchungsportal (öffentl. Endpoint/Embed)
- Tarif-Erweiterung (Wochensatz, Staffeln, Saison)

### einkauf
- `SupplierRating`-Modell (quality/delivery/price)
- `FrameworkContract`-Modell (Rahmenverträge) + Katalogartikel
- 2-stufiger Bestellfreigabe-Workflow (`approved_by`, `/approve`-Endpoint)
- Automatische Bestellvorschläge (Inventar-MinQty → PO)

### produktion (Brücke — MRP-Tiefe bewusst begrenzen)
- BOM-Modell (`bom` + `bom_items`) + CRUD
- `progress`/`work_steps`/`scrap`-Felder auf Order
- Maschinen-Stammdaten-Register (aktuell nur String-ID)
- Material-Verfügbarkeit: Inventar-Abgleich (FE nutzt Fake-Hash)
- QualityCheck-Modell (FE-UI existiert)
- Kalkulation (Soll/Ist-Kosten)

### schichten (Self-Service Pilot-kritisch)
- `shift_swap_requests`-Modell + approve/reject (FE-Tab da)
- Availability-Tabelle (employee × weekday) + Qualifikations-Tabelle
- Echter regelbasierter Auto-Planer (ApplyTemplate ist nur Datums-Kopie)
- `is_minor`-Flag + JArbSchG-Compliance

### rapporte (Solar-Außendienst)
- Signatur-Persistenz (s. cross-cutting)
- Aufmaß-Modell (`Measurement`/`MeasurementPosition`) — FE komplett, BE null
- `weather`-Feld auf WorkReport (Bautagesbericht)
- ReportLine: Material-vs-Leistung-Unterscheidung schärfen

## 🟠 Mobile/Offline (Solar-Pilot)
- PWA/Mobile-Zugangsweg (App ist Electron-Desktop) — rapporte + schichten brauchen Außendienst-Zugang
- `rapporte-client.ts` + `schichten-client` auf offline-queue-fähigen Basis-`client.ts` umstellen (offline-queue.ts existiert)

## 🟡 Leads (CRM, Phase 4 — 2026-06-06)
- **Architektur-Entscheidung:** Lead als **Kontakt-Lifecycle-Status** modellieren, nicht als separate Tabelle. Empfehlung: `contacts.lifecycle_stage` (`lead` → `qualified` → `customer`). Frontend baut die Inbox als gefilterte Sicht.
- Lead-Metadaten am Kontakt: `lead_source` (manual/csv/dialer), `lead_score` (0–100, auto), `lead_temperature_override` (hot/warm/cold, sticky), `lead_status` (new/contacted/qualified/disqualified).
- Endpoints: `GET /api/v1/leads` (= Kontakte mit lifecycle=lead), Status-/Temperatur-Patch, Convert (lead → contact + opt. company + opt. deal). FE nutzt aktuell In-Memory-Mock (`api/hooks/useLeads.ts`) — swapbar.
- Scoring-Regel serverseitig spiegeln (FE-Logik in `computeLeadScore`): Quelle-Basis + Vollständigkeits-Boni.
- Quelle „Dialer": Dialer-Outcomes mit Rückruf-Wunsch sollten automatisch Leads erzeugen (Verknüpfung Dialer↔Leads).

---

## 🟠 kommunikation (Team-Chat + Posteingang, vereintes Modul, Stand 2026-06-08)

> Beim Scharfschalten: echte gRPC-Endpoints existieren überwiegend, aber im **Demo-Mode (MSW)** fehlen Handler — FE-seitig nachgebaut wo nötig. Echtes Backend / Verkabelung durch Luke:

- **🔴 Chat-Reactions NICHT verdrahtet (Luke):** chat-proto deklariert `ToggleReaction/ListReactions/GetReactionSummary`, aber der **chat-Service implementiert sie nicht** (kein `reaction`-Ordner in `internal/chat/`) und das **Gateway exponiert keine chat-Reactions-Route** (route_chat.go hat keine `/reactions`). Einzige echte Impl = `internal/work/reaction` (video/work). `MessageInfo` hat zudem kein `reactions`-Feld (weder proto noch OpenAPI). → To-do Luke: (a) Reactions am chat-Service implementieren ODER work-reaction-Service mitnutzen, (b) Gateway-Route z.B. `POST/GET /api/v1/messages/{id}/reactions`, (c) `reactions` in `MessageInfo` + OpenAPI aufnehmen. **FE-Status (Update 2026-06-11, `507487b9`):** ✅ Migration abgeschlossen — MessageBubble/MessageList nutzen die echten Hooks (`useToggleReaction` + `useReactionSummary`-Batch), der Demo-Store `stores/chatReactions.ts` ist gelöscht; MSW-Handler bedienen die Reaction-Endpoints in-memory.
- **Chat-Volltextsuche:** `GET /api/v1/chat/search?q=&channel_id=` (SearchChat-RPC) existiert im Backend; FE-Hook `useChatSearch` + Demo-Handler (durchsucht Mock-Messages) jetzt gebaut. Realer Index/Ranking + File-Treffer = Luke.
- **✅ File-Upload im Chat (FE verdrahtet):** `POST /api/v1/files/upload` (multipart, nimmt `channel_id` + optional `message_id`) + `GET /{id}/files` + download/thumbnail existieren, `MessageInfo.files` (FileInfo) ist in proto+OpenAPI. FE lädt jetzt beim Senden via `useChatFileUpload` mit `message_id` hoch und rendert `message.files`. **Luke (optional/nice):** echter Storage/Virus-Scan/Thumbnail-Gen statt Stub; `GetFileThumbnailURL` für Bild-Previews verdrahten.
- **Gruppen-DMs, Pin/Lesezeichen, Channel-Notification-Settings:** im chat-proto nicht vorhanden — Neubau.
- **✅ Mentions-Inbox (FE verdrahtet):** `GET /api/v1/messages/mentions` (`GetUserMentions`) + `UserMentionsResponse` in OpenAPI. FE-Hook `useUserMentions` + `MentionsPanel` + Demo-Handler gebaut. Real out-of-the-box.
- **✅ Posteingang (Inbox) scharfgeschaltet (Phase 4, 2026-06-08, FE):** Snooze/Claim/Assign verdrahtet (SnoozePopover/Assignee-Picker via `useEmployees`), Bulk-Toolbar (BulkMarkRead/Archive), Team-Postfächer + Routing-Regeln in `KommunikationSettingsPanel` (FÜR-ALLE, tenant-scoped). **Demo-Handler (MSW) ergänzt** für `snooze`/`unsnooze`/`claim` + Team-Inbox-CRUD+Members + Routing-Rules-CRUD+test (zustandsbehaftet). Echte gRPC-Endpoints existieren — Luke verdrahtet Gateway/Service falls noch offen.
  - **Backend-Neubau nötig (FE läuft mock-first als verdrahtungs-bereites Overlay):**
    - **Inbox-Status** (offen/wartend/gelöst/geschlossen): kein Feld in `InboxMessage`/proto. FE-Overlay `stores/inboxStatus.ts` (persistiert). → `status`-Feld + Filter im `ListMessagesRequest` + Set-Status-RPC.
    - **Threading** (mehrere Msg/Conversation): kein Thread-RPC. FE synthetisiert Seed + persistiert Replies/Notizen in `stores/inboxThread.ts`. → `ListThreadMessages`/Conversation-Modell.
    - **Tags-CRUD:** `repeated string tags` existiert, aber kein Add/Remove-RPC. FE-Overlay `stores/inboxTags.ts`. → `AddTag`/`RemoveTag`-RPC.
    - **Forward:** kein RPC. FE = `ForwardDialog` (Empfänger+Notiz) mit Erfolgs-Toast. → `ForwardMessage`-RPC.
    - **SLA-Tracking:** noch nicht modelliert.
- **Audio/Video aus Chat + Bots/Webhooks/Slash-Commands:** keine RPCs — kompletter Neubau.
- **✅ Synergie + Moduleinstellungen (Phase 5, 2026-06-08, FE):**
  - **Interne Notizen + @Mention im Kunden-Thread:** verdrahtet — `InternalNoteComposer`/`ReplyComposer` nutzen jetzt `MentionTextarea` (wraps chat-`MentionAutocomplete`); interne Notizen landen als `direction:'internal'` im Thread-Overlay (`stores/inboxThread.ts`). **Backend nötig:** interne-Notiz-Persistenz + echtes @Mention-Notify. ✅ Demo zeigt jetzt Namen (2026-06-11, `7a367047`) — Mock-Handler auf camelCase angeglichen. ⚠ Echter API-Swap braucht noch Shape-Adapter (siehe §team FE↔BE-Shape-Mismatch).
  - **Collision-Hinweis „X bearbeitet gerade":** Mock-first deterministisch (`lib/inbox-collision.ts`). **Backend nötig:** Live-Presence-pro-Conversation (Viewers-Event).
  - **Call-Bridge:** Audio/Video-Buttons im Thread-Header → `useMeetingsStore().startCall` (gleicher Pfad wie Team/Kontakte). Echte LiveKit-Calls aus dem Posteingang heraus brauchen `createCall`-Verkabelung + ggf. Kunde→user_id-Mapping.
  - **Slash-Commands + Webhooks:** Mock-Shell (`SlashCommandPalette` /giphy /umfrage /erinnerung; `WebhookConfig`). **Backend nötig:** Bot/Command-Runtime + Webhook-CRUD/-Delivery.
  - **Canned Responses:** CRUD-UI (`CannedResponseManager`) auf Store-Backing (`updateCannedResponse` ergänzt). **Backend nötig:** Canned-Response-CRUD-Endpoints.
  - **Channels-Connect (`ChannelSettingsDialog`):** Mock-Connect (Toast). **Backend nötig:** echte OAuth/Connect-Flows E-Mail/WhatsApp/Widget.
  - **Per-Channel-Mute / eigener Status:** FE-Prefs (`kommunikationPrefs.mutedChannels`, `presence.myStatus`). **Backend nötig:** serverseitige Notification-Routing-Beachtung.

## 🟠 work (Projekte/Aufgaben, Stand 2026-06-08)

> **P1 (Demo-Mode):** MSW-Handler in `mocks/handlers/work.ts` komplett (zustandsbehaftet) — Demo läuft. Kein Backend-Bedarf, das ist nur Mock.
> **P2 (WorkSettingsPanel, settings-komplett):** Persönliche Prefs = lokal (`stores/workPrefs`, kein Backend nötig). Tenant-Settings laufen **mock-first** (`stores/workSettings`, lokal persistiert) — brauchen echtes Backend:

- ✅ **Label-Taxonomie + Task-Labels — BACKEND ERLEDIGT 2026-06-11 (`2b8447b6`, Migrationen 000145+000147):** `work_labels`+`task_labels` (RLS), Label-CRUD `/api/v1/work/labels`, `PUT /tasks/{id}/labels`, `label_ids` im TaskProto, Permission-Seeds `work_labels:*`. ✅ Follow-ups erledigt 2026-06-11 (`d028b8ea`): `label_ids` batch-geladen in Get/ListTasks (GetLabelsByTaskIDs, 1 Query), `filter_label_ids` als tenant-gescopte EXISTS-Clause im task-Repo; zusätzlich CreateTask/ListTasks auf `middleware.GetTenantID(ctx)` (`772483fd`). Offen nur noch: FE-Wiring (Chip-UI/Filter).
- ✅ **Custom-Field-Definitionen (Task) — BACKEND ERLEDIGT 2026-06-11 (`2b8447b6`, Migrationen 000146+000147):** `work_custom_field_definitions` (tenant-scoped, RLS, 9 Feldtypen) + CRUD `/api/v1/work/custom-fields`, Seeds `work_custom_fields:*`. Follow-up FE-Adapter: `field_type`→`type`, `position`→`sortOrder`, Store kennt `dropdown` statt `select`.
- **🟡 Default-Status-Set:** `stores/workSettings.defaultStatuses` mock-first — das Status-Set, mit dem neue Projekte starten. Aktuell seedet der MSW-Create-Handler ein festes Set. → tenant-Setting `default_project_statuses` + Anwendung in `createProject`.
- **🟡 Projekt-Vorlagen löschen:** Liste (`templates_only`) + Umbenennen (PUT name) + „aus Vorlage erstellen" (`from-template`) laufen echt. **Löschen fehlt** (kein `DELETE /projects/{id}`-Endpoint im Client/Spec). → Delete-Project-Endpoint oder Template-Archivierung.
- **🟡 Zeit-Regeln (billable-Default, Stundensatz):** `stores/workSettings.billableByDefault`/`defaultHourlyRate` mock-first. → tenant-Setting; Anwendung beim Anlegen von Time-Entries + Stunden→Rechnung (P4).
- **🟢 P5 (Kalender-Sicht):** KEIN neuer Backend-Bedarf. `WorkCalendarView` bucketet `useTasks({project_id})` nach `due_date`; Drag = Fälligkeit ändern via bestehendes `PUT /tasks/{id}` (`due_date`). Nur ein latentes Komfort-Feld offen: ein `due_date`-Bereichsfilter im `listTasks`-Query (`due_from`/`due_to`) würde bei sehr vielen Tasks das clientseitige Bucketing entlasten — heute irrelevant (page_size 500).

# Hinweis zur Arbeitsweise (Claude = FE, Luke = BE)
Die meisten 🟡-Punkte (FE-Page von Mock-Store auf fertige TanStack-Hooks umstellen) brauchen KEIN Backend von Luke — die Hooks + Endpoints existieren bereits. Luke-Bedarf = nur die 🔴- und cross-cutting-Punkte oben.

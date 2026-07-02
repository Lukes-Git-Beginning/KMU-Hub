---
tags: [datenbank, schema, migrations, ai-first, tenant-isolation, rls]
updated: 2026-06-28
---
# Datenbank

## Überblick
- PostgreSQL 16 mit `pgvector/pgvector:pg16`-Image + Redis 7 (nur Cache, KEIN Dual-Write)
- Änderungen NUR via golang-migrate (`make migrate-create name=xxx`)
- **182 Migrations-Dateien** in `backend/migrations/` (Kopf **000241** auf main (avatar/inventar 2026-06-28) [Welle F: 222 audit_log-append-only + 223 validate 29 tenant-FKs; Welle G: 224 RBAC manager/member operativ-Seed; **Meeting-Parität 2026-06-23: 225 meeting_chat, 226 meeting_cohosts+meetings.locked**]; ⚠ Partitionierungs-Migration in **PR #13** muss jetzt auf **242+** umnummeriert werden (225–241 belegt: Meeting-Parität 225–235, Darien-Track 236–238, avatar/inventar 239–241), da 219–226 auf main belegt = Kollision; Nummern-Luecken durch Reverts/Renumber) (076–116 siehe Sprint-2/3-Liste in der Vorversion dieser Note; **Sprint 4: 117 users_tenant_default_and_fk · 118 rls_foundation · 119 child_tables_tenant_id_backfill · 120–124 RLS-Wellen 2+3 · 125–127 RLS-Welle 4 · 128 fix_hr_document_policy_sysctx · 129 seed_missing_module_permissions** [35 nie geseedete Modul-Permissions admin-only, siehe [[security]] RBAC] **· 130 dialer outcome_id-Partial-Indizes + ~10 FK ON DELETE (R2-P1.8/.9) · 131 seed meetings:write fuer admin+manager+member (R2-P1.5)** — 131er-Review-Gate-Fund: Guard auf Bestands-Funktion braucht Seeds fuer ALLE bisher berechtigten Rollen, nicht nur admin · **132 add_finance_line_tables · 133 backfill_finance_line_tables** = ADR-0007 finance line_items relational, siehe Abschnitt unten · **214 seed_finance_manager_permissions** [manager→finance:read+write, kein delete/admin] **· 215 finance_payments_idempotency_key** [F5 DB-Dedup: idempotency_key + partieller Unique-Index `(tenant_id, idempotency_key) WHERE NOT NULL`])
- **Prod-Stand seit 2026-06-05 Abend:** Migration-Head **`131`** (Code `564f238b`, COSMI_ENV=production scharf — siehe [[deployment]]). **2026-06-08:** Migr. **132/133** (ADR-0007 finance-lines, Commit `3e4c9055`) gepusht → Head **`133`**. **2026-06-09:** Chain PILOT — Migr. **134–136** (password-reset-tokens + booking-pages + RBAC-Seed, Commits `1548a067`+`b4af5739`+`2316d6cd`) → Head **`136`** (auf Server verifiziert 2026-06-10, clean). **2026-06-10:** Marathon-Tag-2-Welle-1 — Migr. **137/138** (advisory_protocols + settings-foundation, Commits `6b211222`+`360f92e6`) gepusht → Head wird **`138`**; nur statisch geprüft (keine lokale Dev-DB), CD-Migrate testet real. **2026-06-10/11:** Welle 2 + Backend-Sessions — Migr. **139–147** (GoBD/E-Rechnung/Kontakt-FK, RLS-Nachzug, Signaturen, Files-Seeds, Work-Labels/Custom-Fields) → Head **`147`**, Prod auf 147 verifiziert (Session 2 `cc5c1cbd`). 000148 reserviert (contract_events-Audit). **2026-06-18 (live gemessen):** Repo-Kopf **000213** (182 `.up.sql`-Dateien, Luecken durch Reverts/Renumber), Production-Kopf **209** (4 Migr. zurueck = CD-Lag zum Mess-Zeitpunkt) — die FE↔Backend-Wiring-Wellen (helpdesk/schichten/hr/wiki/rapporte/inventar/fuhrpark/einkauf/produktion) brachten Migr. 148–213. Volume: `docker_pgdata` (nicht `docker_postgres-data`). psql-User in Production ist **`kmuhub`**, nicht `postgres` — siehe [[troubleshooting]]. **2026-06-19:** Finance/biz-Härtung Wave 3 — Migr. **214/215** (manager-finance-Permissions + payment idempotency_key) → Repo-Kopf **215**. **2026-06-20 (Pre-Launch-Wellen):** Migr. **216** add_currency_to_finance (B6) + **217** drop_line_items_jsonb (Dual-Write entfernt, alle Reads relational — ADR-0007-Abschluss) → main-Kopf **217**. **2026-06-20 (Rigorosum R3 Welle 7a):** Migr. **218** RLS auf `advisory_protocols`+`tenant_settings`/`tenant_module_leads`/`user_settings` (`60e3a250`) → main-Kopf **218**. ⚠ **Kollision:** die Partitionierungs-Migration `partition_ephemeral_log_tables` (events/dialer_call_events/automation_executions deklarativ partitioniert + pg_cron-90d-Retention; audit_log AUSGESCHLOSSEN wg. §257/§147 AO + Hash-Kette) in **PR #13** (`feat/partition-ephemeral-logs`) heisst ebenfalls 218 → muss vor Merge auf **242+** umnummeriert werden (225–241 seit 2026-06-23/28 belegt; NICHT auf main — braucht pg_cron-Image + Maintenance-Window, sonst CD-Auto-Apply-Hazard). **2026-06-21 (Welle E):** Migr. **219** datev_consultant_client_numbers (7c) + **220** user_module_grants + **221** seed_module_grants_permissions → main-Kopf **221**. **2026-06-21 (Welle F, Data-Integrity + P0-1-Rest):** Migr. **222** audit_log_append_only (`BEFORE UPDATE OR DELETE`-Row-Trigger → DB-seitige GoBD/§257/§147-AO-Immutability; INSERT erlaubt, Partition-DROP=DDL unberührt; einziger Test der audit_log mutierte auf Append-Only-Assert umgestellt) + **223** validate_tenant_fks (29 NOT-VALID `tenant_id`-FKs aus 114/115/125/126 via Orphan-Guard-`DO`-Block validiert; golang-migrate atomar → Orphan-Fail rollt zurück, Kopf bleibt 222; forward-only Down=No-op) → main-Kopf **223** (CI grün inkl. `-race`/OpenAPI; CD-Apply gegen Prod beobachten, **bestätigt grün** — keine Orphans, Smoke 24/24). **2026-06-21 (Welle G, Permission-Seed-Nachzug):** Migr. **224** seed_module_manager_member_permissions — die 5 Module booking-pages/schichten/hr:time_*/inventar/einkauf waren komplett admin-only (403 für manager+member, kein operativer Zugriff); manager→voll operativ (alle Actions über 19 Resources), member→Self-Service (schichten:swap read+create, booking-pages:read, inventar:*:read); reiner `role_permissions`-Seed (Permissions existierten, nur Grants fehlten), Resource/Name-Strings 1:1 gegen Seeds verifiziert → main-Kopf **224**. **2026-06-23 (Meeting-Parität W2/W3):** Migr. **225** create_meeting_chat (`meeting_chat` tenant_id+RLS via `enable_tenant_rls`, FK meetings ON DELETE CASCADE, append-only In-Call-Chat) + **226** meeting_cohosts_and_lock (`meeting_cohosts` tenant_id+RLS + `meetings.locked BOOLEAN`-Spalte für server-autoritative Host-Controls) → main-Kopf **226**, **Prod 226 clean verifiziert** (`psql -U kmuhub -d kmuhub -tAc 'SELECT version,dirty FROM schema_migrations'` → `226|f`, beide Tabellen existieren). **2026-06-23/28 (Meeting-Parität 7A–7C + 6A Breakout):** Migr. **227** meeting_crm_link (contact_id/deal_id nullable FK) · **228** meeting_ai_summary · 229–234 (notif-pin/wiki-hr-RLS/task-prio-normalize/helpdesk-hours/retention_policies/wave2-perms) · **235** meeting_breakout_rooms + meeting_breakout_assignments (beide tenant_id NOT NULL + RLS via `enable_tenant_rls`, FK meetings ON DELETE CASCADE, tenant via Parent-Meeting-Subquery; UNIQUE(meeting_id,user_id) für Upsert-Assignment) → main-Kopf **235**, **Prod 235 clean** (CD 2026-06-28 grün, `235|f`, beide Breakout-Tabellen verifiziert). **2026-06-28 (Darien-Backend + avatar/inventar):** Migr. **236** helpdesk_ticket_fields (denormalize assignee/requester_name + description/category + per-Tenant `helpdesk_ticket_counters`-ticket_number) · **237** inbox_thread_messages · **238** inbox_canned_responses (beide reuse `inbox:read/write` → kein Permission-Seed) · **239** add_users_avatar_url (users system-global, custom RLS-Policy 120 → keine neue Policy; `TEXT NOT NULL DEFAULT ''`, speichert presign-object-key) · **240** create_inventory_item_attachments (tenant_id NOT NULL + RLS via `enable_tenant_rls`, FK inventory_items ON DELETE CASCADE, spiegelt fuhrpark vehicle_documents/194) · **241** seed_inventar_attachment_permissions (`inventar:attachment:read/write`→admin, neuer RequirePermission-Guard sonst 403) → main-Kopf **241**, **Prod 241 clean** (CD 2026-06-28 auf `f825b29c` grün).

## RLS-Foundation (Migration 118, Sprint 4 Welle 1a)

Sprint 4 Welle 1 hat die PostgreSQL-Row-Level-Security-Foundation eingezogen. **Noch keine Tabelle hat aktive Policy** — das ist Welle-2-Scope. Migration 118 stellt nur die Helpers bereit:

- `current_tenant_id() RETURNS uuid` — STABLE, liest `app.tenant_id`-GUC, returnt NULL bei leerem oder ungültigem Setting
- `current_user_id() RETURNS uuid` — analog für `app.user_id`
- `current_app_role() RETURNS text` — liest `app.role`
- `is_system_context() RETURNS boolean` — true wenn `app.role = 'system'` (Worker-Bypass)
- `enable_tenant_rls(table_name text)` PROCEDURE — aktiviert RLS auf einer Tabelle mit Standard-Policy: `USING (tenant_id = current_tenant_id() OR is_system_context())` plus identische `WITH CHECK`. FORCE ROW LEVEL SECURITY damit auch der Owner gefiltert wird.
- `enable_tenant_rls_via_join(child, parent, fk, parent_pk)` PROCEDURE — Fallback für Child-Tabellen die tenant transitiv via JOIN ableiten. Kein Aufruf in Welle 1, eigene Spalten bevorzugt.
- Database-Defaults: `ALTER DATABASE kmuhub SET app.tenant_id = ''` (analog für user_id, role) damit `set_config(..., true)` LOCAL in Tx greift.

## RLS-Wiring auf App-Ebene

- **Pool-Hooks** in `backend/internal/database/postgres.go` `AfterRelease` resetten die GUCs beim Release der Connection — Defence-in-Depth gegen non-Tx-Pfade.
- **Tx-Wrapper** `database.BeginRLSTx(ctx, pool)` (neu in `backend/internal/database/transaction.go`): startet Tx + setzt `app.tenant_id/user_id/role` LOCAL via `set_config(..., true)`. Tenant aus `middleware.GetTenantID(ctx)`, im System-Context (`database.WithSystemContext(ctx)`) wird `app.role='system'` gesetzt → Policy bypassed.
- **Worker** wrappen ihren Entry-Context: 10 Sites in `berichte/scheduler`, `automation/trigger`, `fuhrpark`, `email/sync` (worker+engine), `vertraege`, `formulare`, `biz/lexware+bexio`, `inbox/message.StartSnoozeWorker`. Pattern: `ctx = database.WithSystemContext(ctx)` als erstes Statement der Run/Start-Methode.
- **gRPC-Tenant-Trust:** Gateway-Outbound-Interceptor seit Welle 0.6 global aktiv (alle Service-Verbindungen). Welle 1d hat den Inbound-Interceptor in 4 Pilot-0-Services (auth, crm, dialer, work) wired (chat hatte ihn schon seit Welle 0.6).

## Migration 119 — Child-Tabellen-Backfill (Welle 1b)

Vier Child-Tabellen, die ihre Tenant-Zugehörigkeit bisher nur transitiv via FK ableiteten, bekommen eigene `tenant_id`-Spalten + Backfill via JOIN + NOT NULL + FK NOT VALID + VALIDATE + Index:

- `dialer_campaign_contacts` (campaign_id → dialer_campaigns.tenant_id)
- `dialer_agent_status_log` (campaign_id → dialer_campaigns.tenant_id, fallback user_id → users.tenant_id)
- `dialer_call_events` (**dialer_call_session_id** → dialer_call_sessions.tenant_id; nicht `session_id` — echter Spaltenname, war Production-Bug in Anlauf 1)
- `recording_consents` (recording_id → recordings.tenant_id)

Plus `consent_records.tenant_id` von NULLABLE auf NOT NULL promotet (Spalte war seit Migration 111 da, aber nullable).

Backfill-Asserts (RAISE EXCEPTION wenn nach UPDATE noch NULL-Rows existieren) decken Orphan-FK-Rows auf — wenn Migration crasht, ist die DB in keinem inkonsistenten State.
- Index-Konvention: `idx_{table}_{column}`
- **AI-First-Foundation** seit Migration 071 (siehe Abschnitt unten)
- **Seed-Idempotenz:** Migration `000079` (berichte) wurde in `980eba3` um `ON CONFLICT DO NOTHING` erweitert, damit ein Re-Run keine Duplikate erzeugt. Gleiches Muster fuer alle zukuenftigen Seed-Migrations anwenden.

## Tabellen nach Domain

### Core Identity
- `users` — UUID PK, **tenant_id UUID NOT NULL DEFAULT '00000000-...-000000000001'** (Migration 000104, Welle 2D, mit `idx_users_tenant`), email (unique), password_hash (bcrypt), first/last_name, is_active. `auth/postgres_repository.go` SELECTed das Feld jetzt — vorher leer → JWT `tid`-Claim immer leer.
- `roles` — admin, manager, member
- `permissions` — resource:action Pattern (z.B. contacts:write)
- `role_permissions`, `user_roles` — RBAC-Zuordnung
- `refresh_tokens` — Token-Management mit Expiry

### CRM
- `contacts` — first/last_name, email (unique), phone, company_id (FK), position, created_by
- `companies` — name, domain, industry, employee_count, address
- `deals` — name, value (DECIMAL 15,2), currency (EUR default), stage_id, contact_id, company_id, expected_close_date
- `pipeline_stages` — Sales-Pipeline pro Organisation
- `activities` — Polymorphic: calls, meetings, notes, emails, tasks
- `contact_tags`, `company_tags`, `deal_tags` — Tagging-System
- `contact_custom_field_values`, `company_custom_field_values` — JSONB Custom Fields

### Chat
- `channels` — name, is_private, is_dm, is_archived, channel_role ENUM (owner/admin/member)
- `channel_memberships` — user_id, role, last_read_at
- `messages` — text, created_by, thread_id (parent), status
- `message_mentions` — @-Benachrichtigungen
- `reactions` — emoji, user_id, message_id
- `chat_files` — file_url, size, mime_type

### Work
- `projects` — name, project_key (unique), is_template, next_task_number
- `project_members` — role (owner/member/viewer)
- `project_statuses` — Custom pro Projekt, is_closed Flag
- `tasks` — name, description, project_id, assigned_to, status_id, priority, due_date
- `task_dependencies` — DAG für Task-Reihenfolge
- `task_comments`, `task_files`, `task_links` — Diskussion, Dateien, Entity-Links
- `time_entries` — task_id, duration_seconds, user_id, start_time

### Calendar
- `calendars` — type (personal/shared/resource), timezone (Europe/Berlin default), color, is_default
- `calendar_members` — permission (view/edit/admin), color_override
- `events` — start, end, summary, location, all_day, recurrence (RRULE), attendees
- `resources` — Raeume/Equipment, capacity, booking constraints
- `resource_bookings` — Konflikt-Erkennung
- `holidays` — DACH-Feiertage

### Finance
- `company_settings` — Steuernummer, USt-ID, Handelsregister, IBAN, BIC, is_kleinunternehmer
- `finance_number_sequences` — Nummernkreise (RE-2026-001, AN-2026-001) pro Geschaeftsjahr
- `finance_quotes` — Status (draft/sent/accepted/rejected/expired), line_items (JSONB), tax_breakdown
- `finance_invoices` — invoice_date, delivery_date, due_date, payment_terms, company_snapshot (JSONB)
- `finance_payments` — payment_date, amount, method, invoice_id, **idempotency_key** (F5, Migr.215: partieller Unique-Index gegen Doppelzahlung; NULL bei keylosen/Bestands-Zahlungen)
- `finance_credit_notes` — Gutschriften (Reverse Invoices)
- **Positionen seit ADR-0007 (Migr. 132) relational** — `finance_invoice_lines`/`finance_quote_lines`/`finance_credit_note_lines`; `line_items` JSONB bleibt vorerst synchron (Drop Sprint 5). Detail-Abschnitt unten.

### Email/Inbox
- `email_accounts` — IMAP/SMTP/OAuth Konfiguration
- `email_messages` — from, to, subject, body, folder
- `inbox_messages` — Unified Inbox, status (unread/read/starred/archived/snoozed), assigned_to
- `inbox_routing_rules` — Pattern-basierte Zuweisung
- `inbox_teams` — Team-Mailboxen mit Permissions

### Documents
- `documents` — file_id, mime_type, locked_by, locked_until (WOPI)
- `document_versions` — version_number, file_size, etag
- `wopi_locks` — Document-Lock-Management, TTL

### Automation
- `automation_workflows` — trigger (event), condition (JSON), action (JSON), enabled
- `automation_execution_logs` — status, error_message, duration_ms, retries

### Security & Audit
- `security_audit_logs` — action, resource, user_id, result, timestamp
- `login_attempts` — Failed-Login-Tracking für 2FA-Enforcement
- `two_factor_settings` — method, enabled_at, grace_period_until
- `security_tokens` — Sessions, App-Tokens
- `app_passwords` — CalDAV/OAuth App-Passwoerter

### CRM Erweiterungen (Migration 059)
- `pg_trgm` Extension aktiviert (Fuzzy-Matching)
- `contacts.merged_into_id UUID` — Soft-Merge Tracking
- `companies.merged_into_id UUID` — Soft-Merge Tracking
- GIN-Trigram-Index auf `contacts` (first+last name) und `companies` (name)

### Consent Management (Migrations 060, 075)
- `consent_records` — contact_id (FK), consent_type ENUM (marketing_email, marketing_phone, profiling, newsletter, data_processing, data_sharing), granted BOOL, legal_basis ENUM (consent, legitimate_interest, contract, legal_obligation), source, ip_address INET, granted_at, revoked_at
- `gdpr_deletion_requests` — contact_id, requested_by, reason, status (pending/completed), completed_at
- **Migration 075 (2026-04-18, R1-P0.1):** `consent_records.contact_id` FK `ON DELETE CASCADE` → `ON DELETE SET NULL`. Consent-Historie ueberlebt damit GDPR-Loeschung des Kontakts. Regressions-Test: `backend/migrations/migration_000075_test.go`.
- **Aktiv-Check (Consent-Asserter):** `WHERE contact_id=$1 AND consent_type=$2 AND granted=true AND revoked_at IS NULL` — siehe [[security]] Consent Enforcement

### Contacts Owner Index (Migration 062)
- `idx_contacts_owner_id` auf `contacts(owner_id)` — fehlte fuer `ListWithVisibility` Filter

### Finance Erweiterungen (Migration 061)
- `finance_invoices.zugferd_profile VARCHAR(20)` — NULL = plain PDF, sonst 'MINIMUM' / 'BASIC_WL' / 'EN16931'
- `finance_invoices.time_tracking_source JSONB` — Audit-Trail fuer Zeiterfassung→Rechnung

### Finance Line-Items relational (Migrations 132/133, ADR-0007, Sprint 4 S4.FI, 2026-06-08)
- **Migration 132** legt `finance_invoice_lines` / `finance_quote_lines` / `finance_credit_note_lines` an: PK `gen_random_uuid()`, FK auf Parent `ON DELETE CASCADE`, denormalisiertes `tenant_id UUID NOT NULL`, `position/description/quantity/unit_price/tax_rate/line_total`, CHECKs (`quantity>0`, `unit_price>=0`, `tax_rate>=0 AND <=100` — **DACH-sicher**, nicht DE-only 0/7/19, akzeptiert AT/CH-Sätze; `position>=1`). RLS via `enable_tenant_rls()` (Policy `tenant_isolation`). Plus `finance_invoices.locked_at TIMESTAMPTZ` + `locked_by UUID` — ersetzen den `snapshot_data`-JSONB-Lock-Hack (GoBD, `service_gobd.go`).
- **Migration 133** = idempotenter Backfill (NOT-EXISTS-Guard, Position aus Array-Ordinalität, fehlendes `line_total` = `quantity*unit_price`) + Lock-Migration aus `snapshot_data`. End-to-end verifiziert (up/down/up + Idempotenz).
- **Sauberer Cutover (kein Dual-Write/Feature-Flag):** invoice/quote/creditnote-Repos schreiben+lesen relational in atomarer Tx (`pool.Begin`, Bulk-Load via `WHERE <fk>=ANY($1)`, kein N+1, Decimals verlustfrei als String). `app.tenant_id`-GUC kommt automatisch via `PrepareConn`-Hook → Tx erbt ihn (WITH-CHECK erfüllt). `line_items` JSONB bleibt synchron mitbefüllt (Safety-Net + Dashboard-Direktread) → **gRPC/pdf/datev/dashboard/Frontend unverändert** (Proto war schon `repeated LineItem`, kein API-Bruch). **JSONB-DROP deferred Sprint 5.** Commit `3e4c9055`. Tests via testcontainers-go → [[testing]].
- `hr_employee_profiles.hourly_rate DECIMAL(10,2)` — Stundensatz fuer Rechnungsstellung

### Dialer (Migrations 063-067)
- `dialer_campaigns` — name, status (draft/active/paused/completed/archived), mode (preview/power/predictive), settings JSONB, assigned_agent_ids UUID[], contact_count, completed_count, tenant_id
- `dialer_campaign_contacts` — campaign_id (FK CASCADE), contact_id (FK), position, status (pending/in_progress/completed/skipped/callback), outcome_id, callback_at, call_count, UNIQUE(campaign_id, contact_id)
- `dialer_call_sessions` — campaign_contact_id (FK), call_session_id (FK call_sessions, nullable), agent_id, outcome_id, duration_seconds, notes, next_action, appointment_id
- `dialer_call_events` — Append-only Event-Log: dialer_call_session_id (FK CASCADE), event_type, payload JSONB, occurred_at
- `dialer_call_outcomes` — Tenant-konfigurierbar: label, color, is_positive, is_callback, is_appointment, sort_order, tenant_id
- `dialer_agent_status_log` — Audit-Log fuer Agent-Status (Redis ist Live-Quelle): user_id, campaign_id, status, previous_status, changed_at
- **Kritischer Query:** `GetNextPendingContact` nutzt `FOR UPDATE SKIP LOCKED` fuer Phase-2 Power-Dialer-Parallelitaet

### Tenant Isolation (Migrations 069-070)
- `hr_employee_profiles.tenant_id UUID NOT NULL DEFAULT '00000000-...'` + `idx_hr_employee_profiles_tenant`
- `contacts.tenant_id UUID NOT NULL DEFAULT '00000000-...'` + `idx_contacts_tenant`
- `companies.tenant_id UUID NOT NULL DEFAULT '00000000-...'` + `idx_companies_tenant`
- Alle Contact/Company Repository-Queries filtern nach tenant_id
- Single-Tenant Default: Nil-UUID (`00000000-0000-0000-0000-000000000000`)

### Dialer Permissions (Migration 068)
- `dialer:campaigns` (read/write), `dialer:calls` (write), `dialer:agent` (manage), `dialer:outcomes` (manage)
- Zugewiesen an Rollen: admin (alle), manager (alle), member (campaigns:read, calls:write, agent:manage)

### Berichte Permissions (Migration 080)
- `berichte:reports:read` + `berichte:reports:write` als neue Permissions registriert
- Admin-Rolle bekommt beide Rechte; Manager/Member nach Bedarf erweiterbar
- Gated via `middleware.RequirePermission("berichte:reports", "read"|"write")` in `route_berichte.go`

### Formulare (Migration 081, Sprint 1 S1.3)
- `form_schemas` — id UUID PK, tenant_id UUID NOT NULL, title TEXT, description TEXT, fields JSONB NOT NULL DEFAULT '[]', status ENUM(draft/active/archived) DEFAULT 'draft', is_template BOOL DEFAULT false, is_public BOOL DEFAULT false, page_count INT DEFAULT 1, submission_count INT DEFAULT 0 (denorm., per Trigger), created_by UUID, created_at/updated_at/deleted_at (Soft-Delete GoBD). Index: (tenant_id, deleted_at), (tenant_id, status)
- `form_submissions` — id UUID PK, form_schema_id UUID FK ON DELETE SET NULL, tenant_id UUID NOT NULL, answers JSONB NOT NULL, status ENUM(new/read/archived) DEFAULT 'new', submitted_by TEXT NULL, ip_address INET NULL (DSGVO-Kommentar). Index: (form_schema_id, submitted_at DESC), (tenant_id, status)
- `form_webhooks` — id UUID PK, form_schema_id UUID FK ON DELETE CASCADE, tenant_id UUID NOT NULL, url TEXT NOT NULL, secret TEXT NULL (HMAC-SHA256-Key), events TEXT[] DEFAULT ARRAY['submission.created'], active BOOL DEFAULT true, last_triggered_at/last_status. Index: (form_schema_id), (tenant_id)
- `form_webhook_deliveries` — id UUID PK, webhook_id FK CASCADE, submission_id FK CASCADE, tenant_id UUID NOT NULL, payload JSONB NOT NULL, status ENUM(pending/delivered/failed/dead) DEFAULT 'pending', attempt_count INT, max_attempts INT DEFAULT 5, next_attempt_at TIMESTAMPTZ, last_error/last_response_code, created_at/delivered_at. Partial Index: (next_attempt_at) WHERE status = 'pending' (Worker-Effizienz)
- **Worker:** Exp-Backoff 30s→2min→10min→30min→2h, Dead-Letter nach 5 Versuchen, HMAC-Signatur als `X-Cosmi-Signature: sha256=<hex>` Header
- **Permissions:** `formulare:schemas:{read,write}`, `formulare:submissions:{read,write}`, `formulare:webhooks:write`

### Sprint 2 Welle 2A — Handwerk-Module (Migrations 092–099)

Alle Welle-2A-Tabellen tragen `tenant_id UUID NOT NULL DEFAULT '00000000-...-000000000001'` von Anfang an (Option-B-ready). Pflicht-Permissions-Seed je Modul (resource×action × Admin-Grant) als zweites Migration-Paar.

#### Rapporte (Migration 092 + 093)
- `work_reports` — id UUID PK, tenant_id, customer_id UUID NULL (CRM-Stub), title TEXT, description TEXT, status ENUM(draft/submitted/approved/rejected) DEFAULT 'draft', lat NUMERIC(9,6) NULL, lon NUMERIC(9,6) NULL, submitted_at, approved_at, approved_by UUID NULL, rejection_reason TEXT NULL, deleted_at (Soft-Delete). Indizes: `(tenant_id, status) WHERE deleted_at IS NULL`, `(customer_id) WHERE deleted_at IS NULL`
- `report_lines` — id, report_id FK CASCADE, tenant_id, position INT, kind ENUM(work/material/note), description, quantity NUMERIC, unit TEXT, unit_price NUMERIC. Index `(report_id, position)`
- `report_attachments` — id, line_id FK CASCADE, tenant_id, filename, minio_key TEXT (Pattern `tenants/<tid>/rapporte/<report>/<line>/<filename>`), size_bytes, content_type. Index `(report_id)`
- **Permissions:** `rapporte:report:{read,write}`, `rapporte:line:{read,write}`, `rapporte:attachment:{read,write}`

#### Schichten (Migration 094 + 095)
- `shifts` — id, tenant_id, start_at TIMESTAMPTZ, end_at TIMESTAMPTZ, location TEXT, role TEXT, notes, status ENUM(draft/published) DEFAULT 'draft', published_at NULL, capacity INT. Index `(tenant_id, start_at)`, `(tenant_id, status)`
- `shift_assignments` — id, shift_id FK CASCADE, employee_id UUID (FK-Stub Sprint 3), tenant_id, assigned_at, assigned_by UUID. UNIQUE(shift_id, employee_id). Index `(employee_id, shift_id)`
- `shift_templates` — id, tenant_id, name, weekday_pattern JSONB (z.B. `{"mon": [{"start": "08:00", "end": "17:00"}], ...}`), default_role TEXT
- **ArbZG-Pre-Check** (`service.go::validateRestPeriod`): SQL `SELECT MAX(end_at) FROM shifts WHERE employee_id=$1 AND end_at <= $2 AND tenant_id=$3` → Differenz < 11h → `ErrArbzgViolation`. DST-aware via `time.LoadLocation("Europe/Berlin")`.
- **Permissions:** `schichten:shift:{read,write}`, `schichten:assignment:{read,write}`, `schichten:template:{read,write}`

#### Fuhrpark (Migration 096 + 097)
- `vehicles` — id, tenant_id, license_plate TEXT, make/model TEXT, year INT, vin TEXT, fuel_type ENUM, mileage_km INT, tuev_due_date DATE NULL, **tuev_reminder_sent_at TIMESTAMPTZ NULL** (Cron-Idempotenz), assigned_driver_id UUID NULL (FK-Stub Sprint 3). Partial Index `(tenant_id, tuev_due_date) WHERE tuev_due_date IS NOT NULL`
- `vehicle_services` — id, vehicle_id FK CASCADE, tenant_id, scheduled_at, completed_at NULL, kind ENUM(maintenance/inspection/repair/cleaning), cost_cents INT, status ENUM(scheduled/in_progress/completed/cancelled). Index `(vehicle_id, scheduled_at DESC)`
- `vehicle_damages` — id, vehicle_id FK CASCADE, tenant_id, reported_at, description, severity ENUM, status ENUM(reported/in_repair/resolved), photo_keys TEXT[] (MinIO-Keys), repair_cost_cents INT NULL
- **TÜV-Cron** (`worker.go`): `pg_try_advisory_xact_lock(<hash("fuhrpark_tuev_cron")>)` Leader-Election. Scannt 7d-Fenster + 1d-Fenster, schreibt `tuev_reminder_sent_at` (skip bei Stamp <23h alt). Notification-Delivery noch Stub (Sprint-3-Wiring).
- **Permissions:** `fuhrpark:vehicle:{read,write}`, `fuhrpark:service:{read,write}`, `fuhrpark:damage:{read,write}`

#### Vermietung (Migration 098 + 099)
- `rental_objects` — id, tenant_id, name, kind TEXT, daily_rate_cents INT, deposit_cents INT, available BOOL DEFAULT true, deleted_at (Soft-Delete). Index `(tenant_id, deleted_at)`
- `rentals` — id, object_id FK CASCADE, tenant_id, customer_id UUID NULL (CRM-Stub), start_date TIMESTAMPTZ, end_date TIMESTAMPTZ, status ENUM(reserved/active/completed/cancelled), notes. **GIST-Index:** `idx_rentals_object_dates ON rentals USING GIST (object_id, tstzrange(start_date, end_date))` — Doppelbuchung-Schutz auf DB-Ebene
- `rental_inspections` — id, rental_id FK CASCADE, tenant_id, kind ENUM(handover/return), inspector_id UUID, inspected_at, condition_notes TEXT, photo_keys TEXT[] (MinIO-Keys)
- **Overlap-Pre-Check** (`service.go::CheckAvailability`): `SELECT 1 FROM rentals WHERE object_id=$1 AND tstzrange(start_date, end_date) && tstzrange($2, $3) AND status != 'cancelled'` → bei Treffer `ErrRentalConflict`
- **Permissions:** `vermietung:object:{read,write}`, `vermietung:rental:{read,write}`, `vermietung:inspection:{read,write}`

### Sprint 2 Welle 3 — R2-P0 + Option-B Phase 1 (Migrations 105–107)

- **000105 idempotency_keys:** `CREATE TABLE idempotency_keys (key TEXT PK, tenant_id UUID, user_id UUID, method, path, request_hash TEXT, response_status INT, response_body JSONB, created_at, completed_at, expires_at TIMESTAMPTZ DEFAULT NOW()+'24h')` + Indizes `(tenant_id, user_id, created_at DESC)` + `(expires_at)`. Tenant-scoped from day one (Option-B-ready). Cleanup-Goroutine im Gateway tickt 1h und deletet `expires_at < NOW()`. Backend-Middleware `internal/middleware/idempotency.go` startet im **WarnMode** (loggt fehlende Keys, blockt nicht) bis Frontend-Rollout fertig — HardMode in Welle 4. Whitelist `/auth/login|refresh|2fa` weil Token-Rotation nicht idempotent.
- **000106 tenant_id_retrofit_phase1 (Top-20):** PL/pgSQL-Loop `ALTER TABLE %I ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'` + Per-Table-Index ueber 20 Tabellen: deals, activities, tasks, projects, channels, messages, notifications, time_entries, calendar_events, email_messages, inbox_messages, deal_stage_history, pipeline_stages, saved_filters, custom_field_definitions, automations, document_files, recordings, dialer_call_sessions, audit_log. Plus 9 Composite-Hot-Path-Indizes: `idx_deals_tenant_stage`, `idx_activities_tenant_owner`, `idx_tasks_tenant_status`, `idx_messages_tenant_channel`, `idx_notifications_tenant_user`, `idx_time_entries_tenant_user`, `idx_calendar_events_tenant_start`, `idx_recordings_tenant_status`, `idx_audit_log_tenant_time`. Standard-Sentinel `00...000001` (NICHT die alte Nil-UUID `...000000000000` aus 000069/000070). Top-5 Repos voll gewired (deals/activities/tasks/messages/notifications), Rest hat Spalte+Default und wird in Welle 4 gewired.
- **000107 recordings_pre_consent_audit:** `ALTER TABLE recordings ADD COLUMN pre_recording_consent_at TIMESTAMPTZ NULL, ADD COLUMN initiator_consent_id UUID NULL` + `recording_consents.responded_at TIMESTAMPTZ NULL` + Partial-Index `idx_recordings_pre_consent ON recordings(initiator_consent_id) WHERE initiator_consent_id IS NOT NULL`. Audit-Trail fuer R2-P0.4 Initiator-Pre-Consent-Flow. `recording.Service.StartRecording` lehnt 412 Precondition Failed ab wenn `pre_recording_consent_at IS NULL` — neuer Endpoint `POST /api/v1/video/recordings/{id}/initiator-consent` stempelt das Feld nachdem der Initiator den Pre-Dialog bestaetigt.

### Sprint 2 Welle 3.5 — Bugfix-Sweep (Migration 108)

- **000108 idempotency_keys_composite_pk:** `DROP CONSTRAINT idempotency_keys_pkey; ADD PRIMARY KEY (tenant_id, key)`. P0-Fix: ohne Composite-PK koennte ein Idempotency-Key von Tenant A im HardMode als Response von Tenant B replayed werden (Cross-Tenant-Cache-Leak). Application-Layer macht Conflict-Detection ueber den `request_hash`, aber das DB-PK-Constraint ist die letzte Verteidigungslinie. Down-Migration setzt PK absichtlich nur auf `(key)` zurueck — keine FK auf `tenants(id)`, weil der Sentinel `00000000-...-000000000001` aus 000106 (Option-B-Backfill) sonst eine FK-Verletzung waere. Cleanup-FK landet in einer eigenen Migration nach Pilot-1, wenn Legacy-Rows gepurgt sind.
- **Companion-Aenderungen ohne Migration:** Repository-Tenant-Filter-Sweep auf `deal/activity/task/pipelinestage/chat-message/recording postgres_repository.go` (alle UPDATE/DELETE/GetByID/Search filtern jetzt `WHERE id=$1 AND tenant_id=$2` + `RowsAffected==0`-Sentinel). `pipelinestage.scanStage` liest `tenant_id`-Spalte aus 000106 (vorher Scan-Mismatch nach migrate-up). Migration 000106 hat jetzt eine `down.sql` mit explizitem Doc-Kommentar zur bewussten FK-Abwesenheit.

### Sprint 2 Welle 4B — Option-B Phase 2 + Idempotency HardMode-Bereitschaft (Migrations 109–113)

Top-30+ Tabellen-Retrofit auf den verbleibenden Hot-Path-Tabellen, plus Idempotency Complete()-Composite-PK-Fix mit Performance-Index. Alle ALTERs mit `ADD COLUMN IF NOT EXISTS` (defensiv gegen Re-Run und gegen Tabellen, die in 000106 schon retrofittet wurden).

- **000109 option_b_phase2_calendar_work:** 21 tenant_id-Spalten + Index pro Tabelle: `calendars`, `calendar_members`, `event_categories`, `event_attendees`, `event_exceptions`, `event_reminders`, `user_calendar_preferences`, `meetings`, `meeting_attendees`, `meeting_notes`, `meeting_action_items`, `recording_consents`, `resources`, `resource_tags`, `resource_bookings`, `task_comments`, `task_activities`, `task_files`, `task_dependencies`, `project_members`, `project_statuses`. Plus Composite-Index `idx_recordings_meeting_id_tenant_id ON recordings(meeting_id, tenant_id)` fuer Recording-by-Meeting-Lookup ueber FK.
- **000110 option_b_phase2_email_notification:** 11 tenant_id-Spalten + Indizes auf `email_accounts`, `email_folders`, `email_attachments`, `email_signatures`, `email_contact_links`, `team_inboxes`, `team_inbox_members`, `routing_rules`, `notification_preferences`, `notification_mutes`, `notification_quiet_hours`. Subtables zu `email_messages`/`inbox_messages` (die bereits via 000106 tenant_id haben).
- **000111 option_b_phase2_security_crm_aux:** 12 tenant_id-Spalten + Indizes auf `vault_secrets`, `gdpr_export_requests`, `gdpr_erasure_log`, `password_policies`, `password_history`, `ip_access_rules`, `tags`, `contact_tags`, `company_tags`, `deal_tags`, `activity_tags`, `consent_records`. CRM-Auxiliary-Tabellen (Tag-System, Consent) und Security-Subtables.
- **000112 option_b_phase2_automation_exec_channels:** 5 tenant_id-Spalten auf `automation_executions`, `channel_memberships`, `integration_configs`, `bexio_sync_configs`, `lexware_sync_configs`. **JOIN-Backfill-Pattern** fuer Tabellen ohne eigenen owner: `UPDATE automation_executions ae SET tenant_id = a.tenant_id FROM automations a WHERE ae.automation_id = a.id` und analog fuer `channel_memberships` ueber `channels`. Beide bekommen danach `ALTER COLUMN tenant_id SET NOT NULL`. Die drei Integration-Configs bleiben nullable (kein Backfill-Source — App-Code befuellt beim naechsten Touch).
- **000113 idempotency_complete_tenant_pk_alignment:** Pure-Tracking-Migration mit einem zusaetzlichen partial Index `idx_idempotency_keys_tenant_completed ON idempotency_keys(tenant_id, key, completed_at) WHERE completed_at IS NULL`. Ergaenzt 000105's Index `(tenant_id, user_id, created_at DESC)` um Coverage fuer in-flight Lookups (Replay-Detection im HardMode-Pfad). Schema-Aenderung wuerde fuer den App-Code `Complete()`-Composite-PK-Fix nicht notwendig sein, aber Performance-relevant fuer Replay-Detection-Latenz.

**Companion-Aenderungen ohne Migration (im selben Welle-4B-Commit):**
- `idempotency.PostgresRepository.Complete(ctx, tenantID, key, status, body)` — neue Sig mit `WHERE tenant_id = $1 AND key = $2`. Ohne diesen Fix waere die Composite-PK aus 000108 nur halb wirksam (Get war seit Welle 3.5 sicher, Complete bis 4B nicht).
- `cmd/gateway/main.go` HardMode-Env-Flag `IDEMPOTENCY_MODE=hard` (Default WarnMode).
- 16+ Repository-Wirings (work/calendar+meeting+resource, email/*, notification/*, inbox/*, crm/tag+consent+search) mit tenant_id-First-Filter.
- chat/message Cursor-Lookup: ThreadListFilter bekommt TenantID-Field, Cursor-Decode-Query filtert tenant_id (P2-6-Fix).
- crm/activity AddTags: Loop mit N Einzel-INSERTs → Single `INSERT ... SELECT $1, unnest($2::uuid[])` (P3-3-Fix).

### Sprint 3 Welle 2 — Option-B Phase 2 Abschluss (Migrations 114–115, 2026-05-08)

Option-B Phase 2 ist damit **komplett** (~38 Tabellen in zwei Batches). Alle ALTERs defensiv mit `ADD COLUMN IF NOT EXISTS`. JOINs-Backfill-Pattern wie in 000112 wo kein direkter Owner-Bezug vorhanden. Gesamt retrofittete Tabellen: **~123** (85 vor Sprint 3 + 16 + 22 neu). Plan-Inventar: `docs/sprint3-option-b-phase2-plan.md` (~190 Tabellen, Status-Tracking pro Gruppe).

**Standard-Pattern aller Phase-2-Migrations:**
```sql
ALTER TABLE <name> ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE <name> SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
ALTER TABLE <name> ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE <name> ADD CONSTRAINT fk_<name>_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) NOT VALID;
CREATE INDEX IF NOT EXISTS idx_<name>_tenant_id ON <name>(tenant_id);
```

- **000114 option_b_phase2_settings_preferences:** 16 Tabellen in 4 Gruppen — Gruppe A: `user_dashboard_layouts`, `user_preferences`, `task_preferences`; Gruppe B: `document_folders`, `document_file_versions`, `document_shares`, `document_tags`, `document_file_tags`, `document_entity_links`, `wopi_locks`; Gruppe C: `user_sessions`, `app_specific_passwords`, `recovery_codes`, `gdpr_deletion_requests`; Gruppe D (partial): `caldav_push_subscriptions`. Repository-Wirings: `gateway/cached_dashboard_repository.go` nutzt jetzt `tenantID` im Cache-Key; `document/folder` postgres_repository mit `tenant_id`-First-Filter. **Welle-1-Hotfix `c7a9a76` (2026-05-08):** `CREATE TABLE IF NOT EXISTS tenants(id UUID PK, name TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())` + Sentinel-Insert `'00000000-0000-0000-0000-000000000001'` als Bootstrap am Anfang von 000114 nachgereicht — Production-DB war essenziell leer und 000114+115 referenzierten `tenants(id)` als FK ohne dass die Tabelle je angelegt wurde. Lokaler Run von leerer DB haette das gefangen; siehe Lesson in [[troubleshooting]].

- **000115 option_b_phase2_integrations_chat:** 22 Tabellen in 3 Gruppen — Gruppe D (rest): `caldav_sync_versions`, `caldav_change_log`; Gruppe E: 13 Integration-Mapping-Tabellen (`bexio_*`, `lexware_*`, `datev_*`); Gruppe F: `chat_files`, `message_reactions`, `message_mentions`, `call_sessions`, `call_participants`, `guest_sessions`, `guest_channel_config`. Repository-Wirings: `biz/bexio` und `biz/lexware` — neue Methoden `ListEntityMappings`/`ListSyncLogs` mit JOIN-Tenant-Fence. 8 neue Cross-Tenant-Tests fuer bexio/lexware/message_reactions/chat_files.

### Sprint 2 Welle 2D — JWT-Tenant-Hardening (Migration 104)

- **000104 users_tenant_id:** `ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'` + `CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id)`. Defensives `IF NOT EXISTS` weil ein vorheriger Patch ohne Idempotenz-Schutz auf einigen Dev-DBs angekommen war. Kombiniert mit `auth.Claims.TenantID` (JSON `tid`) und `middleware.GetTenantID()`-Helper (fail-closed: leerer/invalider Wert → `ErrMissingTenantID` → 401). Schliesst Welle-1-Altlast (11 Routes mit Placeholder-TenantID) und 5 Cross-Layer-Holes in dialer/helpdesk gRPC + wiki + biz/hr/lexware. Details siehe [[security]] Abschnitt "JWT Tenant-Claim & Cross-Layer-Hardening".

### Sprint 2 Welle 2C — Bugfix-Sweep-Migrations (100–103)

- **000100 rapporte_approve_permission:** Seedet `rapporte:report:approve` Permission (resource=`rapporte:report`, action=`approve`), grant nur an admin-Rolle. Vorher hatten `/approve` und `/reject` Routes die gleiche `rapporte:report:write` Permission wie Edit/Delete — jetzt separates Approver-Recht.
- **000101 vermietung_gist_overlap_unique_inspection:** `ALTER TABLE rentals ADD CONSTRAINT excl_rentals_no_overlap EXCLUDE USING GIST (tenant_id WITH =, object_id WITH =, tstzrange(start_date, end_date) WITH &&) WHERE (status NOT IN ('cancelled','completed'))` — DB-Layer-Race-Condition-Schutz fuer Doppelbuchung. Plus `ALTER TABLE rental_inspections ADD CONSTRAINT uq_rental_inspections_kind UNIQUE (tenant_id, rental_id, kind)` — verhindert zwei handover- oder zwei return-Inspections fuer dasselbe Rental.
- **000102 schichten_shift_assignments_tenant_unique:** `DROP CONSTRAINT uq_shift_assignments_shift_employee` (war non-tenant-scoped) und `ADD CONSTRAINT uq_shift_assignments_tenant_shift_employee UNIQUE (tenant_id, shift_id, employee_id)`.
- **000103 schichten_shift_capacity:** `ALTER TABLE shifts ADD COLUMN capacity INT NULL` — optionales Capacity-Limit fuer Schichten. AssignEmployee prueft `CountAssignments < shift.capacity` vor INSERT (`ErrShiftFull` sonst), kombiniert mit ArbZG-Pre-Check.

### Berichte / Reports (Migration 079)
- `report_definitions` — tenant_id, name, description, module (finanzen/crm/helpdesk/inventar/produktion/cross), kind (system/custom), query_config JSONB, default_format (pdf/csv/xlsx), created_by (FK users SET NULL), is_published, CHECK-Constraints auf module/kind/format
- `report_cache` — tenant_id, definition_id (FK CASCADE), params_hash TEXT (sha256), result JSONB, row_count, computed_at, expires_at, UNIQUE(definition_id, params_hash); Index `idx_report_cache_expires` fuer Cleanup-Job
- `report_schedules` — tenant_id, definition_id (FK CASCADE), cron_expression, recipients TEXT[], format, params JSONB, active, last_run_at/status/error, created_by (FK users SET NULL). Atomares `ClaimSchedule` UPDATE ... WHERE last_run_at=$prev verhindert Double-Run bei Tick-Overlap.
- `report_runs` — Audit-Log: tenant_id, definition_id, schedule_id (FK SET NULL), trigger (manual/scheduled/api), params, duration_ms, row_count, status (success/failed), error, started_at, completed_at
- **Seeds:** 8 System-Berichte auf Platzhalter-Tenant `00000000-…-000000000001` (Umsatz, Offene Posten, Pipeline, Conversion, Activity, Helpdesk-SLA, Stock-Warnings, DATEV-BWA)
- **Indizes:** `idx_report_{definitions,cache,schedules,runs}_tenant_id` + `idx_report_runs_tenant_started` (DESC fuer Audit-List)

## Index-Strategie
- **Composite:** `(project_key, archived_at)`, `(user_id, role, name)`
- **Conditional:** `(status) WHERE status != 'ended'` für aktive Records
- **Case-insensitive:** `LOWER(email)`, `LOWER(name)` für Suche
- **Time-series:** `created_at DESC` für chronologische Queries
- **Trigram (pg_trgm):** `gin_trgm_ops` auf contacts Name + companies Name — Fuzzy Duplicate Detection
- **Foreign Keys:** ON DELETE CASCADE oder SET NULL

## Connection Pool & PG Tuning (2026-04-08)
- MaxConns: 10 pro Service (10×10=100, passend zu PG max_connections=150)
- MinConns: 2, MaxConnLifetime: 1h, MaxConnIdleTime: 30m, HealthCheckPeriod: 1m
- PG Tuning (docker-compose.prod.yml): shared_buffers=4GB, effective_cache_size=12GB, work_mem=64MB, maintenance_work_mem=512MB

## AI-First-DB Foundation (Migrations 071–074, 2026-04-18)

Pragmatische Minimal-Version der "AI-First-DB"-Patterns — bewusst ohne Vendor-Lock-in.

### Migration 071 — Schema-Selbstbeschreibung via `COMMENT ON`
- Top-10-Tabellen kommentiert: `users`, `companies`, `contacts`, `deals`, `activities`, `projects`, `tasks`, `meetings`, `calendar_events`, `dialer_campaigns`
- Kommentare beschreiben: Zweck, Lifecycle, Enum-Semantik, non-obvious Relations
- Abrufbar via `obj_description()` / `col_description()` für MCP-Agenten und DBeaver
- **Konvention ab jetzt:** Jede neue Tabelle bekommt `COMMENT ON TABLE` + zentrale Spalten als Teil der `.up.sql`

### Migration 072/073 — pgvector auf contacts
- `CREATE EXTENSION vector` (Migration 072)
- `contacts.search_text` = GENERATED ALWAYS (first_name + last_name + email + phone + position + notes)
- `contacts.embedding vector(1536)` + `embedding_updated_at`
- HNSW-Index `idx_contacts_embedding` (m=16, ef_construction=64, cosine)
- Kein Backfill-Worker im Scope — Provider-Entscheidung (OpenAI vs EU-lokal) erst nach ZFA-Launch

### Migration 074 — Agent-Read-Only-Rolle
- `cosmi_agent_readonly` mit `SELECT` auf alle Tabellen (inkl. Default Privileges für neue Tabellen)
- Passwort out-of-band (Deploy-Script / Secrets-Manager), nicht in Git
- Zielgruppe: MCP-Server, die Schema-Introspection und Read-Queries für Agenten anbieten

### Vertagt (bewusst nicht im M1-Scope)
- **Embedding-Backfill-Worker** — eigener Sprint nach Provider-Entscheidung (EU vs OpenAI, DSGVO-Implikationen)
- **Hybrid Search (BM25 + Vector + RRF)** — erst wenn Use-Case konkret (z.B. Kunde fragt nach "Kontakten mit Budget-Bedenken")
- **ParadeDB `pg_search`, `pgai`, `pgvectorscale`** — Vendor-Lock + zusätzliche Failure-Domain
- **Agent-Memory-Tabellen** (`agent_conversations`, `agent_memories`, `agent_tool_calls`) — keine konkrete Agent-Anwendung in Cosmi geplant
- **Semantic Catalog YAML** — YAGNI für Single-Tenant-Stage, `COMMENT ON` reicht
- **Spalten-Rename für Naturalness (SNAILS)** — Breaking Change, warten bis nach Multi-Tenancy-Migration (Phase 3)

### Chain PILOT — Auth & Booking (Migrations 134–136, 2026-06-09)

- **000134 create_password_reset_tokens:** Tabelle `password_reset_tokens` — id UUID PK, `tenant_id UUID NOT NULL` (pre-RLS-ready), `user_id UUID FK → users ON DELETE CASCADE`, `token_hash VARCHAR(64) UNIQUE` (SHA-256), `expires_at TIMESTAMPTZ` (1h), `used_at TIMESTAMPTZ NULL`, `created_at`. Single-use, Muster wie refresh_tokens/invitations. RLS-Policy folgt separater Welle.
- **000135 create_booking_pages:** Zwei Tabellen, beide `tenant_id NOT NULL`:
  - `booking_pages` — id, tenant_id, `calendar_id FK`, `slug` mit `UNIQUE(tenant_id, slug)`, company_name, logo_url, `services JSONB`, `availability_rules JSONB` (`{weekdays, slots_per_weekday[{start,end}], slot_duration_min, buffer_min, lead_time_hours, breaks[{start,end}]}`), active.
  - `public_bookings` — id, tenant_id, `booking_page_id FK`, service_id, customer_name/email/phone, notes, date, time_slot, staff_user_id, status, calendar_event_id; `UNIQUE(booking_page_id, customer_email, date, time_slot)` als Doppel-Submit-Backstop.
- **000136 seed_booking_pages_permissions:** RBAC-Seed: Permissions `booking-pages:read`, `booking-pages:write`, `booking-pages:delete` + Grant an admin-Rolle. Muster 000131 (idempotent ON CONFLICT DO NOTHING).

### Marathon Tag 2 Welle 1 — Beratungsprotokoll & Settings (Migrations 137–138, 2026-06-10)

- **000137 advisory_protocols:** Beratungsprotokoll ZFA (Finanzberatung). Tabelle `advisory_protocols` — 57 typisierte Spalten über 8 Spec-Abschnitte (nur `products` als JSONB, Muster finance line_items), `tenant_id NOT NULL`, `contact_id FK`, Status `draft → finalized` (`handed_over_at`), CHECK `risk_class 1-7`. **Immutability service-seitig erzwungen** (Repo-Muster, DB-Status-Filter als Doppelnetz), 10J-Retention (DSGVO Art. 6(1)(c)). Plus contacts-Erweiterung: `referred_by_contact_id` (Self-FK) + `client_segment` (CHAR(1) CHECK A/B/C). RBAC-Seeds `advisory-protocols:{read,write,delete}`.
- **000138 create_settings_foundation:** 3-Ebenen-Settings-Modell (Tenant-Default → Modul-Leiter-Override → User-Override). Drei Tabellen, alle `tenant_id NOT NULL`: `tenant_settings` (PK tenant_id+module_id+key, value JSONB), `user_settings` (PK +user_id, FK users CASCADE), `tenant_module_leads` (PK tenant_id+user_id+module_id, granted_by/granted_at). Key-Namespacing wie FE (`payroll.*`). RBAC-Seeds `module-leads:{read,write}` (write admin-only), `settings:{read,write}`. Resolve serverseitig in `internal/settings` — siehe [[architektur]].

### Marathon Tag 2 Welle 2 — Finance-Compliance & Kontakte-360° (Migrations 139–141, 2026-06-10)

- **000139 gobd_belegarchiv:** `gobd_documents` — revisionssicheres Belegarchiv nach §147 AO (10J-Retention). Immutability by design: KEIN UPDATE/DELETE (service-seitig erzwungen), `retention_until` = 31.12. von (archived_year + 8).
- **000140 finance_incoming_invoices:** E-Rechnung Eingang (ZUGFeRD/Factur-X CII + XRechnung UBL, E-RechV / EU 2014/55/EU). `line_items`/`tax_breakdown` als JSONB (Muster wie outgoing invoices).
- **000141 finance_invoices_contact_id:** `contact_id UUID NULL FK → contacts ON DELETE SET NULL` auf `finance_invoices` für Kontakte-360°-Verknüpfung.

### Backend-Sessions 2026-06-11 — RLS-Nachzug, Signaturen, Files, Work-Erweiterungen (Migrations 142–147)

- **000142 rls_pilot_new_tables:** RLS-Aktivierung auf den drei Chain-PILOT-Tabellen `password_reset_tokens`, `booking_pages`, `public_bookings` (schließt die in 000134/000135 angekündigte „separate Welle").
- **000143 add_signature_to_rapporte_vermietung_vertraege:** `signature_data TEXT NULL` (+Metadaten) auf `work_reports`, vermietung- und vertraege-Tabellen — persistiert die Canvas-EES-Signaturen.
- **000144 seed_files_permissions:** RBAC-Seed `files:{...}` für die generischen Presign-Routen (`POST /api/v1/files/presign-upload`, `GET .../presign-download`) — Muster 000131, ohne Seed 403 für alle.
- **000145 work_labels:** `work_labels` (tenant-scoped, Farbe) + `task_labels` (m:n zu Tasks), RLS.
- **000146 work_custom_field_definitions:** tenant-scoped Custom-Field-Definitionen für Tasks (`field_type` + optionale Select-Options).
- **000147 seed_work_labels_permissions:** RBAC-Seeds `work_labels:*` + `work_custom_fields:*` (Muster 000144).
- **Companion ohne Migration (2026-06-11 Nacht, `d028b8ea`):** `label_ids` wird in GetTask/ListTasks per `GetLabelsByTaskIDs`-Batch geladen (1 Query); `filter_label_ids` filtert als tenant-gescopte `EXISTS`-Subquery auf `task_labels`/`work_labels` im task-Repo.

**Head: 000147** · 000148 reserviert für contract_events-Audit.

## Verwandte Notes
- [[architektur]] — Service-Architektur, Performance-Patterns
- [[api]] — API-Endpoints

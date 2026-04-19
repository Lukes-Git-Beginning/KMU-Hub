---
tags: [datenbank, schema, migrations, ai-first]
updated: 2026-04-19
---
# Datenbank

## Überblick
- PostgreSQL 16 + Redis 7 (nur Cache, KEIN Dual-Write)
- Änderungen NUR via golang-migrate (`make migrate-create name=xxx`)
- 81 Migration-Paare in `backend/migrations/` (Sprint 1 Welle 2: 076 wiki, 077 helpdesk; Welle 5: 079 berichte; Welle 6: 080 seed_berichte_permissions; S1.3: 081 formulare)
- Index-Konvention: `idx_{table}_{column}`
- **AI-First-Foundation** seit Migration 071 (siehe Abschnitt unten)

## Tabellen nach Domain

### Core Identity
- `users` — UUID PK, email (unique), password_hash (bcrypt), first/last_name, is_active
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
- `finance_payments` — payment_date, amount, method, invoice_id
- `finance_credit_notes` — Gutschriften (Reverse Invoices)

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

## Verwandte Notes
- [[architektur]] — Service-Architektur, Performance-Patterns
- [[api]] — API-Endpoints

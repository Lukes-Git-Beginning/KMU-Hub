---
tags: [datenbank, schema, migrations]
updated: 2026-04-09
---
# Datenbank

## Überblick
- PostgreSQL 16 + Redis 7 (nur Cache, KEIN Dual-Write)
- Änderungen NUR via golang-migrate (`make migrate-create name=xxx`)
- 68 Migration-Paare in `backend/migrations/`
- Index-Konvention: `idx_{table}_{column}`

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

### Consent Management (Migration 060)
- `consent_records` — contact_id (FK), consent_type ENUM (marketing_email, marketing_phone, profiling, newsletter, data_processing, data_sharing), granted BOOL, legal_basis ENUM (consent, legitimate_interest, contract, legal_obligation), source, ip_address INET, granted_at, revoked_at
- `gdpr_deletion_requests` — contact_id, requested_by, reason, status (pending/completed), completed_at

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

### Dialer Permissions (Migration 068)
- `dialer:campaigns` (read/write), `dialer:calls` (write), `dialer:agent` (manage), `dialer:outcomes` (manage)
- Zugewiesen an Rollen: admin (alle), manager (alle), member (campaigns:read, calls:write, agent:manage)

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

## Verwandte Notes
- [[architektur]] — Service-Architektur, Performance-Patterns
- [[api]] — API-Endpoints

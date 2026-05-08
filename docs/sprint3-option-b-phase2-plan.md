# Sprint 3 — Option-B Phase 2: Settings/Preferences tenant_id Retrofit Plan

> Erstellt: 2026-05-08  
> Stream: S3.MT.2  
> Ziel: Alle verbleibenden ~25 Tabellen mit tenant_id versehen (Settings, Preferences, CalDAV, Documents, Mappings, Misc)

---

## 1. Übersicht: Was ist bereits erledigt?

### Phase 1 (Migration 000106 — 20 Tabellen, nativ mit DEFAULT)
deals, activities, tasks, projects, channels, messages, notifications, time_entries,
calendar_events, email_messages, inbox_messages, deal_stage_history, pipeline_stages,
saved_filters, custom_field_definitions, automations, document_files, recordings,
dialer_call_sessions, audit_log

### Phase 2 Batch A (Migration 000109 — 21 Tabellen Calendar/Work)
calendars, calendar_members, event_categories, event_attendees, event_exceptions,
event_reminders, user_calendar_preferences, meetings, meeting_attendees, meeting_notes,
meeting_action_items, recording_consents, resources, resource_tags, resource_bookings,
task_comments, task_activities, task_files, task_dependencies, project_members,
project_statuses

### Phase 2 Batch B (Migration 000110 — 11 Tabellen Email/Inbox/Notification)
email_accounts, email_folders, email_attachments, email_signatures, email_contact_links,
team_inboxes, team_inbox_members, routing_rules,
notification_preferences, notification_mutes, notification_quiet_hours

### Phase 2 Batch C (Migration 000111 — 12 Tabellen Security/CRM-Aux)
vault_secrets, gdpr_export_requests, gdpr_erasure_log, password_policies, password_history,
ip_access_rules, tags, contact_tags, company_tags, deal_tags, activity_tags, consent_records

### Phase 2 Batch D (Migration 000112 — 5 Tabellen Automation/Channels)
automation_executions, channel_memberships,
integration_configs, bexio_sync_configs, lexware_sync_configs

### Nativ mit tenant_id erstellt (keine Migration nötig)
hr_company_settings, hr_leave_types, hr_leave_requests, hr_leave_balances,
hr_work_time_entries, hr_break_entries, hr_document_categories, hr_employee_documents,
inventory_items, inventory_movements, stock_warnings,
suppliers, purchase_orders, po_lines,
production_orders, machine_bookings, production_plans,
contracts, contract_parties, contract_reminders,
shifts, shift_templates, shift_assignments,
vehicles, vehicle_services, vehicle_damages,
rental_objects, rentals, rental_inspections,
wiki_categories, wiki_articles, wiki_versions, wiki_attachments, wiki_share_tokens,
sla_policies, ticket_queues, tickets, ticket_messages, canned_responses,
report_definitions, report_cache, report_schedules, report_runs, report_attachments, report_lines,
form_schemas, form_submissions, form_webhooks, form_webhook_deliveries,
dialer_campaigns, dialer_campaign_contacts, dialer_call_outcomes,
validation_rules, workflow_rules,
plugin_installations, work_reports

---

## 2. Delta: Tabellen die noch KEIN tenant_id haben

### Gruppe A — User-Settings/Dashboard-Preferences (PRIO 1 — Migration 000114)

| Tabelle | Migration | FK-Backfill-Pfad | Aktives Repo? |
|---------|-----------|------------------|---------------|
| `user_dashboard_layouts` | 000023 | `user_id → users.tenant_id` | ✅ `gateway/dashboard_repository.go` |
| `user_project_preferences` | 000026 | `user_id → users.tenant_id` | Fraglich (work-Package) |
| `task_entity_links` | 000026 | `task_id → tasks.tenant_id` | ✅ work-Package |
| `task_custom_field_values` | 000026 | `task_id → tasks.tenant_id` | ✅ work-Package |

### Gruppe B — Document-Layer (PRIO 1 — Migration 000114)

| Tabelle | Migration | FK-Backfill-Pfad | Aktives Repo? |
|---------|-----------|------------------|---------------|
| `document_folders` | 000043 | `created_by → users.tenant_id` (kein direkter FK auf tenant) | ✅ `document/folder/` |
| `document_file_versions` | 000043 | `file_id → document_files.tenant_id` | ✅ `document/file/` |
| `document_shares` | 000043 | `shared_with_user_id → users.tenant_id` | ✅ `document/share/` |
| `document_tags` | 000043 | `created_by → users.tenant_id` | ✅ `document/tag/` |
| `document_file_tags` | 000043 | über `document_files.tenant_id` via `file_id` | ✅ `document/tag/` |
| `document_entity_links` | 000043 | über `document_files.tenant_id` via `file_id` (nullable, andere Keys auch) | ✅ `document/file/` |
| `wopi_locks` | 000044 | `file_id → document_files.tenant_id` | ✅ `document/wopi/` |
| `storage_quotas` | 000018 (chat_files) | Singleton (kein FK) → DEFAULT-Sentinel | Fraglich (1 Row pro Tenant) |

### Gruppe C — Security/Auth-Misc (PRIO 2)

| Tabelle | Migration | FK-Backfill-Pfad | Aktives Repo? |
|---------|-----------|------------------|---------------|
| `user_sessions` | 000039 | `user_id → users.tenant_id` | ✅ `auth/postgres_repository.go` |
| `app_specific_passwords` | 000049 | `user_id → users.tenant_id` | ✅ `caldav/postgres_app_password.go` |
| `two_factor_policy` | 000039 | Singleton pro Tenant → kein FK | Fraglich (Tenant-global) |
| `recovery_codes` | 000039 | `user_id → users.tenant_id` | ✅ `auth/` |
| `gdpr_deletion_requests` | 000060 | `contact_id → contacts.tenant_id` | ✅ `security/gdpr/` |

### Gruppe D — CalDAV-Layer (PRIO 2)

| Tabelle | Migration | FK-Backfill-Pfad | Aktives Repo? |
|---------|-----------|------------------|---------------|
| `caldav_push_subscriptions` | 000051 | `user_id → users.tenant_id` | ✅ `caldav/push_subscription.go` |
| `caldav_sync_versions` | 000050 | `collection_id` ist Kalender/Kontakt-ID, kein direkter user-FK | Via collection_type+collection_id lookup |
| `caldav_change_log` | 000050 | Wie caldav_sync_versions | Via collection-Lookup |
| `caldav_settings` | 000050 | Key-Value Singleton (global, kein Tenant-FK) | **GLOBAL — kein tenant_id nötig** |

### Gruppe E — Integration-Mappings (PRIO 2)

| Tabelle | Migration | FK-Backfill-Pfad | Aktives Repo? |
|---------|-----------|------------------|---------------|
| `bexio_entity_mappings` | 000055 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `bexio_field_mappings` | 000055 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `bexio_sync_log` | 000055 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `lexware_entity_mappings` | 000056 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `lexware_field_mappings` | 000056 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `lexware_sync_log` | 000056 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `lexware_webhook_subscriptions` | 000056 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `datev_upload_configs` | 000056 | Direkt für Tenant — kein config_id | Manuell setzen |
| `datev_upload_log` | 000056 | Via `config_id → datev_upload_configs.tenant_id` | ✅ biz-Package |
| `integration_channel_mappings` | 000053 | `config_id → integration_configs.tenant_id` | ✅ biz-Package |
| `integration_account_links` | 000053 | `kmuhub_user_id → users.tenant_id` | ✅ biz-Package |
| `integration_delivery_log` | 000053 | `mapping_id → integration_channel_mappings.tenant_id` | ✅ biz-Package |
| `integration_link_tokens` | 000053 | Prüfen… | ✅ biz-Package |

### Gruppe F — Chat/Presence/Call (PRIO 3)

| Tabelle | Migration | FK-Backfill-Pfad | Aktives Repo? |
|---------|-----------|------------------|---------------|
| `chat_files` | 000018 | `channel_id → channels.tenant_id` | ✅ chat-Package |
| `message_reactions` | 000038 | `message_id → messages.tenant_id` | ✅ chat-Package |
| `message_mentions` | 000017 | `message_id → messages.tenant_id` | ✅ chat-Package |
| `call_sessions` | 000036 | `initiator_id → users.tenant_id` | ✅ chat-Package |
| `call_participants` | 000036 | `call_id → call_sessions.tenant_id` (nach Retrofit) | ✅ chat-Package |
| `presence_config` | 000038 | Singleton pro Tenant (kein FK) | Fraglich |
| `guest_sessions` | 000054 | `channel_id → channels.tenant_id` | ✅ chat-Package |
| `guest_channel_config` | 000054 | `channel_id → channels.tenant_id` | ✅ chat-Package |

### Gruppe G — Misc/System (SKIP oder niedrige Prio)

| Tabelle | Grund |
|---------|-------|
| `caldav_settings` | Global-Config (Key-Value Singleton, kein Tenant-Bezug) → **SKIP** |
| `storage_quotas` | Per-Tenant Singleton — tenant_id wird primärer Key statt UUID → Migrationsformat abweichend |
| `presence_config` | Global-Singleton → **SKIP** |
| `two_factor_policy` | Tenant-global, kein User-FK → **SKIP** (Policy gilt für alle User eines Tenants) |
| `plugin_manifests` | Systemweit — kein Tenant-Bezug → **SKIP** |
| `plugin_permissions` | Über `installation_id → plugin_installations.tenant_id` → niedrige Prio |
| `plugin_kv_store` | Über `installation_id → plugin_installations.tenant_id` → niedrige Prio |
| `plugin_execution_log` | Über `installation_id → plugin_installations.tenant_id` → niedrige Prio |
| `automation_templates` | Prüfen ob Tenant-bezogen — vermutlich system-global → niedrige Prio |
| `industry_templates` | System-Seed-Daten → **SKIP** |
| `public_holidays` | Länder-Seed-Daten → **SKIP** |
| `roles`, `permissions`, `role_permissions`, `user_roles` | RBAC-System → ggf. Tenant-Retrofit Sprint 4 |
| `invitations` | `invited_by → users.tenant_id` → Sprint 3.5 |
| `refresh_tokens` | `user_id → users.tenant_id` → niedrige Prio (kurzlebig) |
| `event_types` | System-Lookup-Table → **SKIP** |
| `events` | Unklar — prüfen |
| `dialer_call_events` | Dialer-Subevents — via `session_id → dialer_call_sessions.tenant_id` |
| `dialer_agent_status_log` | Via `agent_id → users.tenant_id` |

---

## 3. Migration 000114 — Batch 1: Settings/Preferences + Document-Layer

**Scope: 13 Tabellen (Gruppe A + B)**

Priorisiert weil:
- Direkte API-Endpunkte existieren (aktive Repos)
- User-facing Settings (Cross-Tenant-Leakage wäre sichtbar)
- Document-Layer: document_files hat bereits tenant_id, alle Subtabellen brauchen es auch

**Tabellen in 000114:**
1. `user_dashboard_layouts` (user_id → users)
2. `document_folders` (created_by → users)
3. `document_file_versions` (file_id → document_files)
4. `document_shares` (shared_with_user_id → users)
5. `document_tags` (created_by → users)
6. `document_file_tags` (file_id → document_files)
7. `document_entity_links` (kein eindeutiger FK → Sentinel-DEFAULT)
8. `wopi_locks` (file_id → document_files)
9. `user_sessions` (user_id → users)
10. `app_specific_passwords` (user_id → users)
11. `recovery_codes` (user_id → users)
12. `gdpr_deletion_requests` (contact_id → contacts)
13. `caldav_push_subscriptions` (user_id → users)

---

## 4. Migration 000115 (nächste Welle) — Integration-Mappings + Chat

**Scope: ~15 Tabellen (Gruppe D rest + E + F)**

- bexio/lexware/datev/integration mapping-Tabellen (config_id-basierter Backfill)
- chat_files, message_reactions, message_mentions, call_sessions, call_participants
- guest_sessions, guest_channel_config

---

## 5. Repository-Wirings in 000114-Scope

Tabellen mit aktiven Repos die nach der Migration upgedatet werden:
- `gateway/dashboard_repository.go` — user_dashboard_layouts (tenant_id in SELECT/INSERT/DELETE)
- `document/folder/postgres_repository.go` — document_folders (tenant_id Filter)
- `document/file/postgres_repository.go` — document_file_versions (tenant_id Filter)
- `document/share/postgres_repository.go` — document_shares (tenant_id Filter)
- `document/tag/postgres_repository.go` — document_tags + document_file_tags (tenant_id Filter)
- `caldav/push_subscription.go` — caldav_push_subscriptions (tenant_id Filter)
- `auth/postgres_repository.go` — user_sessions, recovery_codes (tenant_id Filter)
- `caldav/postgres_app_password.go` — app_specific_passwords (tenant_id Filter)

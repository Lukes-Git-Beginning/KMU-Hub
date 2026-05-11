-- Sprint 4 Welle 3 — RLS-Aktivierung Phase 2 auf 95 Long-Tail-Tabellen
-- mit garantiertem NOT NULL tenant_id.
-- 2026-05-11
--
-- Builds on:
--   - 000118 (helper functions + enable_tenant_rls procedure)
--   - 000120 (Pilot-0 RLS, 22 tables)
--   - 000121 (kmuhub_app NOSUPERUSER NOBYPASSRLS role)
--
-- Welle 3.1A scope: only tables where tenant_id is NOT NULL by schema
-- constraint. The 44 tables with NULLABLE tenant_id (mostly added via
-- ADD COLUMN in mig 109/110/111/112 without subsequent SET NOT NULL)
-- are deferred to Welle 3.1B, because Welle 3.1.2 sampling found
-- repository-layer INSERTs that omit tenant_id altogether
-- (contact_tags/company_tags/deal_tags, vault_secrets,
-- gdpr_export_requests, bexio_sync_configs). RLS-enabling those tables
-- without first fixing the repos would break tag-add, vault-secret-write,
-- GDPR-request creation, and Bexio-sync-config provisioning.
--
-- 4 EXCLUDED tables (no tenant_id at all, deferred to Welle 4):
--   - ticket_messages   (parent FK to tickets — needs ADD COLUMN or via_join)
--   - plugin_kv_store   (parent FK to plugin_installations — same)
--   - plugin_execution_log (same)
--   - caldav_settings   (system-global key-value table by design — no RLS)
--
-- Migration runs with row_security disabled inside the migration tx so the
-- ALTER TABLE statements themselves are not subject to a (not yet existing)
-- policy. Same pattern as 000118 + 000120.

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- Phase 1 retrofit (mig 000106) — 9 core tables with NOT NULL DEFAULT sentinel.
-- ============================================================================
CALL enable_tenant_rls('calendar_events');
CALL enable_tenant_rls('email_messages');
CALL enable_tenant_rls('inbox_messages');
CALL enable_tenant_rls('notifications');
CALL enable_tenant_rls('time_entries');
CALL enable_tenant_rls('automations');
CALL enable_tenant_rls('saved_filters');
CALL enable_tenant_rls('custom_field_definitions');
CALL enable_tenant_rls('document_files');

-- ============================================================================
-- Option-B Phase 2 batch 1 (mig 000112) — 2 tables SET NOT NULL via JOIN backfill.
-- ============================================================================
CALL enable_tenant_rls('automation_executions');
CALL enable_tenant_rls('channel_memberships');

-- ============================================================================
-- Option-B Phase 2 settings/preferences (mig 000114) — 15 tables.
-- (user_sessions already RLS-enabled in mig 000120; not repeated here.)
-- ============================================================================
CALL enable_tenant_rls('user_dashboard_layouts');
CALL enable_tenant_rls('user_project_preferences');
CALL enable_tenant_rls('task_entity_links');
CALL enable_tenant_rls('task_custom_field_values');
CALL enable_tenant_rls('document_folders');
CALL enable_tenant_rls('document_file_versions');
CALL enable_tenant_rls('document_shares');
CALL enable_tenant_rls('document_tags');
CALL enable_tenant_rls('document_file_tags');
CALL enable_tenant_rls('document_entity_links');
CALL enable_tenant_rls('wopi_locks');
CALL enable_tenant_rls('app_specific_passwords');
CALL enable_tenant_rls('recovery_codes');
CALL enable_tenant_rls('gdpr_deletion_requests');
CALL enable_tenant_rls('caldav_push_subscriptions');

-- ============================================================================
-- Option-B Phase 2 integrations/chat (mig 000115) — 22 tables.
-- ============================================================================
CALL enable_tenant_rls('caldav_sync_versions');
CALL enable_tenant_rls('caldav_change_log');
CALL enable_tenant_rls('bexio_entity_mappings');
CALL enable_tenant_rls('bexio_field_mappings');
CALL enable_tenant_rls('bexio_sync_log');
CALL enable_tenant_rls('lexware_entity_mappings');
CALL enable_tenant_rls('lexware_field_mappings');
CALL enable_tenant_rls('lexware_sync_log');
CALL enable_tenant_rls('lexware_webhook_subscriptions');
CALL enable_tenant_rls('datev_upload_configs');
CALL enable_tenant_rls('datev_upload_log');
CALL enable_tenant_rls('integration_channel_mappings');
CALL enable_tenant_rls('integration_account_links');
CALL enable_tenant_rls('integration_delivery_log');
CALL enable_tenant_rls('integration_link_tokens');
CALL enable_tenant_rls('chat_files');
CALL enable_tenant_rls('message_reactions');
CALL enable_tenant_rls('message_mentions');
CALL enable_tenant_rls('call_sessions');
CALL enable_tenant_rls('call_participants');
CALL enable_tenant_rls('guest_sessions');
CALL enable_tenant_rls('guest_channel_config');

-- ============================================================================
-- Wiki (mig 000076) — 2 tables, tenant_id NOT NULL in CREATE TABLE.
-- (wiki_versions/wiki_attachments/wiki_share_tokens have no tenant_id —
-- deferred to Welle 4 with ADD COLUMN + via_join policy.)
-- ============================================================================
CALL enable_tenant_rls('wiki_categories');
CALL enable_tenant_rls('wiki_articles');

-- ============================================================================
-- Helpdesk (mig 000077) — 4 tables. ticket_messages excluded (no tenant_id;
-- parent FK to tickets — Welle 4 ADD COLUMN or via_join).
-- ============================================================================
CALL enable_tenant_rls('sla_policies');
CALL enable_tenant_rls('ticket_queues');
CALL enable_tenant_rls('tickets');
CALL enable_tenant_rls('canned_responses');

-- ============================================================================
-- Berichte (mig 000079) — 4 tables.
-- ============================================================================
CALL enable_tenant_rls('report_definitions');
CALL enable_tenant_rls('report_cache');
CALL enable_tenant_rls('report_schedules');
CALL enable_tenant_rls('report_runs');

-- ============================================================================
-- Formulare (mig 000081) — 4 tables.
-- ============================================================================
CALL enable_tenant_rls('form_schemas');
CALL enable_tenant_rls('form_submissions');
CALL enable_tenant_rls('form_webhooks');
CALL enable_tenant_rls('form_webhook_deliveries');

-- ============================================================================
-- Welle-2 modules — Inventar (mig 000083), Einkauf (mig 000085),
-- Produktion (mig 000087), Vertraege (mig 000089).
-- All tables have tenant_id NOT NULL in CREATE TABLE.
-- ============================================================================
CALL enable_tenant_rls('inventory_items');
CALL enable_tenant_rls('inventory_movements');
CALL enable_tenant_rls('stock_warnings');
CALL enable_tenant_rls('suppliers');
CALL enable_tenant_rls('purchase_orders');
CALL enable_tenant_rls('po_lines');
CALL enable_tenant_rls('production_orders');
CALL enable_tenant_rls('machine_bookings');
CALL enable_tenant_rls('production_plans');
CALL enable_tenant_rls('contracts');
CALL enable_tenant_rls('contract_parties');
CALL enable_tenant_rls('contract_reminders');

-- ============================================================================
-- Welle-2 modules — Rapporte (mig 000092), Schichten (mig 000094),
-- Fuhrpark (mig 000096), Vermietung (mig 000098).
-- ============================================================================
CALL enable_tenant_rls('work_reports');
CALL enable_tenant_rls('report_lines');
CALL enable_tenant_rls('report_attachments');
CALL enable_tenant_rls('shifts');
CALL enable_tenant_rls('shift_assignments');
CALL enable_tenant_rls('shift_templates');
CALL enable_tenant_rls('vehicles');
CALL enable_tenant_rls('vehicle_services');
CALL enable_tenant_rls('vehicle_damages');
CALL enable_tenant_rls('rental_objects');
CALL enable_tenant_rls('rentals');
CALL enable_tenant_rls('rental_inspections');

-- ============================================================================
-- Finance (mig 000045) — 8 tables.
-- ============================================================================
CALL enable_tenant_rls('company_settings');
CALL enable_tenant_rls('finance_number_sequences');
CALL enable_tenant_rls('finance_quotes');
CALL enable_tenant_rls('finance_invoices');
CALL enable_tenant_rls('finance_credit_notes');
CALL enable_tenant_rls('finance_payments');
CALL enable_tenant_rls('finance_dunning_records');
CALL enable_tenant_rls('finance_dunning_config');

-- ============================================================================
-- Plugins (mig 000057) — 1 table. plugin_kv_store + plugin_execution_log
-- excluded (no tenant_id — Welle 4 ADD COLUMN or via_join through
-- plugin_installations parent).
-- ============================================================================
CALL enable_tenant_rls('plugin_installations');

COMMIT;

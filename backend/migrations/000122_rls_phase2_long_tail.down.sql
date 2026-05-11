-- Reverse of 000122 — drop the tenant_isolation policy and disable RLS
-- on each of the 95 Phase-2-A tables, in reverse order of activation.

SET LOCAL row_security = off;

BEGIN;

-- Plugins
DROP POLICY IF EXISTS tenant_isolation ON plugin_installations;
ALTER TABLE plugin_installations DISABLE ROW LEVEL SECURITY;

-- Finance
DROP POLICY IF EXISTS tenant_isolation ON finance_dunning_config;
DROP POLICY IF EXISTS tenant_isolation ON finance_dunning_records;
DROP POLICY IF EXISTS tenant_isolation ON finance_payments;
DROP POLICY IF EXISTS tenant_isolation ON finance_credit_notes;
DROP POLICY IF EXISTS tenant_isolation ON finance_invoices;
DROP POLICY IF EXISTS tenant_isolation ON finance_quotes;
DROP POLICY IF EXISTS tenant_isolation ON finance_number_sequences;
DROP POLICY IF EXISTS tenant_isolation ON company_settings;
ALTER TABLE finance_dunning_config DISABLE ROW LEVEL SECURITY;
ALTER TABLE finance_dunning_records DISABLE ROW LEVEL SECURITY;
ALTER TABLE finance_payments DISABLE ROW LEVEL SECURITY;
ALTER TABLE finance_credit_notes DISABLE ROW LEVEL SECURITY;
ALTER TABLE finance_invoices DISABLE ROW LEVEL SECURITY;
ALTER TABLE finance_quotes DISABLE ROW LEVEL SECURITY;
ALTER TABLE finance_number_sequences DISABLE ROW LEVEL SECURITY;
ALTER TABLE company_settings DISABLE ROW LEVEL SECURITY;

-- Welle-2 modules — Rapporte/Schichten/Fuhrpark/Vermietung
DROP POLICY IF EXISTS tenant_isolation ON rental_inspections;
DROP POLICY IF EXISTS tenant_isolation ON rentals;
DROP POLICY IF EXISTS tenant_isolation ON rental_objects;
DROP POLICY IF EXISTS tenant_isolation ON vehicle_damages;
DROP POLICY IF EXISTS tenant_isolation ON vehicle_services;
DROP POLICY IF EXISTS tenant_isolation ON vehicles;
DROP POLICY IF EXISTS tenant_isolation ON shift_templates;
DROP POLICY IF EXISTS tenant_isolation ON shift_assignments;
DROP POLICY IF EXISTS tenant_isolation ON shifts;
DROP POLICY IF EXISTS tenant_isolation ON report_attachments;
DROP POLICY IF EXISTS tenant_isolation ON report_lines;
DROP POLICY IF EXISTS tenant_isolation ON work_reports;
ALTER TABLE rental_inspections DISABLE ROW LEVEL SECURITY;
ALTER TABLE rentals DISABLE ROW LEVEL SECURITY;
ALTER TABLE rental_objects DISABLE ROW LEVEL SECURITY;
ALTER TABLE vehicle_damages DISABLE ROW LEVEL SECURITY;
ALTER TABLE vehicle_services DISABLE ROW LEVEL SECURITY;
ALTER TABLE vehicles DISABLE ROW LEVEL SECURITY;
ALTER TABLE shift_templates DISABLE ROW LEVEL SECURITY;
ALTER TABLE shift_assignments DISABLE ROW LEVEL SECURITY;
ALTER TABLE shifts DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_attachments DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_lines DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_reports DISABLE ROW LEVEL SECURITY;

-- Welle-2 modules — Inventar/Einkauf/Produktion/Vertraege
DROP POLICY IF EXISTS tenant_isolation ON contract_reminders;
DROP POLICY IF EXISTS tenant_isolation ON contract_parties;
DROP POLICY IF EXISTS tenant_isolation ON contracts;
DROP POLICY IF EXISTS tenant_isolation ON production_plans;
DROP POLICY IF EXISTS tenant_isolation ON machine_bookings;
DROP POLICY IF EXISTS tenant_isolation ON production_orders;
DROP POLICY IF EXISTS tenant_isolation ON po_lines;
DROP POLICY IF EXISTS tenant_isolation ON purchase_orders;
DROP POLICY IF EXISTS tenant_isolation ON suppliers;
DROP POLICY IF EXISTS tenant_isolation ON stock_warnings;
DROP POLICY IF EXISTS tenant_isolation ON inventory_movements;
DROP POLICY IF EXISTS tenant_isolation ON inventory_items;
ALTER TABLE contract_reminders DISABLE ROW LEVEL SECURITY;
ALTER TABLE contract_parties DISABLE ROW LEVEL SECURITY;
ALTER TABLE contracts DISABLE ROW LEVEL SECURITY;
ALTER TABLE production_plans DISABLE ROW LEVEL SECURITY;
ALTER TABLE machine_bookings DISABLE ROW LEVEL SECURITY;
ALTER TABLE production_orders DISABLE ROW LEVEL SECURITY;
ALTER TABLE po_lines DISABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_orders DISABLE ROW LEVEL SECURITY;
ALTER TABLE suppliers DISABLE ROW LEVEL SECURITY;
ALTER TABLE stock_warnings DISABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_movements DISABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_items DISABLE ROW LEVEL SECURITY;

-- Formulare
DROP POLICY IF EXISTS tenant_isolation ON form_webhook_deliveries;
DROP POLICY IF EXISTS tenant_isolation ON form_webhooks;
DROP POLICY IF EXISTS tenant_isolation ON form_submissions;
DROP POLICY IF EXISTS tenant_isolation ON form_schemas;
ALTER TABLE form_webhook_deliveries DISABLE ROW LEVEL SECURITY;
ALTER TABLE form_webhooks DISABLE ROW LEVEL SECURITY;
ALTER TABLE form_submissions DISABLE ROW LEVEL SECURITY;
ALTER TABLE form_schemas DISABLE ROW LEVEL SECURITY;

-- Berichte
DROP POLICY IF EXISTS tenant_isolation ON report_runs;
DROP POLICY IF EXISTS tenant_isolation ON report_schedules;
DROP POLICY IF EXISTS tenant_isolation ON report_cache;
DROP POLICY IF EXISTS tenant_isolation ON report_definitions;
ALTER TABLE report_runs DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_schedules DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_cache DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_definitions DISABLE ROW LEVEL SECURITY;

-- Helpdesk
DROP POLICY IF EXISTS tenant_isolation ON canned_responses;
DROP POLICY IF EXISTS tenant_isolation ON tickets;
DROP POLICY IF EXISTS tenant_isolation ON ticket_queues;
DROP POLICY IF EXISTS tenant_isolation ON sla_policies;
ALTER TABLE canned_responses DISABLE ROW LEVEL SECURITY;
ALTER TABLE tickets DISABLE ROW LEVEL SECURITY;
ALTER TABLE ticket_queues DISABLE ROW LEVEL SECURITY;
ALTER TABLE sla_policies DISABLE ROW LEVEL SECURITY;

-- Wiki
DROP POLICY IF EXISTS tenant_isolation ON wiki_articles;
DROP POLICY IF EXISTS tenant_isolation ON wiki_categories;
ALTER TABLE wiki_articles DISABLE ROW LEVEL SECURITY;
ALTER TABLE wiki_categories DISABLE ROW LEVEL SECURITY;

-- Option-B Phase 2 integrations/chat (mig 115)
DROP POLICY IF EXISTS tenant_isolation ON guest_channel_config;
DROP POLICY IF EXISTS tenant_isolation ON guest_sessions;
DROP POLICY IF EXISTS tenant_isolation ON call_participants;
DROP POLICY IF EXISTS tenant_isolation ON call_sessions;
DROP POLICY IF EXISTS tenant_isolation ON message_mentions;
DROP POLICY IF EXISTS tenant_isolation ON message_reactions;
DROP POLICY IF EXISTS tenant_isolation ON chat_files;
DROP POLICY IF EXISTS tenant_isolation ON integration_link_tokens;
DROP POLICY IF EXISTS tenant_isolation ON integration_delivery_log;
DROP POLICY IF EXISTS tenant_isolation ON integration_account_links;
DROP POLICY IF EXISTS tenant_isolation ON integration_channel_mappings;
DROP POLICY IF EXISTS tenant_isolation ON datev_upload_log;
DROP POLICY IF EXISTS tenant_isolation ON datev_upload_configs;
DROP POLICY IF EXISTS tenant_isolation ON lexware_webhook_subscriptions;
DROP POLICY IF EXISTS tenant_isolation ON lexware_sync_log;
DROP POLICY IF EXISTS tenant_isolation ON lexware_field_mappings;
DROP POLICY IF EXISTS tenant_isolation ON lexware_entity_mappings;
DROP POLICY IF EXISTS tenant_isolation ON bexio_sync_log;
DROP POLICY IF EXISTS tenant_isolation ON bexio_field_mappings;
DROP POLICY IF EXISTS tenant_isolation ON bexio_entity_mappings;
DROP POLICY IF EXISTS tenant_isolation ON caldav_change_log;
DROP POLICY IF EXISTS tenant_isolation ON caldav_sync_versions;
ALTER TABLE guest_channel_config DISABLE ROW LEVEL SECURITY;
ALTER TABLE guest_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE call_participants DISABLE ROW LEVEL SECURITY;
ALTER TABLE call_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE message_mentions DISABLE ROW LEVEL SECURITY;
ALTER TABLE message_reactions DISABLE ROW LEVEL SECURITY;
ALTER TABLE chat_files DISABLE ROW LEVEL SECURITY;
ALTER TABLE integration_link_tokens DISABLE ROW LEVEL SECURITY;
ALTER TABLE integration_delivery_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE integration_account_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE integration_channel_mappings DISABLE ROW LEVEL SECURITY;
ALTER TABLE datev_upload_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE datev_upload_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE lexware_webhook_subscriptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE lexware_sync_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE lexware_field_mappings DISABLE ROW LEVEL SECURITY;
ALTER TABLE lexware_entity_mappings DISABLE ROW LEVEL SECURITY;
ALTER TABLE bexio_sync_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE bexio_field_mappings DISABLE ROW LEVEL SECURITY;
ALTER TABLE bexio_entity_mappings DISABLE ROW LEVEL SECURITY;
ALTER TABLE caldav_change_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE caldav_sync_versions DISABLE ROW LEVEL SECURITY;

-- Option-B Phase 2 settings/preferences (mig 114)
DROP POLICY IF EXISTS tenant_isolation ON caldav_push_subscriptions;
DROP POLICY IF EXISTS tenant_isolation ON gdpr_deletion_requests;
DROP POLICY IF EXISTS tenant_isolation ON recovery_codes;
DROP POLICY IF EXISTS tenant_isolation ON app_specific_passwords;
DROP POLICY IF EXISTS tenant_isolation ON wopi_locks;
DROP POLICY IF EXISTS tenant_isolation ON document_entity_links;
DROP POLICY IF EXISTS tenant_isolation ON document_file_tags;
DROP POLICY IF EXISTS tenant_isolation ON document_tags;
DROP POLICY IF EXISTS tenant_isolation ON document_shares;
DROP POLICY IF EXISTS tenant_isolation ON document_file_versions;
DROP POLICY IF EXISTS tenant_isolation ON document_folders;
DROP POLICY IF EXISTS tenant_isolation ON task_custom_field_values;
DROP POLICY IF EXISTS tenant_isolation ON task_entity_links;
DROP POLICY IF EXISTS tenant_isolation ON user_project_preferences;
DROP POLICY IF EXISTS tenant_isolation ON user_dashboard_layouts;
ALTER TABLE caldav_push_subscriptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE gdpr_deletion_requests DISABLE ROW LEVEL SECURITY;
ALTER TABLE recovery_codes DISABLE ROW LEVEL SECURITY;
ALTER TABLE app_specific_passwords DISABLE ROW LEVEL SECURITY;
ALTER TABLE wopi_locks DISABLE ROW LEVEL SECURITY;
ALTER TABLE document_entity_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE document_file_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE document_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE document_shares DISABLE ROW LEVEL SECURITY;
ALTER TABLE document_file_versions DISABLE ROW LEVEL SECURITY;
ALTER TABLE document_folders DISABLE ROW LEVEL SECURITY;
ALTER TABLE task_custom_field_values DISABLE ROW LEVEL SECURITY;
ALTER TABLE task_entity_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_project_preferences DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_dashboard_layouts DISABLE ROW LEVEL SECURITY;

-- Option-B Phase 2 batch 1 (mig 112)
DROP POLICY IF EXISTS tenant_isolation ON channel_memberships;
DROP POLICY IF EXISTS tenant_isolation ON automation_executions;
ALTER TABLE channel_memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE automation_executions DISABLE ROW LEVEL SECURITY;

-- Phase 1 retrofit (mig 106)
DROP POLICY IF EXISTS tenant_isolation ON document_files;
DROP POLICY IF EXISTS tenant_isolation ON custom_field_definitions;
DROP POLICY IF EXISTS tenant_isolation ON saved_filters;
DROP POLICY IF EXISTS tenant_isolation ON automations;
DROP POLICY IF EXISTS tenant_isolation ON time_entries;
DROP POLICY IF EXISTS tenant_isolation ON notifications;
DROP POLICY IF EXISTS tenant_isolation ON inbox_messages;
DROP POLICY IF EXISTS tenant_isolation ON email_messages;
DROP POLICY IF EXISTS tenant_isolation ON calendar_events;
ALTER TABLE document_files DISABLE ROW LEVEL SECURITY;
ALTER TABLE custom_field_definitions DISABLE ROW LEVEL SECURITY;
ALTER TABLE saved_filters DISABLE ROW LEVEL SECURITY;
ALTER TABLE automations DISABLE ROW LEVEL SECURITY;
ALTER TABLE time_entries DISABLE ROW LEVEL SECURITY;
ALTER TABLE notifications DISABLE ROW LEVEL SECURITY;
ALTER TABLE inbox_messages DISABLE ROW LEVEL SECURITY;
ALTER TABLE email_messages DISABLE ROW LEVEL SECURITY;
ALTER TABLE calendar_events DISABLE ROW LEVEL SECURITY;

COMMIT;

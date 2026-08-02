-- ============================================================================
-- The four CRM custom-field value tables hold real user data but carry neither
-- tenant_id nor RLS — verified against the local database on 2026-08-02 via
-- pg_class.relrowsecurity + pg_policies:
--   contact_custom_field_values   (contact_id, field_id, value, ...)
--   company_custom_field_values   (company_id, ...)
--   deal_custom_field_values      (deal_id, ...)
--   activity_custom_field_values  (activity_id, ...)
--
-- task_custom_field_values is deliberately absent: it already has its own
-- tenant_id and an active policy (re-checked, not assumed).
--
-- This is not cosmetic. Every read and write path against these tables filters
-- on the parent id alone — contact/postgres_repository.go:379 and :406,
-- company:197/:224, deal:264/:291, activity:283/:310 plus the two batch reads
-- and the merge-duplicates insert — with no tenant predicate anywhere. Anyone
-- holding a foreign contact UUID could read or overwrite that contact's custom
-- field values. A policy closes it once for all of those call sites.
--
-- Why enable_tenant_rls_via_join() and not the column-plus-backfill shape that
-- migration 000126 chose for ticket_messages: 000126's argument was query
-- volume (a plugin execution log scanned in bulk benefits from its own
-- tenant_id index). These tables are only ever reached through the parent id,
-- which is the leading column of their primary key, so the EXISTS subquery
-- resolves through the parent's primary key index — one lookup per row on a
-- set already narrowed to a single entity. Adding a nullable tenant_id and
-- promoting it to NOT NULL would instead require touching all four write paths
-- in Go, including the SELECT-based merge insert in
-- contact/postgres_repository.go:717, and any path missed there breaks writes
-- outright. Migration 000118 marks the join helper as the intended fallback for
-- exactly this case.
-- ============================================================================

CALL enable_tenant_rls_via_join('contact_custom_field_values', 'contacts', 'contact_id');
CALL enable_tenant_rls_via_join('company_custom_field_values', 'companies', 'company_id');
CALL enable_tenant_rls_via_join('deal_custom_field_values', 'deals', 'deal_id');
CALL enable_tenant_rls_via_join('activity_custom_field_values', 'activities', 'activity_id');

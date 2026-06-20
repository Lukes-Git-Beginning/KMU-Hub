-- Rigorosum Runde 3 (2026-06-20), P0-2: enable Row-Level-Security on tables that
-- were created WITHOUT it. Migration 000137 (advisory_protocols) and 000138
-- (settings foundation) added the tables with tenant_id NOT NULL + FK to
-- tenants(id) but never called enable_tenant_rls — leaving MiFID-II advisory
-- data and tenant configuration without a DB-layer isolation backstop.
--
-- All four tables carry their own tenant_id column, so the standard
-- enable_tenant_rls() procedure (mig 000118) applies the tenant_isolation
-- policy directly. Their services (crm → advisory_protocols, auth → settings)
-- both run the TenantInbound interceptor, so the app.tenant_id GUC is set on
-- every request — no phantom-404 risk.
--
-- NUMBERING NOTE: PR #13 (feat/partition-ephemeral-logs) also carries a
-- migration 000218. When that PR is merged it MUST be renumbered to 000219+
-- so main stays gapless and forward-only.

BEGIN;

CALL enable_tenant_rls('advisory_protocols');
CALL enable_tenant_rls('tenant_settings');
CALL enable_tenant_rls('tenant_module_leads');
CALL enable_tenant_rls('user_settings');

COMMIT;

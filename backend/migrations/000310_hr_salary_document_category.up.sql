-- Salary statement document category (hr-salary-document-category).
-- 2026-08-10
--
-- DECIDED by Luke on 2026-08-10: PDF upload over the existing document
-- infrastructure rather than a payroll data model. Cosmi does not compute
-- gross/net pay itself; DACH SMEs get the statement as a PDF out of
-- DATEV/Lexware and just need somewhere to hand it to the employee.
--
-- Seeded the same way as the 000046 system defaults and the four categories
-- 000294 added later: tenant_id is the zero-UUID placeholder, the
-- application layer copies system rows to each tenant on first access, and
-- the insert is idempotent against uq_hr_doc_category_key
-- (UNIQUE (tenant_id, key)).
--
-- visibility = 'employee' so the statement is deliverable to the employee it
-- belongs to (see hr_document_access, migration 000127/000128) without also
-- exposing it to every colleague — hr_only/manager tiers stay out of scope
-- for this category on purpose.

BEGIN;

INSERT INTO hr_document_categories (tenant_id, key, name, visibility, is_system, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000000', 'gehaltsabrechnung', 'Gehaltsabrechnung', 'employee', TRUE, 15)
ON CONFLICT (tenant_id, key) DO NOTHING;

COMMIT;

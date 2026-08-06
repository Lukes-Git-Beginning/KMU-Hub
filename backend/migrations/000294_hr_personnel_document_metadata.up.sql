-- Personalakte (hr-personnel-documents) — metadata columns on hr_employee_documents.
-- 2026-08-06
--
-- hr_employee_documents was built in migration 000046 as a pure link row
-- (employee_id + category_id + file_id) for the document service. The
-- Personalakte tab (desktop PersonnelDocuments.tsx) needs a document that
-- carries its own title, file name/size and an expiry date, and it uploads
-- metadata before any file exists — so file_id becomes optional.
--
-- RLS is untouched: policy hr_document_access (migration 000127, fixed in
-- 000128) keys off tenant_id + the category's visibility tier and neither
-- depends on the columns added here.

BEGIN;

ALTER TABLE hr_employee_documents
    ADD COLUMN IF NOT EXISTS title      VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_name  VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS file_size  VARCHAR(50)  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS expires_at DATE;

-- Metadata-only rows have no file yet. The uq_hr_doc_file unique constraint
-- stays valid: Postgres treats NULLs as distinct, so several file-less
-- documents per employee are allowed.
ALTER TABLE hr_employee_documents ALTER COLUMN file_id DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_hr_employee_documents_tenant
    ON hr_employee_documents(tenant_id, created_at DESC);

-- ============================================================================
-- Category keys the Personalakte UI offers
-- ============================================================================
-- The 000046 seed keys (arbeitsvertrag/zeugnisse/abmahnungen/sonstiges) predate
-- the UI, which offers vertrag/zeugnis/zertifikat/ausweis/sonstiges. Seed the
-- missing four as system categories rather than renaming the old ones — the
-- old keys stay reachable through /hr/employees/{id}/documents.
INSERT INTO hr_document_categories (tenant_id, key, name, visibility, is_system, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000000', 'vertrag',    'Arbeitsvertrag', 'hr_only',  TRUE, 11),
    ('00000000-0000-0000-0000-000000000000', 'zeugnis',    'Zeugnis',        'manager',  TRUE, 12),
    ('00000000-0000-0000-0000-000000000000', 'zertifikat', 'Zertifikat',     'employee', TRUE, 13),
    ('00000000-0000-0000-0000-000000000000', 'ausweis',    'Ausweis',        'hr_only',  TRUE, 14)
ON CONFLICT (tenant_id, key) DO NOTHING;

COMMIT;

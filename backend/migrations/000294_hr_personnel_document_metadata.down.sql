BEGIN;

DELETE FROM hr_document_categories
WHERE tenant_id = '00000000-0000-0000-0000-000000000000'
  AND key IN ('vertrag', 'zeugnis', 'zertifikat', 'ausweis');

DROP INDEX IF EXISTS idx_hr_employee_documents_tenant;

-- file_id can only go back to NOT NULL once the metadata-only rows are gone.
DELETE FROM hr_employee_documents WHERE file_id IS NULL;
ALTER TABLE hr_employee_documents ALTER COLUMN file_id SET NOT NULL;

ALTER TABLE hr_employee_documents
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS file_name,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS expires_at;

COMMIT;

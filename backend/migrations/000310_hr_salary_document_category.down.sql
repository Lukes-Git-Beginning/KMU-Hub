BEGIN;

DELETE FROM hr_document_categories
WHERE tenant_id = '00000000-0000-0000-0000-000000000000'
  AND key = 'gehaltsabrechnung';

COMMIT;

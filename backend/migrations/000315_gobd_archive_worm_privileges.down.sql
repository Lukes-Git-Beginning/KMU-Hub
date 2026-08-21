-- Restore the schema-wide grant from 000121 for the two archive tables.
BEGIN;

GRANT UPDATE, DELETE ON gobd_documents       TO kmuhub_app;
GRANT UPDATE, DELETE ON gobd_document_events TO kmuhub_app;

COMMIT;

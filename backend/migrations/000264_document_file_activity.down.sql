DROP TRIGGER IF EXISTS document_file_activity_append_only ON document_file_activity;
DROP FUNCTION IF EXISTS prevent_document_file_activity_mutation();
DROP TABLE IF EXISTS document_file_activity;

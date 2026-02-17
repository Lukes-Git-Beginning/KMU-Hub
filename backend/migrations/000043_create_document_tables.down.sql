-- Reverse of 000043_create_document_tables.up.sql
-- Drop tables in reverse dependency order

DROP TABLE IF EXISTS document_entity_links;
DROP TABLE IF EXISTS document_file_tags;
DROP TABLE IF EXISTS document_tags;
DROP TABLE IF EXISTS document_shares;
DROP TABLE IF EXISTS document_file_versions;

-- Drop trigger and function before dropping the table
DROP TRIGGER IF EXISTS trg_document_files_search_vector ON document_files;
DROP FUNCTION IF EXISTS document_files_search_vector_update();

DROP TABLE IF EXISTS document_files;
DROP TABLE IF EXISTS document_folders;

DROP TYPE IF EXISTS folder_space_type;

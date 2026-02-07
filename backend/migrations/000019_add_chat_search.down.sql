DROP TRIGGER IF EXISTS chat_files_search_update ON chat_files;
DROP FUNCTION IF EXISTS chat_files_search_vector_update();
DROP INDEX IF EXISTS idx_chat_files_search;
ALTER TABLE chat_files DROP COLUMN IF EXISTS search_vector;

DROP TRIGGER IF EXISTS messages_search_update ON messages;
DROP FUNCTION IF EXISTS messages_search_vector_update();
DROP INDEX IF EXISTS idx_messages_search;
ALTER TABLE messages DROP COLUMN IF EXISTS search_vector;
ALTER TABLE messages DROP COLUMN IF EXISTS lang;

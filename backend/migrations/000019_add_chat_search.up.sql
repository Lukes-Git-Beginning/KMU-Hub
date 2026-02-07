-- Messages: add language column + search_vector
ALTER TABLE messages ADD COLUMN lang TEXT NOT NULL DEFAULT 'german';
ALTER TABLE messages ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_messages_search ON messages USING GIN (search_vector);

-- Dynamic language trigger (reads lang column)
CREATE OR REPLACE FUNCTION messages_search_vector_update() RETURNS TRIGGER AS $$
DECLARE
    ts_config REGCONFIG;
BEGIN
    BEGIN
        ts_config := NEW.lang::REGCONFIG;
    EXCEPTION WHEN OTHERS THEN
        ts_config := 'simple'::REGCONFIG;
    END;
    NEW.search_vector := to_tsvector(ts_config, COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER messages_search_update
    BEFORE INSERT OR UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION messages_search_vector_update();

-- Backfill existing messages
UPDATE messages SET lang = 'german' WHERE is_deleted = FALSE;

-- Chat files: search on filename (always 'simple' config for filenames)
ALTER TABLE chat_files ADD COLUMN search_vector TSVECTOR;
CREATE INDEX idx_chat_files_search ON chat_files USING GIN (search_vector);

CREATE OR REPLACE FUNCTION chat_files_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := to_tsvector('simple', COALESCE(NEW.filename, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER chat_files_search_update
    BEFORE INSERT OR UPDATE ON chat_files
    FOR EACH ROW EXECUTE FUNCTION chat_files_search_vector_update();

-- Backfill existing files
UPDATE chat_files SET filename = filename WHERE is_deleted = FALSE;

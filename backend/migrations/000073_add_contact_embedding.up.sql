-- Add embedding support to contacts for semantic search.
-- search_text is a generated column (auto-maintained) combining name + company + email + notes.
-- embedding is populated asynchronously by a worker (not part of this migration).

ALTER TABLE contacts
    ADD COLUMN search_text TEXT GENERATED ALWAYS AS (
        COALESCE(first_name, '') || ' ' ||
        COALESCE(last_name, '') || ' ' ||
        COALESCE(email, '') || ' ' ||
        COALESCE(phone, '') || ' ' ||
        COALESCE(position, '') || ' ' ||
        COALESCE(notes, '')
    ) STORED,
    ADD COLUMN embedding vector(1536),
    ADD COLUMN embedding_updated_at TIMESTAMPTZ;

CREATE INDEX idx_contacts_embedding ON contacts
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

COMMENT ON COLUMN contacts.search_text IS
'Auto-generated concatenation of searchable contact fields (name, email, phone, position, notes).
 Source for embedding generation and also usable directly with pg_trgm / tsvector.';
COMMENT ON COLUMN contacts.embedding IS
'1536-dim semantic embedding of search_text. NULL until backfilled by embedding worker.
 Provider (OpenAI text-embedding-3-small vs EU-local) to be decided post-launch.';
COMMENT ON COLUMN contacts.embedding_updated_at IS
'Timestamp of last successful embedding generation. Used by the backfill worker to detect stale rows.';

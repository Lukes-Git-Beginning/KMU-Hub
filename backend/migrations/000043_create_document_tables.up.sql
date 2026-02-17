-- Document service tables

-- Enum for folder space types
CREATE TYPE folder_space_type AS ENUM ('personal', 'team', 'project');

-- Document folders (hierarchical via parent_id)
CREATE TABLE document_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES document_folders(id) ON DELETE CASCADE,
    space_type folder_space_type NOT NULL,
    space_id UUID NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    icon VARCHAR(50) NOT NULL DEFAULT 'folder',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_folders_parent ON document_folders(parent_id);
CREATE INDEX idx_document_folders_space ON document_folders(space_type, space_id);

-- Document files
CREATE TABLE document_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    folder_id UUID NOT NULL REFERENCES document_folders(id),
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    storage_key VARCHAR(512) NOT NULL,
    thumbnail_key VARCHAR(512),
    current_version INTEGER NOT NULL DEFAULT 1,
    owner_id UUID NOT NULL REFERENCES users(id),
    is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    content_text TEXT,
    search_vector TSVECTOR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_document_files_folder ON document_files(folder_id) WHERE NOT is_deleted;
CREATE INDEX idx_document_files_owner ON document_files(owner_id);
CREATE INDEX idx_document_files_search ON document_files USING GIN(search_vector);

-- Auto-update search_vector on insert/update
CREATE OR REPLACE FUNCTION document_files_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('german', COALESCE(NEW.filename, '') || ' ' || COALESCE(NEW.content_text, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_document_files_search_vector
    BEFORE INSERT OR UPDATE OF filename, content_text ON document_files
    FOR EACH ROW
    EXECUTE FUNCTION document_files_search_vector_update();

-- Document file versions
CREATE TABLE document_file_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    version_label VARCHAR(255),
    storage_key VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(file_id, version_number)
);

CREATE INDEX idx_document_versions_file ON document_file_versions(file_id);

-- Document shares (files and folders)
CREATE TABLE document_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('file', 'folder')),
    entity_id UUID NOT NULL,
    shared_with_user_id UUID NOT NULL REFERENCES users(id),
    permission VARCHAR(20) NOT NULL CHECK (permission IN ('read', 'write')),
    shared_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(entity_type, entity_id, shared_with_user_id)
);

CREATE INDEX idx_document_shares_entity ON document_shares(entity_type, entity_id);
CREATE INDEX idx_document_shares_user ON document_shares(shared_with_user_id);

-- Document tags
CREATE TABLE document_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(7),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Junction table: files <-> tags
CREATE TABLE document_file_tags (
    file_id UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES document_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- Document entity links (files linked to CRM entities)
CREATE TABLE document_entity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    linked_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(file_id, entity_type, entity_id)
);

CREATE INDEX idx_document_entity_links_file ON document_entity_links(file_id);
CREATE INDEX idx_document_entity_links_entity ON document_entity_links(entity_type, entity_id);

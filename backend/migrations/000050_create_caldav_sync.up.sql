-- CalDAV sync version tracking (per collection)
CREATE TABLE caldav_sync_versions (
    collection_type VARCHAR(20) NOT NULL,
    collection_id UUID NOT NULL,
    sync_version BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_type, collection_id)
);

-- CalDAV change log for incremental sync (RFC 6578)
CREATE TABLE caldav_change_log (
    id BIGSERIAL PRIMARY KEY,
    collection_type VARCHAR(20) NOT NULL,
    collection_id UUID NOT NULL,
    object_path TEXT NOT NULL,
    change_type VARCHAR(10) NOT NULL CHECK (change_type IN ('created', 'modified', 'deleted')),
    sync_version BIGINT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup: changes since a given sync version for a collection
CREATE INDEX idx_caldav_changelog_collection
    ON caldav_change_log(collection_type, collection_id, sync_version);

-- Global CalDAV/CardDAV org settings
CREATE TABLE caldav_settings (
    key VARCHAR(50) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default: CalDAV/CardDAV disabled by default
INSERT INTO caldav_settings (key, value) VALUES ('caldav_carddav_enabled', 'false');

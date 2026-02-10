ALTER TABLE event_exceptions DROP CONSTRAINT IF EXISTS fk_event_exceptions_resource;
ALTER TABLE calendar_events DROP CONSTRAINT IF EXISTS fk_events_resource;
DROP TABLE IF EXISTS resource_bookings;
DROP TABLE IF EXISTS resource_tags;
DROP TABLE IF EXISTS resources;
DROP EXTENSION IF EXISTS btree_gist;

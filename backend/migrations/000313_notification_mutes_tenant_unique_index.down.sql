DROP INDEX idx_notification_mutes_resource;

ALTER TABLE notification_mutes
    ADD CONSTRAINT notification_mutes_user_id_module_id_resource_id_key
    UNIQUE (user_id, module_id, resource_id);

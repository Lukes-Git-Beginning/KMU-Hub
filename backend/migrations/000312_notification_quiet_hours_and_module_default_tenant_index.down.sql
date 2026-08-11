DROP INDEX idx_notification_preferences_module_default;
CREATE UNIQUE INDEX idx_notification_preferences_module_default
    ON notification_preferences(user_id, module_id)
    WHERE event_type_key IS NULL AND module_id IS NOT NULL;

DROP INDEX idx_notification_quiet_hours_user;

ALTER TABLE notification_quiet_hours
    ADD CONSTRAINT notification_quiet_hours_user_id_key UNIQUE (user_id);

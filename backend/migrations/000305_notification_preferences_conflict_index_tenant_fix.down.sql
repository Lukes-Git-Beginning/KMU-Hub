DROP INDEX idx_notification_preferences_event_type;
CREATE UNIQUE INDEX idx_notification_preferences_event_type
    ON notification_preferences(user_id, event_type_key)
    WHERE event_type_key IS NOT NULL;

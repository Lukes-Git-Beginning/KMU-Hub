-- Email and SMS delivery channel toggles, alongside the existing in_app/desktop_push
-- pair. Actual email/SMS delivery is not wired to the dispatcher yet -- these columns
-- only carry the user's choice so a future delivery callback has something to read.
ALTER TABLE notification_preferences ADD COLUMN email BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE notification_preferences ADD COLUMN sms BOOLEAN NOT NULL DEFAULT false;

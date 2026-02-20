DROP TABLE IF EXISTS app_specific_passwords;
ALTER TABLE users DROP COLUMN IF EXISTS caldav_enabled;

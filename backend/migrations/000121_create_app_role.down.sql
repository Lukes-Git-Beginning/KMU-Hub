-- Rollback for 000121: drop kmuhub_app role and its privileges.
-- Reassigns owned objects back to kmuhub before drop so the role can be
-- removed cleanly. Reverses the default privileges as well.

BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kmuhub_app') THEN
        -- Reverse default privileges
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM kmuhub_app;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM kmuhub_app;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM kmuhub_app;

        -- Revoke explicit privileges
        REVOKE ALL ON ALL TABLES IN SCHEMA public FROM kmuhub_app;
        REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM kmuhub_app;
        REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM kmuhub_app;
        REVOKE USAGE ON SCHEMA public FROM kmuhub_app;
        EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM kmuhub_app', current_database());

        -- Drop the role
        DROP ROLE kmuhub_app;
    END IF;
END$$;

COMMIT;

-- Sprint 4 Welle 2 — separate application role for RLS enforcement.
-- 2026-05-11
--
-- The default POSTGRES_USER (kmuhub) is created as a Postgres SUPERUSER by
-- the official postgres image. Superusers bypass row-level security
-- regardless of FORCE ROW LEVEL SECURITY or BYPASSRLS, so the policies
-- introduced in 000120 are effectively no-ops as long as the application
-- connects with that role.
--
-- This migration introduces `kmuhub_app` as a NOSUPERUSER NOBYPASSRLS role
-- and grants it the standard set of CRUD permissions on the public schema.
-- The application's DATABASE_URL must be flipped to kmuhub_app for RLS to
-- actually enforce — done as part of the Welle 2e production deploy
-- (see plan file).
--
-- The migration runner (golang-migrate) keeps using kmuhub-Superuser so
-- future DDL migrations are not blocked.
--
-- Idempotent: safe to re-run, all DO blocks check for prior existence.

BEGIN;

-- 1) Create the role if it does not already exist. Password is a placeholder
--    set by the deploy script (KMUHUB_APP_DB_PASSWORD env). Lokal-Dev nutzt
--    'app_dev' (siehe .env.example).
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kmuhub_app') THEN
        CREATE ROLE kmuhub_app LOGIN
            NOSUPERUSER
            NOBYPASSRLS
            NOCREATEDB
            NOCREATEROLE
            PASSWORD 'change-me-via-alter-role';
    ELSE
        ALTER ROLE kmuhub_app NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
    END IF;
END$$;

-- 2) Database-level connect privilege.
GRANT CONNECT ON DATABASE kmuhub TO kmuhub_app;

-- 3) Schema usage and existing object privileges.
GRANT USAGE ON SCHEMA public TO kmuhub_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO kmuhub_app;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO kmuhub_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO kmuhub_app;

-- 4) Default privileges for objects created by kmuhub in the future
--    (without this every new migration would need a manual GRANT).
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kmuhub_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO kmuhub_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO kmuhub_app;

-- 5) Allow kmuhub_app to set the RLS session GUCs (they are already
--    declared at the database level in 000118, so any role can SET them).
--    No additional grant needed.

COMMIT;

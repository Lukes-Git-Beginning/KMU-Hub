-- Dedicated read-only database role for AI agents accessing the schema via MCP.
-- Separate from application roles to enforce least-privilege on agent-driven queries.
-- Password is set out-of-band (deploy script / secrets manager), not in this migration.

-- Use DO block to make CREATE ROLE idempotent across environments.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cosmi_agent_readonly') THEN
        CREATE ROLE cosmi_agent_readonly WITH LOGIN PASSWORD NULL;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO cosmi_agent_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO cosmi_agent_readonly;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO cosmi_agent_readonly;

-- Future tables get SELECT automatically.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO cosmi_agent_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON SEQUENCES TO cosmi_agent_readonly;

COMMENT ON ROLE cosmi_agent_readonly IS
'Read-only role for AI agents accessing the DB via MCP servers. Password set out-of-band.
 No write grants. Future tables automatically inherit SELECT via default privileges.';

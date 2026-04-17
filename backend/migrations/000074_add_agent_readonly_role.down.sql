-- Reverse 000074: revoke grants and drop the role.

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT ON TABLES FROM cosmi_agent_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT ON SEQUENCES FROM cosmi_agent_readonly;

REVOKE SELECT ON ALL TABLES IN SCHEMA public FROM cosmi_agent_readonly;
REVOKE SELECT ON ALL SEQUENCES IN SCHEMA public FROM cosmi_agent_readonly;
REVOKE USAGE ON SCHEMA public FROM cosmi_agent_readonly;

DROP ROLE IF EXISTS cosmi_agent_readonly;

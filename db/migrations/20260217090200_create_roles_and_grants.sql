-- migrate:up
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'owner') THEN
        CREATE ROLE owner WITH LOGIN SUPERUSER;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'agent') THEN
        CREATE ROLE agent WITH LOGIN NOSUPERUSER;
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO agent;

REVOKE ALL ON TABLE identity FROM agent;
REVOKE ALL ON TABLE self FROM agent;
REVOKE ALL ON TABLE memories FROM agent;
REVOKE ALL ON TABLE tasks FROM agent;
REVOKE ALL ON TABLE task_status FROM agent;
REVOKE ALL ON TABLE system_credentials FROM agent;
REVOKE ALL ON TABLE personal_credentials FROM agent;
REVOKE ALL ON TABLE log FROM agent;

GRANT SELECT ON TABLE identity TO agent;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE self TO agent;
GRANT SELECT, INSERT ON TABLE memories TO agent;
GRANT SELECT, INSERT ON TABLE tasks TO agent;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE task_status TO agent;
GRANT SELECT ON TABLE system_credentials TO agent;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE personal_credentials TO agent;
GRANT INSERT ON TABLE log TO agent;

-- migrate:down
REVOKE ALL ON TABLE identity FROM agent;
REVOKE ALL ON TABLE self FROM agent;
REVOKE ALL ON TABLE memories FROM agent;
REVOKE ALL ON TABLE tasks FROM agent;
REVOKE ALL ON TABLE task_status FROM agent;
REVOKE ALL ON TABLE system_credentials FROM agent;
REVOKE ALL ON TABLE personal_credentials FROM agent;
REVOKE ALL ON TABLE log FROM agent;
REVOKE USAGE ON SCHEMA public FROM agent;

DROP ROLE IF EXISTS agent;
DROP ROLE IF EXISTS owner;

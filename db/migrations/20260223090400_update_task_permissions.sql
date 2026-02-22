-- migrate:up
REVOKE ALL ON TABLE tasks FROM agent;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE tasks TO agent;

-- migrate:down
REVOKE ALL ON TABLE tasks FROM agent;
GRANT SELECT, INSERT ON TABLE tasks TO agent;

-- migrate:up
REVOKE ALL ON TABLE conversation_messages FROM agent;
REVOKE ALL ON TABLE conversations FROM agent;
GRANT SELECT ON TABLE conversations TO agent;
GRANT SELECT ON TABLE conversation_messages TO agent;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE conversations TO owner;
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE conversation_messages TO owner;

-- migrate:down
REVOKE ALL ON TABLE conversation_messages FROM agent;
REVOKE ALL ON TABLE conversations FROM agent;
REVOKE ALL ON TABLE conversation_messages FROM owner;
REVOKE ALL ON TABLE conversations FROM owner;

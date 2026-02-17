-- migrate:up
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER self_set_updated_at
BEFORE UPDATE ON self
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER task_status_set_updated_at
BEFORE UPDATE ON task_status
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER system_credentials_set_updated_at
BEFORE UPDATE ON system_credentials
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER personal_credentials_set_updated_at
BEFORE UPDATE ON personal_credentials
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- migrate:down
DROP TRIGGER IF EXISTS personal_credentials_set_updated_at ON personal_credentials;
DROP TRIGGER IF EXISTS system_credentials_set_updated_at ON system_credentials;
DROP TRIGGER IF EXISTS task_status_set_updated_at ON task_status;
DROP TRIGGER IF EXISTS self_set_updated_at ON self;

DROP FUNCTION IF EXISTS set_updated_at();

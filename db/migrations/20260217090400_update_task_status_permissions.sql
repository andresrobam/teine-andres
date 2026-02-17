-- migrate:up
REVOKE DELETE ON TABLE task_status FROM agent;
REVOKE UPDATE ON TABLE task_status FROM agent;
GRANT UPDATE (status, progress, updated_at) ON TABLE task_status TO agent;

CREATE OR REPLACE FUNCTION prevent_task_status_task_id_update() RETURNS trigger AS $$
BEGIN
    IF NEW.task_id <> OLD.task_id THEN
        RAISE EXCEPTION 'task_status.task_id is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER task_status_prevent_task_id_update
BEFORE UPDATE ON task_status
FOR EACH ROW EXECUTE FUNCTION prevent_task_status_task_id_update();

-- migrate:down
DROP TRIGGER IF EXISTS task_status_prevent_task_id_update ON task_status;
DROP FUNCTION IF EXISTS prevent_task_status_task_id_update();

REVOKE UPDATE (status, progress, updated_at) ON TABLE task_status FROM agent;
GRANT UPDATE, DELETE ON TABLE task_status TO agent;

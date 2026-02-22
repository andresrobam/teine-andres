-- migrate:up
ALTER TABLE tasks ADD COLUMN status task_status_enum NOT NULL DEFAULT 'pending';
ALTER TABLE tasks ADD COLUMN progress jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE tasks ADD COLUMN status_updated_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE tasks ADD COLUMN postponed_until timestamptz NULL;

UPDATE tasks t
SET status = ts.status,
    progress = ts.progress,
    status_updated_at = ts.updated_at,
    postponed_until = ts.postponed_until
FROM task_status ts
WHERE t.id = ts.task_id;

UPDATE tasks t
SET status = 'pending',
    progress = '{}'::jsonb,
    status_updated_at = t.created_at,
    postponed_until = NULL
WHERE NOT EXISTS (SELECT 1 FROM task_status ts WHERE ts.task_id = t.id);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_postponed_until ON tasks(postponed_until) WHERE postponed_until IS NOT NULL;

CREATE OR REPLACE FUNCTION tasks_set_status_updated_at() RETURNS trigger AS $$
BEGIN
    IF NEW.status IS DISTINCT FROM OLD.status
        OR NEW.progress IS DISTINCT FROM OLD.progress
        OR NEW.postponed_until IS DISTINCT FROM OLD.postponed_until THEN
        NEW.status_updated_at = now();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_set_status_updated_at
BEFORE UPDATE ON tasks
FOR EACH ROW EXECUTE FUNCTION tasks_set_status_updated_at();

DROP TABLE IF EXISTS task_status;

-- migrate:down
CREATE TABLE task_status (
    task_id uuid PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
    status task_status_enum NOT NULL DEFAULT 'pending',
    progress jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now(),
    postponed_until timestamptz NULL
);

CREATE INDEX idx_task_status_status ON task_status(status);
CREATE INDEX idx_task_status_postponed_until ON task_status(postponed_until) WHERE postponed_until IS NOT NULL;

INSERT INTO task_status (task_id, status, progress, updated_at, postponed_until)
SELECT id, status, progress, status_updated_at, postponed_until
FROM tasks;

DROP TRIGGER IF EXISTS tasks_set_status_updated_at ON tasks;
DROP FUNCTION IF EXISTS tasks_set_status_updated_at();

ALTER TABLE tasks DROP COLUMN postponed_until;
ALTER TABLE tasks DROP COLUMN status_updated_at;
ALTER TABLE tasks DROP COLUMN progress;
ALTER TABLE tasks DROP COLUMN status;

CREATE TRIGGER task_status_set_updated_at
BEFORE UPDATE ON task_status
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

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

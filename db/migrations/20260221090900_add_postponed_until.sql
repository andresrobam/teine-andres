-- migrate:up
ALTER TABLE task_status ADD COLUMN postponed_until timestamptz NULL;
GRANT UPDATE (postponed_until) ON TABLE task_status TO agent;

CREATE INDEX idx_task_status_postponed_until ON task_status(postponed_until) WHERE postponed_until IS NOT NULL;
-- migrate:down
ALTER TABLE task_status DROP COLUMN postponed_until

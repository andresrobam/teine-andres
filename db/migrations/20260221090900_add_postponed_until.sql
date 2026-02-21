-- Add postponed_until column to task_status table
-- This allows tasks to be postponed until a specific time
-- The main loop will filter out postponed tasks until their time arrives

ALTER TABLE task_status ADD COLUMN postponed_until timestamptz NULL;

GRANT UPDATE (postponed_until) ON TABLE task_status TO agent;

-- Create index for efficient filtering of postponed tasks
CREATE INDEX idx_task_status_postponed_until ON task_status(postponed_until) WHERE postponed_until IS NOT NULL;

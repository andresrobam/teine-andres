-- migrate:up
DROP INDEX IF EXISTS idx_log_created_at;
DROP TABLE IF EXISTS log;

-- migrate:down
CREATE TABLE log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_log_created_at ON log(created_at);

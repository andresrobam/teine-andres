-- migrate:up
CREATE TABLE matrix_sync_state (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    next_batch text,
    updated_at timestamptz NOT NULL DEFAULT now()
);

REVOKE ALL ON TABLE matrix_sync_state FROM agent;

-- Seed with initial row
INSERT INTO matrix_sync_state (next_batch)
VALUES (NULL);

-- migrate:down
DROP TABLE IF EXISTS matrix_sync_state;

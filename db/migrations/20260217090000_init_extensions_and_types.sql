-- migrate:up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'task_status_enum') THEN
        CREATE TYPE task_status_enum AS ENUM (
            'pending',
            'in_progress',
            'blocked',
            'done',
            'cancelled'
        );
    END IF;
END $$;

-- migrate:down
DROP TYPE IF EXISTS task_status_enum;

-- migrate:up
CREATE TABLE identity (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE,
    prompt text NOT NULL
);

CREATE TABLE self (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title text NOT NULL,
    content text NOT NULL,
    tags text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    summary text NOT NULL,
    details text NOT NULL,
    tags text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_task_id uuid REFERENCES tasks(id) ON DELETE SET NULL,
    title text NOT NULL,
    description text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE task_status (
    task_id uuid PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
    status task_status_enum NOT NULL DEFAULT 'pending',
    progress jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE system_credentials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE,
    secret text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE personal_credentials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE,
    secret text NOT NULL,
    source text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_parent_task_id ON tasks(parent_task_id);
CREATE INDEX idx_task_status_status ON task_status(status);
CREATE INDEX idx_memories_created_at ON memories(created_at);
CREATE INDEX idx_log_created_at ON log(created_at);

-- migrate:down
DROP TABLE IF EXISTS log;
DROP TABLE IF EXISTS personal_credentials;
DROP TABLE IF EXISTS system_credentials;
DROP TABLE IF EXISTS task_status;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS memories;
DROP TABLE IF EXISTS self;
DROP TABLE IF EXISTS identity;

-- migrate:up
CREATE TABLE conversations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    model text NOT NULL,
    endpoint text NOT NULL,
    finish_reason text,
    error text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE conversation_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    seq int NOT NULL,
    role text NOT NULL,
    content text,
    reasoning text,
    tool_calls jsonb,
    tool_call_id text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_started_at ON conversations(started_at);
CREATE INDEX idx_conversation_messages_conversation_id ON conversation_messages(conversation_id);
CREATE INDEX idx_conversation_messages_seq ON conversation_messages(seq);

-- migrate:down
DROP INDEX IF EXISTS idx_conversation_messages_seq;
DROP INDEX IF EXISTS idx_conversation_messages_conversation_id;
DROP INDEX IF EXISTS idx_conversations_started_at;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;

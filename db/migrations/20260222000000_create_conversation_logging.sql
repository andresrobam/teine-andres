-- migrate:up
-- Table for conversation sessions
CREATE TABLE conversations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    finish_reason text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb
);

-- Table for chat messages (both requests and responses)
CREATE TABLE chat_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role text NOT NULL, -- 'system', 'user', 'assistant', 'tool'
    content text,
    finish_reason text, -- for assistant messages
    model text, -- which model generated this response
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Table for tool calls
CREATE TABLE tool_calls (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id uuid NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    tool_call_id text NOT NULL, -- the ID from OpenRouter
    tool_name text NOT NULL,
    arguments jsonb NOT NULL,
    response jsonb,
    error text,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);

-- Table for reasoning tokens
CREATE TABLE reasoning_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id uuid NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    reasoning_content text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Indexes for efficient querying
CREATE INDEX idx_chat_messages_conversation_id ON chat_messages(conversation_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at);
CREATE INDEX idx_tool_calls_conversation_id ON tool_calls(conversation_id);
CREATE INDEX idx_tool_calls_message_id ON tool_calls(message_id);
CREATE INDEX idx_reasoning_logs_conversation_id ON reasoning_logs(conversation_id);
CREATE INDEX idx_reasoning_logs_message_id ON reasoning_logs(message_id);

-- Grants for agent role
GRANT SELECT, INSERT ON TABLE conversations TO agent;
GRANT SELECT, INSERT ON TABLE chat_messages TO agent;
GRANT SELECT, INSERT, UPDATE ON TABLE tool_calls TO agent;
GRANT SELECT, INSERT ON TABLE reasoning_logs TO agent;

-- migrate:down
REVOKE ALL ON TABLE reasoning_logs FROM agent;
REVOKE ALL ON TABLE tool_calls FROM agent;
REVOKE ALL ON TABLE chat_messages FROM agent;
REVOKE ALL ON TABLE conversations FROM agent;

DROP TABLE IF EXISTS reasoning_logs;
DROP TABLE IF EXISTS tool_calls;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS conversations;

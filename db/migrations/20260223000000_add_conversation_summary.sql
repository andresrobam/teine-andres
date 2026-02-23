-- migrate:up
-- Add summary column to conversations table
ALTER TABLE conversations ADD COLUMN summary text;

-- Grant update permission to agent role for summary updates
GRANT UPDATE (summary) ON TABLE conversations TO agent;

-- migrate:down
REVOKE UPDATE (summary) ON TABLE conversations FROM agent;
ALTER TABLE conversations DROP COLUMN IF EXISTS summary;

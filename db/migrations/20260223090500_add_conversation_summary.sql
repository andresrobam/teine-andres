-- migrate:up
ALTER TABLE conversations
    ADD COLUMN summary text;

-- migrate:down
ALTER TABLE conversations
    DROP COLUMN IF EXISTS summary;
